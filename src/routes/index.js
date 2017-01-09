#!/usr/bin/node

'use strict'

import mongoose from 'mongoose'
import koarouter from 'koa-router'
import bodyParser from 'koa-bodyparser'
import Promise from 'bluebird'
import docker from './docker'
import models from '../models'
import config from '../config'
import { bringUp } from './util'

mongoose.Promise = Promise
const PREFIX = 'syncing'
const Repo = models.Repository
const router = koarouter({ prefix: '/api/v1' })

const uri = config.isTest ?
              `mongodb://${config.dbhost}/test` :
              `mongodb://${config.dbuser}:${config.dbpasswd}@${config.dbhost}/${config.dbname}`

mongoose.connect(uri, {
  promiseLibrary: Promise,
})

const routerProxy = { router, url: '/' }

if (config.isDev) {
  router.use('/', async (ctx, next) => {
    console.log(ctx.request.method, ctx.request.url)
    await next()
  })
}

['get', 'put', 'post', 'delete', 'use'].forEach(m => {
  routerProxy[m] = function(url, cb) {
    if (typeof cb === 'undefined') {
      cb = url
    } else {
      this.url = url
    }
    this.router[m](this.url, cb)
    return this
  }
})

routerProxy.get('/repositories', (ctx) => {
  return Repo.find({}, { id: false })
    .then(data => ctx.body = data)
})

.use('/repositories/:name', bodyParser({
  onerror: function(err, ctx) {
    if (err) {
      ctx.status = 400
      ctx.body = { message: 'invalid json' }
    }
  }
}))
  .get((ctx) => {
    return Repo.findById(ctx.params.name)
      .then((data) => {
        if (data !== null) {
          ctx.body = data
        } else {
          ctx.status = 404
        }
      })
      .catch(err => {
        console.error('get repo', err)
        ctx.status = 500
        ctx.body = err
      })
  })
  .post((ctx) => {
    const body = ctx.request.body
    body.name = ctx.params.name
    return Repo.create(body)
      .then((repo) => {
        ctx.status = 201
        ctx.body = { message: `sucessfully created new repo: ${body.name}` }
      }, (err) => {
        ctx.status = 400
        ctx.body = { message: err.errmsg }
      })
  })
  .put((ctx) => {
    return Repo.findByIdAndUpdate(ctx.params.name, ctx.request.body, {
      runValidators: true
    })
    .then(() => {
      ctx.body = `${ctx.params.name} updated`
      ctx.status = 204
    })
    .catch(err => {
      console.error('updating', err)
      ctx.status = 500
      ctx.body = err
    })
  })
  .delete((ctx) => {
    return Repo.findByIdAndRemove(ctx.params.name)
      .then(() => ctx.status = 204)
      .catch(err => {
        console.error('deleting', err)
        ctx.status = 500
        ctx.body = err
      })
  })

.get('/repositories/:name/sync', async (ctx) => {
  const name = ctx.params.name
  const config = await Repo.findById(name)
  if (config === null) {
    ctx.status = 404
    return
  }
  const debug = !!ctx.query.debug // dirty hack, convert to boolean
  const opts = {
    Image: config.image,
    Cmd: config.command,
    User: config.user || '',
    Env: config.envs,
    AttachStdin: debug,
    AttachStdout: debug,
    AttachStderr: debug,
    Tty: false,
    OpenStdin: true,
    HostConfig: {
      Binds: [].concat(config.volumes, `${config.storageDir}:/repo`)
    },
    name: `${PREFIX}-${name}`,
  }
  try {
    await bringUp(opts)
    ctx.status = 200
  } catch (e) {
    console.error('bringUp', e)
    ctx.status = 500
    ctx.body = e
  }
})

.get('/containers', (ctx) => {
  return docker.listContainers({ all: true })
    .then((cts) => {
      ctx.body = cts.filter(info => info.Names[0].startsWith(`/${PREFIX}-`))
    })
})
.get('/containers/:repo/inspect', (ctx) => {
  const name = `${PREFIX}-${ctx.params.repo}`
  const ct = docker.getContainer(name)
  return ct.inspect()
    .then((data) => {
      ctx.body = data
    }, (err) => {
      ctx.status = err.statusCode
      ctx.message = err.reason
      ctx.body = err.json
    })
})
.get('/containers/:repo/logs', (ctx) => {
  const name = `${PREFIX}-${ctx.params.repo}`
  const ct = docker.getContainer(name)

  return ct.logs({
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
