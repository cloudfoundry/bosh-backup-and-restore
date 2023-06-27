package configPrinter_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfigPrinter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ConfigPrinter Suite")
}
