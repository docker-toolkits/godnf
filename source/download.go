package source

import (
	"compress/bzip2"
	"fmt"
	"github/luochenglcs/godnf/repodata"
	sqlquery "github/luochenglcs/godnf/source/sqlite"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func decompressBZ2(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening source file: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("error creating destination file: %v", err)
	}
	defer dstFile.Close()

	bz2Reader := bzip2.NewReader(srcFile)

	buf := make([]byte, 32*1024)
	for {
		n, err := bz2Reader.Read(buf)
		if n > 0 {
			if _, err := dstFile.Write(buf[:n]); err != nil {
				return fmt.Errorf("error writing to destination file: %v", err)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading from source file: %v", err)
		}
	}

	return nil
}

func Download(url, dst string) error {
	/*
		client := &getter.Client{
			Src:  url,
			Dst:  dst,
			Mode: getter.ClientModeFile,
		}

		if err := client.Get(); err != nil {
			fmt.Println("Error downloading file:", err)
			return err
		}

		fmt.Println("File downloaded successfully")
	*/
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error fetching URL: %v", err)
	}
	defer resp.Body.Close()

	// 检查HTTP响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error: status code %d", resp.StatusCode)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer out.Close()

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return fmt.Errorf("error writing to file: %v", err)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading response body: %v", err)
		}
	}

	fmt.Println("File downloaded successfully")
	return nil
}

func GetSql(url string, dst string) error {
	fmt.Println("GetSql ", url, " ", dst)
	// Create the target directory
	if dirName := filepath.Dir(dst); dirName != "" {
		if err := os.MkdirAll(dirName, 0o755); err != nil {
			log.Fatal(err)
		}
	}

	err := Download(url, dst)
	if err != nil {
		fmt.Println("ERROR download failed ", url, " ", dst)
		return err
	}

	err = decompressBZ2(dst, dst[:len(dst)-4])
	if err != nil {
		fmt.Println("ERROR decompressBZ2 failed")
		return err
	}

	os.Remove(dst)
	return nil
}

func GetRpm(repoConfs map[string]repodata.RepoConfig, r []sqlquery.ReqRes) error {
	for _, pack := range r {
		var packfile string
		parts := strings.Split(pack.DbPath, "/")
		repoKey := parts[4]
		if pack.Epoch == "" {
			packfile = fmt.Sprintf("%s-%s-%s.%s.rpm", pack.Name, pack.Version, pack.Release, pack.Arch)
		} else {
			packfile = fmt.Sprintf("%s-%s:%s-%s.%s.rpm", pack.Name, pack.Epoch, pack.Version, pack.Release, pack.Arch)
		}

		downurl := fmt.Sprintf("%s/%s", repoConfs[repoKey].BaseURL, packfile)
		dstPath := fmt.Sprintf("%s/%s/%s", "/var/cache/godnf/", repoKey, "packages")

		if err := os.MkdirAll(dstPath, 0o755); err != nil {
			log.Fatal(err)
		}

		dest := fmt.Sprintf("%s/%s", dstPath, packfile)

		fmt.Println(downurl, " ", dest)
		err := Download(downurl, dest)
		if err != nil {
			fmt.Println("ERROR download failed ", downurl, " ", dest)
			return err
		}
	}
	return nil
}
