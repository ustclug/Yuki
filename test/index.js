#!/usr/bin/node

'use strict'

require('../dist')

import { API_PORT, TOKEN_NAME } from '../dist/config'
import test from 'ava'
import { Repository as Repo, User } from '../dist/models'
import DATA from './mock.json'
import mongoose from 'mongoose'
import request from '../dist/request'
import { isListening, getLocalTime } from '../build/Release/addon.node'

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

const API = `http://localhost:${API_PORT}/api/v1`

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

const clone = function(obj) {
  return Object.assign({}, obj)
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

test('List repositories', t => {
  return request(`${API}/repositories`)
    .then(res => {
      t.true(res.ok)
      return res.json()
    })
    .then(res => {
      t.is(res.length, DATA.length)
      for (const r of res) {
        t.truthy(r.name)
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
      t.is(res.image, 'ustcmirror/test:latest')
      t.is(res.interval, '1 1 * * *')
      t.is(res.storageDir, '/tmp/repos/archlinux')
      t.is(res.envs[0], 'RSYNC_USER=asdh')
    })
})

test.serial('Update repository', t => {
  return request(`${API}/repositories/bioc`, {
    image: 'ustcmirror/rsync:latest',
    args: ['echo', '1'],
    volumes: ['/pypi:/tmp/repos/BIOC'],
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
      t.is(res.interval, '48 2 * * *')
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
    storageDir: '/tmp/repos/vim',
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
      t.is(res.interval, '* 5 * * *')
      t.is(res.image, 'ustcmirror/test:latest')
      t.is(res.args[0], 'echo')
      t.is(res.storageDir, '/tmp/repos/vim')
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

test('List users', async t => {
  await request(`${API}/users`, null, 'GET', adminToken)
    .then(res => {
      t.true(res.ok)
      return res.json()
    })
    .then(res => {
      for (const r of res) {
        t.truthy(r.name)
        t.truthy(r.token)
      }
    })

  await request(`${API}/users`, null, 'GET', normalToken)
    .then(res => {
      t.true(res.ok)
      return res.json()
    })
    .then(res => {
      for (const r of res) {
        t.truthy(r.name)
        t.falsy(r.token)
      }
    })
})

test.serial('Create user', async t => {
  await request(`${API}/users/foo`, {
    password: '3rd',
    admin: false
  }, 'POST', clone(adminToken))
    .then(res => {
      t.true(res.ok)
    })

  await request(`${API}/users/foo`, null, 'GET',
                clone(adminToken))
    .then(res => {
      t.true(res.ok)
      return res.json()
    })
    .then(data => {
      t.true(data.name === 'foo')
    })
})

test.serial('Remove user', async t => {
  await request(`${API}/users/foo`, null, 'DELETE',
                clone(adminToken))
    .then(res => {
      t.true(res.ok)
      return res.json()
    })

  await request(`${API}/users/foo`, null, 'GET',
                clone(adminToken))
    .then(res => {
      t.false(res.ok)
    })
})

test('Update user profile', async t => {
  await request(`${API}/users/kiana`, {
    admin: true
  }, 'PUT', clone(normalToken))
    .then(res => {
      t.false(res.ok)
    })

  await request(`${API}/users/kiana`, {
    password: 'asdfqwer'
  }, 'PUT', clone(normalToken))
    .then(res => {
      t.true(res.ok)
    })

  await request(`${API}/users/bronya`, {
    admin: true
  }, 'PUT', clone(adminToken))
    .then(res => {
      t.true(res.ok)
    })
})
