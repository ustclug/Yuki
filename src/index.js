#!/usr/bin/env node

'use strict'

require('./patch')
const config = require('./config')

if (config.isTest || process.argv[2] === 'daemon') {
  require('./server')
} else {
  require('./client')
}
