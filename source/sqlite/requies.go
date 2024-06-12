/*
SELECT pkgKey FROM packages WHERE Name='systemd';
SELECT * FROM requires WHERE pkgKey=12345;

SELECT p.* FROM requires r
JOIN packages p ON r.Name = p.Name
WHERE r.pkgKey = (SELECT pkgKey FROM packages WHERE Name='systemd');
*/
package source

/* github.com/mattn/go-sqlite3 Depends on C, so choose modernc.org/sqlite which doesn't depends on C */
import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github/luochenglcs/godnf/dnflog"

	_ "modernc.org/sqlite"
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
func GetAllRequres(in string, arch string, l int, res *[][]ReqRes, dbpaths []string) {
	if in != "" {
		re, cur, _ := GetRequres(in, arch, dbpaths)
		//add it to new level
		if l >= len(*res) {
			*res = append(*res, []ReqRes{})
		}
		(*res)[l] = append((*res)[l], cur)
		for _, item := range re {
			var existed bool = false
			//var pos int
			for _, row := range *res {
				existed, _ = IsExisted(row, item)
				if existed {
					break
				}
			}
			if !existed {
				GetAllRequres(item.Name, arch, l+1, res, dbpaths)
			}
		}
	}
}

// v1 > v2: 1
// v1 < v2; -1
// v1 = v2: 0

func comparestring(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var num1 int = 0
		var num2 int = 0

		if i < len(parts1) {
			num1, _ = strconv.Atoi(parts1[i])
		}

		if i < len(parts2) {
			num2, _ = strconv.Atoi(parts2[i])
		}

		if num1 < num2 {
			return -1
		} else if num1 > num2 {
			return 1
		}
	}

	return 0
}

// p1 > p2: 1
// p1 < p2; -1
// p1 = p2: 0
func CompVerRelease(p1, p2 ReqRes) int {
	if (comparestring(p1.Version, p2.Version) == 1) ||
		((comparestring(p1.Version, p2.Version) == 0) && (comparestring(p1.Release, p2.Release) == 1)) {
		return 1
	}

	if (comparestring(p1.Version, p2.Version) == 0) && (comparestring(p1.Release, p2.Release) == 0) {
		return 0
	}

	return -1
}

