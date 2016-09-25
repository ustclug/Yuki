#!/usr/bin/node

'use strict'

import config from '../config.json'
import mongoose from 'mongoose'
import koarouter from 'koa-router'
import bodyParser from 'koa-bodyparser'
import Promise from 'bluebird'
import docker from './docker'
import models from '../models'
import {bringUp} from './util'

mongoose.Promise = Promise
const PREFIX = 'syncing'
const Repo = models.Repository
const router = koarouter({ prefix: '/api/v1' })

const isTest = process.env.NODE_ENV === 'test'
const isDev = process.env.NODE_ENV.startsWith('dev')
const uri = isTest ?
              'mongodb://127.0.0.1/test' :
              `mongodb://${config.user}:${config.passwd}@127.0.0.1/${config.dbname}`

mongoose.connect(uri, {
  promiseLibrary: Promise,
})
if (isDev) {
  router.use('/', async (ctx, next) => {
    console.log(ctx.request.method, ctx.request.url)
    await next()
  })
}

const listContainers = Promise.promisify(docker.listContainers, { context: docker })

// NOTE: id can be name of the container
const getContainer = id => Promise.promisifyAll(docker.getContainer(id))

router.get('/repositories/list', async (ctx) => {
  await Repo.find({}, { id: false })
    .then(data => ctx.body = data)
})

router.use('/repositories/:name', bodyParser({
  onerror: function(err, ctx) {
    if (err) {
      ctx.status = 400
      ctx.body = { message: 'invalid json' }
    }
  }
}))
.get('/repositories/:name', async (ctx) => {
  await Repo.findById(ctx.params.name)
    .then((data) => {
      if (data !== null) {
        ctx.body = data
      } else {
        ctx.status = 404
      }
    })
    .catch(err => {
      console.log('get repo', err)
      ctx.status = 500
      ctx.body = err
    })
})
.post('/repositories/:name', async (ctx) => {
  const body = ctx.request.body
  body.name = ctx.params.name
  await Repo.create(body)
    .then((repo) => {
      ctx.body = `sucessfully created new repo: ${body.name}`
      ctx.body = { message: `sucessfully created new repo: ${body.name}` }
    }, (err) => {
      ctx.status = 400
      ctx.body = { message: err.errmsg }
    })
})
.put('/repositories/:name', async (ctx) => {
  await Repo.findByIdAndUpdate(ctx.params.name, ctx.request.body, {
    runValidators: true
  })
  .then(() => ctx.body = `${ctx.params.name} updated`)
  .catch(err => {
    console.log('updating', err)
    ctx.body = err
  })
})
.delete('/repositories/:name', async ctx => {
  await Repo.findByIdAndRemove(ctx.params.name)
  .then(() => ctx.body = `${ctx.params.name} deleted`)
  .catch(err => {
    console.log('updating', err)
    ctx.body = err
  })
})

router.get('/repositories/:name/sync', async (ctx) => {
  const name = ctx.params.name
  try {
    const config = await Repo.findById(name)
    await bringUp({
      Image: config.image,
      Cmd: config.command,
      User: config.user || '',
      Env: config.envs,
      HostConfig: {
        Binds: [].concat(config.volumes, `${config.storageDir}:/data`)
      },
      name: `${PREFIX}-${name}`,
    })
    ctx.body = ''
  } catch (e) {
    /* handle error */
    console.error('bringUp', e)
    ctx.body = e
  }
})

router.get('/containers/list', async (ctx) => {
  await listContainers({ all: true })
    .then((cts) => {
      ctx.body = cts.filter(info => info.Names[0].startsWith(`/${PREFIX}-`))
    })
})
.get('/containers/:repo/inspect', async (ctx) => {
  const name = `${PREFIX}-${ctx.params.repo}`
  const ct = getContainer(name)
  await ct.inspectAsync()
    .then((data) => {
      ctx.body = data
    }, (err) => {
      ctx.status = err.statusCode
      ctx.message = err.reason
      ctx.body = err.json
    })
})
.get('/containers/:repo/logs', async (ctx) => {
  const name = `${PREFIX}-${ctx.params.repo}`
  const ct = getContainer(name)

  await ct.logsAsync({
    stdout: true,
    stderr: true,
    follow: false,
  })
    .then((stream) => {
      ctx.body = stream
    }, (err) => {
      ctx.status = err.statusCode
      ctx.message = err.reason
      ctx.body = err.json
    })
})

export default router.routes()
