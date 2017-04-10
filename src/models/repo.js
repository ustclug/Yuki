#!/usr/bin/node

'use strict'

import mongoose from 'mongoose'

const schema = new mongoose.Schema({
  _id: String,
  // crontab
  interval: {
    required: true,
    type: String,
  },
  image: {
    required: true,
    type: String,
    // Format: image:tag
    match: /^[^:]+:[^:]+$/,
  },
  storageDir: {
    type: String,
    required: true
  },
  logRotCycle: {
    type: Number,
    default: 10
  },
  envs: {
    type: Object,
    default: {}
  },
  volumes: {
    type: Object,
    default: {}
  },
  bindIp: String,
  user: String,
}, { id: false, strict: 'throw' })

function isUndef(val) {
  return typeof val === 'undefined'
}

schema.virtual('upstream')
  .get(function() {
    const defVal = 'unknown'
    if (isUndef(this.image) || isUndef(this.envs)) {
      return
    }
    if (!this.image.startsWith('ustcmirror/')) {
      return defVal
    }
    const img = this.image.splitN('/', 1)[1].splitN(':', 1)[0]
    switch (img) {
      case 'rsync':
      case 'archvsync': {
        const host = this.envs.RSYNC_HOST || '(unknown)'
        const path = this.envs.RSYNC_PATH || '(unknown)'
        return `rsync://${host}/${path}`
      }
      case 'lftpsync': {
        const host = this.envs.LFTPSYNC_HOST || '(unknown)'
        const path = this.envs.LFTPSYNC_PATH || '(unknown)'
        return `${host}/${path}`
      }
      case 'gitsync':
        return this.envs.GITSYNC_URL
      case 'aptsync':
        return this.envs.APTSYNC_URL
      case 'pypi':
        return this.envs.PYPI_MASTER || 'https://pypi.python.org'
      case 'homebrew-bottles':
        return this.envs.HOMEBREW_BOTTLE_DOMAIN || 'http://homebrew.bintray.com'
      case 'rubygems':
        return this.envs.UPSTREAM || 'http://rubygems.org'
      default:
        return defVal
    }
  })

schema.virtual('name')
  .set(function(name) {
    this._id = name
  })

schema.set('toJSON', { versionKey: false, getters: true })
export default mongoose.model('Repository', schema)
