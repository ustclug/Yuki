#!/usr/bin/node

'use strict'

var fetch = require('node-fetch')
fetch.Promise = require('bluebird')

module.exports = function request(url, data, method = 'GET', useropts = {}) {
  var opts = {
    headers: {
      'User-Agent': 'Nagato Yuki/1.0'
    }
  }
  Object.assign(opts, useropts)
  if (data !== null && typeof data === 'object') {
    if (method === 'GET') method = 'POST'
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(data)
  }
  opts.method = method
  return fetch(url, opts)
}
