package main

import (
	"fmt"
	"github/luochenglcs/godnf/dnflog"
	"github/luochenglcs/godnf/install"
	"github/luochenglcs/godnf/repodata"
	"github/luochenglcs/godnf/source"
	sqlquery "github/luochenglcs/godnf/source/sqlite"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/urfave/cli"
)

var installCommand = cli.Command{
	Name:   "install",
	Usage:  "install rpm packages",
	Action: installPacks,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "destdir",
			Usage: "Specify the installation directory",
			Value: "/",
		},
	},
}

func installPacks(clicontext *cli.Context) error {
	destdir := clicontext.String("destdir")

	var packs []string
	for i := 0; i < clicontext.NArg(); i++ {
		dnflog.L.Debug("%d: %s\n", i+1, clicontext.Args().Get(i))
		packs = append(packs, clicontext.Args().Get(i))
	}
	fmt.Println("Update RepoDate")
	repoConfs, err := repodata.GetRepo()
	if err != nil {
		dnflog.L.Error("Error GetRepo")
		return err
	}

	var dbpaths []string
	for key, rc := range repoConfs {
		if rc.Enabled == true {
			dnflog.L.Debug("key: ", key)
			repomd, err := repodata.GetMetadata(rc.BaseURL + "/repodata/repomd.xml")
			if err != nil {
				dnflog.L.Error("Error GetMetadata ", rc.BaseURL)
				return fmt.Errorf("Error GetMetadata ", rc.BaseURL)
			}
			db := fmt.Sprintf("%s/%s%s/%s", destdir, "/var/cache/godnf/", key, strings.TrimPrefix(repomd["primary_db"].Location.Href, "repodata/"))
			db = filepath.Clean(db)
			err = source.GetSql(rc.BaseURL+repomd["primary_db"].Location.Href, db)
			if err != nil {
				dnflog.L.Error("Error GetSql ", rc.BaseURL)
				return fmt.Errorf("Error GetSql ", rc.BaseURL)
			}

			dbpaths = append(dbpaths, db[:len(db)-4])
		}
	}

	dnflog.L.Debug("dbPaths: ", dbpaths)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.Debug)
	for _, pack := range packs {
		if installed, rpmpkg, _ := install.QueryInstalledPkg(destdir, pack); installed {
			fmt.Printf("%s-%s-%s.%s is installed\n", rpmpkg.Name, rpmpkg.Version, rpmpkg.Release, rpmpkg.Arch)
			continue
		}

		var res [][]sqlquery.ReqRes
		sqlquery.GetAllRequres(pack, 0, &res, dbpaths)

		fmt.Printf("\n")
		// Print installing packages
		fmt.Fprintln(w, "==============================================================================================")
		fmt.Fprintln(w, " Package\tArchitecture\tVersion\tRepository")
		fmt.Fprintln(w, "==============================================================================================")
		fmt.Fprintln(w, "Installing:")
		fmt.Fprintln(w, pack)
		fmt.Fprintln(w, "Installing dependencies:")
		for i := 0; i <= len(res)-1; i++ {
			for _, item := range res[i] {
				trimpath := strings.TrimPrefix(item.DbPath, destdir)
				parts := strings.Split(trimpath, "/")

				repoKey := parts[len(parts)-2]
				fmt.Fprintln(w, item.Name, "\t", item.Arch, "\t", item.Version, "-", item.Release, "\t", repoKey)
			}
		}
		w.Flush()

		fmt.Printf("\n")
		for i := 0; i <= len(res)-1; i++ {
			for _, item := range res[i] {
				fmt.Printf("Downloading %s-%s-%s.%s\n", item.Name, item.Version, item.Release, item.Arch)
				err := source.GetRpm(destdir, repoConfs, item)
				if err != nil {
					return fmt.Errorf("Get Rpm failed %v %v", item, err)
				}
			}
		}

		fmt.Printf("\n")
		for i := len(res) - 1; i >= 0; i-- {
			for _, item := range res[i] {
				fmt.Printf("Installing %s-%s-%s.%s\n", item.Name, item.Version, item.Release, item.Arch)
				install.InstallRPM(destdir, item)
			}
		}

	}

	// clean cache
	os.RemoveAll("/var/cache/godnf/")

	return nil
}
