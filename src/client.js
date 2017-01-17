#!/usr/bin/node

'use strict'

const config = require('./config')
const port = config.API_PORT
const request = require('./request')

const repo = process.argv[3]
if (typeof repo === 'undefined') {
  console.error('Repo name is needed')
  process.exit(1)
}
request(`http://127.0.0.1:${port}/api/v1/repositories/${repo}/sync`)
  .then((res) => {
    if (res.ok) {
      console.log('syncing ' + this.repo)
    } else {
      throw new Error('Server return ' + res.status)
    }
  })
  .catch(console.error)
