// Package version provides api model for version information.
package version

// Info contains version information about the application.
type Info struct {
	Major        string `json:"major"        yaml:"major"`
	Minor        string `json:"minor"        yaml:"minor"`
	GitVersion   string `json:"gitVersion"   yaml:"gitVersion"`
	GitCommit    string `json:"gitCommit"    yaml:"gitCommit"`
	GitTreeState string `json:"gitTreeState" yaml:"gitTreeState"`
	BuildDate    string `json:"buildDate"    yaml:"buildDate"`
	GoVersion    string `json:"goVersion"    yaml:"goVersion"`
	Compiler     string `json:"compiler"     yaml:"compiler"`
	Platform     string `json:"platform"     yaml:"platform"`
} // @name VersionInfo
