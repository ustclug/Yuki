#!/usr/bin/node

'use strict'

import mongoose from 'mongoose'
import Url from 'url'

const schema = new mongoose.Schema({
  _id: String,
  size: {
    type: Number,
    default: 0
  },
  lastExitCode: {
    type: Number,
    default: -1,
  },
  lastSuccess: {
    type: Date,
    default: new Date(0)
  }
}, {
  id: false,
  strict: 'throw',
  timestamps: true
})

const virt = schema.virtual('upstream', {
  ref: 'Repository',
  localField: '_id',
  foreignField: '_id',
  justOne: true
})

const _leftSlash = new RegExp('^/+')
const _slashes = new RegExp('([^:]/)/+')
const trimSlash = (s) => s.replace(_leftSlash, '')
const urlJoin = (host, path) => Url.resolve(host, trimSlash(path)).replace(_slashes, '$1')

// FIXME: mongoose won't execute getters in reverse order in 5.0
// https://github.com/Automattic/mongoose/issues/4835
virt.getters.unshift(function(repo) {
  const defVal = 'unknown'
  if (!(repo && repo.image && repo.envs)) {
    return defVal
  }
  if (!repo.image.startsWith('ustcmirror/')) {
    return defVal
  }
  const img = repo.image.splitN('/', 1)[1].splitN(':', 1)[0]
  const { envs } = repo
  switch (img) {
    case 'rsync':
    case 'debian-cd':
    case 'archvsync': {
      const host = envs.RSYNC_HOST || '(unknown)'
      const path = envs.RSYNC_PATH || envs.RSYNC_MODULE || '(unknown)'
      return urlJoin(`rsync://${host}/`, path)
    }
    case 'lftpsync': {
      const host = envs.LFTPSYNC_HOST || '(unknown)'
      const path = envs.LFTPSYNC_PATH || '(unknown)'
      return urlJoin(`${host}/`, path)
    }
    case 'gitsync':
      return envs.GITSYNC_URL
    case 'aptsync':
      return envs.APTSYNC_URL
    case 'pypi':
      return envs.PYPI_MASTER || 'https://pypi.python.org'
    case 'homebrew-bottles':
      return envs.HOMEBREW_BOTTLE_DOMAIN || 'http://homebrew.bintray.com'
    case 'gsutil-rsync':
      return envs.GS_URL || defVal
    case 'rubygems':
      return envs.UPSTREAM || 'http://rubygems.org'
    default:
      return defVal
  }
})

schema.set('toJSON', { versionKey: false, getters: true })
export default mongoose.model('Meta', schema)
