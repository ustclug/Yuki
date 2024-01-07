package info

import (
	"runtime"
)

var (
	Version   string
	BuildDate string
)

var VersionInfo = struct {
	Version   string
	GoVersion string
	BuildDate string
}{
	Version:   Version,
	BuildDate: BuildDate,
	GoVersion: runtime.Version(),
}
