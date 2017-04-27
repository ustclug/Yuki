#!/usr/bin/node

'use strict'

import mongoose from 'mongoose'

const schema = new mongoose.Schema({
  name: {
    type: String,
    required: true
  },
  exitCode: {
    type: Number,
    default: -1
  },
}, {
  id: false,
  strict: 'throw',
  timestamps: true
})

schema.set('toJSON', { versionKey: false, getters: true })
export default mongoose.model('Log', schema)
