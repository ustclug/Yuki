#!/usr/bin/node

'use strict'

import Koa from 'koa'
import Promise from 'bluebird'
import mongoose from 'mongoose'
import { exec } from 'child_process'
import routes from './routes'
import CONFIG from './config'
import logger from './logger'
import scheduler from './scheduler'
import fs from './filesystem'
import { User, Meta } from './models'
import { createMeta } from './repositories'
import {
  cleanImages,
  cleanContainers,
  observeRunningContainers,
  updateImages } from './containers'
import { IS_TEST, EMITTER } from './globals'

const app = new Koa()
const server = require('http').createServer(app.callback())
const io = require('socket.io')(server)
io.on('connection', (socket) => {
  require('./ws/shell')(socket)
})

app.use(routes)
app.on('error', (err) => {
  logger.error('Uncaught error in App: %s', err)
})
process.on('uncaughtException', (err) => {
  logger.error('Uncaught exception: %s', err)
  if (err.code === 'EPIPE') {
    // Ignore EPIPE
    return
  }
  process.exit(1)
})

const dbopts = {
  useMongoClient: true,
  user: CONFIG.get('DB_USER'),
  pass: CONFIG.get('DB_PASSWD'),
  promiseLibrary: Promise,
}
mongoose.Promise = Promise

if (IS_TEST) {
  mongoose.connect('mongodb://127.0.0.1/test', { useMongoClient: true })
} else {
  mongoose.connect('127.0.0.1', CONFIG.get('DB_NAME'), CONFIG.get('DB_PORT'), dbopts)
}
logger.info('Connected to MongoDB')

const listening = server.listen(CONFIG.get('API_PORT'), CONFIG.get('API_ADDR'), () => {
  const addr = listening.address()
  logger.info(`listening on ${addr.address}:${addr.port}`)
})

if (!IS_TEST) {
  logger.info('Cleaning containers')

  cleanContainers()
    .then(observeRunningContainers, (err) => {
      logger.error('Cleaning containers: %s', err)
    })

  scheduler.schedRepos()
  scheduler.addCusJob('updateImages', CONFIG.get('IMAGES_UPDATE_INTERVAL'), () => {
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

  createMeta()
    .catch((e) => logger.error('createMeta: %s', e))

  EMITTER.on('sync-end', (data) => {
    const meta = {
      size: fs.getSize(data.storageDir),
      lastExitCode: data.exitCode
    }
    if (data.exitCode === 0) {
      meta.lastSuccess = Date.now()
    }
    Meta.findByIdAndUpdate(data.name, meta, { upsert: true })
      .catch((e) => logger.error('updateMeta: %s', e))

    CONFIG.get('POST_SYNC').forEach((cmd) => {
      exec(cmd.format(data), { maxBuffer: 1024 * 1024 }, (e, stdout, stderr) => {
        if (e) {
          logger.error('postSync: %s', e)
          logger.error(stderr)
        }
      })
    })
  })

  User.findOne()
    .then((user) => {
      if (user === null) {
        return User.create({
        // root:root
          name: 'root',
          password: '63a9f0ea7bb98050796b649e85481845',
          admin: true
        })
          .then(() => {
            logger.warn('User `root` with password `root` has been created.')
          }, (err) => {
            logger.error('Creating user <root> in empty db: %s', err)
          })
      }
    })
}
