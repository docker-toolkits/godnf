package source

import (
	"database/sql"
	"fmt"
	"github/luochenglcs/godnf/dnflog"
	"log"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

func QueryRepoPkg(dbPath string, name string, strict bool) (bool, []ReqRes, error) {
	parts := strings.Split(dbPath, "/")
	repoKey := parts[len(parts)-2]

	_, err := os.Stat(dbPath)
	if os.IsNotExist(err) {
		return false, nil, fmt.Errorf("not exist db: %s", dbPath)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var querySQL string
	if strict {
		querySQL = `SELECT name, epoch, version, release, arch FROM packages WHERE name=?`
	} else {
		querySQL = `SELECT name, epoch, version, release, arch FROM packages WHERE name LIKE ?`
	}

	rows, err := db.Query(querySQL, name)
	if err != nil {
		log.Fatalf("Error querying data: %v\n", err)
	}
	defer rows.Close()
	var rpmpkg []ReqRes
	for rows.Next() {
		var name, epoch, version, release, arch string
		err = rows.Scan(&name, &epoch, &version, &release, &arch)
		if err != nil {
			log.Fatalf("Error scanning row: %v\n", err)
		}
		dnflog.L.Debug("Name: %s, Epoch: %s, Version: %s, Release: %s\n", name, epoch, version, release)
		var curpkg ReqRes
		curpkg.DbPath = repoKey
		curpkg.Name = name
		curpkg.Version = version
		curpkg.Release = release
		curpkg.Arch = arch
		rpmpkg = append(rpmpkg, curpkg)
	}

	if len(rpmpkg) != 0 {
		return true, rpmpkg, nil
	}

	err = rows.Err()
	if err != nil {
		log.Fatalf("Error during row iteration: %v\n", err)
	}
	return false, nil, fmt.Errorf("not Found In db")
}
