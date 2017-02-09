#!/usr/bin/node

'use strict'

require('../dist')

import { API_PORT, TOKEN_NAME } from '../dist/config'
import test from 'ava'
import { Repository as Repo, User } from '../dist/models'
import DATA from './mock.json'
import mongoose from 'mongoose'
import axios from 'axios'
import { isListening, getLocalTime } from '../build/Release/addon.node'

axios.defaults.validateStatus = (status) => true
const request = axios.create({
  baseURL: `http://localhost:${API_PORT}/api/v1/`
})

test.before(async t => {
  mongoose.createConnection('mongodb://127.0.0.1/test')
  await Repo.remove()
  await Repo.create(DATA)
  await User.remove()
  await User.create([{
    name: 'yuki',
    password: 'longpass',
    admin: true,
  }, {
    name: 'kiana',
    password: 'password',
    admin: false,
  }, {
    name: 'bronya',
    password: 'homu',
    admin: false,
  }])
})

const adminToken = {
  headers: {
    [TOKEN_NAME]: '97e8bb1a932af6db0a33857d644280906c5a0395'
  }
}

const normalToken = {
  headers: {
    [TOKEN_NAME]: 'e6960169400e8607f1826ea6260a32763a68f3bc'
  }
}

test('Addon: isListening()', t => {
  t.true(isListening('127.0.0.1', API_PORT))
  t.true(isListening('localhost', API_PORT))
  t.false(isListening('localhost', 4))
})

test('Addon: getLocalTime()', t => {
  t.truthy(getLocalTime())
  t.truthy(getLocalTime(1485346154))
  t.throws(getLocalTime.bind(null, 1, 2))
})

test.serial('List repositories', t => {
  return request('repositories')
    .then(res => {
      t.is(res.status, 200)
      t.is(res.data.length, DATA.length)
      for (const r of res.data) {
        t.truthy(r.name)
      }
    })
})

test.serial('Remove repository', t => {
  return request.delete('repositories/gmt')
    .then(res => {
      t.is(res.status, 204)
      return request('repositories/gmt')
    })
    .then(res => {
      t.is(res.status, 404)
    })
})

test.serial('Update repository', t => {
  return request.put('repositories/bioc', {
    image: 'ustcmirror/rsync:latest',
    args: ['echo', '1'],
    volumes: ['/pypi:/tmp/repos/BIOC'],
    user: 'mirror'
  })
    .then(res => {
      t.is(res.status, 204)
      return request('repositories/bioc')
    })
    .then(res => {
      t.is(res.status, 200)
      t.is(res.data.interval, '48 2 * * *')
      t.is(res.data.user, 'mirror')
      t.is(res.data.image, 'ustcmirror/rsync:latest')
      t.is(res.data.args[0], 'echo')
      t.is(res.data.args[1], '1')
      t.true(res.data.volumes[0].endsWith('BIOC'))
    })
})

test('Get repository', t => {
  return request('repositories/archlinux')
    .then(res => {
      t.is(res.status, 200)
      t.is(res.data.image, 'ustcmirror/test:latest')
      t.is(res.data.interval, '1 1 * * *')
      t.is(res.data.storageDir, '/tmp/repos/archlinux')
      t.is(res.data.envs[0], 'RSYNC_USER=asdh')
    })
})

test('Create repository', t => {
  return request.post('repositories/vim', {
    image: 'ustcmirror/test:latest',
    interval: '* 5 * * *',
    storageDir: '/tmp/repos/vim',
    args: ['echo', 'vim'],
  })
    .then(res => {
      t.is(res.status, 201)
      return request('repositories/vim')
    })
    .then(res => {
      t.is(res.status, 200)
      t.is(res.data.interval, '* 5 * * *')
      t.is(res.data.image, 'ustcmirror/test:latest')
      t.is(res.data.args[0], 'echo')
      t.is(res.data.storageDir, '/tmp/repos/vim')
    })
})

test('Start a container', t => {
  return request.post('repositories/archlinux/sync', null)
    .then(res => {
      t.is(res.status, 204)
      return request.post('containers/archlinux/wait', null)
    })
    .then(res => {
      t.is(res.status, 200)
      t.is(res.data.StatusCode, 0)
    })

})

test('List users', async t => {
  await request('users', adminToken)
    .then(res => {
      t.is(res.status, 200)
      for (const r of res.data) {
        t.truthy(r.name)
        t.truthy(r.token)
      }
    })

  await request('users', normalToken)
    .then(res => {
      t.is(res.status, 200)
      for (const r of res.data) {
        t.truthy(r.name)
        t.falsy(r.token)
      }
    })
})

test.serial('Create user', async t => {
  await request.post('users/foo', {
    password: '3rd',
    admin: false
  }, adminToken)
    .then(res => {
      t.is(res.status, 201)
    })

  await request('users/foo', adminToken)
    .then(res => {
      t.is(res.status, 200)
      t.true(res.data.name === 'foo')
    })
})

test.serial('Remove user', async t => {
  await request.delete('users/foo', adminToken)
    .then(res => {
      t.is(res.status, 204)
    })

  await request('users/foo', adminToken)
    .then(res => {
      t.is(res.status, 404)
    })
})

test('Update user profile', async t => {
  await request.put('users/kiana', {
    admin: true
  }, normalToken)
    .then(res => {
      t.not(res.status, 204)
    })

  await request.put('users/kiana', {
    password: 'asdfqwer'
  }, normalToken)
    .then(res => {
      t.is(res.status, 204)
    })

  await request.put('users/bronya', {
    admin: true
  }, adminToken)
    .then(res => {
      t.is(res.status, 204)
    })
})
