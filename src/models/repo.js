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
  autoRotLog: {
    type: Boolean,
    default: true
  },
  rotateCycle: {
    type: Number,
    default: 10
  },
  args: [String],
  envs: [String],
  volumes: [String],
  bindIp: String,
  user: String,
}, { id: false })

schema.virtual('name')
  .get(function() {
    return this._id
  })
  .set(function(name) {
    this._id = name
  })

schema.set('toJSON', { versionKey: false, getters: true })
export default mongoose.model('Repository', schema)
