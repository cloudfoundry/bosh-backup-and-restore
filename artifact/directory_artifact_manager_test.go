package artifact_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf/pcf-backup-and-restore/artifact"
)

var _ = Context("ArtifactManager", func() {
	var artifactName = "my-cool-redis"
	var artifactManager = DirectoryArtifactManager{}
	Describe("Create", func() {
		It("creates a directory with the given name", func() {
			_, err := artifactManager.Create(artifactName)
			Expect(err).NotTo(HaveOccurred())
			Expect(artifactName).To(BeADirectory())
		})
	})

	Describe("NoopArtifactCreator", func() {
		Context("when the directory exists", func() {
			BeforeEach(func() {
				err := os.MkdirAll(artifactName, 0700)
				Expect(err).NotTo(HaveOccurred())
			})
			It("does not create a directory", func() {
				_, err := artifactManager.Open(artifactName)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the directory does not exist", func() {
			It("fails", func() {
				_, err := artifactManager.Open(artifactName)
				Expect(err).To(HaveOccurred())
				Expect(artifactName).NotTo(BeADirectory())
			})
		})
	})

	AfterEach(func() {
		Expect(os.RemoveAll(artifactName)).To(Succeed())
	})
})
