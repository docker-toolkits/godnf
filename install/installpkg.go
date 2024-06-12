package install

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github/luochenglcs/godnf/dnflog"
	sqlquery "github/luochenglcs/godnf/source/sqlite"

	"github.com/cavaliergopher/cpio"
	"github.com/cavaliergopher/rpm"
	"github.com/klauspost/compress/zstd"
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
		dnflog.L.Error("Error getting current directory: %v\n", err)
		return
	}
	dnflog.L.Debug("Current directory: %s\n", currentDir)

	err = os.Chdir(destdir)
	if err != nil {
		dnflog.L.Error("Error changing directory: %v\n", err)
		return
	}

	newDir, err := os.Getwd()
	if err != nil {
		dnflog.L.Error("Error getting new directory: %v\n", err)
		return
	}
	dnflog.L.Debug("New directory: %s\n", newDir)

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
	var compression string
	// Check the compression algorithm of the payload
	if compression = pkg.PayloadCompression(); compression != "xz" && compression != "zstd" {
		log.Fatalf("Unsupported compression: %s pkg %s", compression, name)
	}

	var pkgReader io.Reader
	var comperr error
	if compression == "xz" {
		// Attach a reader to decompress the payload
		pkgReader, comperr = xz.NewReader(f)
		if comperr != nil {
			log.Fatal(comperr)
		}
	} else if compression == "zstd" {
		// Attach a reader to decompress the payload
		pkgReader, comperr = zstd.NewReader(f)
		if comperr != nil {
			log.Fatal(comperr)
		}
	}

	// Check the archive format of the payload
	if format := pkg.PayloadFormat(); format != "cpio" {
		log.Fatalf("Unsupported payload format: %s", format)
	}

	// Attach a reader to unarchive each file in the payload
	cpioReader := cpio.NewReader(pkgReader)
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
			dnflog.L.Debug("dir %s", hdr.Name)
			if err := os.MkdirAll(hdr.Name, hdr.FileInfo().Mode()); err != nil {
				log.Fatal(err)
			}
		}

		// solve symlink
		if hdr.Mode&cpio.TypeSymlink == cpio.TypeSymlink {
			dnflog.L.Debug("symlink: ", hdr.Name, "->", hdr.Linkname)
			// Create the target directory
			if dirName := filepath.Dir(hdr.Name); dirName != "" {
				if err := os.MkdirAll(dirName, 0o755); err != nil {
					log.Fatal("MkdirAll ", err)
				}
			}
			dnflog.L.Debug("Lstat: ", hdr.Name, "->", hdr.Linkname)
			if info, err := os.Lstat(hdr.Name); err == nil {
				if info.IsDir() {
					dnflog.L.Debug("Lstat: ", hdr.Name, info)
					empty, err := isDirEmpty(hdr.Name)
					if err != nil {
						log.Fatal("isDirEmpty ", err)
					}

					if !empty {
						if err := moveAll(hdr.Name, hdr.Linkname); err != nil {
							log.Fatal("moveAll ", err)
						}
					}
				}

				err = os.Remove(hdr.Name)
				if err != nil {
					dnflog.L.Error("Error removing existing symlink: %v\n", err)
					return
				}
			}
			dnflog.L.Debug("Symlink: ", hdr.Name, "->", hdr.Linkname)
			if err := os.Symlink(hdr.Linkname, hdr.Name); err != nil {
				log.Fatal(err)
			}
		}

		// solve file
		if hdr.Mode.IsRegular() {
			dnflog.L.Debug("file:%s", hdr.Name)
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
		dnflog.L.Error("Error changing directory: %v\n", err)
		return
	}

	oldDir, err := os.Getwd()
	if err != nil {
		dnflog.L.Error("Error getting new directory: %v\n", err)
		return
	}
	dnflog.L.Debug("New directory: %s\n", oldDir)
}

func InstallRPM(destdir string, rpmpkg sqlquery.ReqRes) {
	if installed, _, _ := sqlquery.QueryInstalledPkg(destdir, rpmpkg.Name, true); installed {
		fmt.Printf("Name: %s-%s-%s is installed\n", rpmpkg.Name, rpmpkg.Version, rpmpkg.Release)
		return
	}

	trimpath := strings.TrimPrefix(rpmpkg.DbPath, destdir)
	parts := strings.Split(trimpath, "/")
	repoKey := parts[len(parts)-2]

	dnflog.L.Debug("Name: %s Version %s Release %s\n", rpmpkg.Name, rpmpkg.Version, rpmpkg.Release)
	var packfile string
	if rpmpkg.Epoch == "" {
		packfile = fmt.Sprintf("%s-%s-%s.%s.rpm", rpmpkg.Name, rpmpkg.Version, rpmpkg.Release, rpmpkg.Arch)
	} else {
		packfile = fmt.Sprintf("%s-%s:%s-%s.%s.rpm", rpmpkg.Name, rpmpkg.Epoch, rpmpkg.Version, rpmpkg.Release, rpmpkg.Arch)
	}
	filepath := fmt.Sprintf("./%s/%s/packages/%s", "/var/cache/godnf/", repoKey, packfile)
	ExtractRPM(destdir, filepath)
	sqlquery.RecordInstalledPkg(destdir, rpmpkg)
}
