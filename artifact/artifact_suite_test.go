package artifact_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestArtifact(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Artifact Suite")
}
