#!/usr/bin/env node

'use strict'

const path = require('path')

function merge(target, src, prefix = '') {
  Object.keys(target).forEach(k => {
    if (typeof src[`${prefix}${k}`] !== 'undefined')
      target[k] = src[`${prefix}${k}`]
  })
}

function setup() {

  const defaults = {
    // For client
    API_ROOT: '',
    API_PORT: 9999
  }
  // Access local server by default
  defaults.API_ROOT = `http://localhost:${defaults.API_PORT}/`

  // should be lower case
  defaults['TOKEN_NAME'] = 'x-mirror-token'

  const fps = ['/etc/ustcmirror/config', path.join(process.env['HOME'], '.ustcmirror/config')]

  for (const fp of fps) {
    let cfg
    try {
      cfg = require(fp)
    } catch (e) {
      if (e.code !== 'MODULE_NOT_FOUND') {
        console.error(`Invalid config: ${fp}`)
        process.exit(1)
      }
      continue
    }
    merge(defaults, cfg)
  }

  merge(defaults, process.env, 'YUKI_')
  return defaults
}

module.exports = setup()
