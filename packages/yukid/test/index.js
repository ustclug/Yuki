#!/usr/bin/node

'use strict'

require('../dist')
import test from 'ava'
import mongoose from 'mongoose'
import DATA from './mock.json'
import Client from '../lib'
import { API_PORT } from '../dist/config'
import { TOKEN_NAME } from '../dist/globals'
import { Repository as Repo, User, Log } from '../dist/models'
import { isListening } from '../build/Release/addon.node'
import { createHash } from 'crypto'

const request = new Client({
  baseUrl: `http://localhost:${API_PORT}/api/v1/`
})

const md5hash = function(text) {
  return createHash('md5').update(text).digest('hex')
}

const calToken = (name, pass) => {
  const hash = createHash('sha1').update(name).update(pass)
  return hash.digest('hex')
}

test.before(async t => {
  mongoose.createConnection('mongodb://127.0.0.1/test')
  await Repo.remove()
  await Repo.create(DATA)
  await User.remove()
  await Log.remove()
  await User.create([{
    name: 'yuki',
    password: md5hash('longpass'),
    admin: true,
  }, {
    name: 'kiana',
    password: md5hash('password'),
    admin: false,
  }, {
    name: 'bronya',
    password: md5hash('homu'),
    admin: false,
  }])
})

const adminToken = {
  headers: {
    [TOKEN_NAME]: calToken('yuki', md5hash('longpass'))
  }
}

const normalToken = {
  headers: {
    [TOKEN_NAME]: calToken('kiana', md5hash('password'))
  }
}

test('Addon: isListening()', t => {
  t.true(isListening('127.0.0.1', API_PORT))
  t.true(isListening('localhost', API_PORT))
  t.false(isListening('localhost', 4))
})

test('String.prototype.splitN', t => {
  const s = 'q=3=4=5'
  t.deepEqual(s.splitN('=', 0), ['q=3=4=5'])
  t.deepEqual(s.splitN('=', 1), ['q', '3=4=5'])
  t.deepEqual(s.splitN('=', 2), ['q', '3', '4=5'])
  t.deepEqual(''.splitN('=', 2), [''])
})

test('Object.entries', t => {
  const obj = { a: 3, b: 4 }
  for (const [k, v] of Object.entries(obj)) {
    t.truthy(k)
    t.truthy(v)
  }
})

test.serial('List repositories', t => {
  return request.get('repositories')
    .then(async res => {
      t.is(res.status, 200)
      const data = await res.json()
      t.is(data.length, DATA.length)
      for (const r of data) {
        t.truthy(r._id)
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
        t.truthy(r._id)
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
        t.truthy(r._id)
        t.truthy(r.token)
      }
    })

  await request.get('users', normalToken)
    .then(async res => {
      t.is(res.status, 200)
      const data = await res.json()
      for (const r of data) {
        t.truthy(r._id)
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
      t.true(data._id === 'foo')
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

  await User.findById('kiana', 'token')
    .then(data => {
      t.is(data.token, calToken('kiana', 'asdfqwer'))
    })

  await request.put('users/bronya', {
    admin: true
  }, adminToken)
    .then(res => {
      t.is(res.status, 204)
    })
})

test('Export config', async t => {
  await request.get('config', null, adminToken)
    .then(res => {
      t.is(res.status, 200)
    })
})

test.after('Import config', async t => {
  await Repo.remove()
  await request.get('config', adminToken)
    .then(res => {
      t.is(res.status, 200)
    })
})
