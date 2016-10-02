#!/usr/bin/env node

'use strict'

var port = process.env.PORT || 9999

function applyPolyFill() {
  require('babel-register')({
    'presets': [
      'es2015',
      'stage-1'
    ],
  })
  require('babel-polyfill')
}

if (!process.env.NODE_ENV) {
  process.env.NODE_ENV = 'production'
}

if (process.env.NODE_ENV === 'test') {
  applyPolyFill()
  module.exports = require('./app.js').listen(port).address()
} else {
  var basename = require('path').basename
  var progName = basename(process.argv[1])

  // process.argv[0] is /usr/bin/node
  switch (progName) {
    case 'yukid':
      applyPolyFill()
      var server = require('./app.js').listen(port, function() {
        var addr = server.address()
        console.log('listening on', addr.address + ':' + addr.port)
      })
      break

    case 'yuki':
      var action = process.argv[2]
      var repo = process.argv[3]
      var request = require('./request')
      switch (action) {
        case 'sync':
          request('http://localhost:' + port + '/api/v1/repositories/' + repo + '/sync')
            .then(function(res) {
              if (res.ok) {
                console.log('syncing ' + repo)
              } else {
                throw new Error('Server return ' + res.status)
              }
            })
            .catch(console.error)
          break;

        default:
          console.error('Unknown action: ' + action)
      }
      break

    default:
      console.error('Unknown program: ' + progName)
  }
}

