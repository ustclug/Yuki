package cron

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCron(t *testing.T) {
	var wg sync.WaitGroup
	now := time.Now()
	next := now.Add(time.Second * 4)
	wg.Add(1)
	cron := New()
	_, err := cron.AddFunc("@every 3s", func() {
		wg.Done()
	})
	assert.Nil(t, err)
	wg.Wait()
	now2 := time.Now()
	assert.True(t, now2.After(now) && now2.Before(next))
}
