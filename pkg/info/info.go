package info

import (
	"runtime"
)

var (
	Version   string
	BuildDate string
	GitCommit string
)

var VersionInfo = struct {
	Version   string
	GoVersion string
	BuildDate string
	GitCommit string
}{
	Version:   Version,
	BuildDate: BuildDate,
	GitCommit: GitCommit,
	GoVersion: runtime.Version(),
}
