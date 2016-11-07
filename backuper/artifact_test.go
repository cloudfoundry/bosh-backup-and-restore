package backuper_test

import (
	"crypto/sha1"
	"fmt"
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

	Describe("NoopArtifactCreator", func() {
		It("does not create a directory", func() {
			_, err := NoopArtifactCreator(artifactName)
			Expect(err).NotTo(HaveOccurred())
			Expect(artifactName).NotTo(BeADirectory())
		})
	})

	Describe("DeploymentMatches", func() {
		var artifact Artifact
		var deploymentName string
		var instance1 *fakes.FakeInstance
		var instance2 *fakes.FakeInstance

		BeforeEach(func() {
			artifactName = "my-cool-redis"
			deploymentName = "my-cool-redis"
			instance1 = new(fakes.FakeInstance)
			instance1.NameReturns("redis")
			instance1.IDReturns("0")

			instance2 = new(fakes.FakeInstance)
			instance2.NameReturns("redis")
			instance2.IDReturns("1")

			Expect(os.Mkdir(deploymentName, 0777)).To(Succeed())

			file, err := os.Create(deploymentName + "/" + "metadata")
			Expect(err).NotTo(HaveOccurred())

			fakeMetadata := []byte(`---
instances:
- instance_name: redis
  instance_id: 0
  checksum: foo
- instance_name: redis
  instance_id: 1
  checksum: foo
`)

			_, err = file.Write(fakeMetadata)
			Expect(err).NotTo(HaveOccurred())
			Expect(file.Close()).To(Succeed())

			artifact, _ = NoopArtifactCreator(artifactName)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
		})

		Context("when the backup on disk matches the current deployment", func() {
			It("returns true", func() {
				match, _ := artifact.DeploymentMatches(deploymentName, []Instance{instance1, instance2})
				Expect(match).To(BeTrue())
			})
		})

		Context("when the backup doesn't match the current deployment", func() {
			JustBeforeEach(func() {
				tooManyInstances := []byte(`---
instances:
- instance_name: redis
  instance_id: 0
  checksum: foo
- instance_name: redis
  instance_id: 1
  checksum: foo
- instance_name: broker
  instance_id: 2
  checksum: foo
`)
				file, err := os.Create(deploymentName + "/" + "metadata")
				Expect(err).NotTo(HaveOccurred())
				_, err = file.Write(tooManyInstances)
				Expect(err).NotTo(HaveOccurred())
				Expect(file.Close()).To(Succeed())
			})

			It("returns false", func() {
				match, _ := artifact.DeploymentMatches(deploymentName, []Instance{instance1, instance2})
				Expect(match).To(BeFalse())
			})
		})

		Context("when an error occurs checking the metadata", func() {
			BeforeEach(func() {
				file, err := os.Create(deploymentName + "/" + "metadata")
				Expect(err).NotTo(HaveOccurred())
				_, err = file.Write([]byte("this is not yaml"))
				Expect(err).NotTo(HaveOccurred())
				Expect(file.Close()).To(Succeed())
			})

			It("returns error", func() {
				_, err := artifact.DeploymentMatches(deploymentName, []Instance{instance1, instance2})
				Expect(err).To(HaveOccurred())
			})
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

	Describe("Checksum", func() {
		var artifact Artifact
		var fakeInstance *fakes.FakeInstance

		BeforeEach(func() {
			artifact, _ = DirectoryArtifactCreator(artifactName)
			fakeInstance = new(fakes.FakeInstance)
			fakeInstance.IDReturns("0")
			fakeInstance.NameReturns("redis")
		})
		Context("file exists", func() {
			JustBeforeEach(func() {
				writer, fileCreationError := artifact.CreateFile(fakeInstance)
				Expect(fileCreationError).NotTo(HaveOccurred())

				writer.Write([]byte("foo bar baz"))
				Expect(writer.Close()).NotTo(HaveOccurred())
			})

			It("returns the checksum for the saved instance data", func() {
				Expect(artifact.CalculateChecksum(fakeInstance)).To(Equal(fmt.Sprintf("%x", sha1.Sum([]byte("foo bar baz")))))
			})
		})
		Context("file doesn't exist", func() {
			It("fails", func() {
				_, err := artifact.CalculateChecksum(fakeInstance)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("AddChecksum", func() {
		var artifact Artifact
		var addChecksumError error
		var fakeInstance *fakes.FakeInstance
		var checksum string

		BeforeEach(func() {
			artifact, _ = DirectoryArtifactCreator(artifactName)
			fakeInstance = new(fakes.FakeInstance)
			fakeInstance.IDReturns("0")
			fakeInstance.NameReturns("redis")
			checksum = "foobar"
		})
		JustBeforeEach(func() {
			addChecksumError = artifact.AddChecksum(fakeInstance, checksum)
		})

		Context("Succesfully creates a checksum file, if none exists", func() {
			It("makes a file", func() {
				Expect(artifactName + "/metadata").To(BeARegularFile())

				expectedMetadata := `---
instances:
- instance_name: redis
  instance_id: "0"
  checksum: foobar`
				Expect(ioutil.ReadFile(artifactName + "/metadata")).To(MatchYAML(expectedMetadata))
			})
		})
		Context("Appends to a checksum file, if already exists", func() {
			BeforeEach(func() {
				firstInstance := new(fakes.FakeInstance)
				firstInstance.IDReturns("0")
				firstInstance.NameReturns("broker")
				Expect(artifact.AddChecksum(firstInstance, "orignal_checksum")).NotTo(HaveOccurred())
			})

			It("appends to file", func() {
				Expect(artifactName + "/metadata").To(BeARegularFile())

				expectedMetadata := `---
instances:
- instance_name: broker
  instance_id: "0"
  checksum: orignal_checksum
- instance_name: redis
  instance_id: "0"
  checksum: foobar`
				Expect(ioutil.ReadFile(artifactName + "/metadata")).To(MatchYAML(expectedMetadata))
			})
		})

		Context("Appends fails, if existing file isn't valid", func() {
			BeforeEach(func() {
				ioutil.WriteFile(artifactName+"/metadata", []byte("not valid yaml"), 0666)
			})

			It("fails", func() {
				Expect(addChecksumError).To(HaveOccurred())
			})
		})

	})

	AfterEach(func() {
		Expect(os.RemoveAll(artifactName)).To(Succeed())
	})
})
