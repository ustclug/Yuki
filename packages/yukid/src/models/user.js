#!/usr/bin/node

'use strict'

import mongoose from 'mongoose'
import { createHash } from 'crypto'

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
  token: {
    min: 40,
    max: 40,
    type: String,
  }
}, { id: false })

const calToken = (name, pass) => {
  const hash = createHash('sha1').update(name).update(pass)
  return hash.digest('hex')
}

schema.pre('findOneAndUpdate', function(next) {
  const cond = this._conditions
  const update = this._update.$set || this._update
  if (cond._id && update.password) {
    this.update(cond, {
      $set: {
        token: calToken(cond._id, update.password)
      }
    })
  }
  return next()
})

schema.virtual('name')
  .get(function() {
    return this._id
  })
  .set(function(name) {
    this._id = name
    this.token = calToken(name, this.password)
  })

schema.set('toJSON', { versionKey: false, getters: true })
export default mongoose.model('User', schema)
