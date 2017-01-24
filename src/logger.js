#!/usr/bin/node

'use strict'

import winston from 'winston'
import { isDev } from './config'
import { getLocalTime } from '../build/Release/addon.node'

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
