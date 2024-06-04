package main

import (
	"fmt"
	"github/luochenglcs/godnf/install"
	"github/luochenglcs/godnf/repodata"
	"github/luochenglcs/godnf/source"
	sqlquery "github/luochenglcs/godnf/source/sqlite"
	"os"
	"strings"
)

func main() {
	repoConfs, err := repodata.GetRepo()
	if err != nil {
		fmt.Println("Error GetRepo")
		return
	}
	fmt.Println("Update RepoDate")
	var dbpaths []string
	for key, rc := range repoConfs {
		if rc.Enabled == true {
			fmt.Println("key: ", key)
			repomd, err := repodata.GetMetadata(rc.BaseURL + "/repodata/repomd.xml")
			if err != nil {
				fmt.Println("Error GetMetadata ", rc.BaseURL)
				return
			}
			db := fmt.Sprintf("%s%s/%s", "/var/cache/godnf/", key, strings.TrimPrefix(repomd["primary_db"].Location.Href, "repodata/"))
			err = source.GetSql(rc.BaseURL+repomd["primary_db"].Location.Href, db)
			if err != nil {
				fmt.Println("Error GetSql ", rc.BaseURL)
				return
			}

			dbpaths = append(dbpaths, db[:len(db)-4])
		}
	}

	fmt.Println("dbPaths: ", dbpaths)
	var res [][]sqlquery.ReqRes
	sqlquery.GetAllRequres("CUnit", 0, &res, dbpaths)

	for _, item := range res {
		source.GetRpm(repoConfs, item)
	}

	for i := len(res) - 1; i >= 0; i-- {
		for _, item := range res[i] {
			parts := strings.Split(item.DbPath, "/")
			repoKey := parts[4]
			fmt.Printf(">>>>level %d Name: %s Version %s Release %s\n", i, item.Name, item.Version, item.Release)
			var packfile string
			if item.Epoch == "" {
				packfile = fmt.Sprintf("%s-%s-%s.%s.rpm", item.Name, item.Version, item.Release, item.Arch)
			} else {
				packfile = fmt.Sprintf("%s-%s:%s-%s.%s.rpm", item.Name, item.Epoch, item.Version, item.Release, item.Arch)
			}
			filepath := fmt.Sprintf("%s/%s/packages/%s", "/var/cache/godnf/", repoKey, packfile)
			install.ExtractRPM(filepath)
		}
	}

	// clean cache
	os.RemoveAll("/var/cache/godnf/")
}
