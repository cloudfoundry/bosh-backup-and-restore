package main

import (
	"fmt"
	"os"

	"github.com/pivotal-cf/bosh-backup-and-restore/artifact"
	"github.com/pivotal-cf/bosh-backup-and-restore/bosh"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/mgutz/ansi"
	"github.com/pivotal-cf/bosh-backup-and-restore/instance"
	"github.com/pivotal-cf/bosh-backup-and-restore/ssh"
	"github.com/pivotal-cf/bosh-backup-and-restore/standalone"
)

var version string

func main() {
	app := cli.NewApp()

	app.Version = version

	app.Name = "BOSH Backup and Restore"
	app.HelpName = "bbr"
	app.Usage = ""

	app.Commands = []cli.Command{
		{
			Name:   "deployment",
			Usage:  "Backup BOSH deployments",
			Flags:  availableDeploymentFlags(),
			Before: validateDeploymentFlags,
			Subcommands: []cli.Command{
				{
					Name:    "pre-backup-check",
					Aliases: []string{"c"},
					Usage:   "Check a deployment can be backed up",
					Action:  preBackupCheck,
				},
				{
					Name:    "backup",
					Aliases: []string{"b"},
					Usage:   "Backup a deployment",
					Action:  backup,
				},
				{
					Name:    "restore",
					Aliases: []string{"r"},
					Usage:   "Restore a deployment from backup",
					Action:  restore,
				},
			},
		},
		{
			Name:   "director",
			Usage:  "Backup BOSH director",
			Flags:  availableDirectorFlags(),
			Before: validateDirectorFlags,
			Subcommands: []cli.Command{
				{
					Name:    "pre-backup-check",
					Aliases: []string{"c"},
					Usage:   "Check a BOSH Director can be backed up",
					Action:  directorPreBackupCheck,
				},
			},
		},
		{
			Name:  "version",
			Usage: "",
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

func preBackupCheck(c *cli.Context) error {
	var deployment = c.Parent().String("deployment")

	backuper, err := makeBackuper(c)
	if err != nil {
		return err
	}

	backupable, err := backuper.CanBeBackedUp(deployment)

	if backupable {
		fmt.Printf("Deployment '%s' can be backed up.\n", deployment)
		return cli.NewExitError("", 0)
	} else {
		fmt.Printf("Deployment '%s' cannot be backed up.\n", deployment)
		return cli.NewExitError(err, 1)
	}
}

func directorPreBackupCheck(c *cli.Context) error {
	var deployment = c.Parent().String("name")

	logger := makeLogger(c)

	deploymentManager := standalone.NewDeploymentManager(logger,
		c.Parent().String("host"),
		c.Parent().String("username"),
		c.Parent().String("private-key-path"),
		instance.NewJobFinder(logger),
		ssh.ConnectionCreator,
	)
	backuper := orchestrator.NewBackuper(artifact.DirectoryArtifactManager{}, logger, deploymentManager)

	backupable, err := backuper.CanBeBackedUp(deployment)

	if backupable {
		fmt.Printf("Director can be backed up.\n")
		return cli.NewExitError("", 0)
	} else {
		fmt.Printf("Director cannot be backed up.\n")
		return cli.NewExitError(err, 1)
	}
}

func backup(c *cli.Context) error {
	var deployment = c.Parent().String("deployment")

	backuper, err := makeBackuper(c)
	if err != nil {
		return err
	}

	backupErr := backuper.Backup(deployment)

	errorCode, errorMessage := orchestrator.ProcessBackupError(backupErr)

	return cli.NewExitError(errorMessage, errorCode)
}

func restore(c *cli.Context) error {
	var deployment = c.Parent().String("deployment")

	restorer, err := makeRestorer(c)
	if err != nil {
		return err
	}

	err = restorer.Restore(deployment)
	return orchestrator.ProcessRestoreError(err)
}

func validateDeploymentFlags(c *cli.Context) error {
	return validateFlags([]string{"target", "username", "password", "deployment"}, c)
}

func validateDirectorFlags(c *cli.Context) error {
	return validateFlags([]string{"name", "host", "username", "private-key-path"}, c)
}

func validateFlags(requiredFlags []string, c *cli.Context) error {
	for _, flag := range requiredFlags {
		if c.String(flag) == "" {
			cli.ShowAppHelp(c)
			return fmt.Errorf("--%v flag is required.", flag)
		}
	}
	return nil
}

func availableDeploymentFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "target, t",
			Value: "",
			Usage: "Target BOSH Director URL",
		},
		cli.StringFlag{
			Name:  "username, u",
			Value: "",
			Usage: "BOSH Director username",
		},
		cli.StringFlag{
			Name:   "password, p",
			Value:  "",
			EnvVar: "BOSH_CLIENT_SECRET",
			Usage:  "BOSH Director password",
		},
		cli.StringFlag{
			Name:  "deployment, d",
			Value: "",
			Usage: "Name of BOSH deployment",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logs",
		},
		cli.StringFlag{
			Name:   "ca-cert",
			Value:  "",
			EnvVar: "CA_CERT",
			Usage:  "Custom CA certificate",
		},
	}
}

func availableDirectorFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "name, n",
			Value: "",
			Usage: "Name for backup",
		},
		cli.StringFlag{
			Name:  "host",
			Value: "",
			Usage: "BOSH Director hostname",
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

func makeBackuper(c *cli.Context) (*orchestrator.Backuper, error) {
	logger := makeLogger(c)
	deploymentManager, err := newDeploymentManager(
		c.Parent().String("target"),
		c.Parent().String("username"),
		c.Parent().String("password"),
		c.Parent().String("ca-cert"),
		logger,
	)

	if err != nil {
		return nil, redCliError(err)
	}

	return orchestrator.NewBackuper(artifact.DirectoryArtifactManager{}, logger, deploymentManager), nil
}

func makeRestorer(c *cli.Context) (*orchestrator.Restorer, error) {
	logger := makeLogger(c)
	deploymentManager, err := newDeploymentManager(
		c.Parent().String("target"),
		c.Parent().String("username"),
		c.Parent().String("password"),
		c.Parent().String("ca-cert"),
		logger,
	)

	if err != nil {
		return nil, redCliError(err)
	}

	return orchestrator.NewRestorer(artifact.DirectoryArtifactManager{}, logger, deploymentManager), nil
}

func newDeploymentManager(targetUrl, username, password, caCert string, logger boshlog.Logger) (orchestrator.DeploymentManager, error) {
	boshClient, err := bosh.BuildClient(targetUrl, username, password, caCert, logger)
	if err != nil {
		return nil, redCliError(err)
	}

	return bosh.NewBoshDeploymentManager(boshClient, logger), nil
}

func makeLogger(c *cli.Context) boshlog.Logger {
	var debug = c.GlobalBool("debug")
	return makeBoshLogger(debug)
}

func redCliError(err error) *cli.ExitError {
	return cli.NewExitError(ansi.Color(err.Error(), "red"), 1)
}

func makeBoshLogger(debug bool) boshlog.Logger {
	if debug {
		return boshlog.NewLogger(boshlog.LevelDebug)
	}
	return boshlog.NewLogger(boshlog.LevelInfo)
}
