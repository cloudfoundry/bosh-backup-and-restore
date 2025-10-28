package instance_test

import (
	. "github.com/cloudfoundry/bosh-backup-and-restore/instance"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Script", func() {
	var (
		script Script
		result string
	)

	JustBeforeEach(func() {
		result = script.JobName()
	})

	Describe("JobName", func() {
		BeforeEach(func() {
			script = Script("/var/vcap/jobs/a-job-name/backup")
		})

		It("returns the job name for a given bosh job script", func() {
			Expect(result).To(Equal("a-job-name"))
		})
	})

	Describe("Name", func() {
		It("returns the job name for a given bosh job script", func() {
			Expect(Script("/var/vcap/jobs/a-job-name/bin/bbr/backup").Name()).To(Equal("backup"))
			Expect(Script("/var/vcap/jobs/a-job-name/bin/bbr/pre-restore-lock").Name()).To(Equal("pre-restore-lock"))
			Expect(Script("/var/vcap/jobs/a-job-name/bin/bbr/restore").Name()).To(Equal("restore"))
			Expect(Script("/var/vcap/jobs/a-job-name/bin/bbr/post-restore-lock").Name()).To(Equal("post-restore-lock"))
			Expect(Script("/var/vcap/jobs/a-job-name/bin/bbr/metadata").Name()).To(Equal("metadata"))
		})
	})
})
