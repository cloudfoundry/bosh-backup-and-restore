package backuper_test

import (
	. "github.com/pivotal-cf/pcf-backup-and-restore/backuper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backuper/Checksum", func() {
	Describe("Match", func() {
		It("returns false if checksums don't match", func() {
			Expect(BackupChecksum{"foo": "bar"}.Match(BackupChecksum{"foo": "baz"})).To(BeFalse())
		})

		It("returns true if checksums match", func() {
			Expect(BackupChecksum{"foo": "bar"}.Match(BackupChecksum{"foo": "bar"})).To(BeTrue())
		})

		It("returns false if keys dont match", func() {
			Expect(BackupChecksum{"foo": "bar"}.Match(BackupChecksum{"foo": "bar", "extra": "nope"})).To(BeFalse())
		})
	})
})
