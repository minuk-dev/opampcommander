// Package version provides utilities for handling version information.
package version

var (
	//nolint:gochecknoglobals
	gitMajor string
	//nolint:gochecknoglobals
	gitMinor string
	// NOTE: The $Format strings are replaced during 'git archive' thanks to the
	// companion .gitattributes file containing 'export-subst' in this same
	// directory.  See also https://git-scm.com/docs/gitattributes
	//nolint:gochecknoglobals
	gitVersion = "v0.0.0-master+$Format:%H$"
	//nolint:gochecknoglobals
	gitCommit = "$Format:%H$" // sha1 from git, output of $(git rev-parse HEAD)
	//nolint:gochecknoglobals
	gitTreeState = ""
	//nolint:gochecknoglobals
	buildDate = "1970-01-01T00:00:00Z" // build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
)
