#!/usr/bin/node

'use strict'

import winston from 'winston'
import { isDev } from './config'

let getLocalTime
try {
  getLocalTime = require('../build/Release/addon.node').getLocalTime;
} catch (e) {
  getLocalTime = require('../build/Debug/addon.node').getLocalTime;
}

export default new (winston.Logger)({
  transports: [
    new (winston.transports.Console)({
      level: isDev ? 'debug' : 'warn',
      formatter: (options) => {
        return `[${getLocalTime()}] ${options.level.toUpperCase()}: ${options.message}`
      }
    })
  ]
})
