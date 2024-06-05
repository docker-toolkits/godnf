package version

const (
	defaultVersion = "v1.0.0"
)

var (
	// Package is filled at linking time
	Package = "github.com/luochenglcs/godnf"

	// Version holds the complete version number. Filled in at linking time.
	Version = defaultVersion

	// Revision is filled with the VCS (e.g. git) revision being used to build
	// the program at linking time.
	Revision = ""
)
