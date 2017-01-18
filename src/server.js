#!/usr/bin/node

'use strict'

import Koa from 'koa'
import routes from './routes'
import mongoose from 'mongoose'
import docker from './docker'
import config from './config'
import logger from './logger'

const app = new Koa()
module.exports = app

app.use(routes)
app.on('error', (err) => {
  console.error('Uncaught error: ', err)
})

const uri = config.isTest ?
  `mongodb://${config.DB_HOST}/test` :
  `mongodb://${config.DB_USER}:${config.DB_PASSWD}@\
${config.DB_HOST}:${config.DB_PORT}/${config.DB_NAME}`

mongoose.Promise = Promise

logger.info('Connecting to MongoDB')
mongoose.connect(uri, {
  promiseLibrary: Promise,
})

const server = app.listen(config.API_PORT, () => {
  const addr = server.address()
  console.log(`listening on ${addr.address}:${addr.port}`)
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
