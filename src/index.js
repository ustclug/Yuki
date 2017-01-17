#!/usr/bin/env node

'use strict'

const config = require('./config')
const port = config.API_PORT

if (config.isTest) {
  require('./server.js').listen(port)
} else {
  const action = process.argv[2]
  switch (action) {
    case 'daemon': {
      const server = require('./server.js').listen(port, () => {
        const addr = server.address()
        console.log('listening on', addr.address + ':' + addr.port)
      })
    }
      break
    case 'sync':
      require('./client.js')
      break;
    default:
      console.error('Unknown action: ' + action)
  }
}
