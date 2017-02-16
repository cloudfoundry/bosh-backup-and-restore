package instance_test

import (
	. "github.com/pivotal-cf/pcf-backup-and-restore/instance"

	. "github.com/onsi/ginkgo"
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
			script = Script("/var/vcap/jobs/a-job-name/p-backup")
		})

		It("returns the job name for a given bosh job script", func() {
			Expect(result).To(Equal("a-job-name"))
		})
	})

	Describe("Name", func() {
		It("returns the job name for a given bosh job script", func() {
			Expect(Script("/var/vcap/jobs/a-job-name/bin/p-backup").Name()).To(Equal("p-backup"))
			Expect(Script("/var/vcap/jobs/a-job-name/bin/p-restore").Name()).To(Equal("p-restore"))
			Expect(Script("/var/vcap/jobs/a-job-name/bin/p-metadata").Name()).To(Equal("p-metadata"))
		})
	})
})
