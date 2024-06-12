package source

import (
	"database/sql"
	"fmt"
	"github/luochenglcs/godnf/dnflog"
	"strings"

	"log"
	"os"
	"path/filepath"
)

func RecordInstalledPkg(destdir string, rpmpkg ReqRes) error {

	dbPath := fmt.Sprintf("%s/%s", destdir, "/var/lib/godnf/godnf_packages.db")
	if dirName := filepath.Dir(dbPath); dirName != "" {
		if err := os.MkdirAll(dirName, 0o755); err != nil {
			log.Fatal(err)
		}
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTableSQL := `CREATE TABLE IF NOT EXISTS packages (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"name" TEXT,
		"epoch" INTEGER,
		"version" TEXT,
		"release" TEXT,
		"arch" TEXT,
		"repo" TEXT
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Error creating table: %v\n", err)
	}

	insertPackageSQL := `INSERT INTO packages (name, epoch, version, release, arch, repo) VALUES (?, ?, ?, ?, ?, ?)`
	trimpath := strings.TrimPrefix(rpmpkg.DbPath, destdir)
	parts := strings.Split(trimpath, "/")

	repoKey := parts[len(parts)-2]
	_, err = db.Exec(insertPackageSQL, rpmpkg.Name, rpmpkg.Epoch, rpmpkg.Version, rpmpkg.Release, rpmpkg.Arch, repoKey)
	if err != nil {
		log.Fatalf("Error inserting data: %v\n", err)
	}

	return nil
}

func QueryInstalledPkg(destdir string, name string, strict bool) (bool, []ReqRes, error) {

	dbPath := fmt.Sprintf("%s/%s", destdir, "/var/lib/godnf/godnf_packages.db")

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
		querySQL = `SELECT name, epoch, version, release, arch, repo FROM packages WHERE name=?`
	} else {
		querySQL = `SELECT name, epoch, version, release, arch, repo FROM packages WHERE name LIKE ?`
	}

	rows, err := db.Query(querySQL, name)
	if err != nil {
		log.Fatalf("Error querying data: %v\n", err)
	}
	defer rows.Close()
	var rpmpkg []ReqRes
	for rows.Next() {
		var name, epoch, version, release, arch, repo string
		err = rows.Scan(&name, &epoch, &version, &release, &arch, &repo)
		if err != nil {
			log.Fatalf("Error scanning row: %v\n", err)
		}
		dnflog.L.Debug("Name: %s, Epoch: %s, Version: %s, Release: %s\n", name, epoch, version, release)
		var curpkg ReqRes
		curpkg.DbPath = repo
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
