// Package cron provides some cron utility functions.
package cron

import (
	"sync"

	"gopkg.in/knight42/cron.v3"
)

// Cron wraps `cron.Cron`.
type Cron struct {
	*cron.Cron
}

// FuncJob is alias of `cron.FuncJob`.
type FuncJob cron.FuncJob

var scheduledJobs sync.Map

func init() {
	scheduledJobs = sync.Map{}
}

// Parse parses job specification.
func Parse(spec string) (cron.Schedule, error) {
	return cron.Parse(spec)
}

// New returns an instance of Cron.
func New() *Cron {
	c := cron.New()
	c.Start()
	return &Cron{c}
}

// AddJob removes the job with the same name first and adds a new job.
func (c *Cron) AddJob(name, spec string, cmd FuncJob) error {
	c.RemoveJob(name)
	id, err := c.Cron.AddFunc(spec, cmd)
	if err != nil {
		return err
	}
	scheduledJobs.Store(name, id)
	return nil
}

// RemoveJob remove the job with the given name.
func (c *Cron) RemoveJob(name string) {
	if v, ok := scheduledJobs.Load(name); ok {
		c.Cron.Remove(v.(cron.EntryID))
		scheduledJobs.Delete(name)
	}
}
