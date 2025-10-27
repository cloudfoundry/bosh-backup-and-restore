package orchestrator_test

import (
	. "github.com/cloudfoundry/bosh-backup-and-restore/orchestrator"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backuper/Checksum", func() {
	var match bool
	var files []string

	Describe("Match", func() {
		Context("if the checksums don't match", func() {

			BeforeEach(func() {
				match, files = BackupChecksum{
					"/var/foo": "checksum1",
					"/var/bar": "checksum2",
					"/var/baz": "checksum3",
				}.Match(BackupChecksum{
					"/var/foo": "checksum11111111",
					"/var/bar": "checksum22222222",
					"/var/baz": "checksum3",
				})
			})

			It("returns false", func() {
				Expect(match).To(BeFalse())
			})

			It("returns a list of files whose checksums don't match", func() {
				Expect(files).To(ConsistOf("/var/foo", "/var/bar"))
			})
		})

		Context("if the checksums match", func() {
			It("returns true", func() {
				match, _ := BackupChecksum{"/var/foo": "bar"}.Match(BackupChecksum{"/var/foo": "bar"})
				Expect(match).To(BeTrue())
			})
		})

		Context("if there are extra keys", func() {
			It("returns false", func() {
				match, _ := BackupChecksum{"/var/foo": "bar"}.Match(BackupChecksum{"/var/foo": "bar", "/tmp/some-extra-thing": "nope"})
				Expect(match).To(BeFalse())
			})

			Context("and the checksums don't match", func() {
				BeforeEach(func() {
					match, files = BackupChecksum{"/var/foo": "bar"}.Match(BackupChecksum{"/var/foo": "baz", "extra": "nope"})
				})

				It("returns false", func() {
					Expect(match).To(BeFalse())
				})

				It("returns a list of files whose checksums don't match", func() {
					Expect(files).To(ConsistOf("/var/foo"))
				})
			})
		})
	})
})
