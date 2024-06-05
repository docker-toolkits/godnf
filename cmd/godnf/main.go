package main

import (
	"fmt"
	"github/luochenglcs/godnf/dnflog"
	"github/luochenglcs/godnf/version"
	"os"

	"github.com/urfave/cli"
)

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(c.App.Name, version.Package, c.App.Version, version.Revision)
	}
	app := cli.NewApp()
	app.Name = "godnf"
	app.Usage = "package manager use go"
	app.Version = version.Version
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "loglevel",
			Usage: "set log level: 0-DEBUG, 1-INFO, 2-WARN, 3-ERROR, default:3",
			Value: 3,
		},
	}

	app.Commands = []cli.Command{
		installCommand,
	}

	var debuglevel int

	app.Before = func(context *cli.Context) error {
		debuglevel = context.GlobalInt("loglevel")
		var err error
		dnflog.L, err = dnflog.NewLogger(dnflog.LogLevel(debuglevel), "")
		if err != nil {
			fmt.Printf("Error creating logger: %v\n", err)
			return err
		}
		defer dnflog.L.Close()
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		dnflog.L.Error("Error running app: %v\n", err)
		os.Exit(1)
	}
}
