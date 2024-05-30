package main

import (
   	"io"
	"log"
	"os"
	"path/filepath"
    "fmt"

	"github.com/cavaliergopher/cpio"
	"github.com/cavaliergopher/rpm"
	"github.com/ulikunitz/xz"
)
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

		// Skip directories and other irregular file types in this example
		if !hdr.Mode.IsRegular() {
			continue
		}

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
	}
}

func main() {
	pkg, err := rpm.Open("vim-enhanced-9.0.2092-2.ocs23.x86_64.rpm")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Package:", pkg)
	fmt.Println("Summary:", pkg.Summary())
    ExtractRPM("kernel-core-6.6.6-2401.0.1.1.ocs23.x86_64.rpm")
    
	// Output:
	// Package: golang-1.17.2-1.el7.x86_64
	// Summary: The Go Programming Language
}

