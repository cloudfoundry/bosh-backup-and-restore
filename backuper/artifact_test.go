package backuper_test

import (
	"io"
	"io/ioutil"
	"os"

	. "github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper/fakes"

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
		var fileCreationError error
		var writer io.Writer
		var fakeInstance *fakes.FakeInstance

		BeforeEach(func() {
			artifact, _ = DirectoryArtifactCreator(artifactName)
			fakeInstance = new(fakes.FakeInstance)
			fakeInstance.IDReturns("0")
			fakeInstance.NameReturns("redis")
		})
		JustBeforeEach(func() {
			writer, fileCreationError = artifact.CreateFile(fakeInstance)
		})
		Context("Can create a file", func() {
			It("creates a file in the artifact directory", func() {
				Expect(artifactName + "/redis-0.tgz").To(BeARegularFile())
			})

			It("writer writes contents to the file", func() {
				writer.Write([]byte("they are taking our jobs"))
				Expect(ioutil.ReadFile(artifactName + "/redis-0.tgz")).To(Equal([]byte("they are taking our jobs")))
			})

			It("does not fail", func() {
				Expect(fileCreationError).NotTo(HaveOccurred())
			})
		})

		Context("Cannot create file", func() {
			BeforeEach(func() {
				fakeInstance.NameReturns("foo/bar/baz")
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
