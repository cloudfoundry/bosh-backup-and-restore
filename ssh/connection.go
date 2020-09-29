package ssh

import (
	"bytes"
	"context"
	"io"
	"sync"

	"time"

	"strings"

	"log"
	"net"
	"os"

	boshhttp "github.com/cloudfoundry/bosh-utils/httpclient"
	proxy "github.com/cloudfoundry/socks5-proxy"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

//go:generate counterfeiter -o fakes/fake_ssh_connection.go . SSHConnection
type SSHConnection interface {
	Stream(cmd string, writer io.Writer) ([]byte, int, error)
	StreamStdin(cmd string, reader io.Reader) ([]byte, []byte, int, error)
	Run(cmd string) ([]byte, []byte, int, error)
	Username() string
}

//go:generate counterfeiter -o fakes/fake_logger.go . Logger
type Logger interface {
	Warn(tag, msg string, args ...interface{})
	Debug(tag, msg string, args ...interface{})
	Error(tag, msg string, args ...interface{})
}

var dialFunc boshhttp.DialContextFunc
var dialFuncMutex sync.RWMutex

func NewConnection(hostName, userName, privateKey string, publicKeyCallback ssh.HostKeyCallback, publicKeyAlgorithm []string, logger Logger) (SSHConnection, error) {
	return NewConnectionWithServerAliveInterval(hostName, userName, privateKey, publicKeyCallback, publicKeyAlgorithm, 60, logger)
}

func NewConnectionWithServerAliveInterval(hostName, userName, privateKey string, publicKeyCallback ssh.HostKeyCallback, publicKeyAlgorithm []string, serverAliveInterval time.Duration, logger Logger) (SSHConnection, error) {
	parsedPrivateKey, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, errors.Wrap(err, "ssh.NewConnection.ParsePrivateKey failed")
	}

	conn := Connection{
		host: defaultToSSHPort(hostName),
		sshConfig: &ssh.ClientConfig{
			User: userName,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(parsedPrivateKey),
			},
			HostKeyCallback:   publicKeyCallback,
			HostKeyAlgorithms: publicKeyAlgorithm,
		},
		logger:              logger,
		serverAliveInterval: serverAliveInterval,
		dialFunc:            createDialContextFunc(),
	}

	return conn, nil
}

type Connection struct {
	host                string
	sshConfig           *ssh.ClientConfig
	logger              Logger
	serverAliveInterval time.Duration
	dialFunc            boshhttp.DialContextFunc
}

func (c Connection) Run(cmd string) (stdout, stderr []byte, exitCode int, err error) {
	stdoutBuffer := bytes.NewBuffer([]byte{})

	stderr, exitCode, err = c.Stream(cmd, stdoutBuffer)

	return stdoutBuffer.Bytes(), stderr, exitCode, errors.Wrap(err, "ssh.Run failed")
}

func (c Connection) Stream(cmd string, stdoutWriter io.Writer) (stderr []byte, exitCode int, err error) {
	errBuffer := bytes.NewBuffer([]byte{})

	exitCode, err = c.runInSession(cmd, stdoutWriter, errBuffer, nil)

	return errBuffer.Bytes(), exitCode, errors.Wrap(err, "ssh.Stream failed")
}

func (c Connection) StreamStdin(cmd string, stdinReader io.Reader) (stdout, stderr []byte, exitCode int, err error) {
	stdoutBuffer := bytes.NewBuffer([]byte{})
	stderrBuffer := bytes.NewBuffer([]byte{})

	exitCode, err = c.runInSession(cmd, stdoutBuffer, stderrBuffer, stdinReader)

	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), exitCode, errors.Wrap(err, "ssh.StreamStdin failed")
}

type sessionClosingOnErrorWriter struct {
	endGameWriter io.Writer
	sshSession    SSHSession
	writerError   error
}

func (w *sessionClosingOnErrorWriter) Write(data []byte) (int, error) {
	n, err := w.endGameWriter.Write(data)
	if err != nil {
		w.writerError = err
		w.sshSession.Close()
	}
	return n, err
}

