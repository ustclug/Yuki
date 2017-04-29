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

// FIXME: Modifying Document._doc is undefined behaviour,
// depends on the implementation
schema.methods.prettySize = function() {
  if (undefined === this.size) return this
  const units = ['B', 'KB', 'MB', 'GB']
  let size = this.size
  for (const unit of units) {
    if (size < 1024) {
      this._doc.size = `${size.toFixed(1)} ${unit}`
      return this
    } else {
      size /= 1024
    }
  }
  this._doc.size = `${size.toFixed(1)} TB`
  return this
}

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
  switch (img) {
    case 'rsync':
    case 'archvsync': {
      const host = repo.envs.RSYNC_HOST || '(unknown)'
      const path = repo.envs.RSYNC_PATH || '(unknown)'
      return urlJoin(`rsync://${host}/`, path)
    }
    case 'lftpsync': {
      const host = repo.envs.LFTPSYNC_HOST || '(unknown)'
      const path = repo.envs.LFTPSYNC_PATH || '(unknown)'
      return urlJoin(`${host}/`, path)
    }
    case 'gitsync':
      return repo.envs.GITSYNC_URL
    case 'aptsync':
      return repo.envs.APTSYNC_URL
    case 'pypi':
      return repo.envs.PYPI_MASTER || 'https://pypi.python.org'
    case 'homebrew-bottles':
      return repo.envs.HOMEBREW_BOTTLE_DOMAIN || 'http://homebrew.bintray.com'
    case 'rubygems':
      return repo.envs.UPSTREAM || 'http://rubygems.org'
    default:
      return defVal
  }
})

schema.set('toJSON', { versionKey: false, getters: true })
export default mongoose.model('Meta', schema)
