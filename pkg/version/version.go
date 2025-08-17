package version

import (
	"fmt"
	"runtime"

	v1version "github.com/minuk-dev/opampcommander/api/v1/version"
)

func Get() v1version.Info {
	return v1version.Info{
		Major:        gitMajor,
		Minor:        gitMinor,
		GitVersion:   dynamicGitVersion.Load().(string),
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
