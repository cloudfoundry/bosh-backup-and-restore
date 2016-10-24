package testssh

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/kr/pty"

	"golang.org/x/crypto/ssh"
)

type Server struct {
	Port int
	net.Listener
	LastUser    string
	LastKey     string
	FailAuth    bool
	FailSession bool
	logger      *log.Logger
}

func New(logWriter io.Writer) *Server {
	t := &Server{logger: log.New(logWriter, "[test-ssh-server] ", log.Lshortfile)}

	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.logger.Fatalf("Failed to listen (%s)", err)
	}
	t.Port = listener.Addr().(*net.TCPAddr).Port
	t.Listener = listener

	go t.HandleRequests()
	return t
}

func (t *Server) Close() {
	t.logger.Printf("Closing server listing on (%d)", t.Port)
	t.Listener.Close()
}

func (t *Server) HandleRequests() {
	t.logger.Printf("Listening on %d...", t.Port)

	for {
		tcpConn, err := t.Accept()
		if err != nil {
			t.logger.Printf("Failed to accept incoming connection (%s)", err)
			break
		}
		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, t.sshConfig())
		if err != nil {
			t.logger.Printf("Failed to handshake (%s)", err)
			break
		}

		t.logger.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())
		go ssh.DiscardRequests(reqs)
		go t.handleChannels(chans)
	}
}

func (t *Server) handleChannels(chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		go t.handleChannel(newChannel)
	}
}

func (t *Server) handleChannel(newChannel ssh.NewChannel) {
	if t := newChannel.ChannelType(); t != "session" {
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		return
	}
	if t.FailSession {
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("Failing due to test"))
		return
	}
	channel, requests, err := newChannel.Accept()
	if err != nil {
		t.logger.Printf("Could not accept channel (%s)", err)
		return
	}

	go t.processChannel(channel, requests)
}
func (t *Server) processChannel(channel ssh.Channel, requests <-chan *ssh.Request) {
	for req := range requests {
		switch req.Type {
		case "shell":
			t.startShellSession(channel, req)
			return
		case "exec":
			t.startExecSession(channel, req)
			return
		}
	}
}
func (t *Server) startShellSession(connection ssh.Channel, req *ssh.Request) {
	bash := exec.Command("bash")

	close := func() {
		connection.Close()
		_, err := bash.Process.Wait()
		if err != nil {
			t.logger.Printf("Failed to exit bash (%s)", err)
		}
		t.logger.Printf("Session closed")
	}

	t.logger.Print("Creating pty...")
	bashf, err := pty.Start(bash)
	if err != nil {
		t.logger.Printf("Could not start pty (%s)", err)
		close()
		return
	}

	var once sync.Once
	go func() {
		io.Copy(connection, bashf)
		once.Do(close)
	}()
	go func() {
		io.Copy(bashf, connection)
		once.Do(close)
	}()

	if len(req.Payload) == 0 {
		req.Reply(true, nil)
	}
}

type execMsg struct {
	Command string
}

func (t *Server) startExecSession(connection ssh.Channel, req *ssh.Request) {
	msg := execMsg{}
	ssh.Unmarshal(req.Payload, &msg)
	defer connection.Close()

	payloadSplit := strings.Split(msg.Command, " ")
	cmd := exec.Command(payloadSplit[0], payloadSplit[1:]...)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		t.logger.Printf("Error in stdout %v", err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.logger.Printf("Error in stderr %v", err)
		return
	}
	input, err := cmd.StdinPipe()
	if err != nil {
		t.logger.Printf("Error in stdin %v", err)
		return
	}

	if err = cmd.Start(); err != nil {
		t.logger.Printf("Error in start %v", err)
		return
	}

	go io.Copy(input, connection)
	io.Copy(connection, stdout)
	io.Copy(connection.Stderr(), stderr)

	if err = cmd.Wait(); err != nil {
		switch err := err.(type) {
		case *exec.ExitError:
			exitCode := err.Sys().(syscall.WaitStatus).ExitStatus()
			serializedValue := make([]byte, 4)
			binary.BigEndian.PutUint32(serializedValue, uint32(exitCode))
			connection.SendRequest("exit-status", false, serializedValue)
			req.Reply(true, nil)
			return
		default:
			t.logger.Printf("Error in cmd %v", err)
			req.Reply(false, nil)
			return

		}

	}
	connection.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
	req.Reply(true, nil)

	return
}

