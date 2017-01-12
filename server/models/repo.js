#!/usr/bin/node

'use strict'

import mongoose from 'mongoose'

const schema = new mongoose.Schema({
  _id: String,
  storageDir: {
    required: true,
    type: String,
  },
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
  logDir: String,
  command: [String],
  envs: [String],
  volumes: [String],
  rm: Boolean,
  debug: Boolean,
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
