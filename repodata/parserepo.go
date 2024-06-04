package repodata

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

// RepoConfig represents the structure of a YUM repo configuration
type RepoConfig struct {
	Name     string
	BaseURL  string
	Enabled  bool
	GPGCheck bool
}

func GetRepo() error {
	root := "/etc/yum.repos.d/"
	repoConfigs := make(map[string]RepoConfig)

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// Load the .repo file
		if err != nil {
			fmt.Printf("error file %q: %v\n", path, err)
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".repo" {
			return nil
		}
		fmt.Println(path)
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

			repoConfigs[rc.Name] = rc
		}

		return nil
	})

	for key, rc := range repoConfigs {
		if rc.Enabled == true {
			fmt.Println("key: ", key)
			fmt.Println("name: ", rc.Name)
			fmt.Println("BaseURL: ", rc.BaseURL)
			fmt.Println("Enabled: ", rc.Enabled)
			fmt.Println("GPGCheck: ", rc.GPGCheck)
		}
	}

	return nil
}
