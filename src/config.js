#!/usr/bin/env node

'use strict'

if (!process.env.NODE_ENV) {
  process.env.NODE_ENV = 'production'
}

module.exports = {
  'dbuser': 'mirror',
  'dbpasswd': 'averylongpass',
  'dbhost': '127.0.0.1',
  'dbname': 'mirror',
  'isProd': process.env.NODE_ENV.startsWith('prod'),
  'isDev': process.env.NODE_ENV.startsWith('dev'),
  'isTest': process.env.NODE_ENV.startsWith('test'),
  'apiport': 9999,
}

