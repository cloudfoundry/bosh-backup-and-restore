package ssh

import (
	"bytes"

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
	session, err := c.connection.NewSession()
	if err != nil {
		return nil, nil, 0, err
	}

	outBuffer := bytes.NewBuffer([]byte{})
	errBuffer := bytes.NewBuffer([]byte{})
	exitCode := 0

	session.Stdout = outBuffer
	session.Stderr = errBuffer
	err = session.Run(cmd)
	if err != nil {
		exitErr, yes := err.(*ssh.ExitError)
		if yes {
			exitCode = exitErr.ExitStatus()
		} else {
			return nil, nil, -1, err
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
