#!/usr/bin/node

'use strict'

require('../dist')

import {API_PORT} from '../dist/config'
import test from 'ava'
import models from '../dist/models'
import data from './mock.json'
import mongoose from 'mongoose'
import request from '../dist/request'
import { isListening } from '../build/Release/addon.node'

const Repo = models.Repository

test.before(async t => {
  mongoose.createConnection('mongodb://127.0.0.1/test')
  await Repo.remove({})
  await Repo.create(data)
})

const API = `http://localhost:${API_PORT}/api/v1`

test('Native addon: isListening', t => {
  t.true(isListening('127.0.0.1', API_PORT))
  t.true(isListening('localhost', API_PORT))
  t.false(isListening('localhost', 4))
})

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
  return request(`${API}/repositories/archlinux`)
    .then(res => {
      t.true(res.ok)
      return res.json()
    })
    .then(res => {
      t.is(res.name, 'archlinux')
      t.is(res.image, 'ustcmirror/test:latest')
      t.true(res.storageDir === '/srv/repo/archlinux')
      t.is(res.envs[0], 'RSYNC_USER=asdh')
    })
})

test.serial('Update repository', t => {
  return request(`${API}/repositories/bioc`, {
    image: 'ustcmirror/rsync:latest',
    args: ['echo', '1'],
    volumes: ['/pypi:/srv/repo/BIOC'],
    user: 'mirror'
  }, 'PUT')
    .then(res => {
      t.true(res.ok)
      return request(`${API}/repositories/bioc`)
    })
    .then(res => {
      t.true(res.ok)
      return res.json()
    })
    .then(res => {
      t.is(res.name, 'bioc')
      t.is(res.user, 'mirror')
      t.is(res.image, 'ustcmirror/rsync:latest')
      t.is(res.args[0], 'echo')
      t.is(res.args[1], '1')
      t.true(res.volumes[0].endsWith('BIOC'))
    })
})

test('New repository', t => {
  return request(`${API}/repositories/vim`, {
    image: 'ustcmirror/test:latest',
    interval: '* 5 * * *',
    storageDir: '/srv/repo/vim',
    args: ['echo', 'vim'],
  })
    .then(res => {
      t.is(res.status, 201)
      return request(`${API}/repositories/vim`)
    })
    .then(res => {
      t.is(res.status, 200)
      return res.json()
    })
    .then(res => {
      t.is(res.name, 'vim')
      t.is(res.image, 'ustcmirror/test:latest')
      t.is(res.args[0], 'echo')
      t.is(res.storageDir, '/srv/repo/vim')
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
