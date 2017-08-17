#!/usr/bin/node

'use strict'

import winston from 'winston'
import CONFIG from './config'
import { IS_TEST } from './globals'
import moment from 'moment'

const getLocalTime = () => moment().local().format('YYYY-MM-DD HH:mm:ss')

const transports = []

const LOGLEVEL = CONFIG.get('LOGLEVEL')
const LOGDIR_ROOT = CONFIG.get('LOGDIR_ROOT')
const TIMESTAMP = CONFIG.get('TIMESTAMP')

if (!IS_TEST) {
  transports.push(new (winston.transports.File)({
    level: LOGLEVEL,
    json: false,
    filename: `${LOGDIR_ROOT}/yukid.log`,
    maxsize: 1024 * 1024 * 10,
    maxFiles: 30,
    formatter: (options) => {
      let msg = options.message
      if (options.meta.stack) {
        // Error
        msg += `\n${options.meta.stack}`
      } else if (options.meta && Object.keys(options.meta).length !== 0) {
        msg += `\n${JSON.stringify(options.meta, null, 4)}`
      }
      return TIMESTAMP ?
        `[${getLocalTime()}] ${options.level.toUpperCase()}: ${msg}` :
        `${options.level.toUpperCase()}: ${msg}`
    }
  }))
} else {
  transports.push(new (winston.transports.Console)({ level: LOGLEVEL }))
}

export default new (winston.Logger)({ transports })
