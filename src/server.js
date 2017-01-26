#!/usr/bin/node

'use strict'

import Koa from 'koa'
import routes from './routes'
import mongoose from 'mongoose'
import docker from './docker'
import CONFIG from './config'
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
  user: CONFIG.DB_USER,
  pass: CONFIG.DB_PASSWD,
  promiseLibrary: Promise,
}
mongoose.Promise = Promise

if (CONFIG.isTest) {
  mongoose.connect('127.0.0.1', 'test')
} else {
  mongoose.connect('127.0.0.1', CONFIG.DB_NAME, CONFIG.DB_PORT, dbopts)
}
logger.info('Connected to MongoDB')

const Repo = models.Repository

const server = app.listen(CONFIG.API_PORT, () => {
  const addr = server.address()
  logger.info(`listening on ${addr.address}:${addr.port}`)
})

if (!CONFIG.isTest) {
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
  .then(() => Repo.find({}, { interval: true, name: true }))
  .then(docs => {
    docs.forEach(doc => {
      schedule.addJob(doc.name, doc.interval)
    })
  })
}
