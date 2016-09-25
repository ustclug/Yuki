#!/usr/bin/node

'use strict'

import mongoose from 'mongoose'

const schema = new mongoose.Schema({
  _id: Number,
  log: [String],
})

schema.virtual('timestamp')
  .get(function() {
    return this._id
  })
  .set(function(v) {
    this._id = v
  })

export default mongoose.model('Log', schema)
