#!/usr/bin/node

'use strict'

import Docker from 'dockerode'
import Promise from 'bluebird'
import CONFIG from './config'
import logger from './logger'
import { IS_PROD } from './globals'
import isListening from 'is-listening'

const daemon = new Map()
daemon.set('tcp', {
  host: CONFIG.DOCKERD_HOST,
  port: CONFIG.DOCKERD_PORT,
  Promise: Promise
})
daemon.set('socket', {
  socketPath: CONFIG.DOCKERD_SOCKET,
  Promise: Promise
})

let docker = null
if (!IS_PROD &&
    isListening(CONFIG.DOCKERD_PORT, CONFIG.DOCKERD_HOST)) {
  // Check synchronously if the socket can be connected
  // with native addon
  logger.debug('dockerd: TCP socket connected')
  docker = new Docker(daemon.get('tcp'))
} else if (isListening(CONFIG.DOCKERD_SOCKET)) {
  logger.debug('dockerd: UNIX local socket connected')
  docker = new Docker(daemon.get('socket'))
}

if (docker === null) {
  logger.error('Unable to connect to docker daemon')
  process.exit(1)
}

export default docker
