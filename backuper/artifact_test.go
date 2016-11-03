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

	})
	Describe("CreateFile", func() {
		var artifact Artifact
		var filename string
		var artifactCreationError error
		var fileCreationError error
		BeforeEach(func() {
			artifact, artifactCreationError = DirectoryArtifactCreator(artifactName)
		})
		JustBeforeEach(func() {
			fileCreationError = artifact.CreateFile(filename)
		})
		Context("Can create a file", func() {
			BeforeEach(func() {
				filename = "foo"
			})
			It("creates a file in the artifact directory", func() {
				Expect(artifactName + "/" + "foo").To(BeARegularFile())
			})

			It("does not fail", func() {
				Expect(fileCreationError).NotTo(HaveOccurred())
			})
		})

		Context("Cannot create file", func() {
			BeforeEach(func() {
				filename = "foo/bar/baz"
			})
			It("fails", func() {
				Expect(fileCreationError).To(HaveOccurred())
			})
		})

	})

	AfterEach(func() {
		Expect(os.RemoveAll(artifactName)).To(Succeed())
	})
})
