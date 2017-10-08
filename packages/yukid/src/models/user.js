#!/usr/bin/node

'use strict'

import mongoose from 'mongoose'

const schema = new mongoose.Schema({
  // username
  _id: {
    required: true,
    type: String,
    match: /^[-\w]+$/
  },
  password: {
    required: true,
    type: String
  },
  admin: {
    type: Boolean,
    default: false
  },
}, { id: false })

schema.virtual('name')
  .get(function() {
    return this._id
  })
  .set(function(name) {
    this._id = name
  })

schema.set('toJSON', { versionKey: false, getters: true })
export default mongoose.model('User', schema)
