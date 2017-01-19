package artifact_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/pivotal-cf/pcf-backup-and-restore/artifact"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper/fakes"

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
		var artifact backuper.Artifact
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
				match, _ := artifact.DeploymentMatches(artifactName, []backuper.Instance{instance1, instance2})
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
				match, _ := artifact.DeploymentMatches(artifactName, []backuper.Instance{instance1, instance2})
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
				_, err := artifact.DeploymentMatches(artifactName, []backuper.Instance{instance1, instance2})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when an error occurs checking if the file exists", func() {
			It("returns error", func() {
				_, err := artifact.DeploymentMatches(artifactName, []backuper.Instance{instance1, instance2})
				Expect(err).To(HaveOccurred())
			})
		})
	})
	Describe("Verify", func() {
		var artifact backuper.Artifact
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

		Context("when the artifact sha's match metafile", func() {
			BeforeEach(func() {
				contents := gzipContents(createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
					"file2": "Gopher names:\nGeorge\nGeoffrey\nGonzo",
				}))
				Expect(ioutil.WriteFile(artifactName+"/redis-0.tgz", contents, 0666)).NotTo(HaveOccurred())
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

		Context("when one of the artifact file's contents don't match the sha", func() {
			BeforeEach(func() {
				contents := gzipContents(createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
					"file2": "Gopher names:\nGeorge\nGeoffrey\nGonzo",
				}))
				Expect(ioutil.WriteFile(artifactName+"/redis-0.tgz", contents, 0666)).NotTo(HaveOccurred())
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
				contents := gzipContents(createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
				}))
				Expect(ioutil.WriteFile(artifactName+"/redis-0.tgz", contents, 0666)).NotTo(HaveOccurred())
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
				contents := gzipContents(createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
				}))
				Expect(ioutil.WriteFile(artifactName+"/redis-1.tgz", contents, 0666)).NotTo(HaveOccurred())
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
		var artifact backuper.Artifact
		var fileCreationError error
		var writer io.Writer
		var fakeInstance *fakes.FakeInstance

		BeforeEach(func() {
			artifact, _ = artifactManager.Create(artifactName, logger)
			fakeInstance = new(fakes.FakeInstance)
			fakeInstance.IndexReturns("0")
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
	Describe("SaveManifest", func() {
		var artifact backuper.Artifact
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
		var artifact backuper.Artifact
		var fileReadError error
		var reader io.Reader
		var fakeInstance *fakes.FakeInstance

		BeforeEach(func() {
			artifact, _ = artifactManager.Open(artifactName, logger)
			fakeInstance = new(fakes.FakeInstance)
			fakeInstance.IndexReturns("0")
			fakeInstance.NameReturns("redis")
		})

		Context("File exists and is readable", func() {
			BeforeEach(func() {
				err := os.MkdirAll(artifactName, 0700)
				Expect(err).NotTo(HaveOccurred())
				_, err = os.Create(artifactName + "/redis-0.tgz")
				Expect(err).NotTo(HaveOccurred())
				err = ioutil.WriteFile(artifactName+"/redis-0.tgz", []byte("backup-content"), 0700)
				Expect(err).NotTo(HaveOccurred())
			})

			JustBeforeEach(func() {
				reader, fileReadError = artifact.ReadFile(fakeInstance)
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
				_, fileReadError = artifact.ReadFile(fakeInstance)
				Expect(fileReadError).To(HaveOccurred())
			})
		})
	})

	Describe("Checksum", func() {
		var artifact backuper.Artifact
		var fakeInstance *fakes.FakeInstance

		BeforeEach(func() {
			artifact, _ = artifactManager.Create(artifactName, logger)
			fakeInstance = new(fakes.FakeInstance)
			fakeInstance.IndexReturns("0")
			fakeInstance.NameReturns("redis")
		})
		Context("file exists", func() {
			JustBeforeEach(func() {
				writer, fileCreationError := artifact.CreateFile(fakeInstance)
				Expect(fileCreationError).NotTo(HaveOccurred())

				contents := gzipContents(createTarWithContents(map[string]string{
					"readme.txt": "This archive contains some text files.",
					"gopher.txt": "Gopher names:\nGeorge\nGeoffrey\nGonzo",
					"todo.txt":   "Get animal handling license.",
				}))

				writer.Write(contents)
				Expect(writer.Close()).NotTo(HaveOccurred())
			})

			It("returns the checksum for the saved instance data", func() {
				Expect(artifact.CalculateChecksum(fakeInstance)).To(Equal(
					backuper.BackupChecksum{
						"readme.txt": fmt.Sprintf("%x", sha1.Sum([]byte("This archive contains some text files."))),
						"gopher.txt": fmt.Sprintf("%x", sha1.Sum([]byte("Gopher names:\nGeorge\nGeoffrey\nGonzo"))),
						"todo.txt":   fmt.Sprintf("%x", sha1.Sum([]byte("Get animal handling license."))),
					}))
			})
		})
		Context("invalid tar file", func() {
			JustBeforeEach(func() {
				writer, fileCreationError := artifact.CreateFile(fakeInstance)
				Expect(fileCreationError).NotTo(HaveOccurred())

				contents := gzipContents([]byte("this ain't a tarball"))

				writer.Write(contents)
				Expect(writer.Close()).NotTo(HaveOccurred())
			})

			It("fails to read", func() {
				_, err := artifact.CalculateChecksum(fakeInstance)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("invalid gz file", func() {
			JustBeforeEach(func() {
				writer, fileCreationError := artifact.CreateFile(fakeInstance)
				Expect(fileCreationError).NotTo(HaveOccurred())

				contents := createTarWithContents(map[string]string{
					"readme.txt": "This archive contains some text files.",
				})

				writer.Write(contents)
				Expect(writer.Close()).NotTo(HaveOccurred())
			})

			It("fails to read", func() {
				_, err := artifact.CalculateChecksum(fakeInstance)
				Expect(err).To(HaveOccurred())
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
		var artifact backuper.Artifact
		var addChecksumError error
		var fakeInstance *fakes.FakeInstance
		var checksum map[string]string

		BeforeEach(func() {
			artifact, _ = artifactManager.Create(artifactName, logger)
			fakeInstance = new(fakes.FakeInstance)
			fakeInstance.IndexReturns("0")
			fakeInstance.NameReturns("redis")
			checksum = map[string]string{"filename": "foobar"}
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
  instance_index: "0"
  checksums:
    filename: foobar`
				Expect(ioutil.ReadFile(artifactName + "/metadata")).To(MatchYAML(expectedMetadata))
			})
		})
		Context("Appends to a checksum file, if already exists", func() {
			BeforeEach(func() {
				firstInstance := new(fakes.FakeInstance)
				firstInstance.IndexReturns("0")
				firstInstance.NameReturns("broker")
				Expect(artifact.AddChecksum(firstInstance, map[string]string{"filename1": "orignal_checksum"})).NotTo(HaveOccurred())
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
		var artifact backuper.Artifact
		var fetchChecksumError error
		var fakeInstance *fakes.FakeInstance
		var checksum backuper.BackupChecksum
		BeforeEach(func() {
			fakeInstance = new(fakes.FakeInstance)
		})
		JustBeforeEach(func() {
			var artifactOpenError error
			artifact, artifactOpenError = artifactManager.Open(artifactName, logger)
			Expect(artifactOpenError).NotTo(HaveOccurred())

			checksum, fetchChecksumError = artifact.FetchChecksum(fakeInstance)
		})
		Context("the instance is found in metadata", func() {
			BeforeEach(func() {
				fakeInstance.NameReturns("foo")
				fakeInstance.IndexReturns("bar")

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
				Expect(checksum).To(Equal(backuper.BackupChecksum{"filename1": "orignal_checksum"}))
			})
		})
		Context("the instance is not found in metadata", func() {
			BeforeEach(func() {
				fakeInstance.NameReturns("not-foo")
				fakeInstance.IndexReturns("bar")

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

func deleteTestMetadata(deploymentName string) {
	Expect(os.RemoveAll(deploymentName)).To(Succeed())
}
func gzipContents(contents []byte) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	gzipStream := gzip.NewWriter(bytesBuffer)
	gzipStream.Write(contents)

	Expect(gzipStream.Close()).NotTo(HaveOccurred())
	return bytesBuffer.Bytes()
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
