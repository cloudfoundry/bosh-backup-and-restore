package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/mgutz/ansi"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/bosh"
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

	app.Flags = []cli.Flag{
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
	app.Before = func(c *cli.Context) error {
		requiredFlags := []string{"target", "username", "password", "deployment"}

		for _, flag := range requiredFlags {
			if c.GlobalString(flag) == "" {
				return fmt.Errorf("--%v flag is required.", flag)
			}
		}

		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:    "backup",
			Aliases: []string{"b"},
			Usage:   "add a task to the list",
			Action: func(c *cli.Context) error {
				var logger boshlog.Logger

				if c.GlobalBool("debug") {
					logger = boshlog.NewLogger(boshlog.LevelDebug)
				} else {
					logger = boshlog.NewLogger(boshlog.LevelError)
				}

				factory := director.NewFactory(logger)

				config, err := director.NewConfigFromURL(c.GlobalString("target"))
				if err != nil {
					return cli.NewExitError(ansi.Color(
						"Target director URL is malformed",
						"red"), 1)
				}

				config.Username = c.GlobalString("username")
				config.Password = c.GlobalString("password")

				if c.GlobalString("ca-cert") != "" {
					cert, err := ioutil.ReadFile(c.GlobalString("ca-cert"))
					if err != nil {
						return cli.NewExitError(ansi.Color(err.Error(), "red"), 1)
					}
					config.CACert = string(cert)
				}

				boshDirector, err := factory.New(config, director.NewNoopTaskReporter(), director.NewNoopFileReporter())
				if err != nil {
					return err
				}

				backuper := backuper.New(bosh.New(boshDirector, director.NewSSHOpts))

				if err := backuper.Backup(c.GlobalString("deployment")); err != nil {
					return cli.NewExitError(ansi.Color(err.Error(), "red"), 1)
				}
				return nil
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
