package main

import (
	"fmt"
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
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output in logs",
		},
	}

	app.Commands = []cli.Command{
		installCommand,
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running app: %v\n", err)
		os.Exit(1)
	}
}
