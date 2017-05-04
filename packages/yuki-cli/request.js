#!/usr/bin/node

'use strict'

const fetch = require('node-fetch');
const { version } = require('./package.json')
const Url = require('url')

fetch.Promise = require('bluebird')

const serialize = JSON.stringify

const isNotValid = (obj) => typeof obj === 'undefined' || obj === null

const isBuffer = (val) => {
  return toString.call(val) === '[object Uint8Array]';
}

const isObject = (val) => {
  return val !== null && typeof val === 'object';
}

const isFunction = (val) => {
  return toString.call(val) === '[object Function]';
}

const isStream = (val) => {
  return isObject(val) && isFunction(val.pipe);
}

const merge = (...objs) => {
  const result = {}
  function assignValue(val, key) {
    if (typeof result[key] === 'object' && typeof val === 'object') {
      result[key] = merge(result[key], val);
    } else {
      result[key] = val;
    }
  }
  for (const obj of objs) {
    if (isNotValid(obj)) {
      continue
    }
    Object.keys(obj).forEach(key => {
      assignValue(obj[key], key)
    })
  }
  return result;
}

const normalizeUrl = (u) => {
  // not absolute
  if (!/^([a-z][a-z\d\+\-\.]*:)?\/\//i.test(u)) u = `http://${u}`
  if (!u.endsWith('/')) u += '/'
  return u
}

const methodsNoBody = ['get', 'head', 'delete']
const methodsWithBody = ['patch', 'put', 'post']
const defaults = {
  headers: {
    'User-Agent': `Nagato Yuki/${version}`,
    'Content-Type': 'application/json'
  },
  url: '',
  baseUrl: '',
  method: 'get',
  timeout: 0,
  follow: 20
}

class Client {
  constructor(cfg) {
    // assert(typeof cfg === 'object')
    this.common = merge(defaults, cfg)
  }

  request(cfg) {
    cfg = merge(this.common, cfg)
    const url = Url.resolve(normalizeUrl(cfg.baseUrl), cfg.url)
    delete cfg.url
    delete cfg.baseUrl

    if (!isStream(cfg.body) &&
        !isBuffer(cfg.body) &&
        isObject(cfg.body))
    {
      cfg.body = serialize(cfg.body)
      if (cfg.method.toLowerCase() === 'get') {
        cfg.method = 'post'
      }
    }

    return fetch(url, cfg)
      .then(async res => {
        if (!res.ok) {
          res.error = await res.json()
        }
        return res
      })
  }
}

methodsNoBody.forEach(function noBody(method) {
  Client.prototype[method] = function(url, cfg = {}) {
    return this.request(merge(cfg, {
      method,
      url
    }))
  }
})

methodsWithBody.forEach(function withBody(method) {
  Client.prototype[method] = function(url, body = {}, cfg = {}) {
    return this.request(merge(cfg, {
      body,
      method,
      url
    }))
  }
})

module.exports = Client
