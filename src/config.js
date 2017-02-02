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
  BIND_ADDRESS: '',
  CT_LABEL: 'syncing',
  CT_NAME_PREFIX: 'syncing',
  LOGDIR_ROOT: '/var/log/ustcmirror',
  IMAGES_UPGRADE_INTERVAL: '1 * * * *',
  OWNER: `${process.getuid()}:${process.getgid()}`,
  isProd: process.env.NODE_ENV.startsWith('prod'),
  isDev: process.env.NODE_ENV.startsWith('dev'),
  isTest: process.env.NODE_ENV.startsWith('test'),

  // For client
  API_ROOT: '',
}

const path = require('path')
const fps = ['/etc/ustcmirror/config', path.join(process.env['HOME'], '.ustcmirror/config')]

defaultCfg.API_ROOT = `http://localhost:${defaultCfg.API_PORT}/`

for (const fp of fps) {
  let cfg
  try {
    cfg = require(fp)
  } catch (e) {
    if (e.code !== 'MODULE_NOT_FOUND') {
      throw e
    }
    continue
  }
  Object.assign(defaultCfg, cfg)
}


if (!(defaultCfg.isTest ||
    process.argv[2] !== 'daemon' ||
    defaultCfg.BIND_ADDRESS))
{
  console.error('Need to specify <BIND_ADDRESS> in configuration')
  process.exit(1)
}

if (!defaultCfg.API_ROOT.startsWith('http://')) {
  defaultCfg.API_ROOT = `http://${defaultCfg.API_ROOT}`
}
if (!defaultCfg.API_ROOT.endsWith('/')) {
  defaultCfg.API_ROOT += '/'
}

// should be lower case
defaultCfg['TOKEN_NAME'] = 'x-mirror-token'

defaultCfg._images = [
  'ustcmirror/gitsync:latest',
  'ustcmirror/rsync:latest',
  'ustcmirror/lftpsync:latest',
]

if (defaultCfg.isDev) {
  console.log('Configuration:', JSON.stringify(defaultCfg, null, 4))
}

module.exports = defaultCfg
