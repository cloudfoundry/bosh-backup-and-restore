package flags

import (
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func Validate(requiredFlags []string, c *cli.Context) error {
	if containsHelpFlag(c) {
		return nil
	}

	for _, flag := range requiredFlags {
		if c.String(flag) == "" {
			cli.ShowSubcommandHelp(c)
			return redCliError(errors.Errorf("--%v flag is required.", flag))
		}
	}
	return nil
}

func ValidateDeployment(c *cli.Context) error {
	deploymentError := redCliError(errors.New("provide one of '--deployment' or '--all-deployments' flags."))

	if (c.String("deployment") != "" && c.Bool("all-deployments")) || (c.String("deployment") == "" && !c.Bool("all-deployments")) {
		cli.ShowSubcommandHelp(c)
		return deploymentError
	}

	return nil
}

func containsHelpFlag(c *cli.Context) bool {
	for _, arg := range c.Args() {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func redCliError(err error) *cli.ExitError {
	return cli.NewExitError(ansi.Color(err.Error(), "red"), 1)
}
