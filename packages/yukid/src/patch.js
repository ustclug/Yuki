#!/usr/bin/node

'use strict'

if (typeof String.prototype.splitN !== 'function') {
  String.prototype.splitN = function(sep, n) {
    const parts = this.split(sep)
    if (parts.length > n) {
      const rest = parts.slice(n).join(sep)
      return parts.slice(0, n).concat(rest)
    }
    return parts
  }
}

// String interpolation
if (typeof String.prototype.format !== 'function') {
  const pattern = /(\${([\w_]+)})/g
  String.prototype.format = function(params = {}) {
    return this.replace(pattern, function() {
      const key = arguments[2]
      const val = params[key]
      return (val === undefined || val === null) ? '' : val
    })
  }
}

if (typeof Object.entries !== 'function') {
  Object.entries = function entries(obj) {
    var entrys = []
    for (var key in obj) {
      if (obj.hasOwnProperty(key)) {
        entrys.push([key, obj[key]])
      }
    }
    return entrys
  }
}

