package backuper_test

import (
	"fmt"

	"github.com/cloudfoundry/bosh-cli/director/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
)

var _ = Describe("Backuper", func() {
	var (
		boshDirector *fakes.FakeDirector
		b            backuper.Backuper
		deployment   *fakes.FakeDeployment
	)

	BeforeEach(func() {
		boshDirector = new(fakes.FakeDirector)
		b = backuper.New(boshDirector)
		deployment = &fakes.FakeDeployment{
			NameStub: func() string { return "deploymentName" },
		}
	})
	It("found a deployment", func() {
		boshDirector.FindDeploymentReturns(deployment, nil)
		deployment.ManifestReturns("not relevant", nil)

		Expect(b.Backup("deploymentName")).To(Not(HaveOccurred()))

		Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal("deploymentName"))
		Expect(deployment.ManifestCallCount()).To(Equal(1))
	})

	It("had an error while fetching manifest", func() {
		boshDirector.FindDeploymentReturns(deployment, nil)
		deployment.ManifestReturns("", fmt.Errorf("Deployment 'deploymentName' not found"))

		err := b.Backup("deploymentName")

		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("Deployment 'deploymentName' not found")))

		Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal("deploymentName"))
		Expect(deployment.ManifestCallCount()).To(Equal(1))
	})

	It("had an error while finding deployment", func() {
		boshDirector.FindDeploymentReturns(deployment, fmt.Errorf("Some error"))

		Expect(b.Backup("deploymentName")).To(HaveOccurred())

		Expect(deployment.ManifestCallCount()).To(Equal(0))
		Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal("deploymentName"))
	})
})
