// Package cron provides some cron utility functions.
package cron

import (
	"sync"

	"github.com/robfig/cron/v3"
)

// Cron wraps `cron.Cron`.
type Cron struct {
	inner *cron.Cron
}

// FuncJob is alias of `cron.FuncJob`.
type FuncJob = cron.FuncJob

var scheduledJobs sync.Map

// Parse parses job specification.
func Parse(spec string) (cron.Schedule, error) {
	return cron.ParseStandard(spec)
}

// New returns an instance of Cron.
func New() *Cron {
	c := cron.New()
	c.Start()
	return &Cron{c}
}

func (c *Cron) Entries() []cron.Entry {
	return c.inner.Entries()
}

// Jobs returns a map of job names to job.
func (c *Cron) Jobs() map[string]cron.Entry {
	ret := map[string]cron.Entry{}
	entries := c.inner.Entries()
	id2name := map[cron.EntryID]string{}
	scheduledJobs.Range(func(key, value interface{}) bool {
		name := key.(string)
		id := value.(cron.EntryID)
		id2name[id] = name
		return true
	})
	for _, entry := range entries {
		name, ok := id2name[entry.ID]
		if !ok {
			continue
		}
		ret[name] = entry
	}
	return ret
}

func (c *Cron) AddFunc(spec string, cmd func()) error {
	_, err := c.inner.AddFunc(spec, cmd)
	return err
}

// AddJob removes the job with the same name first and adds a new job.
func (c *Cron) AddJob(name, spec string, cmd FuncJob) error {
	c.RemoveJob(name)
	id, err := c.inner.AddFunc(spec, cmd)
	if err != nil {
		return err
	}
	scheduledJobs.Store(name, id)
	return nil
}

// HasJob returns whether the given job exists.
func (c *Cron) HasJob(name string) bool {
	_, ok := scheduledJobs.Load(name)
	return ok
}

// RemoveJob remove the job with the given name.
func (c *Cron) RemoveJob(name string) {
	if v, ok := scheduledJobs.Load(name); ok {
		c.inner.Remove(v.(cron.EntryID))
		scheduledJobs.Delete(name)
	}
}
