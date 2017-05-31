package ssh

import (
	"bytes"
	"io"

	"time"

	"strings"

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

type Logger interface {
	Warn(tag, msg string, args ...interface{})
}

func NewConnection(hostName, userName, privateKey string, logger Logger) (SSHConnection, error) {
	return NewConnectionWithServerAliveInterval(hostName, userName, privateKey, 60, logger)
}

func NewConnectionWithServerAliveInterval(hostName, userName, privateKey string, serverAliveInterval time.Duration, logger Logger) (SSHConnection, error) {
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
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
		logger:              logger,
		serverAliveInterval: serverAliveInterval,
	}

	return conn, nil
}

type Connection struct {
	host                string
	sshConfig           *ssh.ClientConfig
	logger              Logger
	serverAliveInterval time.Duration
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

func (c Connection) runInSession(cmd string, stdout, stderr io.Writer, stdin io.Reader) (int, error) {
	connection, err := ssh.Dial("tcp", c.host, c.sshConfig)
	if err != nil {
		return -1, errors.Wrap(err, "ssh.Dial failed")
	}
	defer connection.Close()

	session, err := connection.NewSession()
	if err != nil {
		return -1, errors.Wrap(err, "ssh.NewSession failed")
	}

	stopKeepAliveLoop := c.startKeepAliveLoop(session)
	defer close(stopKeepAliveLoop)

	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr

	var exitCode int

	err = session.Run(cmd)
	if err == nil {
		exitCode = 0
	} else {
		switch err := err.(type) {
		case *ssh.ExitError:
			exitCode = err.ExitStatus()
		default:
			return -1, errors.Wrap(err, "ssh.Session.Run failed")
		}
	}
	return exitCode, nil
}

func (c Connection) startKeepAliveLoop(session *ssh.Session) chan struct{} {
	terminate := make(chan struct{})
	go func() {
		for {
			select {
			case <-terminate:
				return
			default:
				_, err := session.SendRequest("keepalive@bbr", true, nil)
				if err != nil {
					c.logger.Warn("ssh", "keepalive failed: %+v", err)
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
