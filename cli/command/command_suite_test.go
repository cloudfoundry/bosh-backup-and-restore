package command

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBbr(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BBR Cmd Suite")
}
