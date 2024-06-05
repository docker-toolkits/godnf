package main

import (
	"fmt"
	"github/luochenglcs/godnf/install"
	"github/luochenglcs/godnf/repodata"
	"github/luochenglcs/godnf/source"
	sqlquery "github/luochenglcs/godnf/source/sqlite"
	"strings"

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
		fmt.Printf("%d: %s\n", i+1, clicontext.Args().Get(i))
		packs = append(packs, clicontext.Args().Get(i))
	}

	repoConfs, err := repodata.GetRepo()
	if err != nil {
		fmt.Println("Error GetRepo")
		return err
	}

	fmt.Println("Update RepoDate")
	var dbpaths []string
	for key, rc := range repoConfs {
		if rc.Enabled == true {
			fmt.Println("key: ", key)
			repomd, err := repodata.GetMetadata(rc.BaseURL + "/repodata/repomd.xml")
			if err != nil {
				fmt.Println("Error GetMetadata ", rc.BaseURL)
				return fmt.Errorf("Error GetMetadata ", rc.BaseURL)
			}
			db := fmt.Sprintf("%s/%s%s/%s", destdir, "/var/cache/godnf/", key, strings.TrimPrefix(repomd["primary_db"].Location.Href, "repodata/"))
			err = source.GetSql(rc.BaseURL+repomd["primary_db"].Location.Href, db)
			if err != nil {
				fmt.Println("Error GetSql ", rc.BaseURL)
				return fmt.Errorf("Error GetSql ", rc.BaseURL)
			}

			dbpaths = append(dbpaths, db[:len(db)-4])
		}
	}
	fmt.Println("dbPaths: ", dbpaths)
	for _, pack := range packs {
		if installed, _ := install.QueryInstalledPkg(destdir, pack); installed {
			fmt.Printf("Already Install : Name: %sn", pack)
			continue
		}

		var res [][]sqlquery.ReqRes
		sqlquery.GetAllRequres(pack, 0, &res, dbpaths)

		for _, item := range res {
			source.GetRpm(destdir, repoConfs, item)
		}

		for i := len(res) - 1; i >= 0; i-- {
			for _, item := range res[i] {
				fmt.Printf(">>>level %d \n", i)
				install.InstallRPM(destdir, item)
			}
		}

	}

	// clean cache
	//os.RemoveAll("/var/cache/godnf/")

	return nil
}
