#!/usr/bin/env node

'use strict'

import {
  IS_DEV,
  IS_TEST
} from './globals'
import path from 'path'

function merge(target, src, prefix = '') {
  Object.keys(target).forEach(k => {
    if (typeof src[`${prefix}${k}`] !== 'undefined')
      target[k] = src[`${prefix}${k}`]
  })
}

const setup = () => {
  const defaults = {
    DB_USER: '',
    DB_PASSWD: '',
    DB_HOST: '127.0.0.1',
    DB_NAME: 'mirror',
    DB_PORT: 27017,
    API_PORT: 9999,
    API_ADDR: '127.0.0.1',
    DOCKERD_PORT: 2375,
    DOCKERD_HOST: '127.0.0.1',
    DOCKERD_SOCKET: '/var/run/docker.sock',
    BIND_ADDRESS: '',
    CT_LABEL: 'syncing',
    CT_NAME_PREFIX: 'syncing',
    LOGDIR_ROOT: '/var/log/ustcmirror',
    IMAGES_UPDATE_INTERVAL: '1 * * * *',
    OWNER: `${process.getuid()}:${process.getgid()}`,
    TIMESTAMP: true,
    FILESYSTEM: {
      type: 'fs'
    },
    LOGLEVEL: '',
  }

  const fps = ['/etc/ustcmirror/config', path.join(process.env['HOME'], '.ustcmirror/config')]

  for (const fp of fps) {
    let cfg
    try {
      cfg = require(fp)
    } catch (e) {
      if (e.code !== 'MODULE_NOT_FOUND') {
        console.error('Invalid config at:', fp)
        console.error(e)
        process.exit(1)
      }
      continue
    }
    merge(defaults, cfg)
  }

  merge(defaults, process.env, 'YUKI_')
  if (typeof defaults.TIMESTAMP === 'string') {
    defaults.TIMESTAMP = defaults.TIMESTAMP === 'true'
  }

  if (!defaults.LOGLEVEL) {
    defaults.LOGLEVEL = IS_DEV ? 'debug' : 'warn'
  }

  if (!IS_TEST) {
    if (!/(error|warn|info|verbose|debug|silly)/.test(defaults.LOGLEVEL))
    {
      console.error(`Invalid LOGLEVEL: ${defaults.LOGLEVEL}`)
      process.exit(1)
    }
    if (typeof defaults.FILESYSTEM !== 'object') {
      console.error('Invalid FILESYSTEM: %j', defaults.FILESYSTEM)
      process.exit(1)
    }
  }

  if (IS_DEV) {
    console.log('Configuration:', JSON.stringify(defaults, null, 4))
  }
  return defaults
}

export default setup()
