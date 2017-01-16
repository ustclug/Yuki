#!/usr/bin/node

'use strict'

require('../dist')

import {apiPort} from '../dist/config'
import test from 'ava'
import models from '../dist/models'
import data from './mock.json'
import mongoose from 'mongoose'
import request from '../dist/request'

const Repo = models.Repository

test.before(async t => {
  mongoose.createConnection('mongodb://127.0.0.1/test')
  await Repo.remove({})
  await Repo.create(data)
})

const API = `http://localhost:${apiPort}/api/v1`

test('List repositories', t => {
  return request(`${API}/repositories`)
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

test.serial('Get repository', t => {
  return request(`${API}/repositories/pypi`)
    .then(res => {
      t.true(res.ok)
      return res.json()
    })
    .then(res => {
      t.is(res.name, 'pypi')
      t.is(res.image, 'ustcmirror/test:latest')
      t.true(res.storageDir === '/pypi')
      t.is(res.user, '')
      t.is(res.envs[0], 'RSYNC_PASS=asdh')
    })
})

test.serial('Update repository', t => {
  return request(`${API}/repositories/pypi`, {
    image: 'ustcmirror/rsync:latest',
    args: ['echo', '1'],
    volumes: ['/pypi:/srv/repo/newpypi'],
    user: 'mirror'
  }, 'PUT')
    .then(res => {
      t.true(res.ok)
      return request(`${API}/repositories/pypi`)
    })
    .then(res => {
      t.true(res.ok)
      return res.json()
    })
    .then(res => {
      t.is(res.name, 'pypi')
      t.is(res.user, 'mirror')
      t.is(res.image, 'ustcmirror/rsync:latest')
      t.is(res.args[0], 'echo')
      t.is(res.args[1], '1')
      t.true(res.volumes[0].endsWith('newpypi'))
    })
})

test('New repository', t => {
  return request(`${API}/repositories/bioc`, {
    image: 'ustcmirror/test:latest',
    interval: '* * * * *',
    storageDir: '/bioc',
    args: ['rsync', 'somewhere'],
    user: 'mirror'
  })
    .then(res => {
      t.is(res.status, 201)
      return request(`${API}/repositories/bioc`)
    })
    .then(res => {
      t.is(res.status, 200)
      return res.json()
    })
    .then(res => {
      t.is(res.name, 'bioc')
      t.is(res.user, 'mirror')
      t.is(res.image, 'ustcmirror/test:latest')
      t.is(res.args[0], 'rsync')
      t.is(res.storageDir, '/bioc')
    })
})

test('Remove repository', t => {
  return request(`${API}/repositories/gmt`, null, 'DELETE')
    .then(res => {
      t.is(res.status, 204)
      return request(`${API}/repositories/gmt`)
    })
    .then(res => {
      t.is(res.status, 404)
    })
})

test('Start a container', t => {
  return request(`${API}/repositories/archlinux/sync`, null, 'POST')
    .then(res => {
      t.is(res.status, 204)
      return res.json()
    })
    .then(() => {
      return request(`${API}/containers/archlinux/wait`, null, 'POST')
    })
    .then(res => {
      t.is(res.status, 200)
      return res.json()
    })
    .then(data => t.is(data.StatusCode, 0))

})
