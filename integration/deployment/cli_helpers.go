package deployment

import (
	. "github.com/onsi/gomega"
)

type helpText struct {
	output []byte
}

func (h helpText) outputString() string {
	return string(h.output)
}

func ShowsTheDeploymentHelpText(helpText *helpText) {
	Expect(helpText.outputString()).To(ContainSubstring("--target"))
	Expect(helpText.outputString()).To(ContainSubstring("Target BOSH Director URL"))

	Expect(helpText.outputString()).To(ContainSubstring("--username"))
	Expect(helpText.outputString()).To(ContainSubstring("BOSH Director username"))

	Expect(helpText.outputString()).To(ContainSubstring("--password"))
	Expect(helpText.outputString()).To(ContainSubstring("BOSH Director password"))

	Expect(helpText.outputString()).To(ContainSubstring("--deployment"))
	Expect(helpText.outputString()).To(ContainSubstring("Name of BOSH deployment"))
}
