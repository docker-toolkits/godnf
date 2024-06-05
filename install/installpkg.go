package install

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	sqlquery "github/luochenglcs/godnf/source/sqlite"

	"github.com/cavaliergopher/cpio"
	"github.com/cavaliergopher/rpm"
	"github.com/ulikunitz/xz"
)

func moveAll(sourceDir, targetDir string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == sourceDir {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(targetDir, relPath)

		destDir := filepath.Dir(destPath)
		err = os.MkdirAll(destDir, os.ModePerm)
		if err != nil {
			return err
		}

		err = os.Rename(path, destPath)
		if err != nil {
			return err
		}

		return nil
	})
}
func isDirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()

	entries, err := f.Readdirnames(0)
	if err != nil {
		return false, err
	}

	return len(entries) == 0, nil
}

func ExtractRPM(destdir string, name string) {
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}
	fmt.Printf("Current directory: %s\n", currentDir)

	err = os.Chdir(destdir)
	if err != nil {
		fmt.Printf("Error changing directory: %v\n", err)
		return
	}

	newDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting new directory: %v\n", err)
		return
	}
	fmt.Printf("New directory: %s\n", newDir)

	// Open a package file for reading
	f, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Read the package headers
	pkg, err := rpm.Read(f)
	if err != nil {
		log.Fatal(err)
	}

	// Check the compression algorithm of the payload
	if compression := pkg.PayloadCompression(); compression != "xz" {
		log.Fatalf("Unsupported compression: %s", compression)
	}

	// Attach a reader to decompress the payload
	xzReader, err := xz.NewReader(f)
	if err != nil {
		log.Fatal(err)
	}

	// Check the archive format of the payload
	if format := pkg.PayloadFormat(); format != "cpio" {
		log.Fatalf("Unsupported payload format: %s", format)
	}

	// Attach a reader to unarchive each file in the payload
	cpioReader := cpio.NewReader(xzReader)
	for {
		// Move to the next file in the archive
		hdr, err := cpioReader.Next()
		if err == io.EOF {
			break // no more files
		}
		if err != nil {
			log.Fatal(err)
		}

		// solve path
		if hdr.Mode.IsDir() {
			fmt.Println(hdr.Name)
			if err := os.MkdirAll(hdr.Name, hdr.FileInfo().Mode()); err != nil {
				log.Fatal(err)
			}
		}

		// solve symlink
		if hdr.Mode&cpio.TypeSymlink == cpio.TypeSymlink {
			fmt.Println(hdr.Name, "->", hdr.Linkname)
			// Create the target directory
			if dirName := filepath.Dir(hdr.Name); dirName != "" {
				if err := os.MkdirAll(dirName, 0o755); err != nil {
					log.Fatal(err)
				}
			}

			if _, err := os.Lstat(hdr.Name); err == nil {
				empty, err := isDirEmpty(hdr.Name)
				if err != nil {
					log.Fatal(err)
				}
				if empty == false {
					if err := moveAll(hdr.Name, hdr.Linkname); err != nil {
						log.Fatal(err)
					}
				}

				err = os.Remove(hdr.Name)
				if err != nil {
					fmt.Printf("Error removing existing symlink: %v\n", err)
					return
				}
			}

			if err := os.Symlink(hdr.Linkname, hdr.Name); err != nil {
				log.Fatal(err)
			}
		}

		// solve file
		if hdr.Mode.IsRegular() {
			fmt.Println(hdr.Name)
			// Create the target directory
			if dirName := filepath.Dir(hdr.Name); dirName != "" {
				if err := os.MkdirAll(dirName, 0o755); err != nil {
					log.Fatal(err)
				}
			}

			// Create and write the file
			outFile, err := os.Create(hdr.Name)
			if err != nil {
				log.Fatal(err)
			}
			if _, err := io.Copy(outFile, cpioReader); err != nil {
				outFile.Close()
				log.Fatal(err)
			}
			outFile.Close()
			os.Chmod(hdr.Name, hdr.FileInfo().Mode())
		}

	}

	err = os.Chdir(currentDir)
	if err != nil {
		fmt.Printf("Error changing directory: %v\n", err)
		return
	}

	oldDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting new directory: %v\n", err)
		return
	}
	fmt.Printf("New directory: %s\n", oldDir)
}

