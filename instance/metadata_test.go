package instance_test

import (
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metadata", func() {
	It("can be created with raw metadata YAML", func() {
		rawMetadata := []byte(`---
backup:
  name: foo
restore:
  name: bar`)

		m, err := NewJobMetadata(rawMetadata)

		Expect(err).NotTo(HaveOccurred())
		Expect(m.Backup.Name).To(Equal("foo"))
		Expect(m.Restore.Name).To(Equal("bar"))
	})

	It("fails when provided invalid YAML", func() {
		rawMetadata := []byte(`arrrr`)

		_, err := NewJobMetadata(rawMetadata)

		Expect(err).To(MatchError(ContainSubstring("failed to unmarshal job metadata")))
	})

	It("has an optional `should_be_locked_before` field", func() {
		rawMetadata := []byte(`---
backup:
  name: foo
  should_be_locked_before:
  - job_name: job1
    release: release1
  - job_name: job2
    release: release2
restore:
  name: bar
`)

		m, err := NewJobMetadata(rawMetadata)

		Expect(err).NotTo(HaveOccurred())
		Expect(m.Backup.Name).To(Equal("foo"))
		Expect(m.Restore.Name).To(Equal("bar"))
		Expect(m.Backup.ShouldBeLockedBefore).To(ConsistOf(
			LockBefore{JobName: "job1", Release: "release1"}, LockBefore{JobName: "job2", Release: "release2"},
		))
	})

	It("errors if either the job name or release are missing", func() {
		rawMetadata := []byte(`---
backup:
  name: foo
  should_be_locked_before:
  - job_name: job1
    release: release1
  - job_name: job2
restore:
  name: bar
`)

		_, err := NewJobMetadata(rawMetadata)

		Expect(err).To(HaveOccurred())
	})
})
