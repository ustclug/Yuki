package cron

import (
	"sync"

	"gopkg.in/knight42/cron.v3"
)

type Cron struct {
	*cron.Cron
}

type FuncJob cron.FuncJob

var scheduledJobs sync.Map

func init() {
	scheduledJobs = sync.Map{}
}

func Parse(spec string) (cron.Schedule, error) {
	return cron.Parse(spec)
}

func New() *Cron {
	c := cron.New()
	c.Start()
	return &Cron{c}
}

func (c *Cron) AddJob(name, spec string, cmd FuncJob) error {
	c.RemoveJob(name)
	id, err := c.Cron.AddFunc(spec, cmd)
	if err != nil {
		return err
	}
	scheduledJobs.Store(name, id)
	return nil
}

func (c *Cron) RemoveJob(name string) {
	if v, ok := scheduledJobs.Load(name); ok {
		c.Cron.Remove(v.(cron.EntryID))
		scheduledJobs.Delete(name)
	}
}
