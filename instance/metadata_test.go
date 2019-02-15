package instance_test

import (
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metadata", func() {
	It("can be created with raw metadata YAML", func() {
		rawMetadata := `---
backup_name: foo
restore_name: bar`

		m, err := ParseJobMetadata(rawMetadata)

		Expect(err).NotTo(HaveOccurred())
		Expect(m.BackupName).To(Equal("foo"))
		Expect(m.RestoreName).To(Equal("bar"))
	})

	It("fails when provided invalid YAML", func() {
		rawMetadata := "arrrr"

		_, err := ParseJobMetadata(rawMetadata)

		Expect(err).To(MatchError(ContainSubstring("failed to unmarshal job metadata")))
	})

	It("has an optional `backup_should_be_locked_before` field", func() {
		rawMetadata := `---
backup_name: foo
restore_name: bar
backup_should_be_locked_before:
- job_name: job1
  release: release1
- job_name: job2
  release: release2
`

		m, err := ParseJobMetadata(rawMetadata)

		Expect(err).NotTo(HaveOccurred())
		Expect(m.BackupName).To(Equal("foo"))
		Expect(m.RestoreName).To(Equal("bar"))
		Expect(m.BackupShouldBeLockedBefore).To(ConsistOf(
			LockBefore{JobName: "job1", Release: "release1"}, LockBefore{JobName: "job2", Release: "release2"},
		))
	})

	It("errors if either the job name or release are missing from backup_should_be_locked_before", func() {
		rawMetadata := `---
backup_name: foo
restore_name: bar
backup_should_be_locked_before:
- job_name: job1
  release: release1
- job_name: job2
`

		_, err := ParseJobMetadata(rawMetadata)

		Expect(err).To(MatchError(ContainSubstring("both job name and release should be specified for should be locked before")))
	})

	It("has an optional `restore_should_be_locked_before` field", func() {
		rawMetadata := `---
backup_name: foo
restore_name: bar
restore_should_be_locked_before:
- job_name: job1
  release: release1
- job_name: job2
  release: release2
`

		m, err := ParseJobMetadata(rawMetadata)

		Expect(err).NotTo(HaveOccurred())
		Expect(m.BackupName).To(Equal("foo"))
		Expect(m.RestoreName).To(Equal("bar"))
		Expect(m.RestoreShouldBeLockedBefore).To(ConsistOf(
			LockBefore{JobName: "job1", Release: "release1"}, LockBefore{JobName: "job2", Release: "release2"},
		))
	})

	It("errors if either the job name or release are missing", func() {
		rawMetadata := `---
backup_name: foo
restore_name: bar
restore_should_be_locked_before:
- job_name: job1
  release: release1
- job_name: job2
`

		_, err := ParseJobMetadata(rawMetadata)

		Expect(err).To(MatchError(ContainSubstring("both job name and release should be specified for should be locked before")))
	})
})
