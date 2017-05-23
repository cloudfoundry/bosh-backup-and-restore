package ssh

import (
	"bytes"
	"io"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

//go:generate counterfeiter -o fakes/fake_ssh_connection.go . SSHConnection
type SSHConnection interface {
	Stream(cmd string, writer io.Writer) ([]byte, int, error)
	StreamStdin(cmd string, reader io.Reader) ([]byte, []byte, int, error)
	Run(cmd string) ([]byte, []byte, int, error)
	Cleanup() error
	Username() string
}

func ConnectionCreator(hostName, userName, privateKey string) (SSHConnection, error) {
	parsedPrivateKey, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, errors.Wrap(err, "ssh.ConnectionCreator.ParsePrivateKey failed")
	}

	conn := Connection{
		host: hostName,
		sshConfig: &ssh.ClientConfig{
			User: userName,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(parsedPrivateKey),
			},
		},
	}

	return conn, nil
}

type Connection struct {
	host      string
	sshConfig *ssh.ClientConfig
}

func (c Connection) Run(cmd string) ([]byte, []byte, int, error) {
	outBuffer := bytes.NewBuffer([]byte{})
	errBuffer, exitCode, err := c.Stream(cmd, outBuffer)
	return outBuffer.Bytes(), errBuffer, exitCode, errors.Wrap(err, "ssh.Run.Stream failed")
}

func (c Connection) Stream(cmd string, writer io.Writer) ([]byte, int, error) {
	errBuffer := bytes.NewBuffer([]byte{})

	exitCode, err := c.runInSession(cmd, writer, errBuffer, nil)

	return errBuffer.Bytes(), exitCode, err
}

func (c Connection) StreamStdin(cmd string, stdinReader io.Reader) (stdout, stderr []byte, exitCode int, err error) {
	outBuffer := bytes.NewBuffer([]byte{})
	errBuffer := bytes.NewBuffer([]byte{})

	exitCode, err = c.runInSession(cmd, outBuffer, errBuffer, stdinReader)

	return outBuffer.Bytes(), errBuffer.Bytes(), exitCode, err
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

	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr

	var exitCode int

	err = session.Run(cmd)
	if err == nil {
		exitCode = 0
	} else {
		exitErr, yes := err.(*ssh.ExitError)
		if yes {
			exitCode = exitErr.ExitStatus()
		} else {
			return -1, errors.Wrap(err, "ssh.Session.Run failed")
		}
	}
	return exitCode, nil
}

func (c Connection) Username() string {
	return c.sshConfig.User
}

func (c Connection) Cleanup() error {
	return nil
}
