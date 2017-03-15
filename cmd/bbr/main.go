package main

import (
	"fmt"
	"os"

	"github.com/pivotal-cf/bosh-backup-and-restore/artifact"
	"github.com/pivotal-cf/bosh-backup-and-restore/factory"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/mgutz/ansi"
)

var version string

func main() {
	app := cli.NewApp()

	app.Version = version

	app.Name = "Pivotal Backup and Restore"
	app.HelpName = "Pivotal Backup and Restore"

	app.Flags = availableFlags()
	app.Before = validateFlags
	app.Commands = []cli.Command{

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
	}

	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}

func preBackupCheck(c *cli.Context) error {
	var deployment = c.GlobalString("deployment")

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

func backup(c *cli.Context) error {
	var deployment = c.GlobalString("deployment")

	backuper, err := makeBackuper(c)
	if err != nil {
		return err
	}

	backupErr := backuper.Backup(deployment)

	errorCode, errorMessage := orchestrator.ProcessBackupError(backupErr)

	return cli.NewExitError(errorMessage, errorCode)
}

func restore(c *cli.Context) error {
	var deployment = c.GlobalString("deployment")

	restorer, err := makeRestorer(c)
	if err != nil {
		return err
	}

	err = restorer.Restore(deployment)
	return orchestrator.ProcessRestoreError(err)
}

func validateFlags(c *cli.Context) error {
	requiredFlags := []string{"target", "username", "password", "deployment"}

	for _, flag := range requiredFlags {
		if c.GlobalString(flag) == "" {
			return fmt.Errorf("--%v flag is required.", flag)
		}
	}
	return nil
}

func availableFlags() []cli.Flag {
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

func makeBackuper(c *cli.Context) (*orchestrator.Backuper, error) {
	logger := makeLogger(c)
	boshClient, err := makeBoshClient(c, logger)
	if err != nil {
		return nil, cli.NewExitError(ansi.Color(err.Error(), "red"), 1)
	}
	deploymentManager := makeDeploymentManager(boshClient, logger)
	return orchestrator.NewBackuper(boshClient, artifact.DirectoryArtifactManager{}, logger, deploymentManager), nil
}

func makeRestorer(c *cli.Context) (*orchestrator.Restorer, error) {
	logger := makeLogger(c)
	boshClient, err := makeBoshClient(c, logger)
	if err != nil {
		return nil, cli.NewExitError(ansi.Color(err.Error(), "red"), 1)
	}
	deploymentManager := makeDeploymentManager(boshClient, logger)
	return orchestrator.NewRestorer(boshClient, artifact.DirectoryArtifactManager{}, logger, deploymentManager), nil
}

func makeBoshClient(c *cli.Context, logger boshlog.Logger) (orchestrator.BoshClient, error) {
	targetUrl := c.GlobalString("target")
	username := c.GlobalString("username")
	password := c.GlobalString("password")
	caCert := c.GlobalString("ca-cert")

	return factory.BuildClient(targetUrl, username, password, caCert, logger)
}

func makeDeploymentManager(boshClient orchestrator.BoshClient, logger boshlog.Logger) orchestrator.DeploymentManager {
	return orchestrator.NewBoshDeploymentManager(boshClient, logger)
}

func makeLogger(c *cli.Context) boshlog.Logger {
	var debug = c.GlobalBool("debug")
	var logger = makeBoshLogger(debug)
	return logger
}

func makeBoshLogger(debug bool) boshlog.Logger {
	if debug {
		return boshlog.NewLogger(boshlog.LevelDebug)
	}
	return boshlog.NewLogger(boshlog.LevelInfo)
}
