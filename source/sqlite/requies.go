/*
SELECT pkgKey FROM packages WHERE Name='systemd';
SELECT * FROM requires WHERE pkgKey=12345;

SELECT p.* FROM requires r
JOIN packages p ON r.Name = p.Name
WHERE r.pkgKey = (SELECT pkgKey FROM packages WHERE Name='systemd');
*/
package source

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type queryRes struct {
	Name    string
	Flags   sql.NullString
	Epoch   sql.NullString
	Version sql.NullString
	Release sql.NullString
}

type ReqRes struct {
	DbPath  string
	Name    string
	Epoch   string
	Version string
	Release string
	Arch    string
}

func IsExisted(res []ReqRes, item ReqRes) (existed bool, pos int) {
	for i, r := range res {
		if r.Name != item.Name {
			continue
		}
		/*
			if r.Epoch != item.Epoch {
				continue
			}
			if r.Version != item.Version {
				continue
			}
			if r.Release != item.Release {
				continue
			}
		*/
		return true, i
	}

	return false, 0
}

// TODO: Better traversal
func GetAllRequres(in string, l int, res *[][]ReqRes, dbpaths []string) {
	if in != "" {
		re, _ := GetRequres(in, dbpaths)
		for _, item := range re {
			var existed bool = false
			//var pos int
			for _, row := range *res {
				existed, _ = IsExisted(row, item)
				if existed == true {
					break
				}
			}

			if existed == false {
				//add it to new level
				if l >= len(*res) {
					*res = append(*res, []ReqRes{})
				}
				(*res)[l] = append((*res)[l], item)
				GetAllRequres(item.Name, l+1, res, dbpaths)
			}
		}
	}
}

