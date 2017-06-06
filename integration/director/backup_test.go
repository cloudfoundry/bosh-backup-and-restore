package director

import (
	"io/ioutil"
	"os"

	. "github.com/pivotal-cf/bosh-backup-and-restore/integration"
	"github.com/pivotal-cf/bosh-backup-and-restore/testcluster"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"path"

	"archive/tar"
	"io"
	"time"

	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backup", func() {
	var backupWorkspace string
	var session *gexec.Session
	var directorIP string

	BeforeEach(func() {
		var err error
		backupWorkspace, err = ioutil.TempDir(".", "backup-workspace-")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(backupWorkspace)).To(Succeed())
	})

	JustBeforeEach(func() {
		session = binary.Run(
			backupWorkspace,
			[]string{"BOSH_CLIENT_SECRET=admin"},
			"director",
			"--artifactname", "my-director",
			"--host", directorIP,
			"--username", "foobar",
			"--private-key-path", pathToPrivateKeyFile,
			"--debug",
			"backup",
		)
	})

	Context("When there is a director instance", func() {
		var directorInstance *testcluster.Instance

		BeforeEach(func() {
			directorInstance = testcluster.NewInstance()
			directorInstance.CreateUser("foobar", readFile(pathToPublicKeyFile))
			directorIP = directorInstance.Address()
		})

		AfterEach(func() {
			directorInstance.DieInBackground()
		})

		Context("and there is a backup script", func() {
			BeforeEach(func() {
				directorInstance.CreateFiles("/var/vcap/jobs/bosh/bin/bbr/backup")
			})

			Context("and the backup script succeeds", func() {
				BeforeEach(func() {
					directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/backup", `#!/usr/bin/env sh
set -u
printf "backupcontent1" > $BBR_ARTIFACT_DIRECTORY/backupdump1
printf "backupcontent2" > $BBR_ARTIFACT_DIRECTORY/backupdump2
`)
				})

				It("successfully backs up the director", func() {
					By("exiting zero", func() {
						Expect(session.ExitCode()).To(BeZero())
					})

					backupFolderPath := path.Join(backupWorkspace, "my-director")
					boshBackupFilePath := path.Join(backupFolderPath, "/bosh-0-bosh.tar")
					metadataFilePath := path.Join(backupFolderPath, "/metadata")

					By("creating a backup directory which contains a backup artifact and a metadata file", func() {
						Expect(backupFolderPath).To(BeADirectory())
						Expect(boshBackupFilePath).To(BeARegularFile())
						Expect(metadataFilePath).To(BeARegularFile())
					})

					By("having successfully run the backup script, using the $BBR_ARTIFACT_DIRECTORY variable", func() {
						Expect(filesInTar(boshBackupFilePath)).To(ConsistOf("backupdump1", "backupdump2"))
						Expect(contentsInTar(boshBackupFilePath, "backupdump1")).To(Equal("backupcontent1"))
						Expect(contentsInTar(boshBackupFilePath, "backupdump2")).To(Equal("backupcontent2"))
					})

					By("correctly populating the metadata file", func() {
						metadataContents := ParseMetadata(metadataFilePath)

						currentTimezone, _ := time.Now().Zone()
						Expect(metadataContents.BackupActivityMetadata.StartTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))
						Expect(metadataContents.BackupActivityMetadata.FinishTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))

						Expect(metadataContents.InstancesMetadata).To(HaveLen(1))
						Expect(metadataContents.InstancesMetadata[0].InstanceName).To(Equal("bosh"))
						Expect(metadataContents.InstancesMetadata[0].InstanceIndex).To(Equal("0"))

						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Name).To(Equal("bosh"))
						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums).To(HaveLen(2))
						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums["./bosh/backupdump1"]).To(Equal(ShaFor("backupcontent1")))
						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums["./bosh/backupdump2"]).To(Equal(ShaFor("backupcontent2")))

						Expect(metadataContents.CustomArtifactsMetadata).To(BeEmpty())
					})

					By("printing the backup progress to the screen", func() {
						Expect(session.Out).To(gbytes.Say(fmt.Sprintf("INFO - Running pre-checks for backup of my-director...")))
						//Expect(session.Out).To(gbytes.Say("INFO - Scripts found:"))
						Expect(session.Out).To(gbytes.Say("INFO - bosh/bosh/backup"))
						Expect(session.Out).To(gbytes.Say(fmt.Sprintf("INFO - Starting backup of my-director...")))
						Expect(session.Out).To(gbytes.Say("INFO - Running pre-backup scripts..."))
						Expect(session.Out).To(gbytes.Say("INFO - Done."))
						Expect(session.Out).To(gbytes.Say("INFO - Running backup scripts..."))
						Expect(session.Out).To(gbytes.Say("INFO - Backing up bosh on bosh/0..."))
						Expect(session.Out).To(gbytes.Say("INFO - Done."))
						Expect(session.Out).To(gbytes.Say("INFO - Running post-backup scripts..."))
						Expect(session.Out).To(gbytes.Say("INFO - Done."))
						Expect(session.Out).To(gbytes.Say("INFO - Copying backup -- [^-]*-- from bosh/0..."))
						Expect(session.Out).To(gbytes.Say("INFO - Finished copying backup -- from bosh/0..."))
						Expect(session.Out).To(gbytes.Say("INFO - Starting validity checks"))
						//Expect(session.Out).To(gbytes.Say(`DEBUG - Calculating shasum for local file ./bosh/backupdump[12]`))
						//Expect(session.Out).To(gbytes.Say(`DEBUG - Calculating shasum for local file ./bosh/backupdump[12]`))
						//Expect(session.Out).To(gbytes.Say("DEBUG - Calculating shasum for remote files"))
						//Expect(session.Out).To(gbytes.Say("DEBUG - Comparing shasums"))
						Expect(session.Out).To(gbytes.Say("INFO - Finished validity checks"))
					})

					By("cleaning up backup artifacts from the remote", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
					})
				})
			})

			Context("but the backup script fails", func() {
				BeforeEach(func() {
					directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/backup", "echo 'NOPE!'; exit 1")
				})

				It("fails to backup the director", func() {
					By("returning exit code 1", func() {
						Expect(session.ExitCode()).To(Equal(1))
					})
				})
			})

			Context("but the backup artifact directory already exists", func() {
				BeforeEach(func() {
					directorInstance.CreateDir("/var/vcap/store/bbr-backup")
				})

				It("fails to backup the director", func() {
					By("exiting non-zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					By("printing a log message saying the director instance cannot be backed up", func() {
						Expect(string(session.Err.Contents())).To(ContainSubstring("Directory /var/vcap/store/bbr-backup already exists on instance bosh/0"))
					})

					By("not deleting the existing artifact directory", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeTrue())
					})
				})
			})
		})

		Context("if there are no backup scripts", func() {
			BeforeEach(func() {
				directorInstance.CreateFiles("/var/vcap/jobs/bosh/bin/bbr/not-a-backup-script")
			})

			It("fails to backup the director", func() {
				By("returning exit code 1", func() {
					Expect(session.ExitCode()).To(Equal(1))
				})

				By("printing an error", func() {
					Expect(string(session.Err.Contents())).To(ContainSubstring("Deployment 'my-director' has no backup scripts"))
				})
			})
		})
	})

	Context("When the director does not resolve", func() {
		BeforeEach(func() {
			directorIP = "no:22"
		})

		It("fails to backup the director", func() {
			By("returning exit code 1", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			By("printing an error", func() {
				Expect(string(session.Err.Contents())).To(ContainSubstring("no such host"))
			})
		})
	})
})

func getTarReader(path string) *tar.Reader {
	reader, err := os.Open(path)
	Expect(err).NotTo(HaveOccurred())
	tarReader := tar.NewReader(reader)
	return tarReader
}

func filesInTar(path string) []string {
	tarReader := getTarReader(path)

	filenames := []string{}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
		info := header.FileInfo()
		if !info.IsDir() {
			filenames = append(filenames, info.Name())
		}
	}
	return filenames
}

func contentsInTar(tarFile, file string) string {
	tarReader := getTarReader(tarFile)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
		info := header.FileInfo()
		if !info.IsDir() && info.Name() == file {
			contents, err := ioutil.ReadAll(tarReader)
			Expect(err).NotTo(HaveOccurred())
			return string(contents)
		}
	}
	Fail("File " + file + " not found in tar " + tarFile)
	return ""
}
