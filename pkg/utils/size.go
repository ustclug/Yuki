package utils

import (
	"fmt"
)

func PrettySize(size int64) string {
	if size < 0 {
		return "unknown"
	}
	const n = float64(1024)
	a := float64(size)
	units := []string{"B", "KiB", "MiB", "GiB"}
	for _, u := range units {
		if a < n {
			return fmt.Sprintf("%.1f %s", a, u)
		}
		a /= n
	}
	return fmt.Sprintf("%.1f TiB", a)
}
