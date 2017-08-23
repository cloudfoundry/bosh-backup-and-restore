package orderer_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestOrderer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Orderer Suite")
}
