package source

import (
	"compress/bzip2"
	"fmt"
	sqlquery "github/luochenglcs/godnf/source/sqlite"
	"io"
	"net/http"
	"os"
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

func GetSql(url string) error {
	err := Download(url, "/var/cache/godnf/BaseOS/baseos-primary.sqlite.bz2")
	if err != nil {
		fmt.Println("ERROR download failed ", url, " /var/cache/godnf/BaseOS/baseos-primary.sqlite.bz2")
		return err
	}

	err = decompressBZ2("/var/cache/godnf/BaseOS/baseos-primary.sqlite.bz2", "/var/cache/godnf/BaseOS/baseos-primary.sqlite")
	if err != nil {
		fmt.Println("ERROR decompressBZ2 failed")
		return err
	}

	os.Remove("/var/cache/godnf/BaseOS/baseos-primary.sqlite.bz2")
	return nil
}

func GetRpm(baseurl string, r []sqlquery.ReqRes) error {
	for _, pack := range r {
		var packfile string
		if pack.Epoch == "" {
			packfile = fmt.Sprintf("%s-%s-%s.%s.rpm", pack.Name, pack.Version, pack.Release, pack.Arch)
		} else {
			packfile = fmt.Sprintf("%s-%s:%s-%s.%s.rpm", pack.Name, pack.Epoch, pack.Version, pack.Release, pack.Arch)
		}
		downurl := fmt.Sprintf("%s/%s", baseurl, packfile)
		dest := fmt.Sprintf("%s/%s", "/var/cache/godnf/BaseOS/packages/", packfile)
		fmt.Println(downurl, " ", dest)
		err := Download(downurl, dest)
		if err != nil {
			fmt.Println("ERROR download failed ", downurl, " ", dest)
			return err
		}
	}
	return nil
}