func RecordInstalledPkg(destdir string, rpmpkg sqlquery.ReqRes) error {

	dbPath := fmt.Sprintf("%s/%s", destdir, "/var/lib/godnf/godnf_packages.db")
	if dirName := filepath.Dir(dbPath); dirName != "" {
		if err := os.MkdirAll(dirName, 0o755); err != nil {
			log.Fatal(err)
		}
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTableSQL := `CREATE TABLE IF NOT EXISTS packages (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"name" TEXT,
		"epoch" INTEGER,
		"version" TEXT,
		"release" TEXT
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Error creating table: %v\n", err)
	}

	insertPackageSQL := `INSERT INTO packages (name, epoch, version, release) VALUES (?, ?, ?, ?)`
	_, err = db.Exec(insertPackageSQL, rpmpkg.Name, rpmpkg.Epoch, rpmpkg.Version, rpmpkg.Release)
	if err != nil {
		log.Fatalf("Error inserting data: %v\n", err)
	}

	return nil
}

func QueryInstalledPkg(destdir string, name string) (bool, error) {

	dbPath := fmt.Sprintf("%s/%s", destdir, "/var/lib/godnf/godnf_packages.db")

	_, err := os.Stat(dbPath)
	if os.IsNotExist(err) {
		return false, fmt.Errorf("Not exist db")
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	querySQL := `SELECT name, epoch, version, release FROM packages WHERE name=?`
	rows, err := db.Query(querySQL, name)
	if err != nil {
		log.Fatalf("Error querying data: %v\n", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var epoch string
		var version string
		var release string
		err = rows.Scan(&name, &epoch, &version, &release)
		if err != nil {
			log.Fatalf("Error scanning row: %v\n", err)
		}
		fmt.Printf("Name: %s, Epoch: %s, Version: %s, Release: %s\n", name, epoch, version, release)

		return true, nil
	}

	err = rows.Err()
	if err != nil {
		log.Fatalf("Error during row iteration: %v\n", err)
	}
	return false, fmt.Errorf("Not Found In db")
}

func InstallRPM(destdir string, rpmpkg sqlquery.ReqRes) {
	if installed, _ := QueryInstalledPkg(destdir, rpmpkg.Name); installed {
		fmt.Printf("Already Install : Name: %s, Epoch: %s, Version: %s, Release: %s\n", rpmpkg.Name, rpmpkg.Epoch, rpmpkg.Version, rpmpkg.Release)
		return
	}

	trimpath := strings.TrimPrefix(rpmpkg.DbPath, destdir)
	parts := strings.Split(trimpath, "/")
	repoKey := parts[len(parts)-2]

	fmt.Printf("Name: %s Version %s Release %s\n", rpmpkg.Name, rpmpkg.Version, rpmpkg.Release)
	var packfile string
	if rpmpkg.Epoch == "" {
		packfile = fmt.Sprintf("%s-%s-%s.%s.rpm", rpmpkg.Name, rpmpkg.Version, rpmpkg.Release, rpmpkg.Arch)
	} else {
		packfile = fmt.Sprintf("%s-%s:%s-%s.%s.rpm", rpmpkg.Name, rpmpkg.Epoch, rpmpkg.Version, rpmpkg.Release, rpmpkg.Arch)
	}
	filepath := fmt.Sprintf("./%s/%s/packages/%s", "/var/cache/godnf/", repoKey, packfile)
	ExtractRPM(destdir, filepath)
	RecordInstalledPkg(destdir, rpmpkg)
}
