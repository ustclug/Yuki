#!/usr/bin/node

'use strict'

import Docker from 'dockode'

let docker

if (!process.env.NODE_ENV.startsWith('prod')) {
  docker = new Docker({
    host: '127.0.0.1',
    port: 2375
  })
} else {
  docker = new Docker({
    socketPath: '/var/run/docker.sock'
  })
}

export default docker
