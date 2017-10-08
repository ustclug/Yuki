require('../dist')
import test from 'ava'
import jwt from 'jsonwebtoken'
import mongoose from 'mongoose'
import DATA from './mock.json'
import Client from '../lib'
import CONFIG from '../dist/config'
import { TOKEN_NAME } from '../dist/globals'
import { Repository as Repo, User } from '../dist/models'

const request = new Client({
  baseUrl: `http://localhost:${CONFIG.get('API_PORT')}/api/v1/`
})

const calToken = (name) => {
  return jwt.sign({ name }, 'iamasecret', { noTimestamp: true })
}

test.before(async t => {
  mongoose.createConnection('mongodb://127.0.0.1/test', { useMongoClient: true })
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
    authorization: `Bearer ${calToken('yuki')}`
  }
}
const normalToken = {
  headers: {
    cookie: `${TOKEN_NAME}=${calToken('kiana')}`
  }
}

test('String.prototype.splitN', t => {
  const s = 'q=3=4=5'
  t.deepEqual(s.splitN('=', 0), ['q=3=4=5'])
  t.deepEqual(s.splitN('=', 1), ['q', '3=4=5'])
  t.deepEqual(s.splitN('=', 2), ['q', '3', '4=5'])
  t.deepEqual(''.splitN('=', 2), [''])
})

test('String.prototype.format', t => {
  const s = 'a${foo}${bar}'
  t.is(s.format({}), 'a')
  t.is(s.format({ a: 3 }), 'a')
  t.is(s.format({ foo: 1, bar: 2 }), 'a12')
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
  return request.delete('repositories/gmt', normalToken)
    .then(res => {
      t.is(res.status, 204)
      return request.get('repositories/gmt', normalToken)
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
  }, normalToken)
    .then(res => {
      t.is(res.status, 204)
      return request.get('repositories/bioc', normalToken)
    })
    .then(async res => {
      t.is(res.status, 200)
      const data = (await res.json())[0]
      t.is(data.interval, '48 2 * * *')
      t.is(data.user, 'mirror')
      t.is(data.image, 'ustcmirror/rsync:latest')
      t.true(data.volumes['/pypi'] === '/tmp/repos/BIOC')
    })
})

test('Get repository', t => {
  return request.get('repositories/archlinux', normalToken)
    .then(async res => {
      const data = (await res.json())[0]
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
  }, normalToken)
    .then(res => {
      t.is(res.status, 201)
      return request.get('repositories/vim', normalToken)
    })
    .then(async res => {
      t.is(res.status, 200)
      const data = (await res.json())[0]
      t.is(data.interval, '* 5 * * *')
      t.is(data.image, 'ustcmirror/test:latest')
      t.is(data.storageDir, '/tmp/repos/vim')
    })
})

test('List users', async t => {
  await request.get('users', adminToken)
    .then(async res => {
      t.is(res.status, 200)
      const data = await res.json()
      for (const r of data) {
        t.truthy(r._id)
      }
    })

  await request.get('users', normalToken)
    .then(async res => {
      t.is(res.status, 200)
      const data = await res.json()
      for (const r of data) {
        t.truthy(r._id)
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

  await request.put('users/bronya', {
    admin: true
  }, adminToken)
    .then(res => {
      t.is(res.status, 204)
    })
})

test('Export config', async t => {
  await request.get('config', adminToken)
    .then(res => {
      t.is(res.status, 200)
    })
})

test.serial('Import config', async t => {
  const repo = {
    "_id": "xbmc",
    "interval": "2 2 * * *",
    "image": "ustcmirror/test:latest",
    "storageDir": "/tmp/repos/xbmc.git"
  }
  await request.post('config', repo, adminToken)
    .then(res => {
      t.is(res.status, 201)
    })
})

test.serial('Reload config', async t => {
  await request.post('reload', null, adminToken)
    .then(res => {
      t.is(res.status, 204)
    })
})

test('Start a container', t => {
  return request.post('containers/archlinux', null, normalToken)
    .then(res => {
      t.is(res.status, 201)
      return request.post('containers/archlinux/wait', null, normalToken)
    })
    .then(async res => {
      t.is(res.status, 200)
      const data = await res.json()
      t.is(data.StatusCode, 0)
    })
})

test.after('List meta', async t => {
  await request.get('meta')
    .then(async res => {
      t.is(res.status, 200)
      const data = await res.json()
      for (const r of data) {
        t.not(r.lastSuccess, undefined)
        t.not(r.lastExitCode, undefined)
        t.not(r.size, undefined)
        t.not(r.upstream, undefined)
      }
    })
})
