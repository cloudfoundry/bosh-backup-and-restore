package bosh_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBosh(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bosh Suite")
}

func MustHaveEnv(keyname string) string {
	val := os.Getenv(keyname)
	Expect(val).NotTo(BeEmpty(), "Need "+keyname+" for the test")
	return val
}
