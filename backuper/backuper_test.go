package backuper_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper/fakes"
)

var (
	boshClient *fakes.FakeBoshClient
	b          backuper.Backuper
)
var _ = Describe("Backuper", func() {
	BeforeEach(func() {
		boshClient = new(fakes.FakeBoshClient)
		b = backuper.New(boshClient)
	})
	It("found a deployment", func() {
		boshClient.CheckDeploymentExistsReturns(true, nil)

		Expect(b.Backup("deploymentName")).To(BeNil())

		Expect(boshClient.CheckDeploymentExistsArgsForCall(0)).To(Equal("deploymentName"))
	})

	It("did not find a deployment", func() {
		boshClient.CheckDeploymentExistsReturns(false, nil)

		err := b.Backup("deploymentName")
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("Deployment 'deploymentName' not found")))

		Expect(boshClient.CheckDeploymentExistsArgsForCall(0)).To(Equal("deploymentName"))
	})

	It("had an error", func() {
		boshClient.CheckDeploymentExistsReturns(false, fmt.Errorf("Some error"))
		backuper.New(boshClient)

		Expect(b.Backup("deploymentName")).To(HaveOccurred())

		Expect(boshClient.CheckDeploymentExistsArgsForCall(0)).To(Equal("deploymentName"))
	})
})
