#!/usr/bin/node

'use strict'

import '../'
import test from 'ava'
import Promise from 'bluebird'
import models from '../models'
import data from './mock.json'
import mongoose from 'mongoose'
import fetch from 'node-fetch'

const Repo = models.Repository

test.before(async t => {
  mongoose.createConnection('mongodb://127.0.0.1/test', {
    promiseLibrary: Promise,
  })
  mongoose.Promise = Promise
  fetch.Promise = Promise
  await Repo.remove({})
  for (const r of data) {
    await Repo.create(r)
  }
})

function request(url, data, method = 'GET') {
  if (data !== null && typeof data === 'object') {
    if (method === 'GET') method = 'POST'
    return fetch(url, {
      method: method,
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(data)
    })
  }
  return fetch(url, { method })
}

const API = 'http://localhost:9999/api/v1'

test('List repositories', t => {
  return fetch(`${API}/repositories/list`)
    .then(res => {
      t.true(res.ok)
      return res.json()
    })
    .then(res => {
      t.is(res.length, data.length)
      for (const r of res) {
        t.true(typeof r.name !== 'undefined')
      }
    })
})

test('Get repository', t => {
  return fetch(`${API}/repositories/pypi`)
    .then(res => {
      t.true(res.ok)
      return res.json()
    })
    .then(res => {
      t.is(res.name, 'pypi')
      t.is(res.image, 'ustclug/alpine:latest')
      t.is(res.rm, true)
      t.is(res.user, '')
      t.is(res.debug, false)
      t.is(res.storageDir, '/pypi')
      t.is(res.envs[0], 'RSYNC_PASS=asdh')
    })
})

test('Update repository', t => {
  return request(`${API}/repositories/pypi`, {
    image: 'alpine:edge',
    command: ['echo', '1'],
    user: 'mirror'
  }, 'PUT')
    .then(res => {
      t.true(res.ok)
      return fetch(`${API}/repositories/pypi`)
    })
    .then(res => {
      t.true(res.ok)
      return res.json()
    })
    .then(res => {
      t.is(res.name, 'pypi')
      t.is(res.user, 'mirror')
      t.is(res.image, 'alpine:edge')
      t.is(res.command[0], 'echo')
      t.is(res.command[1], '1')
    })
})

test('New repository', t => {
  return request(`${API}/repositories/bioc`, {
    image: 'mongo:latest',
    storageDir: '/zxc/asd',
    interval: '* * * * *',
    command: ['rsync', 'somewhere'],
    user: 'mirror'
  })
    .then(res => {
      t.is(res.status, 200)
      return fetch(`${API}/repositories/bioc`)
    })
    .then(res => {
      t.is(res.status, 200)
      return res.json()
    })
    .then(res => {
      t.is(res.name, 'bioc')
      t.is(res.user, 'mirror')
      t.is(res.image, 'mongo:latest')
      t.is(res.command[0], 'rsync')
    })
})

test('Remove repository', t => {
  return request(`${API}/repositories/gmt`, null, 'DELETE')
    .then(res => {
      t.is(res.status, 200)
      return fetch(`${API}/repositories/gmt`)
    })
    .then(res => {
      t.is(res.status, 404)
    })
})
