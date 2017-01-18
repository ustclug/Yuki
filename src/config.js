#!/usr/bin/env node

'use strict'

if (!process.env.NODE_ENV) {
  process.env.NODE_ENV = 'production'
}

const defaultCfg = {
  DB_USER: 'mirror',
  DB_PASSWD: 'averylongpass',
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
  OWNER: `${process.getuid()}:${process.getgid()}`,
  isProd: process.env.NODE_ENV.startsWith('prod'),
  isDev: process.env.NODE_ENV.startsWith('dev'),
  isTest: process.env.NODE_ENV.startsWith('test'),
}

module.exports = defaultCfg

const fs = require('fs')
const path = require('path')

const fp = path.join(process.env.HOME, '.ustcmirror.json')
let exist
try {
  fs.statSync(fp)
  exist = true
} catch (e) {
  exist = false
}

let userCfg
// Throw error if JSON is invalid
exist ? (userCfg = require(fp)) : (userCfg = {})

Object.assign(defaultCfg, userCfg)

if (!defaultCfg.isTest && !defaultCfg.BIND_ADDR) {
  console.error('Need to specify <BIND_ADDR> in conf')
  process.exit(1)
}
