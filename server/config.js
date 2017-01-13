#!/usr/bin/env node

'use strict'

if (!process.env.NODE_ENV) {
  process.env.NODE_ENV = 'production'
}

module.exports = {
  'dbUser': 'mirror',
  'dbPasswd': 'averylongpass',
  'dbHost': '127.0.0.1',
  'dbName': 'mirror',
  'dbPort': 27017,
  'isProd': process.env.NODE_ENV.startsWith('prod'),
  'isDev': process.env.NODE_ENV.startsWith('dev'),
  'isTest': process.env.NODE_ENV.startsWith('test'),
  'apiPort': 9999,
}
