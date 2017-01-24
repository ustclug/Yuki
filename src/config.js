#!/usr/bin/env node

'use strict'

if (!process.env.NODE_ENV) {
  process.env.NODE_ENV = 'production'
}

const defaultCfg = {
  // For server
  DB_USER: '',
  DB_PASSWD: '',
  DB_HOST: '127.0.0.1',
  DB_NAME: 'mirror',
  DB_PORT: 27017,
  API_PORT: 9999,
  DOCKERD_PORT: 2375,
  DOCKERD_HOST: '127.0.0.1',
  DOCKERD_SOCKET: '/var/run/docker.sock',
  BIND_ADDR: '',
  CT_LABEL: 'syncing',
  CT_NAME_PREFIX: 'syncing',
  LOGDIR_ROOT: '/home/mirror/log',
  OWNER: `${process.getuid()}:${process.getgid()}`,
  isProd: process.env.NODE_ENV.startsWith('prod'),
  isDev: process.env.NODE_ENV.startsWith('dev'),
  isTest: process.env.NODE_ENV.startsWith('test'),

  // For client
  API_URL: 'http://localhost:9999/',
}

module.exports = defaultCfg

const fs = require('fs')
const fp = '/etc/ustcmirror/daemon.json'

let exist
try {
  fs.statSync(fp)
  exist = true
} catch (e) {
  exist = false
}

// Throw error if JSON is invalid
const userCfg = exist ? require(fp) : {}

Object.assign(defaultCfg, userCfg)

if (!defaultCfg.isTest && !defaultCfg.BIND_ADDR) {
  console.error(`Need to specify <BIND_ADDR> in ${fp}`)
  process.exit(1)
}
