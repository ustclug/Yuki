#!/usr/bin/node

'use strict'

import mongoose from 'mongoose'

const schema = new mongoose.Schema({
  _id: String,
  lastSuccess: Date,
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
  .get(function() {
    return this._id
  })
  .set(function(name) {
    this._id = name
  })

export default mongoose.model('Log', schema)
