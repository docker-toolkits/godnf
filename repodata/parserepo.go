package repodata

import (
	"bufio"
	"fmt"
	"github/luochenglcs/godnf/dnflog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/ini.v1"
)

// RepoConfig represents the structure of a YUM repo configuration
type RepoConfig struct {
	Name     string
	BaseURL  string
	Enabled  bool
	GPGCheck bool
}

func getVerAndArch() (release, arch string) {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		dnflog.L.Error("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var versionID string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VERSION_ID=") {
			versionID = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), `"`)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		dnflog.L.Error("Error reading file: %v\n", err)
		return
	}
	dnflog.L.Debug(versionID)
	return versionID, archMap[runtime.GOARCH]
}

func GetRepo() (map[string]RepoConfig, error) {
	root := "/etc/yum.repos.d/"
	repoConfigs := make(map[string]RepoConfig)
	release, arch := getVerAndArch()
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// Load the .repo file
		if err != nil {
			dnflog.L.Error("error file %q: %v\n", path, err)
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".repo" {
			return nil
		}
		dnflog.L.Debug(path)
		cfg, err := ini.Load(path)
		if err != nil {
			return nil
		}
		repoConfigs = make(map[string]RepoConfig)

		// Iterate through the sections in the .repo file
		for _, section := range cfg.Sections() {
			if section.Name() == "DEFAULT" {
				continue
			}

			// Parse the section into a RepoConfig struct
			rc := RepoConfig{
				Name:     section.Name(),
				BaseURL:  section.Key("baseurl").String(),
				Enabled:  section.Key("enabled").MustBool(false),
				GPGCheck: section.Key("gpgcheck").MustBool(false),
			}

			rc.BaseURL = strings.Replace(rc.BaseURL, "$releasever", release, 1)
			rc.BaseURL = strings.Replace(rc.BaseURL, "$basearch", arch, 1)
			repoConfigs[rc.Name] = rc
		}

		return nil
	})

	for key, rc := range repoConfigs {
		if rc.Enabled {
			fmt.Printf("reponame:%s url:%s\n", key, rc.BaseURL)
			dnflog.L.Debug("key: ", key)
			dnflog.L.Debug("name: ", rc.Name)
			dnflog.L.Debug("BaseURL: ", rc.BaseURL)
			dnflog.L.Debug("Enabled: ", rc.Enabled)
			dnflog.L.Debug("GPGCheck: ", rc.GPGCheck)
		}
	}

	return repoConfigs, nil
}
