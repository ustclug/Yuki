#!/usr/bin/node

//;(function() {
'use strict'

import schedule from 'node-schedule'
import logger from './logger'
import { bringUp, queryOpts, autoRemove } from './util'

//const sche = new schedule()

class Scheduler {
  constructor() {
    this._paused = false
  }

  async addJob(name, spec) {
    if (typeof schedule.scheduledJobs[name] !== 'undefined') {
      schedule.scheduledJobs[name].cancel()
    }
    const opts = await queryOpts({ name, debug: false })
    if (opts === null) {
      logger.warn(`no such repository: ${name}`)
      return
    }

    schedule.scheduleJob(name, spec, async () => {
      let ct
      try {
        ct = await bringUp(opts)
      } catch (e) {
        logger.error(`bringUp ${name}: %s`, e)
        return
      }
      autoRemove(ct)
        .catch(e => logger.error(`Removing ${name}: %s`, e))
    })
    logger.info(`Scheduled ${name}`)
  }

  cancelJob(name) {
    return schedule.cancelJob(name)
  }

  pause() {
    if (this._paused) return
    Object.values(schedule.scheduledJobs)
      .forEach(job => {
        job.cancel(true) // reschedule
      })
    this._paused = true
  }

  resume() {
    if (!this._paused) return
    this._paused = false
  }

}

export default new Scheduler()
