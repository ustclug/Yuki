#!/usr/bin/node

'use strict'

require('../dist')

import test from 'ava'
import mongoose from 'mongoose'
import DATA from './mock.json'
import Client from '../dist/request'
import { API_PORT, TOKEN_NAME } from '../dist/config'
import { Repository as Repo, User } from '../dist/models'
import { isListening, getLocalTime } from '../build/Release/addon.node'

const request = new Client({
  baseUrl: `http://localhost:${API_PORT}/api/v1/`
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
  return request.get('repositories')
    .then(async res => {
      t.is(res.status, 200)
      const data = await res.json()
      t.is(data.length, DATA.length)
      for (const r of data) {
        t.truthy(r.name)
      }
    })
})

test.serial('Remove repository', t => {
  return request.delete('repositories/gmt')
    .then(res => {
      t.is(res.status, 204)
      return request.get('repositories/gmt')
    })
    .then(res => {
      t.is(res.status, 404)
    })
})

test.serial('Update repository', t => {
  return request.put('repositories/bioc', {
    image: 'ustcmirror/rsync:latest',
    cmd: ['echo', '1'],
    volumes: {
      '/pypi': '/tmp/repos/BIOC'
    },
    user: 'mirror'
  })
    .then(res => {
      t.is(res.status, 204)
      return request.get('repositories/bioc')
    })
    .then(async res => {
      t.is(res.status, 200)
      const data = await res.json()
      t.is(data.interval, '48 2 * * *')
      t.is(data.user, 'mirror')
      t.is(data.image, 'ustcmirror/rsync:latest')
      t.is(data.cmd[0], 'echo')
      t.is(data.cmd[1], '1')
      t.true(data.volumes['/pypi'].endsWith('BIOC'))
    })
})

test('Get repository', t => {
  return request.get('repositories/archlinux')
    .then(async res => {
      const data = await res.json()
      t.is(res.status, 200)
      t.is(data.image, 'ustcmirror/test:latest')
      t.is(data.interval, '1 1 * * *')
      t.is(data.storageDir, '/tmp/repos/archlinux')
      t.is(data.envs['RSYNC_USER'], 'asdh')
    })
})

test('Filter repositories', t => {
  return request.get('repositories?type=gitsync')
    .then(async res => {
      t.is(res.status, 200)
      const data = await res.json()
      t.is(data.length, 2)
      for (const r of data) {
        t.truthy(r.name)
        t.truthy(r.interval)
        t.is(r.image, 'ustcmirror/gitsync:latest')
      }
    })
})

test('Create repository', t => {
  return request.post('repositories/vim', {
    image: 'ustcmirror/test:latest',
    interval: '* 5 * * *',
    storageDir: '/tmp/repos/vim',
    cmd: ['echo', 'vim'],
  })
    .then(res => {
      t.is(res.status, 201)
      return request.get('repositories/vim')
    })
    .then(async res => {
      t.is(res.status, 200)
      const data = await res.json()
      t.is(data.interval, '* 5 * * *')
      t.is(data.image, 'ustcmirror/test:latest')
      t.is(data.cmd[0], 'echo')
      t.is(data.storageDir, '/tmp/repos/vim')
    })
})

test('Start a container', t => {
  return request.post('repositories/archlinux/sync', null)
    .then(res => {
      t.is(res.status, 204)
      return request.post('containers/archlinux/wait', null)
    })
    .then(async res => {
      t.is(res.status, 200)
      const data = await res.json()
      t.is(data.StatusCode, 0)
    })

})

test('List users', async t => {
  await request.get('users', adminToken)
    .then(async res => {
      t.is(res.status, 200)
      const data = await res.json()
      for (const r of data) {
        t.truthy(r.name)
        t.truthy(r.token)
      }
    })

  await request.get('users', normalToken)
    .then(async res => {
      t.is(res.status, 200)
      const data = await res.json()
      for (const r of data) {
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

  await request.get('users/foo', adminToken)
    .then(async res => {
      t.is(res.status, 200)
      const data = await res.json()
      t.true(data.name === 'foo')
    })
})

test.serial('Remove user', async t => {
  await request.delete('users/foo', adminToken)
    .then(res => {
      t.is(res.status, 204)
    })

  await request.get('users/foo', adminToken)
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
