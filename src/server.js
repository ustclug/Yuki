#!/usr/bin/node

'use strict'

import Koa from 'koa'
import Promise from 'bluebird'
import routes from './routes'
import mongoose from 'mongoose'
import CONFIG from './config'
import logger from './logger'
import schedule from './scheduler'
import { updateImages, cleanImages, cleanContainers } from './util'

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

const server = app.listen(CONFIG.API_PORT, CONFIG.API_ADDR, () => {
  const addr = server.address()
  logger.info(`listening on ${addr.address}:${addr.port}`)
})

if (!CONFIG.isTest) {
  logger.info('Cleaning containers')

  Promise.all([
    cleanContainers({ running: true }),
    cleanContainers({ exited: true })
  ])
  .then(() => schedule.schedRepos())
  .catch((err) => logger.error('Cleaning containers: %s', err))

  schedule.addCusJob('updateImages', CONFIG.IMAGES_UPDATE_INTERVAL, () => {
    logger.info('Updating images')
    updateImages()
    .then(cleanImages, (err) => {
      logger.error('Pulling images: %s', err)
    })
    .catch((err) => {
      logger.error('Cleaning images: %s', err)
    })
  })
  logger.info('images-update scheduled')
}
