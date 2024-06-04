// parse repomd.xml
// https://pkg.go.dev/github.com/oneumyvakin/rpmeta#pkg-functions
package repodata

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/oneumyvakin/rpmeta"
)

func GetMetadata(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("ERROR")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("ERROR: status code ", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return err
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

	fmt.Printf("Revision :%s\n", r.Revision)
	for key, data := range repomds {
		fmt.Printf("Type :%s\n", key)
		fmt.Printf("Checksum: %s %s\n", data.Checksum.Type, data.Checksum.Value)
		fmt.Printf("OpenChecksum: %s %s\n", data.OpenChecksum.Type, data.OpenChecksum.Value)
		fmt.Printf("Location: %s\n", data.Location.Href)
		fmt.Printf("TimeStamp: %s\n", data.Timestamp)
		fmt.Printf("Size: %s\n", data.Size)
		fmt.Printf("OpenSize: %s\n", data.OpenSize)
	}

	return nil
}
