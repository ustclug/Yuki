// Package common provides some helper functions.
package common

import "os"

// DirExists checks whether given path is an existing directory.
func DirExists(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return stat.IsDir()
}
