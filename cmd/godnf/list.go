package main

import (
	"fmt"
	"github/luochenglcs/godnf/dnflog"
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

var listCommand = cli.Command{
	Name:   "list",
	Usage:  "list rpm packages",
	Action: listPacks,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "installed",
			Usage: "list installed rpm packages",
		},
	},
}

func listPacks(clicontext *cli.Context) error {

	showInstalled := clicontext.Bool("installed")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.Debug)

	if showInstalled {
		_, rpmlists, err := sqlquery.QueryInstalledPkg("", "%", false)
		if err != nil {
			return err
		}
		fmt.Printf("\n")
		// Print installing packages
		fmt.Fprintln(w, "==============================================================================================")
		fmt.Fprintln(w, " Package\tArchitecture\tVersion\tRepository")
		fmt.Fprintln(w, "==============================================================================================")
		fmt.Fprintln(w, "Installed")

		for _, item := range rpmlists {
			fmt.Fprintln(w, item.Name, "\t", item.Arch, "\t", item.Version, "-", item.Release, "\t", item.DbPath)
		}

		w.Flush()

		return nil
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
		db := fmt.Sprintf("%s/%s%s/%s", "", "/var/cache/godnf/", key, strings.TrimPrefix(repomd["primary_db"].Location.Href, "repodata/"))
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

	var allRpmpkgs []sqlquery.ReqRes
	for _, db := range dbpaths {
		_, rpmpkgs, err := sqlquery.QueryRepoPkg(db, "%", false)
		if err != nil {
			return err
		}
		allRpmpkgs = append(allRpmpkgs, rpmpkgs[:]...)
	}

	fmt.Printf("\n")
	// Print installing packages
	fmt.Fprintln(w, "==============================================================================================")
	fmt.Fprintln(w, " Package\tArchitecture\tVersion\tRepository")
	fmt.Fprintln(w, "==============================================================================================")
	fmt.Fprintln(w, "Repo packages")

	for _, item := range allRpmpkgs {
		_, installrpmlists, _ := sqlquery.QueryInstalledPkg("", item.Name, true)
		var isinstalled bool = false
		for _, installedpkg := range installrpmlists {
			if sqlquery.CompVerRelease(installedpkg, item) == 0 && installedpkg.Arch == item.Arch {
				isinstalled = true
				break
			}
		}
		if isinstalled {
			fmt.Fprintln(w, item.Name, "\t", item.Arch, "\t", item.Version+"-"+item.Release, "\t", item.DbPath, "    ", "@Installed")
		} else {
			fmt.Fprintln(w, item.Name, "\t", item.Arch, "\t", item.Version+"-"+item.Release, "\t", item.DbPath)
		}

	}

	w.Flush()

	// clean cache
	os.RemoveAll("/var/cache/godnf/")

	return nil
}
