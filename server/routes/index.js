#!/usr/bin/node

'use strict'

import mongoose from 'mongoose'
import koarouter from 'koa-router'
import bodyParser from 'koa-bodyparser'
import Promise from 'bluebird'
import docker from './docker'
import models from '../models'
import config from '../config'
import logger from '../logger'
import { bringUp } from './util'

mongoose.Promise = Promise
const PREFIX = 'syncing'
const Repo = models.Repository
const router = koarouter({ prefix: '/api/v1' })

const uri = config.isTest ?
  `mongodb://${config.dbHost}/test` :
  `mongodb://${config.dbUser}:${config.dbPasswd}@${config.dbHost}:${config.dbPort}/${config.dbName}`

mongoose.connect(uri, {
  promiseLibrary: Promise,
})

const routerProxy = { router, url: '/' }

function setErrMsg(ctx, msg) {
  ctx.body = { message: msg }
}

if (config.isDev) {
  router.use('/', async (ctx, next) => {
    logger.debug(ctx.request.method, ctx.request.url)
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
      setErrMsg(ctx, 'invalid json')
    }
  }
}))
  .get((ctx) => {
    return Repo.findById(ctx.params.name)
      .then((data) => {
        if (data !== null) {
          // dirty workaround to get rid of _id
          // cannot exclude _id in query
          // since name is a virtual key which depends on _id
          data = data.toJSON()
          delete data['_id']
          ctx.body = data
        } else {
          ctx.status = 404
        }
      })
      .catch(err => {
        logger.error('Get repo:', err)
        ctx.status = 500
        setErrMsg(ctx, err)
      })
  })
  .post((ctx) => {
    const body = ctx.request.body
    body.name = ctx.params.name
    return Repo.create(body)
      .then((repo) => {
        ctx.status = 201
        ctx.body = {}
      }, (err) => {
        ctx.status = 400
        setErrMsg(ctx, err.errmsg)
      })
  })
  .put((ctx) => {
    return Repo.findByIdAndUpdate(ctx.params.name, ctx.request.body, {
      runValidators: true
    })
    .then(() => {
      ctx.body = {}
      ctx.status = 204
    })
    .catch(err => {
      logger.error('updating', err)
      ctx.status = 500
      setErrMsg(ctx, err)
    })
  })
  .delete((ctx) => {
    return Repo.findByIdAndRemove(ctx.params.name)
      .then(() => ctx.status = 204)
      .catch(err => {
        logger.error('deleting', err)
        ctx.status = 500
        setErrMsg(ctx, err)
      })
  })

.get('/repositories/:name/sync', async (ctx) => {
  const name = ctx.params.name
  const cfg = await Repo.findById(name)
  if (cfg === null) {
    ctx.status = 404
    return
  }
  const debug = !!ctx.query.debug // dirty hack, convert to boolean
  const opts = {
    Image: cfg.image,
    Cmd: cfg.command,
    User: cfg.user || '',
    Env: cfg.envs,
    AttachStdin: debug,
    AttachStdout: debug,
    AttachStderr: debug,
    Tty: false,
    OpenStdin: true,
    Labels: {
      'syncing': ''
    },
    HostConfig: {
      Binds: cfg.volumes
    },
    name: `${PREFIX}-${name}`,
  }
  let ct
  try {
    ct = await bringUp(opts)
  } catch (e) {
    logger.debug(`Syncing ${name}: `, e.json.message)
    ctx.status = e.statusCode
    ctx.body = e.json
  }
  if (!debug) {
    ct.wait()
      .then(() => ct.remove({ v: true }))
      .catch(e => logger.error(JSON.stringify(e)))
  }
  ctx.status = 200
  ctx.body = {}
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
