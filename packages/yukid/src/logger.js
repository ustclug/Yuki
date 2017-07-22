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
      return TIMESTAMP ?
        `[${getLocalTime()}] ${options.level.toUpperCase()}: ${options.message}` :
        `${options.level.toUpperCase()}: ${options.message}`
    }
  }))
} else {
  transports.push(new (winston.transports.Console)({ level: LOGLEVEL }))
}

export default new (winston.Logger)({ transports })
