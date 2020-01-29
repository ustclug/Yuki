package utils

import (
	"fmt"
	"os"
)

func CheckError(err error) {
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
