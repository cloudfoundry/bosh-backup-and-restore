package ssh

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net"
	"testing"
	"time"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	gossh "golang.org/x/crypto/ssh"
)

// startSingleSessionServer accepts one SSH connection on a local port and hands
// its first session channel to handle. It returns a connection to that server
// and a channel that is closed once the server goroutine finishes.
func startSingleSessionServer(t *testing.T, handle func(gossh.NewChannel)) (SSHConnection, <-chan struct{}) {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	private := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	signer, err := gossh.NewSignerFromKey(key)
	if err != nil {
		t.Fatal(err)
	}
	config := &gossh.ServerConfig{NoClientAuth: true}
	config.AddHostKey(signer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() }) //nolint:errcheck
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close() //nolint:errcheck
		server, channels, requests, err := gossh.NewServerConn(conn, config)
		if err != nil {
			return
		}
		defer server.Close() //nolint:errcheck
		go gossh.DiscardRequests(requests)
		incoming := <-channels
		if incoming == nil {
			return
		}
		handle(incoming)
	}()
	connection, err := NewConnection(listener.Addr().String(), "test", string(private), gossh.FixedHostKey(signer.PublicKey()), nil, boshlog.NewLogger(boshlog.LevelNone))
	if err != nil {
		t.Fatal(err)
	}
	return connection, done
}

func TestSessionClosedBeforeExecIsNotSuccess(t *testing.T) {
	connection, done := startSingleSessionServer(t, func(incoming gossh.NewChannel) {
		channel, _, err := incoming.Accept()
		if err != nil {
			return
		}
		// Tear down the session before acknowledging any exec request.
		channel.Close() //nolint:errcheck
	})
	_, _, code, err := connection.Run("must-not-run")
	if err == nil || code == 0 {
		t.Fatalf("unexecuted command reported success: code=%d err=%v", code, err)
	}
	<-done
}

func TestCommandExitingBeforeReadingAllStdinIsSuccess(t *testing.T) {
	connection, done := startSingleSessionServer(t, func(incoming gossh.NewChannel) {
		channel, requests, err := incoming.Accept()
		if err != nil {
			return
		}
		for request := range requests {
			if request.Type != "exec" {
				request.Reply(false, nil) //nolint:errcheck
				continue
			}
			request.Reply(true, nil) //nolint:errcheck
			// Exit successfully without reading stdin, as tar does once it has
			// read the end-of-archive marker. The pause lets the client fill
			// the channel window, so it is blocked writing stdin when the
			// channel closes.
			time.Sleep(300 * time.Millisecond)
			if _, err := channel.SendRequest("exit-status", false, gossh.Marshal(struct{ Status uint32 }{0})); err != nil {
				return
			}
			channel.Close() //nolint:errcheck
		}
	})
	// Far larger than the SSH channel window, so the client is still writing
	// stdin when the remote side closes the channel.
	stdin := bytes.NewReader(make([]byte, 8*1024*1024))
	_, _, code, err := connection.StreamStdin("exits-early", stdin)
	if err != nil || code != 0 {
		t.Fatalf("successful command reported failure: code=%d err=%v", code, err)
	}
	<-done
}
