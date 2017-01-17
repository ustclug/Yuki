#!/usr/bin/node

'use strict'

import Docker from 'dockode'
import Promise from 'bluebird'
import config from './config'

let docker = null
const daemon = new Map()
daemon.set('tcp', {
  host: '127.0.0.1',
  port: 2375,
  promiseLibrary: Promise
})
daemon.set('socket', {
  socketPath: '/var/run/docker.sock',
  promiseLibrary: Promise
})

if (!config.isProd) {
  for (const type of ['tcp', 'socket']) {
    try {
      docker = new Docker(daemon.get(type))
      break
    } catch (e) {
      // ignore
    }
  }
} else {
  docker = new Docker(daemon.get('socket'))
}

if (docker === null) {
  console.error('Unable to connect to docker daemon')
  process.exit(1)
}

export default docker
