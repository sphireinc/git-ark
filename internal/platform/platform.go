package platform

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
}

func Current() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
	}
}

func String() string {
	info := Current()
	return fmt.Sprintf("version=%s commit=%s build_date=%s go=%s %s/%s", info.Version, info.Commit, info.BuildDate, info.GoVersion, info.GOOS, info.GOARCH)
}

func ModuleVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" {
		return bi.Main.Version
	}
	return Version
}