func getRequirePkgname(req queryRes, arch, dbpath string) (res ReqRes, err error) {

	var lastestName string = ""
	var qArch string
	var maxP ReqRes
	var needcomp string = ""
	if req.Flags.Valid {
		needcomp = req.Flags.String
		if req.Epoch.Valid {
			maxP.Epoch = req.Epoch.String
		}
		if req.Version.Valid {
			maxP.Version = req.Version.String
		}
		if req.Release.Valid {
			maxP.Release = req.Release.String
		}
	}

	db, err := sql.Open("sqlite", dbpath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	//var res []ReqRes
	query := fmt.Sprintf("SELECT p.Name,p.Epoch,p.Version,p.Release,p.Arch FROM packages p JOIN provides pr ON p.pkgKey = pr.pkgKey WHERE pr.Name='%s';", req.Name)
	//fmt.Printf("query %s\n", query)
	reqquery, err := db.Query(query)
	if err != nil {
		log.Fatalf("Error executing query provides: %v", err)
	}
	defer reqquery.Close()

	for reqquery.Next() {
		var p2 ReqRes
		err := reqquery.Scan(&p2.Name, &p2.Epoch, &p2.Version, &p2.Release, &qArch)
		if err != nil {
			log.Fatalf("Error scanning row reqquery: %v", err)
		}

		if qArch != "noarch" && qArch != arch {
			continue
		}
		maxP.Arch = qArch
		if maxP.Version == "" && maxP.Release == "" {
			lastestName = p2.Name
			maxP.Epoch = p2.Epoch
			maxP.Version = p2.Version
			maxP.Release = p2.Release
		} else {
			if needcomp == "" || needcomp == "GE" {
				if CompVerRelease(p2, maxP) != -1 {
					lastestName = p2.Name
					maxP.Epoch = p2.Epoch
					maxP.Version = p2.Version
					maxP.Release = p2.Release
				}
			} else if needcomp == "EQ" {
				if CompVerRelease(p2, maxP) == 0 {
					lastestName = p2.Name
					maxP.Epoch = p2.Epoch
					maxP.Version = p2.Version
					maxP.Release = p2.Release
				}
			} else {
				if CompVerRelease(p2, maxP) == -1 {
					lastestName = p2.Name
					maxP.Epoch = p2.Epoch
					maxP.Version = p2.Version
					maxP.Release = p2.Release
				}
			}
		}
	}

	/* No rpm package is queried from the tables 'provides' if lastestName == "",  query from the files table */
	if lastestName == "" {
		query = fmt.Sprintf("SELECT p.Name,p.Epoch,p.Version,p.Release,p.Arch FROM packages p JOIN files pr ON p.pkgKey = pr.pkgKey WHERE pr.Name='%s';", req.Name)
		filequery, err := db.Query(query)
		if err != nil {
			log.Fatalf("Error executing query files: %v", err)
		}
		defer filequery.Close()

		for filequery.Next() {
			var p2 ReqRes
			err := filequery.Scan(&p2.Name, &p2.Epoch, &p2.Version, &p2.Release, &qArch)
			if err != nil {
				log.Fatalf("Error scanning row filequery: %v", err)
			}

			if qArch != "noarch" && qArch != arch {
				continue
			}

			maxP.Arch = qArch
			if maxP.Version == "" && maxP.Release == "" {
				lastestName = p2.Name
				maxP.Epoch = p2.Epoch
				maxP.Version = p2.Version
				maxP.Release = p2.Release
			} else {
				if needcomp == "" || needcomp == "GE" {
					if CompVerRelease(p2, maxP) != -1 {
						lastestName = p2.Name
						maxP.Epoch = p2.Epoch
						maxP.Version = p2.Version
						maxP.Release = p2.Release
					}
				} else if needcomp == "EQ" {
					if CompVerRelease(p2, maxP) == 0 {
						lastestName = p2.Name
						maxP.Epoch = p2.Epoch
						maxP.Version = p2.Version
						maxP.Release = p2.Release
					}
				} else {
					if CompVerRelease(p2, maxP) == -1 {
						lastestName = p2.Name
						maxP.Epoch = p2.Epoch
						maxP.Version = p2.Version
						maxP.Release = p2.Release
					}
				}
			}
		}
	}

	// Not found in current db, record it
	if lastestName == "" {
		return ReqRes{}, fmt.Errorf("not Found")
	}

	//fmt.Printf("Name: %s | %s | %s | %s\n", lastestName, max_epoch, max_version, max_release)
	var resultPkg ReqRes
	resultPkg.DbPath = dbpath
	resultPkg.Name = lastestName
	resultPkg.Version = maxP.Version
	resultPkg.Release = maxP.Release
	resultPkg.Epoch = maxP.Epoch
	resultPkg.Arch = maxP.Arch

	return resultPkg, nil
}

func getRequresInfo(in, arch, dbpath string) ([]queryRes, ReqRes, error) {
	db, err := sql.Open("sqlite", dbpath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	//query := `SELECT pkgKey,Name,Epoch,arch,Version,Release FROM provides WHERE Name=?;`
	query := `SELECT p.pkgKey,p.Name,p.Epoch,p.arch,p.Version,p.Release FROM packages p JOIN provides pr ON p.pkgKey = pr.pkgKey WHERE pr.Name=?;`
	packrows, err := db.Query(query, in)
	if err != nil {
		log.Fatalf("Error getRequresInfo executing query provides: %v", err)
	}
	defer packrows.Close()

	var latestPkgKey int = -1
	var max_epoch, max_version, max_release string
	var lastestarch string = arch
	var qarch, Name string
	var currentpkg ReqRes
	//fmt.Println("Data from 'packages' table:")
	for packrows.Next() {
		var pkgKey int
		var Epoch, Version, Release string
		err := packrows.Scan(&pkgKey, &Name, &Epoch, &qarch, &Version, &Release)
		if err != nil {
			log.Fatalf("Error scanning row packrows: %v", err)
		}

		if qarch != "noarch" && qarch != arch {
			continue
		}
		lastestarch = qarch
		//fmt.Printf("pkgKey: %d, Name: %s, Arch: %s, Version: %s, Release %s\n", pkgKey, Name, qarch, Version, Release)
		if latestPkgKey == -1 {
			latestPkgKey = pkgKey
			//max_epoch = Epochd
			max_version = Version
			max_release = Release
		} else {
			if (strings.Compare(Version, max_version) == 1) ||
				((strings.Compare(Version, max_version) == 0) && (strings.Compare(Release, max_release) != -1)) {
				latestPkgKey = pkgKey
				//max_epoch = Epoch
				max_version = Version
				max_release = Release
			}
		}
	}

	// Don't find package in current db
	if latestPkgKey == -1 {
		return nil, ReqRes{}, fmt.Errorf("Not Found Package in db")
	}
	currentpkg.Name = Name
	currentpkg.DbPath = dbpath
	currentpkg.Epoch = max_epoch
	currentpkg.Arch = lastestarch
	currentpkg.Version = max_version
	currentpkg.Release = max_release

	//fmt.Printf("Max pkgKey: %d,  Version: %s, Release %s\n", latestPkgKey, max_version, max_release)
	query = `SELECT Name,Flags,Epoch,Version,Release FROM requires WHERE pkgKey=?;`
	reqrows, err := db.Query(query, latestPkgKey)
	if err != nil {
		log.Fatalf("Error executing query requires:  %v", err)
	}
	defer reqrows.Close()

	var requires []queryRes
	for reqrows.Next() {
		var req queryRes
		err := reqrows.Scan(&req.Name, &req.Flags, &req.Epoch, &req.Version, &req.Release)
		if err != nil {
			log.Fatalf("Error scanning row reqrows: %v", err)
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
	return requires, currentpkg, nil
}

func GetRequres(in string, arch string, dbpaths []string) ([]ReqRes, ReqRes, error) {
	var reqinfo []queryRes
	var cur ReqRes

	for _, db := range dbpaths {
		r, c, err := getRequresInfo(in, arch, db)
		if err == nil {
			if cur.Name == "" {
				cur = c
				reqinfo = r
			} else {
				if CompVerRelease(c, cur) == 1 {
					cur = c
					reqinfo = r

				}
			}
		}
	}

	if cur.Name == "" {
		dnflog.L.Info("Pkg %s >> %s-%s-%s\n", in, cur.Name, cur.Version, cur.Release)
	}

	var res []ReqRes
	for _, item := range reqinfo {
		var maxpkg ReqRes
		for _, db := range dbpaths {
			t, err := getRequirePkgname(item, arch, db)
			if err == nil {
				if maxpkg.Name == "" {
					maxpkg = t
				} else {
					if CompVerRelease(t, maxpkg) == 1 {
						maxpkg = t
					}
				}
			}
		}

		/* if maxpkg.Name != 0, mean have requires pkg not found in db */
		if maxpkg.Name == "" {
			dnflog.L.Error("Not Such Package ", item)
			log.Fatalf("Not Such Package: %v", item)
			return nil, ReqRes{}, fmt.Errorf("Not Such Package ", item)
		}

		if existed, _ := IsExisted(res, maxpkg); !existed {
			res = append(res, maxpkg)
		}
	}

	dnflog.L.Debug("---->%s %v<------\n", in, cur)
	for _, pack := range res {
		if pack.Epoch == "" {
			dnflog.L.Debug("%s-%s-%s.%s\n", pack.Name, pack.Version, pack.Release, pack.Arch)
		} else {
			dnflog.L.Debug("%s-%s:%s-%s.%s\n", pack.Name, pack.Epoch, pack.Version, pack.Release, pack.Arch)
		}
	}
	dnflog.L.Debug("---->%s<------\n", in)

	return res, cur, nil
}
