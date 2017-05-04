#!/usr/bin/node

'use strict'

//
// ONLY CONSTANTS ARE ALLOWED
// BE AWARE OF NOT INTRODUCING ANY OTHER SUBMODULES!
//
const EventEmitter = require('events')

if (!process.env.NODE_ENV) {
  process.env.NODE_ENV = 'production'
}

export default {
  EMITTER: new EventEmitter(),
  IS_DEV: process.env.NODE_ENV.startsWith('dev'),
  IS_TEST: process.env.NODE_ENV.startsWith('test'),
  IS_PROD: process.env.NODE_ENV.startsWith('prod'),
  TOKEN_NAME: 'x-mirror-token' // should be lower-case
}
