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
	"sync"
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
	var wg sync.WaitGroup
	ch := make(chan string, len(repoConfs))

	for key, rc := range repoConfs {
		dnflog.L.Debug("key: ", key)
		repomd, err := repodata.GetMetadata(rc.BaseURL + "/repodata/repomd.xml")
		if err != nil {
			dnflog.L.Error("Error GetMetadata ", rc.BaseURL)
			return fmt.Errorf("error GetMetadata %s", rc.BaseURL)
		}
		db := fmt.Sprintf("%s/%s%s/%s", destdir, "/var/cache/godnf/", key, strings.TrimPrefix(repomd["primary_db"].Location.Href, "repodata/"))
		db = filepath.Clean(db)
		dbpaths = append(dbpaths, db[:len(db)-4])
		wg.Add(1)
		downurl := fmt.Sprintf("%s/%s", rc.BaseURL, repomd["primary_db"].Location.Href)

		go func(url, dbstore string) {
			defer wg.Done()
			err := source.GetSql(url, dbstore)
			if err != nil {
				dnflog.L.Error("Error GetSql ", url)
				ch <- err.Error()
				return
			}
			ch <- "success"
		}(downurl, db)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for msg := range ch {
		if msg != "success" {
			fmt.Println("Error:", msg)
			return fmt.Errorf("get db Error: %s", msg)
		}
	}

	dnflog.L.Debug("dbPaths: ", dbpaths)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.Debug)
	for _, pack := range packs {
		if installed, rpmpkgs, _ := sqlquery.QueryInstalledPkg(destdir, pack, true); installed {
			for _, rpmpkg := range rpmpkgs {
				fmt.Printf("%s-%s-%s.%s is installed\n", rpmpkg.Name, rpmpkg.Version, rpmpkg.Release, rpmpkg.Arch)
			}

			continue
		}

		var res [][]sqlquery.ReqRes
		arch := repodata.GetRuntimeArch()
		sqlquery.GetAllRequres(pack, arch, 0, &res, dbpaths)

		var totalpkg int = 0
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
				fmt.Fprintln(w, item.Name, "\t", item.Arch, "\t", item.Version+"-"+item.Release, "\t", repoKey)
				totalpkg++
			}
		}
		w.Flush()

		fmt.Printf("\n")
		getrpmch := make(chan string, totalpkg)
		for i := 0; i <= len(res)-1; i++ {
			for _, item := range res[i] {
				if installed, _, _ := sqlquery.QueryInstalledPkg(destdir, item.Name, true); installed {
					fmt.Printf("Name: %s-%s-%s is installed\n", item.Name, item.Version, item.Release)
					continue
				}
				wg.Add(1)
				go func(pkg sqlquery.ReqRes) {
					defer wg.Done()
					err := source.GetRpm(destdir, repoConfs, pkg)
					if err != nil {
						getrpmch <- fmt.Sprintf("get Rpm failed %v %v", pkg, err)
						return
					}
					fmt.Printf("Download %s-%s-%s.%s\n", pkg.Name, pkg.Version, pkg.Release, pkg.Arch)
					getrpmch <- "success"
				}(item)
			}
		}

		go func() {
			wg.Wait()
			close(getrpmch)
		}()

		for msg := range getrpmch {
			if msg != "success" {
				fmt.Println("Error:", msg)
				return fmt.Errorf("get db Error: %s", msg)
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
