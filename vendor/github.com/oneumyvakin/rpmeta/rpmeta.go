// Package rpmeta contains structures for RPM/YUM repository meta files (repodata)
package rpmeta

// RepoMd defines /repodata/repomd.xml structure:
//     <repomd>
//         <revision>1485854918</revision>
//         <data type="filelists">...</data>
//         <data type="primary">...</data>
//         <data type="primary_db">...</data>
//         <data type="other_db">...</data>
//         <data type="other">...</data>
//         <data type="filelists_db">...</data>
//     </repomd>
type RepoMd struct {
	Revision string       `xml:"revision"`
	Data     []RepoMdData `xml:"data"`
}

// RepoMdData defines <data> structure:
//     <data type="primary">
//         <checksum type="sha256">dabe2ce5481d23de1f4f52bdcfee0f9af98316c9e0de2ce8123adeefa0dd08b9</checksum>
//         <open-checksum type="sha256">e1e2ffd2fb1ee76f87b70750d00ca5677a252b397ab6c2389137a0c33e7b359f</open-checksum>
//         <location href="repodata/dabe2ce5481d23de1f4f52bdcfee0f9af98316c9e0de2ce8123adeefa0dd08b9-primary.xml.gz"/>
//         <timestamp>1485854918</timestamp>
//         <size>134</size>
//         <open-size>167</open-size>
//     </data>
type RepoMdData struct {
	Type         string             `xml:"type,attr"`
	Checksum     RepoMdChecksum     `xml:"checksum"`
	OpenChecksum RepoMdChecksum     `xml:"open-checksum"`
	Location     RepoMdDataLocation `xml:"location"`
	Timestamp    string             `xml:"timestamp"`
	Size         string             `xml:"size"`
	OpenSize     string             `xml:"open-size"`
}

// RepoMdChecksum defines <checksum> structure:
//     <checksum type="sha256">dabe2ce5481d23de1f4f52bdcfee0f9af98316c9e0de2ce8123adeefa0dd08b9</checksum>
type RepoMdChecksum struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// RepoMdDataLocation defines <location> structure:
//     <checksum type="sha256">dabe2ce5481d23de1f4f52bdcfee0f9af98316c9e0de2ce8123adeefa0dd08b9</checksum>
type RepoMdDataLocation struct {
	Href string `xml:"href,attr"`
}

// Metadata defines <metadata> structure:
//     <metadata xmlns="http://linux.duke.edu/metadata/common" xmlns:rpm="http://linux.duke.edu/metadata/rpm" packages="13">
//         <package type="rpm">...</package>
//         <package type="rpm">...</package>
//     </metadata>
type Metadata struct {
	PackagesCount int               `xml:"packages,attr"`
	Packages      []MetadataPackage `xml:"package"`
}

// MetadataPackage defines <metadata><package> structure:
//     <package type="rpm">
//         <name>plesk-gems-pre</name>
//         <arch>x86_64</arch>
//         <version epoch="0" ver="0.0.1" rel="centos7.16070614"/>
//         <checksum type="sha256" pkgid="YES">93c40cd196172a17e2c0a0a8f640bb0ea677b9295e82f8084fe7cc2e5a61ed89</checksum>
//         <summary>This package contains prerequisites for ruby gems installation</summary>
//         <description></description>
//         <packager>Plesk &lt;info@plesk.com&gt;</packager>
//         <url></url>
//         <time file="1467794790" build="1467794786"/>
//         <size package="2116" installed="0" archive="124"/>
//         <location href="packages/plesk-gems-pre-0.0.1-centos7.16070614.x86_64.rpm"/>
//     </package>
type MetadataPackage struct {
	Type     string             `xml:"type,attr"`
	Name     string             `xml:"name"`
	Arch     string             `xml:"arch"`
	Version  MetadataVersion    `xml:"version"`
	Checksum RepoMdChecksum     `xml:"checksum"`
	Time     MetadataTime       `xml:"time"`
	Location RepoMdDataLocation `xml:"location"`
}

// MetadataVersion defines <version> structure:
//     <version epoch="0" ver="0.0.1" rel="centos7.16070614"/>
type MetadataVersion struct {
	Epoch string `xml:"epoch,attr"`
	Ver   string `xml:"ver,attr"`
	Rel   string `xml:"rel,attr"`
}

// MetadataTime defines <time> structure:
//     <time file="1467794790" build="1467794786"/>
type MetadataTime struct {
	File  string `xml:"file,attr"`
	Build string `xml:"build,attr"`
}