func (t *Server) sshConfig() *ssh.ServerConfig {
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			t.LastUser = conn.User()
			t.LastKey = string(key.Marshal())
			if t.FailAuth {
				return nil, fmt.Errorf("Auth fails")
			}
			return nil, nil
		},
	}
	private, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		t.logger.Fatal("Failed to parse private key")
	}
	config.AddHostKey(private)
	return config
}

var privateKey = []byte("-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEAyYiwxZo49jiNrEy11PyfvNLK04fyxgHdtHXh9v4eyRPgdeVR\nhxWpl+U7wqxZtS9SFMT4cI/e//Pwt+0WlcTPEftP9V8b7aABajD4hXdYEWbaJTJo\n92JLwGD5ILdMtXZKigo9aoAxG7J+6jl9g6PAWNA53MUHaqubVwTp4qdv3wii11r3\n9LlhXMihmwyo1mBNiOrijyRJrUEsXEu9+qufSHbylIHyFYbEwycm/vqhgWc5TcaI\nomDc83cNjdGtgJrhFKYC3tBNAtP0lP7ypN8bq5pLAZ9ks2VTVH2mhq2wttq7K/yv\nTt4pOa+UqSZVMxzVMDRNp6KtIO/sgn+bmBVo9wIDAQABAoIBABMQCeB3CQJJMSVm\nECD4UEe1DJhbmJwgGw9xwxDw0oqkhavBKCgF5YfHmBJ+6PFZa4MpanKDOU2ujktn\ncqZx+kAyLEsCVwrwApI/1ZISStNCjknMbd9QfefRhF8S13+mk8Bg3ZRQUdTT2mtf\nSr8D4zLDZ2W5gU0WtFfT0CevPMa0yCrO5GyARrrYEG186SErhIsTAav/pWDUdUzn\nRuSIoitIcIbz2I1Bs6aeYGMzGMmVgNb13qt+vleZW3nppqiptyfZGles2HXF3H2r\nRIu7XtdlHWCqCS4Jq41dp7j42rh46EQAQbp7lkm4rGGrAg9YOKGC1xrKM3OPWIZ7\nKxqjcfkCgYEA/oGeyj4B0yfxL7yAAFn0+6bv1++buQQlpsVapU3JZ5iSfMktJ1+R\nL2Iv+1nf5OEUk0GjGhJUq/S3+lVbx3BTOwXlZYw9UWv8IHEqP1lVifDDJFbFVKQv\nuuRwvDfgouH209n2JJzgbfpQRO6pB8dY7LjtD9Q91IkHbm6aBtCjZrUCgYEAyrd7\nigmcyRdIDOqtQdwT3VwIJKPbRHKvUX/KqEXTcgsNEBrakr/HC65wAVVB4wuu2Of5\naCWGj6I2VnDd8Yf8AAU+UI+FTtGDSGkN+G0dgz6Jz6ntG2m9EQX9wXBvKEobQJu4\nFrPrR/mU/kbXgZsUesA7CJFjg7zrLFuvAPGg0HsCgYAsI4LMhHCAlH7JzqFMbk2E\nj3EtPAr/zW5SPAv6e0EgzF8rcSB5oaNmWlsD9pRT940/9LQ6w08X+3sk2UTvk9V7\neQxNzkKcKmQxpC61ieLB55WQadQTV95HRXMf0XkOBq5uE3ES7Hon2K+vJMz/4lzT\nwUar5h1LDPDTAC+KWwjbuQKBgQDFfUF+tmSnR+Yqp0pJekVkBy/rujJ4mZ4RMQVX\nMEeRuBBu2yqLgwhAah22PsAkmJIrwLsq6jwQnICBcA3ZK5im0HTn+RpvMg/LMIWq\nu2rgHMIXrL1RUo8eEY8osAeq4Z9xLwOGIpwaD51Gp+911YZ7G+GnNDUV96vJGD0D\nF2OLFwKBgBoL1GL+MjXnVB9ntHtL2hwBjWsq4jRH18hQ1La8qSM4/3B4j3PyeDBl\nD16uXDNo8h9hgR2klKz6rcFspU+Zaw6g4pnr+RPYoHSRIgOB4QrDGrQLKxB756Gd\nx4F5w8wMFGnNScc/c9NAqOtijhuIhurs8qTZ3xOFiQ5W2mxTzKP+\n-----END RSA PRIVATE KEY-----\n")
