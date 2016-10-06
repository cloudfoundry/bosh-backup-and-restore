package main

import (
	"os"

	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/boshclient"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	app.Name = "Pivotal Backup and Restore"

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
			Name:  "password, p",
			Value: "",
			Usage: "BOSH Director password",
		},
		cli.StringFlag{
			Name:  "deployment, d",
			Value: "",
			Usage: "Name of BOSH deployment",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "backup",
			Aliases: []string{"b"},
			Usage:   "add a task to the list",
			Action: func(c *cli.Context) error {
				client := boshclient.New(c.GlobalString("target"), c.GlobalString("username"), c.GlobalString("password"))
				backuper := backuper.New(client)
				if err := backuper.Backup(c.GlobalString("deployment")); err != nil {
					return cli.NewExitError(err.Error(), 1)
				}
				return nil
			},
		},
	}

	app.Run(os.Args)
}
