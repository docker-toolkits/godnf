package install

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

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

func ExtractRPM(name string) {
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
}
