package server

import (
	"os"
	"strings"
)

var (
	IsTest bool = false
)

func init() {
	if strings.HasPrefix(os.Getenv("YUKID_ENV"), "test") {
		IsTest = true
	}
}
