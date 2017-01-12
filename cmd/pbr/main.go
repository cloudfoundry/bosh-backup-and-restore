package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/mgutz/ansi"
	"github.com/pivotal-cf/pcf-backup-and-restore/artifact"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/bosh"
	"github.com/pivotal-cf/pcf-backup-and-restore/ssh"
	"github.com/urfave/cli"

	"github.com/cloudfoundry/bosh-cli/director"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
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

	backuperInstance, err := makeBackuper(c)
	if err != nil {
		return err
	}

	backupErr := backuperInstance.Backup(deployment)
	return processBackupError(backupErr)
}

func restore(c *cli.Context) error {
	var deployment = c.GlobalString("deployment")

	backuperInstance, err := makeBackuper(c)
	if err != nil {
		return err
	}

	err = backuperInstance.Restore(deployment)
	return processRestoreError(err)
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
			EnvVar: "BOSH_PASSWORD",
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

func processBackupError(err backuper.Error) error {
	switch {
	case err.IsCleanup():
		return cli.NewExitError(ansi.Color(err.Error(), "yellow"), 2)
	case err.IsPostBackup():
		return cli.NewExitError(ansi.Color(err.Error(), "red"), 42)
	case err.IsFatal():
		return cli.NewExitError(ansi.Color(err.Error(), "red"), 1)
	case err.IsNil():
		return nil
	default:
		return err
	}
}


func processRestoreError(err error) error {
	switch err := err.(type) {
	case backuper.CleanupError:
		return cli.NewExitError(ansi.Color(err.Error(), "yellow"), 2)
	case backuper.PostBackupUnlockError:
		return cli.NewExitError(ansi.Color(err.Error(), "red"), 42)
	case error:
		return cli.NewExitError(ansi.Color(err.Error(), "red"), 1)
	default:
		return err
	}
}

func makeBackuper(c *cli.Context) (*backuper.Backuper, error) {
	var debug = c.GlobalBool("debug")
	var targetUrl = c.GlobalString("target")
	var username = c.GlobalString("username")
	var password = c.GlobalString("password")
	var caCert = c.GlobalString("ca-cert")

	var logger = makeBoshLogger(debug)
	boshDirector, err := makeBoshDirectorClient(targetUrl, username, password, caCert, logger)
	if err != nil {
		return nil, err
	}
	boshClient := bosh.New(boshDirector, director.NewSSHOpts, ssh.ConnectionCreator, logger)
	deploymentManager := backuper.NewBoshDeploymentManager(boshClient, logger)
	return backuper.New(boshClient, artifact.DirectoryArtifactManager{}, logger, deploymentManager), nil
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

	config.Username = username
	config.Password = password

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
