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

.post('/repositories/:name/sync', async (ctx) => {
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
    AttachStdin: false,
    AttachStdout: false,
    AttachStderr: false,
    Tty: false,
    OpenStdin: true,
    Labels: {
      'syncing': ''
    },
    HostConfig: {
      Binds: cfg.volumes,
      NetworkMode: 'host',
      RestartPolicy: {
        Name: 'on-failure',
        MaximumRetryCount: 2
      },
    },
    name: `${PREFIX}-${name}`,
  }
  opts.Env.push(`REPO=${name}`)
  if (debug) opts.Env.push(`DEBUG=${debug}`)

  let ct
  try {
    ct = await bringUp(opts)
  } catch (e) {
    ctx.throw(e)
  }
  if (!debug) {
    ct.wait()
      .then(() => ct.remove({ v: true }))
      .catch(ctx.throw)
  }
  ctx.status = 204
})

.get('/containers', (ctx) => {
  return docker.listContainers({ all: true })
    .then((cts) => {
      ctx.body = cts.filter(info => typeof info.Labels['syncing'] !== 'undefined')
    })
})
.delete('/containers/:repo', (ctx) => {
  const name = `${PREFIX}-${ctx.params.repo}`
  const ct = docker.getContainer(name)
  return ct.remove({ v: true })
    .then(() => ctx.status = 204)
    .catch(ctx.throw)
})
.post('/containers/:repo/wait', (ctx) => {
  const name = `${PREFIX}-${ctx.params.repo}`
  const ct = docker.getContainer(name)
  return ct.wait()
    .then((res) => {
      ctx.status = 200
      ctx.body = res
    })
    .catch(err => {
      ctx.status = err.statusCode
      ctx.message = err.reason
      ctx.body = err.json
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
    })
    .catch(err => {
      ctx.status = err.statusCode
      //ctx.message = err.reason
      // FIXME: Inconsistent behaviour because of docker-modem
      // err.json is null
      //ctx.body = err.json
      setErrMsg(ctx, err.reason)
    })
});

['start', 'stop', 'restart', 'pause', 'unpause'].forEach(action => {
  router.post(`/containers/:repo/${action}`, ctx => {
    const name = `${PREFIX}-${ctx.params.repo}`
    const ct = docker.getContainer(name)
    return ct[action]()
      .then(() => ctx.status = 204)
      .catch(err => {
        ctx.status = err.statusCode
        ctx.message = err.reason
        ctx.body = err.json
      })
  })
})

export default router.routes()
