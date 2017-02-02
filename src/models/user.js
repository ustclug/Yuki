#!/usr/bin/node

'use strict'

import mongoose from 'mongoose'
import { createHash } from 'crypto'

const schema = new mongoose.Schema({
  _id: String, // username,
  password: {
    required: true,
    type: String
  },
  admin: {
    type: Boolean,
    default: false
  },
  token: {
    min: 40,
    max: 40,
    type: String,
  }
}, { id: false })

schema.virtual('name')
  .get(function() {
    return this._id
  })
  .set(function(name) {
    this._id = name
    const hash = createHash('sha1').update(this._id).update(this.password)
    this.token = hash.digest('hex')
  })

schema.set('toJSON', { versionKey: false, getters: true })
export default mongoose.model('User', schema)
