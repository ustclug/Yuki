#!/usr/bin/node

'use strict'

import Docker from 'dockode'
import Promise from 'bluebird'
import config from './config'
import logger from './logger'

let isListening
try {
  isListening = require('../build/Release/addon.node').isListening;
} catch (e) {
  isListening = require('../build/Debug/addon.node').isListening;
}

const daemon = new Map()
daemon.set('tcp', {
  host: config.DOCKERD_HOST,
  port: config.DOCKERD_PORT,
  promiseLibrary: Promise
})
daemon.set('socket', {
  socketPath: config.DOCKERD_SOCKET,
  promiseLibrary: Promise
})

let docker = null
if (!config.isProd &&
    isListening(config.DOCKERD_HOST, config.DOCKERD_PORT)) {
  // Check synchronously if the socket can be connected
  // with native addon
  logger.debug('dockerd: TCP socket connected')
  docker = new Docker(daemon.get('tcp'))
} else if (isListening(config.DOCKERD_SOCKET)) {
  logger.debug('dockerd: UNIX local socket connected')
  docker = new Docker(daemon.get('socket'))
}

if (docker === null) {
  logger.error('Unable to connect to docker daemon')
  process.exit(1)
}

export default docker
