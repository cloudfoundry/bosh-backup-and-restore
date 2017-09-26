package command

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/mgutz/ansi"
	"github.com/urfave/cli"
)

type DeploymentPreBackupCheck struct{}

func NewDeploymentPreBackupCheckCommand() DeploymentPreBackupCheck {
	return DeploymentPreBackupCheck{}
}
func (d DeploymentPreBackupCheck) Cli() cli.Command {
	return cli.Command{
		Name:    "pre-backup-check",
		Aliases: []string{"c"},
		Usage:   "Check a deployment can be backed up",
		Action:  d.Action,
	}
}
func (d DeploymentPreBackupCheck) Action(c *cli.Context) error {
	var deployment = c.Parent().String("deployment")
	backuper, err := factory.BuildDeploymentBackupChecker(
		c.Parent().String("target"),
		c.Parent().String("username"),
		c.Parent().String("password"),
		c.Parent().String("ca-cert"),
		c.GlobalBool("debug"),
		c.Bool("with-manifest"))

	if err != nil {
		return redCliError(err)
	}

	backupable, checkErr := backuper.CanBeBackedUp(deployment)

	if backupable {
		fmt.Printf("Deployment '%s' can be backed up.\n", deployment)
		return cli.NewExitError("", 0)
	} else {
		fmt.Printf("Deployment '%s' cannot be backed up.\n", deployment)
		writeStackTrace(checkErr.PrettyError(true))
		return cli.NewExitError(checkErr.Error(), 1)
	}
}

func redCliError(err error) *cli.ExitError {
	return cli.NewExitError(ansi.Color(err.Error(), "red"), 1)
}
