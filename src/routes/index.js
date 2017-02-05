#!/usr/bin/node

'use strict'

import path from 'path'
import stream from 'stream'
import koarouter from 'koa-router'
import bodyParser from 'koa-bodyparser'
import Promise from 'bluebird'
import docker from '../docker'
import { Repository as Repo, User} from '../models'
import CONFIG from '../config'
import logger from '../logger'
import schedule from '../scheduler'
import auth from './auth'
import { bringUp, autoRemove, dirExists, makeDir, queryOpts } from '../util'

const PREFIX = CONFIG.CT_NAME_PREFIX
const LABEL = CONFIG.CT_LABEL
const router = koarouter({ prefix: '/api/v1' })

const routerProxy = { router, url: '/' }

function setErrMsg(ctx, msg) {
  ctx.body = { message: msg }
}

function isAuthorized(ctx, next) {
  if (!ctx.state.authorized) {
    ctx.body = { message: 'unauthorized access' }
    logger.warn(`Unauthorized: ${ctx.method} ${ctx.request.url}`)
    return ctx.status = 401
  }
  return next()
}

function isAdmin(ctx, next) {
  if (!ctx.state.isAdmin) {
    ctx.body = { message: 'Operation not permitted. Please concat administrator' }
    logger.warn(`Not permitted: ${ctx.state.username} ${ctx.method} ${ctx.request.url}`)
    return ctx.status = 401
  }
  return next()
}

if (CONFIG.isDev) {
  router.use(async (ctx, next) => {
    await next()
    logger.debug(ctx.request.method, ctx.status, ctx.request.url)
  })
}

['get', 'put', 'post', 'delete', 'use'].forEach(m => {
  routerProxy[m] = function(url, ...cb) {
    if (typeof url === 'string') {
      this.url = url
      this.router[m].call(this.router, url, ...cb)
    } else {
      this.router[m].call(this.router, this.url, url, ...cb)
    }
    return this
  }
})

const JSONParser = bodyParser({
  onerror: function(err, ctx) {
    if (err) {
      ctx.status = 400
      setErrMsg(ctx, 'invalid json')
    }
  }
})

router.use(auth)

routerProxy
  .get('/auth', isAuthorized, (ctx) => {
    if (ctx.state.username) {
      ctx.body = {
        name: ctx.state.username,
        admin: ctx.state.isAdmin
      }
    } else {
      logger.warn('Auth: impossible empty username')
      setErrMsg(ctx, 'unable to find matched user')
      ctx.status = 404
    }
  })
  .post(JSONParser, async (ctx) => {
    const name = ctx.request.body.username
    const pwHash = ctx.request.body.password
    const token = await User.findOne({
      _id: name,
      password: pwHash
    }, {
      _id: false,
      password: false
    })
    if (token === null) {
      logger.warn(`Auth: invalid user or password: ${name}`)
      setErrMsg(ctx, 'invalid user or password')
      return ctx.status = 404
    }
    ctx.body = token
    logger.info(`${name} login`)
  })

routerProxy.get('/repositories', (ctx) => {
  return Repo.find()
    .then(data => ctx.body = data)
})

.use('/repositories/:name', JSONParser)
  .get((ctx) => {
    const name = ctx.params.name
    return Repo.findById(name)
      .then((data) => {
        if (data !== null) {
          ctx.body = data
        } else {
          ctx.status = 404
          setErrMsg(ctx, `no such repository: ${ctx.params.name}`)
        }
      })
      .catch(err => {
        logger.error(`Get repo ${name}: %s`, err)
        ctx.status = 500
        setErrMsg(ctx, err)
      })
  })
  .post(isAuthorized, (ctx) => {
    const body = ctx.request.body
    body.name = ctx.params.name
    if (!dirExists(body.storageDir)) {
      setErrMsg(ctx, `no such directory: ${body.storageDir}`)
      logger.error(`Creating ${body.name}: no such directory: ${body.storageDir}`)
      return ctx.status = 400
    }
    return Repo.create(body)
      .then((repo) => {
        ctx.status = 201
        ctx.body = {}
        schedule.addJob(repo.name, repo.interval)
      }, (err) => {
        logger.error(`Creating ${body.name}: %s`, err)
        ctx.status = 400
        setErrMsg(ctx, err.errmsg)
      })
  })
  .put(isAuthorized, (ctx) => {
    const name = ctx.params.name
    return Repo.findByIdAndUpdate(name, ctx.request.body, {
      runValidators: true
    })
    .then((repo) => {
      // would automatically cancel the existing job
      schedule.addJob(repo._id /* name */, repo.interval)
      ctx.status = 204
    })
    .catch(err => {
      logger.error(`Updating ${name}: %s`, err)
      ctx.status = 500
      setErrMsg(ctx, err.errmsg)
    })
  })
  .delete(isAuthorized, (ctx) => {
    const name = ctx.params.name
    return Repo.findByIdAndRemove(name)
      .then(() => {
        schedule.cancelJob(name)
        ctx.status = 204
      })
      .catch(err => {
        logger.error(`Deleting ${name}: %s`, err)
        ctx.status = 500
        setErrMsg(ctx, err.errmsg)
      })
  })

