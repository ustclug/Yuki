#!/usr/bin/node

'use strict'

import winston from 'winston'
import { LOGLEVEL, TIMESTAMP } from './config'
import { getLocalTime } from '../build/Release/addon.node'

export default new (winston.Logger)({
  transports: [
    new (winston.transports.Console)({
      level: LOGLEVEL,
      formatter: (options) => {
        return TIMESTAMP ?
          `[${getLocalTime()}] ${options.level.toUpperCase()}: ${options.message}` :
          `${options.level.toUpperCase()}: ${options.message}`
      }
    })
  ]
})
