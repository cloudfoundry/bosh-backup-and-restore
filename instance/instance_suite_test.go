package instance_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
	"golang.org/x/crypto/ssh"
)

func TestInstance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Instance Suite")
}

var defaultPrivateKey string //nolint:unused

var _ = BeforeSuite(func() {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	Expect(err).NotTo(HaveOccurred())

	defaultPrivateKeyBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)

	defaultPrivateKey = string(defaultPrivateKeyBytes)
})

func publicKeyForDocker(privateKey string) string { //nolint:unused
	parsedPrivateKey, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		Fail("Cant parse key")
	}

	return "ssh-rsa " + base64.StdEncoding.EncodeToString(parsedPrivateKey.PublicKey().Marshal())
}

var _ = AfterSuite(func() {
	testcluster.WaitForContainersToDie()
})
