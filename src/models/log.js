#!/usr/bin/node

'use strict'

import mongoose from 'mongoose'

const schema = new mongoose.Schema({
  _id: String,
  lastSuccess: Date,
  exitCode: {
    type: Number,
    default: -1
  },
  status: {
    enum: ['success', 'failure', 'running', 'unknown'],
    default: 'unknown',
    type: String
  },
}, {
  id: false,
  strict: 'throw',
  timestamps: true
})

schema.virtual('name')
  .set(function(name) {
    this._id = name
  })

schema.set('toJSON', { versionKey: false, getters: true })
export default mongoose.model('Log', schema)
