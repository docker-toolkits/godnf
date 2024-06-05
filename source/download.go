package source

import (
	"compress/bzip2"
	"fmt"
	"github/luochenglcs/godnf/dnflog"
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

	dnflog.L.Debug("File downloaded successfully")
	return nil
}

func GetSql(url string, dst string) error {
	dnflog.L.Debug("GetSql ", url, " ", dst)
	// Create the target directory
	if dirName := filepath.Dir(dst); dirName != "" {
		if err := os.MkdirAll(dirName, 0o755); err != nil {
			log.Fatal(err)
		}
	}

	err := Download(url, dst)
	if err != nil {
		dnflog.L.Error("ERROR download failed ", url, " ", dst)
		return err
	}

	err = decompressBZ2(dst, dst[:len(dst)-4])
	if err != nil {
		dnflog.L.Error("ERROR decompressBZ2 failed")
		return err
	}

	os.Remove(dst)
	return nil
}

func GetRpm(destdir string, repoConfs map[string]repodata.RepoConfig, pack sqlquery.ReqRes) error {

	dnflog.L.Debug("GetRpm %s %s\n", pack.DbPath, pack.Name)

	var packfile string
	trimpath := strings.TrimPrefix(pack.DbPath, destdir)
	parts := strings.Split(trimpath, "/")
	if len(parts) <= 2 {
		return fmt.Errorf("Not Such Packages")
	}
	repoKey := parts[len(parts)-2]
	if pack.Epoch == "" {
		packfile = fmt.Sprintf("%s-%s-%s.%s.rpm", pack.Name, pack.Version, pack.Release, pack.Arch)
	} else {
		packfile = fmt.Sprintf("%s-%s:%s-%s.%s.rpm", pack.Name, pack.Epoch, pack.Version, pack.Release, pack.Arch)
	}

	downurl1 := fmt.Sprintf("%s/%s", repoConfs[repoKey].BaseURL, packfile)
	downurl2 := fmt.Sprintf("%s/%s/%s", repoConfs[repoKey].BaseURL, "Packages", packfile)
	dstPath := fmt.Sprintf("%s/%s/%s/%s", destdir, "/var/cache/godnf/", repoKey, "packages")

	if err := os.MkdirAll(dstPath, 0o755); err != nil {
		log.Fatal(err)
	}

	dest := fmt.Sprintf("%s/%s", dstPath, packfile)

	dnflog.L.Debug(downurl1, " ", dest)
	err1 := Download(downurl1, dest)
	if err1 != nil {
		dnflog.L.Debug(downurl2, " ", dest)
		err2 := Download(downurl2, dest)
		if err2 != nil {
			dnflog.L.Error("ERROR download failed ", downurl2, " ", dest)
			return err2
		}
	}

	return nil
}
