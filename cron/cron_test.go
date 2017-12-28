package cron

import (
	"sync"
	"time"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestCron(t *testing.T) {
	var wg sync.WaitGroup
	now := time.Now()
	wg.Add(1)
	cron := New()
	cron.AddFunc("@every 3s", func() {
		defer wg.Done()
	})
	cron.Start()
	wg.Wait()
	now2 := time.Now()
	assert.True(t, now2.After(now))
}
