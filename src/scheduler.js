#!/usr/bin/node

'use strict'

import path from 'path'
import schedule from 'node-schedule'
import logger from './logger'
import { Repository as Repo } from './models'
import CONFIG from './config'
import { bringUp, queryOpts, autoRemove, dirExists, makeDir } from './util'

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
      logger.warn(`scheduler: could not get options of ${name} from db`)
      return false
    }
    const cfg = await Repo.findById(name, 'storageDir')
    if (!dirExists(cfg.storageDir)) {
      logger.warn(`scheduler: no such directory: ${cfg.storageDir}`)
      return false
    }

    schedule.scheduleJob(name, spec, async () => {
      const opts = await queryOpts({ name, debug: false })
      if (opts === null) {
        logger.error(`Job ${name}: could not get options from db`)
        return
      }
      const cfg = await Repo.findById(name, 'storageDir')
      if (!dirExists(cfg.storageDir)) {
        logger.warn(`Job ${name}: no such directory: ${cfg.storageDir}`)
        return
      }

      const logdir = path.join(CONFIG.LOGDIR_ROOT, name)
      try {
        makeDir(logdir)
      } catch (e) {
        logger.error(`Job ${name}: ${e.message}`)
        return
      }

      let ct
      try {
        ct = await bringUp(opts)
      } catch (e) {
        return logger.error(`bringUp ${name}: %s`, e)
      }
      logger.debug(`Syncing ${name}`)
      autoRemove(ct)
        .catch(e => logger.error(`Removing ${name}: %s`, e))
    })

    logger.info(`${name} scheduled`)
    return true
  }

  addCusJob(name, spec, cb) {
    return schedule.scheduleJob(name, spec, cb)
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
    Repo.find({}, { interval: true })
    .forEach(r => {
      schedule.scheduledJobs[r._id].reschedule(r.interval)
    })
    this._paused = false
  }

}

export default new Scheduler()
