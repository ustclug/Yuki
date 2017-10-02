#!/usr/bin/node

'use strict'

import Promise from 'bluebird'
import mongoose from 'mongoose'
import { URL } from 'url'

import logger from '../logger'
import CONFIG from '../config'
import { IS_TEST } from '../globals'

mongoose.Promise = Promise

if (IS_TEST) {
  mongoose.connect('mongodb://127.0.0.1:27017/test', { useMongoClient: true })
} else {
  const uri = new URL('mongodb://127.0.0.1/')
  uri.port = CONFIG.get('DB_PORT')
  uri.pathname = `/${CONFIG.get('DB_NAME')}`
  uri.username = CONFIG.get('DB_USER')
  uri.password = CONFIG.get('DB_PASSWD')
  mongoose.connect(uri.toString(), {
    useMongoClient: true,
    promiseLibrary: Promise,
  })
}
logger.info('Connected to MongoDB')

import Repo from './repo'
import User from './user'
import Log from './log'
import Meta from './meta'

export default {
  Repository: Repo,
  User,
  Meta,
  Log
}
