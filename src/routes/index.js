#!/usr/bin/node

'use strict'

import stream from 'stream'
import koarouter from 'koa-router'
import bodyParser from 'koa-bodyparser'
import docker from '../docker'
import models from '../models'
import config from '../config'
import logger from '../logger'
import { bringUp, queryOpts, autoRemove } from '../util'

const PREFIX = config.CT_NAME_PREFIX
const LABEL = config.CT_LABEL
const Repo = models.Repository
const router = koarouter({ prefix: '/api/v1' })

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
          setErrMsg(ctx, `No such repository: ${ctx.params.name}`)
        }
      })
      .catch(err => {
        logger.error(`Get repo ${name}: %s`, err)
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
      logger.error(`Updating ${ctx.params.name}: %s`, err)
      ctx.status = 500
      setErrMsg(ctx, err)
    })
  })
  .delete((ctx) => {
    return Repo.findByIdAndRemove(ctx.params.name)
      .then(() => ctx.status = 204)
      .catch(err => {
        logger.error(`Deleting ${ctx.params.name}: %s`, err)
        ctx.status = 500
        setErrMsg(ctx, err)
      })
  })

.post('/repositories/:name/sync', async (ctx) => {
  const name = ctx.params.name
  const debug = !!ctx.query.debug // dirty hack, convert to boolean
  const opts = await queryOpts({ name, debug })
  if (opts === null) {
    ctx.status = 404
    setErrMsg(ctx, `no such repository: ${name}`)
    return
  }

  let ct
  try {
    ct = await bringUp(opts)
  } catch (e) {
    logger.error(`bringUp ${name}: %s`, e)
    return ctx.status = 500
  }

  if (!debug)
    autoRemove(ct)
    .catch(e => logger.error(`Removing ${name}: %s`, e))

  ctx.status = 204
})

.get('/containers', (ctx) => {
  return docker.listContainers({ all: true })
    .then((cts) => {
      ctx.body = cts.filter(info => typeof info.Labels[LABEL] !== 'undefined')
    })
})
.delete('/containers/:repo', (ctx) => {
  const name = `${PREFIX}-${ctx.params.repo}`
  const ct = docker.getContainer(name)
  return ct.remove({ v: true })
    .then(() => ctx.status = 204)
    .catch(err => {
      logger.error('Delete repo: %s', err)
      ctx.body = err.json
      ctx.status = err.statusCode
    })
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
  const follow = !!ctx.query.follow

  const ct = docker.getContainer(name)
  const opts = {
    stdout: true,
    stderr: true,
    follow
  }
  if (!follow) {
    return ct.logs(opts)
      .then(data => {
        ctx.body = data
      })
      .catch(err => {
        ctx.status = err.statusCode
        //ctx.message = err.reason
        // FIXME: Inconsistent behaviour because of docker-modem
        // err.json is null
        //ctx.body = err.json
        setErrMsg(ctx, err.reason)
      })
  } else {
    return ct.logs(opts)
      .then(s => {
        const logStream = new stream.PassThrough()
        s.on('end', () => logStream.end())
        ct.modem.demuxStream(s, logStream, logStream)
        ctx.body = logStream
      })
      .catch(err => {
        ctx.status = err.statusCode
        setErrMsg(ctx, err.reason)
      })
  }
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
