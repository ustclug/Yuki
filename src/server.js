#!/usr/bin/node

'use strict'

import Koa from 'koa'
import routes from './routes'
import mongoose from 'mongoose'
import docker from './docker'
import config from './config'
import logger from './logger'
import models from './models'
import schedule from './scheduler'

const app = new Koa()
module.exports = app

app.use(routes)
app.on('error', (err) => {
  console.error('Uncaught error: ', err)
})

const dbopts = {
  user: config.DB_USER,
  pass: config.DB_PASSWD,
  promiseLibrary: Promise,
}
mongoose.Promise = Promise

logger.info('Connecting to MongoDB')
if (config.isTest) {
  mongoose.connect('127.0.0.1', 'test')
} else {
  mongoose.connect('127.0.0.1', config.DB_NAME, config.DB_PORT, dbopts)
}

const Repo = models.Repository

const server = app.listen(config.API_PORT, () => {
  const addr = server.address()
  logger.info(`listening on ${addr.address}:${addr.port}`)
})

// cleanup
if (config.isProd) {
  docker.listContainers({
    all: true,
    filters: {
      label: {
        syncing: true
      },
      status: {
        exited: true
      }
    }
  })
  .then(cts => {
    logger.info('Cleaning exited containers')
    cts.forEach(info => {
      const ct = docker.getContainer(info.Id)
      ct.remove({ v: true })
        .catch(console.error)
    })
  })
}

if (!config.isTest) {
  Repo.find({}, { interval: true, name: true })
  .then(docs => {
    docs.forEach(doc => {
      schedule.addJob(doc.name, doc.interval)
    })
  })
}
