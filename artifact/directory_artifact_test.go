package artifact_test

import (
	"archive/tar"
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/pivotal-cf/bosh-backup-and-restore/artifact"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DirectoryArtifact", func() {
	var artifactName = "my-cool-redis"
	var artifactManager = DirectoryArtifactManager{}
	var logger = boshlog.NewWriterLogger(boshlog.LevelDebug, GinkgoWriter, GinkgoWriter)

	AfterEach(func() {
		Expect(os.RemoveAll(artifactName)).To(Succeed())
	})

	Describe("DeploymentMatches", func() {
		var artifact orchestrator.Artifact
		var instance1 *fakes.FakeInstance
		var instance2 *fakes.FakeInstance

		BeforeEach(func() {
			artifactName = "my-cool-redis"
			instance1 = new(fakes.FakeInstance)
			instance1.NameReturns("redis")
			instance1.IndexReturns("0")

			instance2 = new(fakes.FakeInstance)
			instance2.NameReturns("redis")
			instance2.IndexReturns("1")

			artifact, _ = artifactManager.Open(artifactName, logger)
		})

		Context("when the backup on disk matches the current deployment", func() {
			BeforeEach(func() {
				createTestMetadata(artifactName, `---
instances:
- instance_name: redis
  instance_index: 0
  checksum: foo
- instance_name: redis
  instance_index: 1
  checksum: foo
`)
			})

			It("returns true", func() {
				match, _ := artifact.DeploymentMatches(artifactName, []orchestrator.Instance{instance1, instance2})
				Expect(match).To(BeTrue())
			})
		})

		Context("when the backup doesn't match the current deployment", func() {
			BeforeEach(func() {
				createTestMetadata(artifactName, `---
instances:
- instance_name: redis
  instance_index: 0
  checksum: foo
- instance_name: redis
  instance_index: 1
  checksum: foo
- instance_name: broker
  instance_index: 2
  checksum: foo
`)
			})

			It("returns false", func() {
				match, _ := artifact.DeploymentMatches(artifactName, []orchestrator.Instance{instance1, instance2})
				Expect(match).To(BeFalse())
			})
		})

		Context("when an error occurs unmarshaling the metadata", func() {
			BeforeEach(func() {
				Expect(os.Mkdir(artifactName, 0777)).To(Succeed())
				file, err := os.Create(artifactName + "/" + "metadata")
				Expect(err).NotTo(HaveOccurred())
				_, err = file.Write([]byte("this is not yaml"))
				Expect(err).NotTo(HaveOccurred())
				Expect(file.Close()).To(Succeed())
			})

			It("returns error", func() {
				_, err := artifact.DeploymentMatches(artifactName, []orchestrator.Instance{instance1, instance2})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when an error occurs checking if the file exists", func() {
			It("returns error", func() {
				_, err := artifact.DeploymentMatches(artifactName, []orchestrator.Instance{instance1, instance2})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Valid", func() {
		var artifact orchestrator.Artifact
		var verifyResult bool
		var verifyError error

		JustBeforeEach(func() {
			var err error
			artifact, err = artifactManager.Open(artifactName, logger)
			Expect(err).NotTo(HaveOccurred())
			verifyResult, verifyError = artifact.Valid()
		})
		BeforeEach(func() {
			Expect(os.Mkdir(artifactName, 0777)).To(Succeed())
		})

		Context("when the default artifact sha's match metafile", func() {
			BeforeEach(func() {
				contents := createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
					"file2": "Gopher names:\nGeorge\nGeoffrey\nGonzo",
				})

				Expect(ioutil.WriteFile(artifactName+"/redis-0.tar", contents, 0666)).NotTo(HaveOccurred())

				createTestMetadata(artifactName, fmt.Sprintf(`---
instances:
- instance_name: redis
  instance_index: 0
  checksums:
    file1: %x
    file2: %x
`, sha1.Sum([]byte("This archive contains some text files.")),
					sha1.Sum([]byte("Gopher names:\nGeorge\nGeoffrey\nGonzo"))))
			})

			It("returns true", func() {
				Expect(verifyError).NotTo(HaveOccurred())
				Expect(verifyResult).To(BeTrue())
			})
		})

		Context("when the named artifact sha matches the metadata file", func() {
			BeforeEach(func() {
				contents := createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
				})

				Expect(ioutil.WriteFile(artifactName+"/foo_redis.tar", contents, 0666)).NotTo(HaveOccurred())

				createTestMetadata(artifactName, fmt.Sprintf(`---
blobs:
- blob_name: foo_redis
  checksums:
    file1: %x
`, sha1.Sum([]byte("This archive contains some text files."))))
			})

			It("returns true", func() {
				Expect(verifyError).NotTo(HaveOccurred())
				Expect(verifyResult).To(BeTrue())
			})
		})

		Context("when the named artifact sha doesn't match the metadata file", func() {
			BeforeEach(func() {
				contents := createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
				})

				Expect(ioutil.WriteFile(artifactName+"/foo_redis.tar", contents, 0666)).NotTo(HaveOccurred())

				createTestMetadata(artifactName, fmt.Sprintf(`---
blobs:
- blob_name: foo_redis
  checksums:
    file1: %x
`, sha1.Sum([]byte("you fools!"))))
			})

			It("returns false", func() {
				Expect(verifyError).NotTo(HaveOccurred())
				Expect(verifyResult).To(BeFalse())
			})
		})

		Context("when one of the default artifact file's contents don't match the sha", func() {
			BeforeEach(func() {
				contents := createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
					"file2": "Gopher names:\nGeorge\nGeoffrey\nGonzo",
				})
				Expect(ioutil.WriteFile(artifactName+"/redis-0.tar", contents, 0666)).NotTo(HaveOccurred())
				createTestMetadata(artifactName, fmt.Sprintf(`---
instances:
- instance_name: redis
  instance_index: 0
  checksums:
    file1: %x
    file2: %x
`, sha1.Sum([]byte("This archive contains some text files.")),
					sha1.Sum([]byte("Gopher names:\nNo Goper names"))))
			})

			It("returns false", func() {
				Expect(verifyError).NotTo(HaveOccurred())
				Expect(verifyResult).To(BeFalse())
			})
		})

		Context("when one of there is an extra file in the backed metadata", func() {
			BeforeEach(func() {
				contents := createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
				})
				Expect(ioutil.WriteFile(artifactName+"/redis-0.tar", contents, 0666)).NotTo(HaveOccurred())
				createTestMetadata(artifactName, fmt.Sprintf(`---
instances:
- instance_name: redis
  instance_index: 0
  checksums:
    file1: %x
    file2: %x
`, sha1.Sum([]byte("This archive contains some text files.")),
					sha1.Sum([]byte("Gopher names:\nNot present"))))
			})

			It("returns false", func() {
				Expect(verifyResult).To(BeFalse())
				Expect(verifyError).NotTo(HaveOccurred())
			})
		})

		Context("metadata describes a file that dosen't exist", func() {
			BeforeEach(func() {
				createTestMetadata(artifactName, fmt.Sprintf(`---
instances:
- instance_name: redis
	instance_index: 0
	checksums:
		file1: %x
`, sha1.Sum([]byte("This archive contains some text files."))))
			})

			It("returns false", func() {
				Expect(verifyResult).To(BeFalse())
			})
			It("returns an error", func() {
				Expect(verifyError).To(HaveOccurred())
			})
		})

		Context("metadata file dosen't exist", func() {
			BeforeEach(func() {
				contents := createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
				})
				Expect(ioutil.WriteFile(artifactName+"/redis-1.tar", contents, 0666)).NotTo(HaveOccurred())
			})

			It("returns false", func() {
				Expect(verifyResult).To(BeFalse())
			})
			It("returns an error", func() {
				Expect(verifyError).To(HaveOccurred())
			})
		})
	})

	Describe("CreateFile", func() {
		var artifact orchestrator.Artifact
		var fileCreationError error
		var writer io.Writer
		var fakeBackupBlob *fakes.FakeBackupBlob

		BeforeEach(func() {
			artifact, _ = artifactManager.Create(artifactName, logger)
			fakeBackupBlob = new(fakes.FakeBackupBlob)
			fakeBackupBlob.IndexReturns("0")
			fakeBackupBlob.NameReturns("redis")
		})
		JustBeforeEach(func() {
			writer, fileCreationError = artifact.CreateFile(fakeBackupBlob)
		})
		Context("with a default backup blob", func() {
			Context("Can create a file", func() {
				It("creates a file in the artifact directory", func() {
					Expect(artifactName + "/redis-0.tar").To(BeARegularFile())
				})

				It("writer writes contents to the file", func() {
					writer.Write([]byte("lalala a file"))
					Expect(ioutil.ReadFile(artifactName + "/redis-0.tar")).To(Equal([]byte("lalala a file")))
				})

				It("does not fail", func() {
					Expect(fileCreationError).NotTo(HaveOccurred())
				})
			})
		})
		Context("with a named backup blob", func() {
			BeforeEach(func() {
				fakeBackupBlob.IsNamedReturns(true)
				fakeBackupBlob.NameReturns("my-backup-artifact")
			})

			It("creates the named file in the artifact directory", func() {
				Expect(artifactName + "/my-backup-artifact.tar").To(BeARegularFile())
			})

			It("writer writes contents to the file", func() {
				writer.Write([]byte("lalala a file"))
				Expect(ioutil.ReadFile(artifactName + "/my-backup-artifact.tar")).To(Equal([]byte("lalala a file")))
			})

			It("does not fail", func() {
				Expect(fileCreationError).NotTo(HaveOccurred())
			})
		})

		Context("Cannot create file", func() {
			BeforeEach(func() {
				fakeBackupBlob.NameReturns("foo/bar/baz")
			})
			It("fails", func() {
				Expect(fileCreationError).To(HaveOccurred())
			})
		})

	})

	Describe("SaveManifest", func() {
		var artifact orchestrator.Artifact
		var saveManifestError error
		BeforeEach(func() {
			artifactName = "foo-bar"
			artifact, _ = artifactManager.Create(artifactName, logger)
		})
		JustBeforeEach(func() {
			saveManifestError = artifact.SaveManifest("contents")
		})
		It("does not fail", func() {
			Expect(saveManifestError).NotTo(HaveOccurred())
		})

		It("writes contents to a file", func() {
			Expect(ioutil.ReadFile(artifactName + "/manifest.yml")).To(Equal([]byte("contents")))
		})
	})

	Describe("ReadFile", func() {
		var artifact orchestrator.Artifact
		var fileReadError error
		var reader io.Reader
		var fakeBackupBlob *fakes.FakeBackupBlob

		BeforeEach(func() {
			artifact, _ = artifactManager.Open(artifactName, logger)
			fakeBackupBlob = new(fakes.FakeBackupBlob)
			fakeBackupBlob.IndexReturns("0")
			fakeBackupBlob.NameReturns("redis")
		})

		Context("default backup blob - file exists and is readable", func() {
			BeforeEach(func() {
				err := os.MkdirAll(artifactName, 0700)
				Expect(err).NotTo(HaveOccurred())
				_, err = os.Create(artifactName + "/redis-0.tar")
				Expect(err).NotTo(HaveOccurred())
				err = ioutil.WriteFile(artifactName+"/redis-0.tar", []byte("backup-content"), 0700)
				Expect(err).NotTo(HaveOccurred())
			})

			JustBeforeEach(func() {
				reader, fileReadError = artifact.ReadFile(fakeBackupBlob)
			})

			It("does not fail", func() {
				Expect(fileReadError).NotTo(HaveOccurred())
			})

			It("reads the correct file", func() {
				contents, err := ioutil.ReadAll(reader)

				Expect(err).NotTo(HaveOccurred())
				Expect(contents).To(ContainSubstring("backup-content"))
			})
		})

		Context("named backup blob - file exists and is readable", func() {
			BeforeEach(func() {
				fakeBackupBlob.IsNamedReturns(true)
				fakeBackupBlob.NameReturns("foo-bar")

				err := os.MkdirAll(artifactName, 0700)
				Expect(err).NotTo(HaveOccurred())
				_, err = os.Create(artifactName + "/foo-bar.tar")
				Expect(err).NotTo(HaveOccurred())
				err = ioutil.WriteFile(artifactName+"/foo-bar.tar", []byte("backup-content"), 0700)
				Expect(err).NotTo(HaveOccurred())
			})

			JustBeforeEach(func() {
				reader, fileReadError = artifact.ReadFile(fakeBackupBlob)
			})

			It("does not fail", func() {
				Expect(fileReadError).NotTo(HaveOccurred())
			})

			It("reads the correct file", func() {
				contents, err := ioutil.ReadAll(reader)

				Expect(err).NotTo(HaveOccurred())
				Expect(contents).To(ContainSubstring("backup-content"))
			})
		})

		Context("File is not readable", func() {
			It("fails", func() {
				_, fileReadError = artifact.ReadFile(fakeBackupBlob)
				Expect(fileReadError).To(HaveOccurred())
			})
		})
	})

	Describe("Checksum", func() {
		var artifact orchestrator.Artifact
		var fakeBackupBlob *fakes.FakeBackupBlob

		BeforeEach(func() {
			fakeBackupBlob = new(fakes.FakeBackupBlob)
			fakeBackupBlob.IndexReturns("0")
			fakeBackupBlob.NameReturns("redis")
		})
		JustBeforeEach(func() {
			artifact, _ = artifactManager.Create(artifactName, logger)
		})
		Context("file exists", func() {
			Context("default backup blob", func() {
				JustBeforeEach(func() {
					writer, fileCreationError := artifact.CreateFile(fakeBackupBlob)
					Expect(fileCreationError).NotTo(HaveOccurred())

					contents := createTarWithContents(map[string]string{
						"readme.txt": "This archive contains some text files.",
						"gopher.txt": "Gopher names:\nGeorge\nGeoffrey\nGonzo",
						"todo.txt":   "Get animal handling license.",
					})

					writer.Write(contents)
					Expect(writer.Close()).NotTo(HaveOccurred())
				})

				It("returns the checksum for the saved instance data", func() {
					Expect(artifact.CalculateChecksum(fakeBackupBlob)).To(Equal(
						orchestrator.BackupChecksum{
							"readme.txt": fmt.Sprintf("%x", sha1.Sum([]byte("This archive contains some text files."))),
							"gopher.txt": fmt.Sprintf("%x", sha1.Sum([]byte("Gopher names:\nGeorge\nGeoffrey\nGonzo"))),
							"todo.txt":   fmt.Sprintf("%x", sha1.Sum([]byte("Get animal handling license."))),
						}))
				})
			})

			Context("named backup blob", func() {
				BeforeEach(func() {
					fakeBackupBlob.IsNamedReturns(true)
				})
				JustBeforeEach(func() {
					writer, fileCreationError := artifact.CreateFile(fakeBackupBlob)
					Expect(fileCreationError).NotTo(HaveOccurred())

					contents := createTarWithContents(map[string]string{
						"readme.txt": "This archive contains some text files.",
						"gopher.txt": "Gopher names:\nGeorge\nGeoffrey\nGonzo",
						"todo.txt":   "Get animal handling license.",
					})

					writer.Write(contents)
					Expect(writer.Close()).NotTo(HaveOccurred())
				})

				It("returns the checksum for the saved instance data", func() {
					Expect(artifact.CalculateChecksum(fakeBackupBlob)).To(Equal(
						orchestrator.BackupChecksum{
							"readme.txt": fmt.Sprintf("%x", sha1.Sum([]byte("This archive contains some text files."))),
							"gopher.txt": fmt.Sprintf("%x", sha1.Sum([]byte("Gopher names:\nGeorge\nGeoffrey\nGonzo"))),
							"todo.txt":   fmt.Sprintf("%x", sha1.Sum([]byte("Get animal handling license."))),
						}))
				})
			})
		})

		Context("invalid tar file", func() {
			JustBeforeEach(func() {
				writer, fileCreationError := artifact.CreateFile(fakeBackupBlob)
				Expect(fileCreationError).NotTo(HaveOccurred())

				contents := []byte("this ain't a tarball")

				writer.Write(contents)
				Expect(writer.Close()).NotTo(HaveOccurred())
			})

			It("fails to read", func() {
				_, err := artifact.CalculateChecksum(fakeBackupBlob)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("file doesn't exist", func() {
			It("fails", func() {
				_, err := artifact.CalculateChecksum(fakeBackupBlob)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("AddChecksum", func() {
		var artifact orchestrator.Artifact
		var addChecksumError error
		var fakeBackupBlob *fakes.FakeBackupBlob
		var checksum map[string]string

		BeforeEach(func() {
			artifact, _ = artifactManager.Create(artifactName, logger)
			fakeBackupBlob = new(fakes.FakeBackupBlob)
			fakeBackupBlob.IndexReturns("0")
			fakeBackupBlob.NameReturns("redis")
			checksum = map[string]string{"filename": "foobar"}
		})
		JustBeforeEach(func() {
			addChecksumError = artifact.AddChecksum(fakeBackupBlob, checksum)
		})

		Context("Succesfully creates a checksum file, if none exists, with a default backup blob", func() {
			It("makes a file", func() {
				Expect(artifactName + "/metadata").To(BeARegularFile())

				expectedMetadata := `---
instances:
- instance_name: redis
  instance_index: "0"
  checksums:
    filename: foobar`
				Expect(ioutil.ReadFile(artifactName + "/metadata")).To(MatchYAML(expectedMetadata))
			})
		})

		Context("Succesfully creates a checksum file, if none exists, with a named backup blob", func() {
			BeforeEach(func() {
				fakeBackupBlob.IsNamedReturns(true)
				fakeBackupBlob.NameReturns("my-amazing-artifact")
			})

			It("makes a file", func() {
				Expect(artifactName + "/metadata").To(BeARegularFile())

				expectedMetadata := `---
instances: []
blobs:
- blob_name: my-amazing-artifact
  checksums:
    filename: foobar`
				Expect(ioutil.ReadFile(artifactName + "/metadata")).To(MatchYAML(expectedMetadata))
			})
		})

		Context("Appends to a checksum file, if already exists, with a default backup blob", func() {
			BeforeEach(func() {
				anotherRemoteArtifact := new(fakes.FakeBackupBlob)
				anotherRemoteArtifact.IndexReturns("0")
				anotherRemoteArtifact.NameReturns("broker")
				Expect(artifact.AddChecksum(anotherRemoteArtifact, map[string]string{"filename1": "orignal_checksum"})).NotTo(HaveOccurred())
			})

			It("appends to file", func() {
				Expect(artifactName + "/metadata").To(BeARegularFile())

				expectedMetadata := `---
instances:
- instance_name: broker
  instance_index: "0"
  checksums:
    filename1: orignal_checksum
- instance_name: redis
  instance_index: "0"
  checksums:
    filename: foobar`
				Expect(ioutil.ReadFile(artifactName + "/metadata")).To(MatchYAML(expectedMetadata))
			})
		})

		Context("Appends to a checksum file, if already exists, with a named backup blob", func() {
			BeforeEach(func() {
				fakeBackupBlob.IsNamedReturns(true)
				anotherRemoteArtifact := new(fakes.FakeBackupBlob)
				anotherRemoteArtifact.NameReturns("broker")
				anotherRemoteArtifact.IsNamedReturns(true)
				Expect(artifact.AddChecksum(anotherRemoteArtifact, map[string]string{"filename1": "orignal_checksum"})).NotTo(HaveOccurred())
			})

			It("appends to file", func() {
				Expect(artifactName + "/metadata").To(BeARegularFile())

				expectedMetadata := `---
instances: []
blobs:
- blob_name: broker
  checksums:
    filename1: orignal_checksum
- blob_name: redis
  checksums:
    filename: foobar`
				Expect(ioutil.ReadFile(artifactName + "/metadata")).To(MatchYAML(expectedMetadata))
			})
		})

		Context("Appends to a checksum file, if already exists, with a default backup blob and a named backup blob", func() {
			BeforeEach(func() {
				fakeBackupBlob.IsNamedReturns(true)
				anotherRemoteArtifact := new(fakes.FakeBackupBlob)
				anotherRemoteArtifact.NameReturns("broker")
				anotherRemoteArtifact.IndexReturns("0")
				Expect(artifact.AddChecksum(anotherRemoteArtifact, map[string]string{"filename1": "orignal_checksum"})).NotTo(HaveOccurred())
			})

			It("appends to file", func() {
				Expect(artifactName + "/metadata").To(BeARegularFile())

				expectedMetadata := `---
instances:
- instance_name: broker
  instance_index: "0"
  checksums:
    filename1: orignal_checksum
blobs:
- blob_name: redis
  checksums:
    filename: foobar`
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

	Describe("FetchChecksum", func() {
		var artifact orchestrator.Artifact
		var fetchChecksumError error
		var fakeBlob *fakes.FakeBackupBlob
		var checksum orchestrator.BackupChecksum
		BeforeEach(func() {
			fakeBlob = new(fakes.FakeBackupBlob)
		})
		JustBeforeEach(func() {
			var artifactOpenError error
			artifact, artifactOpenError = artifactManager.Open(artifactName, logger)
			Expect(artifactOpenError).NotTo(HaveOccurred())

			checksum, fetchChecksumError = artifact.FetchChecksum(fakeBlob)
		})
		Context("the named backup blob is found in metadata", func() {
			BeforeEach(func() {
				fakeBlob.IsNamedReturns(true)
				fakeBlob.NameReturns("foo")

				createTestMetadata(artifactName, `---
instances: []
blobs:
- blob_name: foo
  checksums:
    filename1: orignal_checksum`)
			})

			It("dosen't fail", func() {
				Expect(fetchChecksumError).NotTo(HaveOccurred())
			})

			It("fetches the checksum", func() {
				Expect(checksum).To(Equal(orchestrator.BackupChecksum{"filename1": "orignal_checksum"}))
			})
		})

		Context("the default backup blob is found in metadata", func() {
			BeforeEach(func() {
				fakeBlob.NameReturns("foo")
				fakeBlob.IndexReturns("bar")

				createTestMetadata(artifactName, `---
instances:
- instance_name: foo
  instance_index: "bar"
  checksums:
    filename1: orignal_checksum`)
			})

			It("dosen't fail", func() {
				Expect(fetchChecksumError).NotTo(HaveOccurred())
			})

			It("fetches the checksum", func() {
				Expect(checksum).To(Equal(orchestrator.BackupChecksum{"filename1": "orignal_checksum"}))
			})
		})
		Context("the default backup blob is not found in metadata", func() {
			BeforeEach(func() {
				fakeBlob.NameReturns("not-foo")
				fakeBlob.IndexReturns("bar")

				createTestMetadata(artifactName, `---
instances:
- instance_name: foo
  instance_index: "bar"
  checksums:
    filename1: orignal_checksum`)
			})

			It("dosen't fail", func() {
				Expect(fetchChecksumError).ToNot(HaveOccurred())
			})

			It("returns nil", func() {
				Expect(checksum).To(BeNil())
			})
		})

		Context("the named backup blob is not found in metadata", func() {
			BeforeEach(func() {
				fakeBlob.NameReturns("not-foo")
				fakeBlob.IsNamedReturns(true)

				createTestMetadata(artifactName, `---
instances:
- instance_name: foo
  instance_index: "bar"
  checksums:
    filename1: orignal_checksum`)
			})

			It("dosen't fail", func() {
				Expect(fetchChecksumError).ToNot(HaveOccurred())
			})

			It("returns nil", func() {
				Expect(checksum).To(BeNil())
			})
		})

		Context("the instance is not found in metadata", func() {
			BeforeEach(func() {
				fakeBlob.NameReturns("not-foo")
				fakeBlob.IndexReturns("bar")

				createTestMetadata(artifactName, `---
instances:
- instance_name: foo
  instance_index: "bar"
  checksums:
    filename1: orignal_checksum`)
			})

			It("dosen't fail", func() {
				Expect(fetchChecksumError).ToNot(HaveOccurred())
			})

			It("returns nil", func() {
				Expect(checksum).To(BeNil())
			})
		})

		Context("if existing file isn't valid", func() {
			BeforeEach(func() {
				createTestMetadata(artifactName, "not valid yaml")
			})

			It("fails", func() {
				Expect(fetchChecksumError).To(HaveOccurred())
			})
		})

	})
})

func createTestMetadata(deploymentName, metadata string) {
	Expect(os.MkdirAll(deploymentName, 0777)).To(Succeed())

	file, err := os.Create(deploymentName + "/" + "metadata")
	Expect(err).NotTo(HaveOccurred())

	_, err = file.Write([]byte(metadata))
	Expect(err).NotTo(HaveOccurred())
	Expect(file.Close()).To(Succeed())
}

func createTarWithContents(files map[string]string) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	tarFile := tar.NewWriter(bytesBuffer)

	for filename, contents := range files {
		hdr := &tar.Header{
			Name: filename,
			Mode: 0600,
			Size: int64(len(contents)),
		}
		if err := tarFile.WriteHeader(hdr); err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
		if _, err := tarFile.Write([]byte(contents)); err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
	}
	if err := tarFile.Close(); err != nil {
		Expect(err).NotTo(HaveOccurred())
	}
	Expect(tarFile.Close()).NotTo(HaveOccurred())
	return bytesBuffer.Bytes()
}