func (c Connection) newClient() (*ssh.Client, error) {
	conn, err := c.dialFunc(context.Background(), "tcp", c.host)
	if err != nil {
		return nil, err
	}

	client, chans, reqs, err := ssh.NewClientConn(conn, c.host, c.sshConfig)
	if err != nil {
		return nil, err
	}

	return ssh.NewClient(client, chans, reqs), nil
}

func createDialContextFunc() boshhttp.DialContextFunc {
	dialFuncMutex.RLock()
	haveDialer := dialFunc != nil
	dialFuncMutex.RUnlock()

	if haveDialer {
		return dialFunc
	}

	dialFuncMutex.Lock()
	defer dialFuncMutex.Unlock()

	socksProxy := proxy.NewSocks5Proxy(proxy.NewHostKey(), log.New(os.Stdout, "sock5-proxy", log.LstdFlags), 60*time.Second)
	dialFunc = boshhttp.SOCKS5DialContextFuncFromEnvironment(&net.Dialer{}, socksProxy)
	return dialFunc
}

//go:generate counterfeiter -o fakes/fake_ssh_session.go . SSHSession
type SSHSession interface {
	Run(cmd string) error
	SendRequest(name string, wantReply bool, payload []byte) (bool, error)
	Close() error
}

type SSHSessionBuilder = func(client *ssh.Client, stdin io.Reader, stdout, stderr io.Writer) (SSHSession, error)

func buildSSHSessionImpl(client *ssh.Client, stdin io.Reader, stdout, stderr io.Writer) (SSHSession, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, errors.Wrap(err, "ssh.NewSession failed")
	}

	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr

	return session, nil
}

var buildSSHSession = buildSSHSessionImpl

func (c Connection) runInSession(cmd string, stdout, stderr io.Writer, stdin io.Reader) (int, error) {
	client, err := c.newClient()
	if err != nil {
		return -1, errors.Wrap(err, "ssh.Dial failed")
	}
	defer client.Close()

	stdoutWrappingWriter := &sessionClosingOnErrorWriter{endGameWriter: stdout, sshSession: nil}
	session, err := buildSSHSession(client, stdin, stdoutWrappingWriter, stderr)
	if err != nil {
		return -1, err
	}
	stdoutWrappingWriter.sshSession = session

	c.logger.Debug("bbr", "Trying to execute '%s' on remote", cmd)

	stopKeepAliveLoop := c.startKeepAliveLoop(session)
	defer close(stopKeepAliveLoop)

	err = session.Run(cmd)

	if stdoutWrappingWriter.writerError != nil {
		return -1, errors.Wrap(stdoutWrappingWriter.writerError, "stdout.Write failed")
	}

	if err != nil {
		switch err := err.(type) {
		case *ssh.ExitError:
			return err.ExitStatus(), nil
		case *ssh.ExitMissingError:
			c.logger.Error("bbr", "Did the network just fail? It looks like my ssh session ended suddenly without getting an exit status from the remote VM.")
			return -1, errors.Wrap(err, "ssh session ended before returning an exit code")
		default:
			return -1, errors.Wrap(err, "ssh.Session.Run failed")
		}
	}

	return 0, nil
}

func (c Connection) startKeepAliveLoop(session SSHSession) chan struct{} {
	terminate := make(chan struct{})
	go func() {
		for {
			select {
			case <-terminate:
				return
			default:
				_, err := session.SendRequest("keepalive@bbr", true, nil)
				if err != nil {
					c.logger.Debug("ssh", "keepalive failed: %+v", err)
				}
				time.Sleep(time.Second * c.serverAliveInterval)
			}
		}
	}()
	return terminate
}

func (c Connection) Username() string {
	return c.sshConfig.User
}

func defaultToSSHPort(host string) string {
	parts := strings.Split(host, ":")
	if len(parts) == 2 {
		return host
	} else {
		return host + ":22"
	}
}
