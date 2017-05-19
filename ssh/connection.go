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
	conn := Connection{
		host: hostName,
		user: userName,
	}

	parsedPrivateKey, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, errors.Wrap(err, "ssh.ParsePrivateKey")
	}

	sshConfig := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(parsedPrivateKey),
		},
	}

	connection, err := ssh.Dial("tcp", hostName, sshConfig)
	if err != nil {
		return nil, errors.Wrap(err, "ssh.Dial")
	}

	conn.connection = connection

	return conn, nil
}

type Connection struct {
	host       string
	user       string
	connection *ssh.Client
	session    *ssh.Session
}

func (c Connection) Run(cmd string) ([]byte, []byte, int, error) {
	outBuffer := bytes.NewBuffer([]byte{})
	errBuffer, exitCode, err := c.Stream(cmd, outBuffer)
	return outBuffer.Bytes(), errBuffer, exitCode, errors.Wrap(err, "Stream")
}

func (c Connection) Stream(cmd string, writer io.Writer) ([]byte, int, error) {
	session, err := c.connection.NewSession()
	if err != nil {
		return nil, 0, errors.Wrap(err, "connection.NewSession")
	}

	errBuffer := bytes.NewBuffer([]byte{})
	exitCode := 0

	session.Stdout = writer
	session.Stderr = errBuffer
	err = session.Run(cmd)
	if err != nil {
		exitErr, yes := err.(*ssh.ExitError)
		if yes {
			exitCode = exitErr.ExitStatus()
		} else {
			return nil, -1, errors.Wrap(err, "session.Run")
		}

	}

	return errBuffer.Bytes(), exitCode, nil
}

func (c Connection) StreamStdin(cmd string, reader io.Reader) (stdout, stderr []byte, exitCode int, err error) {
	session, err := c.connection.NewSession()

	outBuffer := bytes.NewBuffer([]byte{})
	errBuffer := bytes.NewBuffer([]byte{})

	session.Stdin = reader
	session.Stdout = outBuffer
	session.Stderr = errBuffer

	err = session.Run(cmd)

	if err != nil {
		exitErr, yes := err.(*ssh.ExitError)
		if yes {
			exitCode = exitErr.ExitStatus()
		} else {
			return nil, nil, -1, errors.Wrap(err, "expected ssh exiterror")
		}
	}

	return outBuffer.Bytes(), errBuffer.Bytes(), exitCode, nil
}

func (c Connection) Username() string {
	return c.user
}

func (c Connection) Cleanup() error {
	return nil
}
