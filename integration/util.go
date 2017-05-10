package integration

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
)

type helpText struct {
	output []byte
}

func (h helpText) outputString() string {
	return string(h.output)
}

func ShowsTheHelpText(helpText *helpText) {

	It("displays the usable flags", func() {
		Expect(helpText.outputString()).To(ContainSubstring("--target"))
		Expect(helpText.outputString()).To(ContainSubstring("Target BOSH Director URL"))

		Expect(helpText.outputString()).To(ContainSubstring("--username"))
		Expect(helpText.outputString()).To(ContainSubstring("BOSH Director username"))

		Expect(helpText.outputString()).To(ContainSubstring("--password"))
		Expect(helpText.outputString()).To(ContainSubstring("BOSH Director password"))

		Expect(helpText.outputString()).To(ContainSubstring("--deployment"))
		Expect(helpText.outputString()).To(ContainSubstring("Name of BOSH deployment"))
	})
}

func mockDirectorWith(director *mockhttp.Server, info mockhttp.MockedResponseBuilder, vmsResponse []mockhttp.MockedResponseBuilder, sshResponse []mockhttp.MockedResponseBuilder, downloadManifestResponse []mockhttp.MockedResponseBuilder, cleanupResponse []mockhttp.MockedResponseBuilder) {
	director.VerifyAndMock(AppendBuilders(
		[]mockhttp.MockedResponseBuilder{info},
		vmsResponse,
		sshResponse,
		downloadManifestResponse,
		cleanupResponse,
	)...)

}
