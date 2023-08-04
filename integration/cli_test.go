package integration

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cli", func() {
	Context("bbr with no arguments", func() {
		It("displays the usable flags", func() {
			output := string(binary.Run("", []string{""}).Out.Contents())

			showsTheMainHelpText(output)
		})
	})

	Context("bbr with an unknown command", func() {
		It("displays an error and the help text", func() {
			output := string(binary.Run("", []string{""}, "rubbish").Out.Contents())

			Expect(output).To(ContainSubstring("Error command 'rubbish' not found"))
			showsTheMainHelpText(output)
		})
	})

	Context("help", func() {
		It("displays an error and the help text", func() {
			output := string(binary.Run("", []string{""}, "help").Out.Contents())

			showsTheMainHelpText(output)
		})
	})

	Context("version", func() {
		It("displays an error and the help text", func() {
			output := string(binary.Run("", []string{""}, "version").Out.Contents())

			Expect(output).To(ContainSubstring(fmt.Sprintf("bbr version %s", version)))
		})
	})
})

func showsTheMainHelpText(output string) {
	Expect(output).To(ContainSubstring(`SUBCOMMANDS:
   backup
   backup-cleanup
   restore
   restore-cleanup
   pre-backup-check`))

	Expect(output).To(ContainSubstring(`USAGE:
   bbr command [command options] [subcommand] [subcommand options]`))
}
