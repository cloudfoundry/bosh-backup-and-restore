package ssh

import (
	"bytes"
	"io"
	"os"

	"github.com/pivotal-cf/pcf-backup-and-restore/bosh"
	"golang.org/x/crypto/ssh"
)

func ConnectionCreator(hostName, userName, privateKey string) (bosh.SSHConnection, error) {
	conn := Connection{
		host: hostName,
		user: userName,
	}

	parsedPrivateKey, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(parsedPrivateKey),
		},
	}

	connection, err := ssh.Dial("tcp", hostName, sshConfig)
	if err != nil {
		return nil, err
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
	return outBuffer.Bytes(), errBuffer, exitCode, err
}

func (c Connection) Stream(cmd string, writer io.Writer) ([]byte, int, error) {
	session, err := c.connection.NewSession()
	if err != nil {
		return nil, 0, err
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
			return nil, -1, err
		}

	}

	return errBuffer.Bytes(), exitCode, nil
}

func (c Connection) StreamStdin(cmd string, reader io.Reader) ([]byte, int, error) {
	session, err := c.connection.NewSession()

	errBuffer := bytes.NewBuffer([]byte{})

	session.Stdin = reader
	session.Stdout = os.Stdout
	session.Stderr = errBuffer

	exitCode := 0
	err = session.Run(cmd)

	if err != nil {
		exitErr, yes := err.(*ssh.ExitError)
		if yes {
			exitCode = exitErr.ExitStatus()
		} else {
			return nil, -1, err
		}
	}

	return errBuffer.Bytes(), exitCode, nil
}

func (c Connection) Username() string {
	return c.user
}

func (c Connection) Cleanup() error {
	return nil
}
