package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/mgutz/ansi"
	"github.com/pivotal-cf/bosh-backup-and-restore/artifact"
	"github.com/pivotal-cf/bosh-backup-and-restore/bosh"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/ssh"
	"github.com/urfave/cli"

	"github.com/cloudfoundry/bosh-cli/director"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/pivotal-cf/bosh-backup-and-restore/instance"
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

	boshDirector, err := makeBoshDirector(c, logger)
	if err != nil {
		return nil, err
	}
	boshClient := makeBoshClient(boshDirector, logger)
	deploymentManager := makeDeploymentManager(boshClient, logger)
	return orchestrator.NewBackuper(boshClient, artifact.DirectoryArtifactManager{}, logger, deploymentManager), nil
}

func makeDeploymentManager(boshClient orchestrator.BoshDirector, logger boshlog.Logger) orchestrator.DeploymentManager {
	return orchestrator.NewBoshDeploymentManager(boshClient, logger)
}
func makeBoshClient(boshDirector director.Director, logger boshlog.Logger) orchestrator.BoshDirector {
	boshClient := bosh.New(boshDirector, director.NewSSHOpts, ssh.ConnectionCreator, logger, instance.NewJobFinder(logger))
	return boshClient
}

func makeBoshDirector(c *cli.Context, logger boshlog.Logger) (director.Director, error) {
	var targetUrl = c.GlobalString("target")
	var username = c.GlobalString("username")
	var password = c.GlobalString("password")
	var caCert = c.GlobalString("ca-cert")
	boshDirector, err := makeBoshDirectorClient(targetUrl, username, password, caCert, logger)
	return boshDirector, err
}

func makeLogger(c *cli.Context) boshlog.Logger {
	var debug = c.GlobalBool("debug")
	var logger = makeBoshLogger(debug)
	return logger
}

func makeRestorer(c *cli.Context) (*orchestrator.Restorer, error) {
	logger := makeLogger(c)

	boshDirector, err := makeBoshDirector(c, logger)
	if err != nil {
		return nil, err
	}
	boshClient := makeBoshClient(boshDirector, logger)
	deploymentManager := makeDeploymentManager(boshClient, logger)
	return orchestrator.NewRestorer(boshClient, artifact.DirectoryArtifactManager{}, logger, deploymentManager), nil
}

func makeBoshLogger(debug bool) boshlog.Logger {
	if debug {
		return boshlog.NewLogger(boshlog.LevelDebug)
	}
	return boshlog.NewLogger(boshlog.LevelInfo)
}

func makeBoshDirectorClient(targetUrl, username, password, caCert string, logger boshlog.Logger) (director.Director, error) {
	config, err := director.NewConfigFromURL(targetUrl)
	if err != nil {
		return nil, cli.NewExitError(ansi.Color(
			"Target director URL is malformed",
			"red"), 1)
	}

	config.Client = username
	config.ClientSecret = password

	if caCert != "" {
		cert, err := ioutil.ReadFile(caCert)
		if err != nil {
			return nil, cli.NewExitError(ansi.Color(err.Error(), "red"), 1)
		}
		config.CACert = string(cert)
	}

	factory := director.NewFactory(logger)

	return factory.New(config, director.NewNoopTaskReporter(), director.NewNoopFileReporter())
}
