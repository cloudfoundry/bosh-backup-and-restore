package main

import (
	"os"

	"github.com/urfave/cli"

	"fmt"

	"github.com/cloudfoundry/bosh-backup-and-restore/cli/command"
	"github.com/cloudfoundry/bosh-backup-and-restore/cli/flags"
)

var version string

func main() {
	cli.AppHelpTemplate = helpTextTemplate
	cli.SubcommandHelpTemplate = subCommandTextTemplate
	cli.CommandHelpTemplate = commandTextTemplate

	app := cli.NewApp()

	app.Version = version
	app.Name = "bbr"
	app.Usage = "BOSH Backup and Restore"
	app.HideHelp = true
	app.CommandNotFound = commandNotFoundFunc

	app.Commands = []cli.Command{
		{
			Name:   "deployment",
			Usage:  "Backup BOSH deployments",
			Flags:  availableDeploymentFlags(),
			Before: validateDeploymentFlags,
			Subcommands: []cli.Command{
				command.NewDeploymentPreBackupCheckCommand().Cli(),
				command.NewDeploymentBackupCommand().Cli(),
				command.NewDeploymentRestoreCommand().Cli(),
				command.NewDeploymentBackupCleanupCommand().Cli(),
				command.NewDeploymentRestoreCleanupCommand().Cli(),
			},
		},
		{
			Name:   "director",
			Usage:  "Backup BOSH director",
			Flags:  availableDirectorFlags(),
			Before: validateDirectorFlags,
			Subcommands: []cli.Command{
				command.NewDirectorPreBackupCheckCommand().Cli(),
				command.NewDirectorBackupCommand().Cli(),
				command.NewDirectorRestoreCommand().Cli(),
				command.NewDirectorBackupCleanupCommand().Cli(),
				command.NewDirectorRestoreCleanupCommand().Cli(),
			},
		},
		{
			Name:    "help",
			Aliases: []string{"h"},
			Usage:   "Shows a list of commands or help for one command",
			Action:  versionAction,
		},
		{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "Shows the version",
			Action: func(c *cli.Context) error {
				cli.ShowVersion(c)
				return nil
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}

func commandNotFoundFunc(c *cli.Context, msg string) {
	fmt.Printf("Error command '%s' not found\n\n", msg)
	cli.ShowAppHelp(c) //nolint:errcheck
}

func versionAction(c *cli.Context) error {
	cli.ShowAppHelp(c) //nolint:errcheck
	return nil
}

func validateDeploymentFlags(c *cli.Context) error {
	err := flags.Validate([]string{"target", "username", "password"}, c)
	if err != nil {
		return err
	}

	err = flags.ValidateDeployment(c)
	if err != nil {
		return err
	}

	return nil
}

func validateDirectorFlags(c *cli.Context) error {
	return flags.Validate([]string{"host", "username", "private-key-path"}, c)
}

func availableDeploymentFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:   "target, t",
			Value:  "",
			EnvVar: "BOSH_ENVIRONMENT",
			Usage:  "BOSH Director URL",
		},
		cli.StringFlag{
			Name:   "username, u",
			Value:  "",
			EnvVar: "BOSH_CLIENT",
			Usage:  "BOSH Director username",
		},
		cli.StringFlag{
			Name:   "password, p",
			Value:  "",
			EnvVar: "BOSH_CLIENT_SECRET",
			Usage:  "BOSH Director password",
		},
		cli.StringFlag{
			Name:   "deployment, d",
			Value:  "",
			EnvVar: "BOSH_DEPLOYMENT",
			Usage:  "Name of BOSH deployment. Omit if '--all-deployments' is provided",
		},
		cli.StringFlag{
			Name:   "ca-cert",
			Value:  "",
			EnvVar: "CA_CERT,BOSH_CA_CERT",
			Usage:  "Path or value of BOSH Director custom CA certificate",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logs",
		},
		cli.BoolFlag{
			Name:  "all-deployments",
			Usage: "Run command for all deployments. Omit if '--deployment' is provided. Currently only supported for: pre-backup-check, backup and backup-cleanup",
		},
	}
}

func availableDirectorFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "host",
			Value: "",
			Usage: "BOSH Director hostname, with an optional port. Port defaults to 22",
		},
		cli.StringFlag{
			Name:  "username, u",
			Value: "",
			Usage: "BOSH Director SSH username",
		},
		cli.StringFlag{
			Name:  "private-key-path, key",
			Value: "",
			Usage: "BOSH Director SSH private key",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logs",
		},
	}
}
