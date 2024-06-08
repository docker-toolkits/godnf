// parse repomd.xml
// https://pkg.go.dev/github.com/oneumyvakin/rpmeta#pkg-functions
package repodata

import (
	"encoding/xml"
	"fmt"
	"github/luochenglcs/godnf/dnflog"
	"io"
	"net/http"
	"strings"

	"github.com/oneumyvakin/rpmeta"
)

var archMap = map[string]string{
	"amd64":   "x86_64",
	"386":     "i386",
	"686":     "i686",
	"arm64":   "aarch64",
	"arm":     "arm",
	"ppc64":   "ppc64",
	"ppc64le": "ppc64le",
	"s390x":   "s390x",
}

func GetMetadata(url string) (map[string]rpmeta.RepoMdData, error) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("ERROR ", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("ERROR: status code ", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}

	htmlContent := string(body)

	reader := strings.NewReader(htmlContent)

	decoder := xml.NewDecoder(reader)

	var r rpmeta.RepoMd

	err = decoder.Decode(&r)
	if err != nil {
		fmt.Println("ERROR: xml Decode ", err)
	}
	repomds := make(map[string]rpmeta.RepoMdData)

	for _, data := range r.Data {
		repomds[data.Type] = data
	}

	dnflog.L.Debug("Revision :%s\n", r.Revision)
	for key, data := range repomds {
		dnflog.L.Debug("Type :%s\n", key)
		dnflog.L.Debug("Checksum: %s %s\n", data.Checksum.Type, data.Checksum.Value)
		dnflog.L.Debug("OpenChecksum: %s %s\n", data.OpenChecksum.Type, data.OpenChecksum.Value)
		dnflog.L.Debug("Location: %s\n", data.Location.Href)
		dnflog.L.Debug("TimeStamp: %s\n", data.Timestamp)
		dnflog.L.Debug("Size: %s\n", data.Size)
		dnflog.L.Debug("OpenSize: %s\n", data.OpenSize)
	}

	return repomds, nil
}
