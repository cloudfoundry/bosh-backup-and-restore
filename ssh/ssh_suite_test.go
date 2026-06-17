package ssh_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/cloudfoundry/bosh-backup-and-restore/testcluster"
)

func TestSsh(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ssh Suite")
}

var defaultPrivateKey string
var fixturesDir string

var _ = SynchronizedBeforeSuite(func() []byte {
	testcluster.PullDockerImage()
	return []byte{}
}, func(data []byte) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	Expect(err).NotTo(HaveOccurred())

	defaultPrivateKeyBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)

	defaultPrivateKey = string(defaultPrivateKeyBytes)

	fixturesDir = os.Getenv("FIXTURES_DIR")
	Expect(fixturesDir).NotTo(BeEmpty())
})

var _ = AfterSuite(func() {
	testcluster.WaitForContainersToDie()
})
