package main

import (
	"fmt"
	"os"

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
	}
	app.Commands = []cli.Command{
		{
			Name:    "backup",
			Aliases: []string{"b"},
			Usage:   "add a task to the list",
			Action: func(c *cli.Context) error {
				fmt.Println("Your backup is complete")
				return nil
			},
		},
	}

	app.Run(os.Args)
}
