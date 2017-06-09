package backup_test

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf/bosh-backup-and-restore/backup"
)

var _ = Context("BackupManager", func() {
	var artifactName string
	var artifactManager = BackupDirectoryManager{}
	var err error

	BeforeEach(func() {
		artifactName = fmt.Sprintf("my-cool-redis-%d", config.GinkgoConfig.ParallelNode)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(artifactName)).To(Succeed())
	})

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
				Expect(artifactName).NotTo(BeADirectory())
				_, err := artifactManager.Open(artifactName, nil)
				Expect(err).To(MatchError(ContainSubstring("failed opening the directory")))
			})
		})
	})

	Describe("Exists", func() {
		Context("when the artifact exists", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(artifactName, 0777)).To(Succeed())
			})

			It("returns true", func() {
				Expect(artifactManager.Exists(artifactName)).To(BeTrue())
			})
		})

		Context("when the artifact doesn't exist", func() {
			It("returns false", func() {
				Expect(artifactManager.Exists(artifactName)).To(BeFalse())
			})
		})
	})
})
