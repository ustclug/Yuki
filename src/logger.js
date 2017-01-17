#!/usr/bin/node

'use strict'

import winston from 'winston'
import { isDev } from './config'

export default new (winston.Logger)({
  transports: [
    new (winston.transports.Console)({
      level: isDev ? 'debug' : 'warn',
      timestamp: () => new Date().toISOString(),
      formatter: (options) => {
        return `${options.timestamp()} ${options.level.toUpperCase()}: ${options.message}`
      }
    })
  ]
})
