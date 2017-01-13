#!/usr/bin/node

'use strict'

import mongoose from 'mongoose'

const schema = new mongoose.Schema({
  _id: String,
  interval: {
    required: true,
    type: String,
  },
  image: {
    default: 'ustclug/mirror:latest',
    type: String,
    // Format: image:tag
    match: /^[^:]+:[^:]+$/,
  },
  command: [String],
  envs: [String],
  volumes: [String],
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
