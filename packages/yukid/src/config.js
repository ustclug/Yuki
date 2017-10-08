#!/usr/bin/env node

'use strict'

import {
  IS_DEV,
  IS_TEST
} from './globals'
import path from 'path'

function merge(target, src, prefix = '') {
  Object.keys(target).forEach((k) => {
    if (typeof src[`${prefix}${k}`] !== 'undefined')
      target[k] = src[`${prefix}${k}`]
  })
}

const readConfig = function() {
  const defaults = {
    DB_USER: '',
    DB_PASSWD: '',
    DB_HOST: '127.0.0.1',
    DB_NAME: 'mirror',
    DB_PORT: 27017,
    API_PORT: 9999,
    API_ADDR: '127.0.0.1',
    JWT_SECRET: '',
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
    LDAP: {
      enabled: false,
      url: '',
      searchBase: '',
      ca: ''
    },
    POST_SYNC: [],
    LOGLEVEL: '',
  }

  const fps = ['/etc/ustcmirror/config', path.join(process.env['HOME'], '.ustcmirror/config')]

  for (const fp of fps) {
    let cfg
    try {
      // TODO: purge cache
      cfg = require(fp)
    } catch (e) {
      if (e.code !== 'MODULE_NOT_FOUND') {
        console.error('Invalid config at:', fp)
        console.error(e)
        throw e
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
      throw new Error('Invalid config')
    }
    if (typeof defaults.FILESYSTEM !== 'object') {
      console.error('Invalid FILESYSTEM: %j', defaults.FILESYSTEM)
      throw new Error('Invalid config')
    }
    if (!defaults.JWT_SECRET) {
      console.error('Please provide `JWT_SECRET` in your config')
      throw new Error('Invalid config')
    }
  }

  if (IS_DEV) {
    console.log('Configuration:', JSON.stringify(defaults, null, 4))
  }
  return defaults
}

class Config {
  constructor() {
    try {
      this._config = readConfig()
    } catch (e) {
      process.exit(1)
    }
  }

  reload() {
    try {
      this._config = readConfig()
    } catch (e) {
      return
    }
  }

  get(key) {
    return this._config[key]
  }

  set(key, val) {
    this._config[key] = val
  }

  list() {
    return this._config
  }
}

export default new Config()
