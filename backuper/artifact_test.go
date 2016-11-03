package backuper_test

import (
	"os"

	. "github.com/pivotal-cf/pcf-backup-and-restore/backuper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artifact", func() {
	var artifactName = "dave"
	Describe("DirectoryArtifactCreator", func() {
		It("creates a directory with the given name", func() {
			_, err := DirectoryArtifactCreator(artifactName)
			Expect(err).NotTo(HaveOccurred())
			Expect(artifactName).To(BeADirectory())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(artifactName)).To(Succeed())
		})
	})
})
