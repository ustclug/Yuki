#!/usr/bin/node

'use strict'

import path from 'path'
import schedule from 'node-schedule'
import logger from './logger'
import { Repository as Repo } from './models'
import CONFIG from './config'
import { bringUp, queryOpts, dirExists, makeDir } from './util'

class Scheduler {
  constructor() {
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

      try {
        await bringUp(opts)
      } catch (e) {
        return logger.error(`bringUp ${name}: %s`, e)
      }
      logger.debug(`Syncing ${name}`)
    })

    logger.info(`${name} scheduled`)
    return true
  }

  isScheduled(name) {
    return !!schedule.scheduledJobs[name]
  }

  addCusJob(name, spec, cb) {
    return schedule.scheduleJob(name, spec, cb)
  }

  cancelJob(name) {
    return schedule.cancelJob(name)
  }

  schedRepos() {
    return Repo.find({}, { interval: true, name: true })
    .then(docs => {
      docs.forEach(doc => {
        // this -> scheduler instance
        this.addJob(doc.name, doc.interval)
      })
    })
  }
}

export default new Scheduler()
