package utils

import "testing"

func TestDirExists(t *testing.T) {
	if !DirExists("/") {
		t.Fatal("/ is not a directory?")
	}
}
