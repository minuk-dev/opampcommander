package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"

	v1version "github.com/minuk-dev/opampcommander/api/v1/version"
)

// Get returns the version information.
// When the binary is built without goreleaser ldflags (e.g. go install, go build),
// it falls back to VCS info embedded by the Go toolchain via debug.ReadBuildInfo.
func Get() v1version.Info {
	gitVer := gitVersion
	commit := gitCommit
	treeState := gitTreeState

	if strings.Contains(gitVer, "$Format:") {
		if info, ok := debug.ReadBuildInfo(); ok {
			gitVer, commit, treeState = versionFromBuildInfo(info)
		}
	}

	return v1version.Info{
		Major:        gitMajor,
		Minor:        gitMinor,
		GitVersion:   gitVer,
		GitCommit:    commit,
		GitTreeState: treeState,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

const shortHashLen = 12

func versionFromBuildInfo(info *debug.BuildInfo) (string, string, string) {
	var revision, modified string

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}

	gitVer := buildGitVersion(info.Main.Version, revision)
	treeState := buildTreeState(revision, modified)

	return gitVer, revision, treeState
}

func buildGitVersion(mainVersion, revision string) string {
	switch {
	case mainVersion != "" && mainVersion != "(devel)":
		return mainVersion
	case revision != "":
		short := revision
		if len(short) > shortHashLen {
			short = short[:shortHashLen]
		}

		return "v0.0.0-dev+" + short
	default:
		return "v0.0.0-dev"
	}
}

func buildTreeState(revision, modified string) string {
	switch {
	case modified == "true":
		return "dirty"
	case revision != "":
		return "clean"
	default:
		return ""
	}
}
