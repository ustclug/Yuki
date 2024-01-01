package utils

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func PollUntilTimeout(t *testing.T, timeout time.Duration, f func() bool) {
	timer := time.NewTimer(timeout)
	t.Cleanup(func() {
		if !timer.Stop() {
			<-timer.C
		}
	})
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
loop:
	for {
		select {
		case <-ticker.C:
			if f() {
				break loop
			}
		case <-timer.C:
			t.Fatal("Timeout")
		}
	}
}

func WriteFile(t *testing.T, path, content string) {
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
