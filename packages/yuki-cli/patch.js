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