func getRequirePkgname(requires *[]queryRes, dbpath string) (res []ReqRes, err error) {
	db, err := sql.Open("sqlite3", dbpath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	//var res []ReqRes
	var max_epoch, max_version, max_release string
	var query string
	var notfound []queryRes
	for _, req := range *requires {
		if req.Flags.Valid {
			var queryBuilder strings.Builder
			queryBuilder.WriteString(fmt.Sprintf("SELECT p.Name, p.Epoch, p.Version,p.Release,p.Arch FROM packages p JOIN provides pr ON p.pkgKey = pr.pkgKey WHERE pr.Name='%s'", req.Name))
			if req.Flags.String == "EQ" {
				/* TODO: */
				if req.Epoch.Valid {
					queryBuilder.WriteString(fmt.Sprintf(" AND pr.Epoch='%s'", req.Epoch.String))
				}
				if req.Version.Valid {
					queryBuilder.WriteString(fmt.Sprintf(" AND pr.Version='%s'", req.Version.String))
				}
				if req.Release.Valid {
					queryBuilder.WriteString(fmt.Sprintf(" AND pr.Release='%s'", req.Release.String))
				}
			} else if req.Flags.String == "GE" {
				if req.Epoch.Valid {
					queryBuilder.WriteString(fmt.Sprintf(" AND (pr.Epoch>'%s'", req.Epoch.String))
				}
				if req.Version.Valid {
					queryBuilder.WriteString(fmt.Sprintf(" OR pr.Version>='%s'", req.Version.String))
				}
				if req.Release.Valid {
					queryBuilder.WriteString(fmt.Sprintf(" OR pr.Release>='%s'", req.Release.String))
				}
				queryBuilder.WriteString(fmt.Sprintf(")"))
			} else {
				fmt.Println("TODO")
			}
			queryBuilder.WriteString(fmt.Sprintf(";"))
			query = queryBuilder.String()
		} else {
			query = fmt.Sprintf("SELECT p.Name,p.Epoch,p.Version,p.Release,p.Arch FROM packages p JOIN provides pr ON p.pkgKey = pr.pkgKey WHERE pr.Name='%s';", req.Name)
		}
		//fmt.Printf("query %s\n", query)
		reqquery, err := db.Query(query)
		if err != nil {
			log.Fatalf("Error executing query: %v", err)
		}
		defer reqquery.Close()

		var lastestName string = ""
		var Arch string
		for reqquery.Next() {
			var Name, Epoch, Version, Release string
			err := reqquery.Scan(&Name, &Epoch, &Version, &Release, &Arch)
			if err != nil {
				log.Fatalf("Error scanning row: %v", err)
			}
			if lastestName == "" {
				lastestName = Name
				max_version = Version
				max_release = Release
			} else {
				// TODO :11.ocs < 2.ocs， it is unreasonable
				if (strings.Compare(Version, max_version) == 1) ||
					((strings.Compare(Version, max_version) == 0) && (strings.Compare(Release, max_release) != -1)) {
					lastestName = Name
					max_version = Version
					max_release = Release
				}
			}
		}

		/* No rpm package is queried from the tables 'provides' if lastestName == "",  query from the files table */
		if lastestName == "" {
			query = fmt.Sprintf("SELECT p.Name,p.Epoch,p.Version,p.Release,p.Arch FROM packages p JOIN files pr ON p.pkgKey = pr.pkgKey WHERE pr.Name='%s';", req.Name)
			filequery, err := db.Query(query)
			if err != nil {
				log.Fatalf("Error executing query: %v", err)
			}
			defer filequery.Close()

			for filequery.Next() {
				var Name, Epoch, Version, Release string
				err := filequery.Scan(&Name, &Epoch, &Version, &Release, &Arch)
				if err != nil {
					log.Fatalf("Error scanning row: %v", err)
				}
				if lastestName == "" {
					lastestName = Name
					max_version = Version
					max_release = Release
				} else {
					if (strings.Compare(Version, max_version) == 1) ||
						((strings.Compare(Version, max_version) == 0) && (strings.Compare(Release, max_release) != -1)) {
						lastestName = Name
						max_version = Version
						max_release = Release
					}
				}
			}
		}

		// Not found in current db, record it
		if lastestName == "" {
			notfound = append(notfound, req)
		}

		//fmt.Printf("Name: %s | %s | %s | %s\n", lastestName, max_epoch, max_version, max_release)
		var resultPkg ReqRes
		resultPkg.DbPath = dbpath
		resultPkg.Name = lastestName
		resultPkg.Version = max_version
		resultPkg.Release = max_release
		resultPkg.Epoch = max_epoch
		resultPkg.Arch = Arch

		existed, _ := IsExisted(res, resultPkg)
		if existed == false {
			res = append(res, resultPkg)
		}
	}
	*requires = notfound

	return res, nil
}

func getRequresInfo(in, dbpath string) ([]queryRes, error) {
	db, err := sql.Open("sqlite3", dbpath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	query := `SELECT pkgKey,Name,arch,Version,Release FROM packages WHERE Name=?;`
	packrows, err := db.Query(query, in)
	if err != nil {
		log.Fatalf("Error executing query: %v", err)
	}
	defer packrows.Close()

	var latestPkgKey int = -1
	var max_version, max_release string

	//fmt.Println("Data from 'packages' table:")
	for packrows.Next() {
		var pkgKey int
		var Name, arch, Version, Release string
		err := packrows.Scan(&pkgKey, &Name, &arch, &Version, &Release)
		if err != nil {
			log.Fatalf("Error scanning row: %v", err)
		}
		//fmt.Printf("pkgKey: %d, Name: %s, Arch: %s, Version: %s, Release %s\n", pkgKey, Name, arch, Version, Release)
		if latestPkgKey == -1 {
			latestPkgKey = pkgKey
			max_version = Version
			max_release = Release
		} else {
			if (strings.Compare(Version, max_version) == 1) ||
				((strings.Compare(Version, max_version) == 0) && (strings.Compare(Release, max_release) != -1)) {
				latestPkgKey = pkgKey
				max_version = Version
				max_release = Release
			}
		}
	}

	// Don't find package in current db
	if latestPkgKey == -1 {
		return nil, fmt.Errorf("Not Found Package in db")
	}

	//fmt.Printf("Max pkgKey: %d,  Version: %s, Release %s\n", latestPkgKey, max_version, max_release)
	query = `SELECT Name,Flags,Epoch,Version,Release FROM requires WHERE pkgKey=?;`
	reqrows, err := db.Query(query, latestPkgKey)
	if err != nil {
		log.Fatalf("Error executing query: %v", err)
	}
	defer reqrows.Close()

	var requires []queryRes
	for reqrows.Next() {
		var req queryRes
		err := reqrows.Scan(&req.Name, &req.Flags, &req.Epoch, &req.Version, &req.Release)
		if err != nil {
			log.Fatalf("Error scanning row: %v", err)
		}
		//fmt.Printf("Name: %s\n", req.Name)
		/* req.Name (systemd-rpm-macros = 255-4.ocs23 if rpm-build) TODO:pattern */
		if strings.Contains(req.Name, " if ") {
			pattern := `\(([^=]+) \s*=\s*([\d\w.-]+)\s*if\s*([\w.-]+)\)`
			compileRegex := regexp.MustCompile(pattern)
			match := compileRegex.FindStringSubmatch(req.Name)
			//fmt.Printf("match: %s\n", match[1])
			req.Name = match[1]
		}
		requires = append(requires, req)
	}
	return requires, nil
}

func GetRequres(in string, dbpaths []string) ([]ReqRes, error) {
	var reqinfo []queryRes
	var err error
	for _, db := range dbpaths {
		reqinfo, err = getRequresInfo(in, db)
		if err == nil {
			break
		}
	}

	var res []ReqRes
	for _, db := range dbpaths {
		tmp, _ := getRequirePkgname(&reqinfo, db)
		if len(tmp) != 0 {
			for _, item := range tmp {
				fmt.Printf("out >> %s-%s-%s.%s\n", item.Name, item.Version, item.Release, item.Arch)
			}
			res = append(res, tmp[:]...)
		}
	}

	if len(reqinfo) != 0 {
		return nil, fmt.Errorf("Not Such Package ", reqinfo)
	}

	fmt.Printf("---->%s<------\n", in)
	for _, pack := range res {
		if pack.Epoch == "" {
			fmt.Printf("%s-%s-%s.%s\n", pack.Name, pack.Version, pack.Release, pack.Arch)
		} else {
			fmt.Printf("%s-%s:%s-%s.%s\n", pack.Name, pack.Epoch, pack.Version, pack.Release, pack.Arch)
		}
	}
	fmt.Printf("---->%s<------\n", in)

	return res, nil
}