.post('/repositories/:name/sync', isAuthorized, async (ctx) => {
  const name = ctx.params.name
  const debug = !!ctx.query.debug // dirty hack, convert to boolean

  const cfg = await Repo.findById(name)
  if (cfg === null) {
    setErrMsg(ctx, `no such repository: ${name}`)
    return ctx.status = 404
  }

  if (!dirExists(cfg.storageDir)) {
    setErrMsg(ctx, `no such directory: ${cfg.storageDir}`)
    logger.warn(`Sync ${name}: no such directory: ${cfg.storageDir}`)
    return ctx.status = 404
  }

  const logdir = path.join(CONFIG.LOGDIR_ROOT, name)
  try {
    makeDir(logdir)
  } catch (e) {
    setErrMsg(ctx, e.message)
    logger.error(`Sync ${name}: ${e.message}`)
    return ctx.status = 404
  }

  const opts = await queryOpts({ name, debug })

  let ct
  try {
    ct = await bringUp(opts)
  } catch (e) {
    logger.error(`bringUp ${name}: %s`, e)
    ctx.body = e.json
    return ctx.status = e.statusCode
  }

  if (!debug) {
    autoRemove(ct)
    .catch(e => logger.error(`Removing ${name}: %s`, e))
    ctx.status = 204
  } else {
    return ct.logs({
      stdout: true,
      stderr: true,
      follow: true
    })
      .then(s => {
        const logStream = new stream.PassThrough()
        s.on('end', () => {
          logStream.end()
        })
        ct.modem.demuxStream(s, logStream, logStream)
        ctx.body = logStream
      })
      .catch(err => {
        ctx.status = err.statusCode
        setErrMsg(ctx, err.reason)
      })
  }

})

.get('/containers', (ctx) => {
  return docker.listContainers({ all: true })
    .then((cts) => {
      ctx.body = cts.filter(info => typeof info.Labels[LABEL] !== 'undefined')
    })
})
.delete('/containers/:repo', isAuthorized, (ctx) => {
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
.get('/containers/:repo/inspect', isAuthorized, (ctx) => {
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
.get('/containers/:repo/logs', isAuthorized, (ctx) => {
  const name = `${PREFIX}-${ctx.params.repo}`
  const follow = !!ctx.query.follow
  const tail = ctx.query.tail || 'all'

  const ct = docker.getContainer(name)
  const opts = {
    stdout: true,
    stderr: true,
    tail,
    follow
  }
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
})

const actions = ['start', 'stop', 'restart', 'pause', 'unpause']
actions.forEach(action => {
  router.post(`/containers/:repo/${action}`, isAuthorized, ctx => {
    const name = `${PREFIX}-${ctx.params.repo}`
    const ct = docker.getContainer(name)
    return ct[action]()
      .then(() => ctx.status = 204)
      .catch(err => {
        logger.error(`${action} container: %s`, err)
        ctx.status = err.statusCode
        ctx.message = err.reason
        ctx.body = err.json
      })
  })
})

routerProxy.post('/images/update', isAuthorized, async (ctx) => {
  return Promise.all(CONFIG._images.map(tag => docker.pull(tag)))
    .then((data) => {
      ctx.status = 200
    })
    .catch(err => {
      logger.error('Update images: %s', err)
      ctx.status = err.statusCode
      ctx.message = err.reason
      ctx.body = err.json
    })
})

routerProxy.get('/users', isAuthorized, async (ctx) => {
  let users = null
  if (ctx.state.isAdmin) {
    // only hide password
    users = await User.find({}, { password: false })
  } else {
    // only return username
    users = await User.find({}, { name: true })
  }
  return ctx.body = users
})
.put('/users/:name', isAuthorized, JSONParser, ctx => {
  const name = ctx.params.name
  if (!ctx.state.isAdmin && name !== ctx.state.username) {
    setErrMsg(ctx, 'operation not permitted')
    logger.warn(`${ctx.state.username} ${ctx.method} ${ctx.params.name}`)
    return ctx.status = 401
  }
  return User.findByIdAndUpdate(name, ctx.request.body, {
    runValidators: true
  })
  .then((data) => {
    if (data === null) {
      setErrMsg(ctx, `no such user: ${name}`)
      return ctx.status = 404
    }
    ctx.status = 204
  })
  .catch(err => {
    logger.error(`Updating user ${name}: %s`, err)
    ctx.status = 500
    setErrMsg(ctx, err.errmsg)
  })
})

.use('/users/:name', isAuthorized, isAdmin, JSONParser)
.get(async ctx => {
  const name = ctx.params.name
  const user = await User.findById(name, { password: false })
  if (user === null) {
    ctx.status = 404
    setErrMsg(ctx, `no such user ${name}`)
  } else {
    ctx.body = user
  }
})
.post(ctx => {
  const body = ctx.request.body
  const newUser = {
    name: ctx.params.name,
    password: body.password,
    admin: !!body.admin
  }
  return User.create(newUser)
  .then(() => {
    ctx.status = 201
    ctx.body = {}
  }, err => {
    logger.error(`Creating user ${ctx.params.name}: %s`, err)
    ctx.status = 400
    setErrMsg(ctx, err.errmsg)
  })
})
.delete(ctx => {
  const name = ctx.params.name
  return User.findByIdAndRemove(name)
  .then(() => {
    ctx.status = 204
  }, err => {
    logger.error(`Removing user ${ctx.params.name}: %s`, err)
    ctx.status = 500
    setErrMsg(ctx, err.errmsg)
  })

})

routerProxy.use('/config', isAuthorized)
.get((ctx) => {
  return Repo.find()
    .sort({ _id: 1 }).exec()
    .then(docs => {
      ctx.body = docs
    })
    .catch(console.error)
})
.post(isAdmin, JSONParser, (ctx) => {
  const repos = ctx.request.body
  return Repo.create(repos)
    .then(() => {
      ctx.status = 200
    })
    .catch(console.error)
})

export default router.routes()
