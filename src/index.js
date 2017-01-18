#!/usr/bin/env node

'use strict'

const config = require('./config')

if (config.isTest || process.argv[2] === 'daemon') {
  require('./server')
} else {
  require('./client')
}
