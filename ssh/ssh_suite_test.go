package ssh_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
)

func TestSsh(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ssh Suite")
}

var defaultPrivateKey string

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
})

var _ = AfterSuite(func() {
	testcluster.WaitForContainersToDie()
})
