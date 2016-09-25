#!/usr/bin/env node

'use strict'

require('babel-register')({
  'presets': [
    'es2015',
    'stage-1'
  ],
})
require('babel-polyfill')
var app = require('./app.js')
var port = process.env.PORT || 9999

app.listen(port, function() {
  console.log('listening on', port)
})
