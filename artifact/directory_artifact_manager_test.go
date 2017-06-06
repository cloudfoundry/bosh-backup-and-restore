package artifact_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf/bosh-backup-and-restore/artifact"
)

var _ = Context("ArtifactManager", func() {
	var artifactName = "my-cool-redis"
	var artifactManager = DirectoryArtifactManager{}
	var err error

	Describe("Create", func() {
		JustBeforeEach(func() {
			_, err = artifactManager.Create(artifactName, nil)
		})

		Context("when the directory exists", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(artifactName, 0777)).To(Succeed())
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(ContainSubstring("failed creating directory")))
			})
		})

		Context("when the directory doesnt exist", func() {
			It("creates a directory with the given name", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(artifactName).To(BeADirectory())
			})
		})
	})

	Describe("Open", func() {
		Context("when the directory exists", func() {
			BeforeEach(func() {
				err := os.MkdirAll(artifactName, 0700)
				Expect(err).NotTo(HaveOccurred())
			})

			It("does not create a directory", func() {
				_, err := artifactManager.Open(artifactName, nil)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the directory does not exist", func() {
			It("fails", func() {
				_, err := artifactManager.Open(artifactName, nil)
				Expect(err).To(MatchError(ContainSubstring("failed opening the directory")))
				Expect(artifactName).NotTo(BeADirectory())
			})
		})
	})

	Describe("Exists", func() {
		var exists bool

		JustBeforeEach(func() {
			exists = artifactManager.Exists(artifactName)
		})

		Context("when the artifact exists", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(artifactName, 0777)).To(Succeed())
			})

			It("returns true", func() {
				Expect(exists).To(BeTrue())
			})
		})

		Context("when the artifact doesn't exist", func() {
			It("returns false", func() {
				Expect(exists).To(BeFalse())
			})
		})
	})

	AfterEach(func() {
		Expect(os.RemoveAll(artifactName)).To(Succeed())
	})
})
