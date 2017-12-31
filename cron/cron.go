package cron

import (
	"sync"
	"gopkg.in/robfig/cron.v2"
)

type scheduledJobs struct {
	sync.RWMutex
	items map[string]cron.EntryID
}
type Cron struct {
	*cron.Cron
}
type FuncJob cron.FuncJob

var ScheduledJobs scheduledJobs

func init() {
	ScheduledJobs = scheduledJobs{
		items: make(map[string]cron.EntryID),
	}
}

func (jobs *scheduledJobs) Get(name string) (cron.EntryID, bool) {
	jobs.RLock()
	defer jobs.RUnlock()
	id, ok := jobs.items[name]
	return id, ok
}

func (jobs *scheduledJobs) Remove(name string) {
	jobs.Lock()
	defer jobs.Unlock()
	delete(jobs.items, name)
}

func (jobs *scheduledJobs) Set(name string, id cron.EntryID) {
	jobs.Lock()
	defer jobs.Unlock()
	jobs.items[name] = id
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
	ScheduledJobs.Set(name, id)
	return nil
}

func (c *Cron) RemoveJob(name string) {
	if id, ok := ScheduledJobs.Get(name); ok {
		c.Cron.Remove(id)
		ScheduledJobs.Remove(name)
	}
}
