#!/usr/bin/node

'use strict'

import Docker from 'dockerode'

//export default new Docker({
  //socketPath: '/var/run/docker.sock'
//})

export default new Docker({
  host: '127.0.0.1',
  port: 2375
})
