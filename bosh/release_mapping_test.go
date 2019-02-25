package bosh_test

import (
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewBoshManifestReleaseMapping", func() {
	Context("FindReleaseName", func() {
		It("parses a v2 manifest and finds a release name associated with an instance group and job", func() {
			var manifest = `---
instance_groups:
- name: red1
  instances: 1
  jobs:
  - name: redis-server
    release: redis
`
			releaseMapping, err := NewBoshManifestReleaseMapping(manifest)
			Expect(err).NotTo(HaveOccurred())
			Expect(releaseMapping.FindReleaseName("red1", "redis-server")).To(Equal("redis"))
		})

		It("parses a v1 manifest and finds a release name associated with an instance group and job", func() {
			manifest := `---
jobs:
- name: red1
  instances: 1
  templates:
  - name: redis-server
    release: redis
`
			releaseMapping, err := NewBoshManifestReleaseMapping(manifest)
			Expect(err).NotTo(HaveOccurred())
			Expect(releaseMapping.FindReleaseName("red1", "redis-server")).To(Equal("redis"))
		})

		It("parses a manifest with two jobs from the same release correctly", func() {
			var manifest = `---
instance_groups:
- name: red1
  instances: 1
  jobs:
  - name: redis-server
    release: redis
- name: red2
  instances: 1
  jobs:
  - name: redis-client
    release: redis
`
			releaseMapping, err := NewBoshManifestReleaseMapping(manifest)
			Expect(err).NotTo(HaveOccurred())

			Expect(releaseMapping.FindReleaseName("red1", "redis-server")).To(Equal("redis"))
			Expect(releaseMapping.FindReleaseName("red2", "redis-client")).To(Equal("redis"))
		})

		It("parses a manifest with two jobs from the same instance group", func() {
			manifest := `---
instance_groups:
- name: red1
  instances: 1
  jobs:
  - name: redis-server
    release: redis
- name: red2
  instances: 1
  jobs:
  - name: redis-server
    release: redis
  - name: redis-client
    release: redis
`
			releaseMapping, err := NewBoshManifestReleaseMapping(manifest)
			Expect(err).NotTo(HaveOccurred())

			Expect(releaseMapping.FindReleaseName("red2", "redis-client")).To(Equal("redis"))
			Expect(releaseMapping.FindReleaseName("red2", "redis-server")).To(Equal("redis"))
		})

		It("parses a manifest with two identically-named jobs from different releases", func() {
			manifest := `---
instance_groups:
- name: red1
  instances: 1
  jobs:
  - name: redis-server
    release: redis-2.0
- name: red2
  instances: 1
  jobs:
  - name: redis-server
    release: redis-2.5
`

			releaseMapping, err := NewBoshManifestReleaseMapping(manifest)
			Expect(err).NotTo(HaveOccurred())

			Expect(releaseMapping.FindReleaseName("red1", "redis-server")).To(Equal("redis-2.0"))
			Expect(releaseMapping.FindReleaseName("red2", "redis-server")).To(Equal("redis-2.5"))
		})

		It("errors when trying to find release name for a missing instance group name", func() {
			manifest := `---
instance_groups:
- name: red1
  instances: 1
  jobs:
  - name: redis-server
    release: redis-2.0
`
			releaseMapping, err := NewBoshManifestReleaseMapping(manifest)
			Expect(err).NotTo(HaveOccurred())

			_, err = releaseMapping.FindReleaseName("red2", "redis-server")
			Expect(err).To(MatchError(ContainSubstring("error finding release name for job")))
		})

		It("errors when trying to find release name for a missing job name", func() {
			manifest := `---
instance_groups:
- name: red1
  instances: 1
  jobs:
  - name: redis-server
    release: redis-2.0
`
			releaseMapping, err := NewBoshManifestReleaseMapping(manifest)
			Expect(err).NotTo(HaveOccurred())

			_, err = releaseMapping.FindReleaseName("red1", "redis-client")
			Expect(err).To(MatchError(ContainSubstring("error finding release name for job")))
		})
	})

	FContext("IsJobBackupOneRestoreAll", func() {
		It("parses a v1 manifest and finds the bbr.backup_one_restore_all property for an instance group and job", func() {
			var manifest = `---
jobs:
- name: red1
  instances: 1
  templates:
  - name: redis-server
    release: redis
    properties:
      bbr:
        backup_one_restore_all: false
`

			releaseMapping, err := NewBoshManifestReleaseMapping(manifest)
			Expect(err).ToNot(HaveOccurred())

			backupOneRestoreAll, err := releaseMapping.IsJobBackupOneRestoreAll("red1", "redis-server")
			Expect(err).ToNot(HaveOccurred())

			Expect(backupOneRestoreAll).To(BeFalse())
		})

		It("parses a v2 manifest and finds the bbr.backup_one_restore_all property for an instance group and job", func() {
			var manifest = `---
instance_groups:
- name: red1
  instances: 1
  jobs:
  - name: redis-server
    release: redis
    properties:
      bbr:
        backup_one_restore_all: true
- name: red2
  instances: 1
  jobs:
  - name: redis-server
    release: redis
    properties:
      bbr:
        backup_one_restore_all: false
  - name: redis-client
    release: redis
    properties:
      bbr:
        backup_one_restore_all: true
`
			releaseMapping, err := NewBoshManifestReleaseMapping(manifest)
			Expect(err).NotTo(HaveOccurred())

			Expect(releaseMapping.IsJobBackupOneRestoreAll("red1", "redis-server")).To(BeTrue())
			Expect(releaseMapping.IsJobBackupOneRestoreAll("red2", "redis-server")).To(BeFalse())
			Expect(releaseMapping.IsJobBackupOneRestoreAll("red2", "redis-client")).To(BeTrue())
		})

		It("errors when trying to find release name for a missing instance group name", func() {
			manifest := `---
instance_groups:
- name: red1
  instances: 1
  jobs:
  - name: redis-server
    release: redis-2.0
    properties:
      bbr:
        backup_one_restore_all: true

`
			releaseMapping, err := NewBoshManifestReleaseMapping(manifest)
			Expect(err).NotTo(HaveOccurred())

			_, err = releaseMapping.IsJobBackupOneRestoreAll("red2", "redis-server")
			Expect(err).To(MatchError(ContainSubstring("error finding job redis-server in instance group red2")))
		})

		It("returns false if the 'backup_one_restore_all' is not set", func() {
			manifest := `---
instance_groups:
- name: red1
  instances: 1
  jobs:
  - name: redis-server
    release: redis-2.0
    properties:
      bbr:
        something_else: true
`
			releaseMapping, err := NewBoshManifestReleaseMapping(manifest)
			Expect(err).NotTo(HaveOccurred())

			Expect(releaseMapping.IsJobBackupOneRestoreAll("red1", "redis-server")).To(BeFalse())
		})
	})

	It("errors when manifest is not valid yaml", func() {
		manifest := "% THIS IS NOT VALID YAML %"

		_, err := NewBoshManifestReleaseMapping(manifest)
		Expect(err).To(MatchError(ContainSubstring("error unmarshalling manifest yaml")))
	})
})
