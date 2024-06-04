package main

import (
	"fmt"
	"github/luochenglcs/godnf/install"
	"github/luochenglcs/godnf/repodata"
	"github/luochenglcs/godnf/source"
	sqlquery "github/luochenglcs/godnf/source/sqlite"
)

func main() {
	repodata.GetRepo()
	repodata.GetMetadata("https://mirrors.opencloudos.tech/opencloudos-stream/releases/23/BaseOS/x86_64/Packages/repodata/repomd.xml")
	source.GetSql("https://mirrors.opencloudos.tech/opencloudos-stream/releases/23/BaseOS/x86_64/Packages/repodata/bcc24c95ed9205808a055d9a0e64ecc5c453b8b8569dd7c404a19976937971b0-primary.sqlite.bz2")
	var res [][]sqlquery.ReqRes
	sqlquery.GetAllRequres("systemd", 0, &res)

	for _, item := range res {
		source.GetRpm("https://mirrors.opencloudos.tech/opencloudos-stream/releases/23/BaseOS/x86_64/Packages/", item)
	}

	for i := len(res) - 1; i >= 0; i-- {
		for _, item := range res[i] {
			fmt.Printf(">>>>level %d Name: %s Version %s Release %s\n", i, item.Name, item.Version, item.Release)
			var packfile string
			if item.Epoch == "" {
				packfile = fmt.Sprintf("%s-%s-%s.%s.rpm", item.Name, item.Version, item.Release, item.Arch)
			} else {
				packfile = fmt.Sprintf("%s-%s:%s-%s.%s.rpm", item.Name, item.Epoch, item.Version, item.Release, item.Arch)
			}
			filepath := fmt.Sprintf("%s/%s", "/var/cache/godnf/BaseOS/packages/", packfile)
			install.ExtractRPM(filepath)
		}
	}

}
