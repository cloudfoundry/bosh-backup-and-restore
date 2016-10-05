package boshclient_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBoshclient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Boshclient Suite")
}
