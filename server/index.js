#!/usr/bin/env node

'use strict'

const config = require('./config')
const port = config.apiPort

if (config.isTest) {
  require('./app.js').listen(port)
} else {
  const action = process.argv[2]
  switch (action) {
    case 'daemon': {
      const server = require('./app.js').listen(port, () => {
        const addr = server.address()
        console.log('listening on', addr.address + ':' + addr.port)
      })
    }
      break
    case 'sync': {
      const request = require('./request')
      const repo = process.argv[3]
      if (typeof repo === 'undefined') {
        console.error('Repo name is needed')
        process.exit(1)
      }
      request(`http://127.0.0.1:${port}/api/v1/repositories/${repo}/sync`)
        .then((res) => {
          if (res.ok) {
            console.log('syncing ' + repo)
          } else {
            throw new Error('Server return ' + res.status)
          }
        })
        .catch(console.error)
    }
      break;
    default:
      console.error('Unknown action: ' + action)
  }
}
