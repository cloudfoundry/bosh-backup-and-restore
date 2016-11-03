package backuper_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing/iotest"

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
		var reader io.Reader
		var fileCreationError error

		BeforeEach(func() {
			artifact, _ = DirectoryArtifactCreator(artifactName)
		})
		JustBeforeEach(func() {
			fileCreationError = artifact.CreateFile(filename, reader)
		})
		Context("Can create a file", func() {
			BeforeEach(func() {
				filename = "foo"
				reader = bytes.NewBufferString("they are taking our jobs")
			})
			It("creates a file in the artifact directory", func() {
				Expect(artifactName + "/" + "foo").To(BeARegularFile())
			})

			It("writes the contents of the reader to the file", func() {
				Expect(ioutil.ReadFile(artifactName + "/" + "foo")).To(Equal([]byte("they are taking our jobs")))
			})

			It("does not fail", func() {
				Expect(fileCreationError).NotTo(HaveOccurred())
			})

			Context("cannot drain file", func() {
				BeforeEach(func() {
					reader = iotest.TimeoutReader(bytes.NewBufferString("123"))
				})

				It("returns error", func() {
					Expect(fileCreationError).To(HaveOccurred())
				})
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
