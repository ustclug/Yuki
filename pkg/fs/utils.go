package fs

import (
	"os"
)

// dirExists checks whether given path is an existing directory.
func dirExists(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return stat.IsDir()
}
