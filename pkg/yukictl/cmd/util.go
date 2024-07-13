package cmd

import "strings"

const suffixYAML = ".yaml"

func stripSuffix(s string) string {
	return strings.TrimSuffix(s, suffixYAML)
}
