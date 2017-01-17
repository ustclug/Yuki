#!/usr/bin/node

'use strict'

import Koa from 'koa'
import routes from './routes'
import mongoose from 'mongoose'
import docker from './docker'
import config from './config'

const app = new Koa()

app.use(routes)
app.on('error', (err) => {
  console.error('Uncaught error: ', err)
})

const uri = config.isTest ?
  `mongodb://${config.DB_HOST}/test` :
  `mongodb://${config.DB_USER}:${config.DB_PASSWD}@\
${config.DB_HOST}:${config.DB_PORT}/${config.DB_NAME}`
mongoose.Promise = Promise

mongoose.connect(uri, {
  promiseLibrary: Promise,
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
    console.log('Cleaning exited containers')
    cts.forEach(info => {
      const ct = docker.getContainer(info.Id)
      ct.remove({ v: true })
        .catch(console.error)
    })
  })
}

module.exports = app
