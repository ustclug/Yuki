#!/usr/bin/node

'use strict'

import fs from 'fs'
import { createGunzip } from 'zlib'
import path from 'path'
import stream from 'stream'
import koarouter from 'koa-router'
import Promise from 'bluebird'
import docker from '../docker'
import { Repository as Repo, User, Meta } from '../models'
import CONFIG from '../config'
import logger from '../logger'
import schedule from '../scheduler'
import { bringUp, dirExists, updateImages,
  makeDir, myStat, queryOpts,
  updateMeta, tailStream
} from '../util'

const PREFIX = CONFIG.CT_NAME_PREFIX
const LABEL = CONFIG.CT_LABEL
const router = new koarouter({ prefix: '/api/v1' })
const readDir = Promise.promisify(fs.readdir)

const routerProxy = { router, url: '/' }

function setErrMsg(ctx, msg) {
  ctx.body = { message: msg }
}

function isLoggedIn(ctx, next) {
  if (!ctx.state.isLoggedIn) {
    ctx.body = { message: 'unauthenticated access' }
    logger.warn(`Unauthenticated: ${ctx.method} ${ctx.request.url}`)
    return ctx.status = 401
  }
  return next()
}

function isAdmin(ctx, next) {
  if (!ctx.state.isAdmin) {
    ctx.body = { message: 'Operation not permitted. Please concat administrator.' }
    logger.warn(`Unauthorized: ${ctx.state.username} ${ctx.method} ${ctx.request.url}`)
    return ctx.status = 401
  }
  return next()
}

function getContainer(repo) {
  let spec = ''
  if (repo.startsWith('id:')) {
    spec = repo.slice(3)
  } else {
    spec = `${PREFIX}-${repo}`
  }
  return docker.getContainer(spec)
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

router.use(function jsonOnly(ctx, next) {
  if (ctx.accepts('json') === false) {
    setErrMsg(ctx, 'Only JSON is accepted.')
    return ctx.status = 400
  }
  return next()
})

/**
 * @apiVersion 0.1.0
 */

/**
 * @apiDefine CommonErr
 * @apiError {String} message Error message.
 */

/**
 * @apiDefine AccessToken
 * @apiHeader {String} x-mirror-token Users unique access-key.
 */

routerProxy
  /**
   * @api {get} /auth Request info of User
   * @apiName Whoami
   * @apiGroup Auth
   *
   * @apiUse AccessToken
   *
   * @apiSuccess {String} name Name of the User
   * @apiSuccess {Boolean} admin Whether User is admin
   *
   * @apiUse CommonErr
   */
  .get('/auth', isLoggedIn, (ctx) => {
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
  /**
   * @api {post} /auth Log in to remote registry
   * @apiName Login
   * @apiGroup Auth
   *
   * @apiParam {String} username User's name
   * @apiParam {String} password User's password
   *
   * @apiSuccess {String} token Token of the User
   *
   * @apiUse CommonErr
   */
  .post(async (ctx) => {
    const name = ctx.body.username
    const pwHash = ctx.body.password
    const token = await User.findOne({
      _id: name,
      password: pwHash
    }, {
      _id: false,
      password: false
    })
    if (token === null) {
      logger.warn(`Auth: invalid user or password: <${name}>`)
      setErrMsg(ctx, 'invalid user or password')
      return ctx.status = 404
    }
    ctx.body = token
    logger.info(`${name} login`)
  })

routerProxy
  .get('/meta', (ctx) => {
    const availKeys = ['name', 'size', 'lastSuccess']
    let key = ctx.query.key || 'name'
    if (availKeys.indexOf(key) < 0) {
      ctx.status = 400
      return setErrMsg(ctx, `invalid key: ${key}`)
    }
    const order = +ctx.query.order || 1
    if (key === 'name') key = '_id'
    return Meta.find()
      .populate('upstream')
      .sort({ [key]: order })
      .then((docs) => docs.map(r => r.toJSON()))
      .then((data) => ctx.body = data)
  })
  .get('/meta/:name', (ctx) => {
    const name = ctx.params.name
    return Meta.findById(name)
      .populate('upstream')
      .then(r => {
        if (r === null) {
          ctx.status = 404
          return setErrMsg(ctx, `no such repository: ${ctx.params.name}`)
        }
        ctx.body = r.toJSON()
      })
  })

/**
 * @api {get} /repositories List repositories
 * @apiName ListRepositories
 * @apiGroup Repositories
 *
 * @apiParam {String} type Basename of the Docker image
 *
 * @apiSuccess {Object[]} repos(virtual field) List of repositories
 * @apiSuccess {String} .interval Task interval
 * @apiSuccess {String} .image Name of the Docker image
 * @apiSuccess {String} .storageDir Path to storage directory
 * @apiSuccess {Boolean} .autoRotLog Whether rotates log automatically
 * @apiSuccess {Number} .rotateCycle Number of the cycle versions to save
 * @apiSuccess {String[]} .args Arguments passed to image
 * @apiSuccess {String[]} .envs Environment variables
 * @apiSuccess {String[]} .volumes Volumes to be mount
 * @apiSuccess {String} .bindIp Local ip to be bound
 * @apiSuccess {String} .user Owner of the storage directory
 *
 * @apiUse CommonErr
 */
routerProxy.get('/repositories', (ctx) => {
  const type = ctx.query.type
  const query = type ? {
    'image': {
      '$regex': `ustcmirror/${type}`
    }
  } : null
  return Repo.find(query, { image: 1, interval: 1 })
    .sort({ _id: 1 })
    .exec()
    .then(data => data.map(r => {
      r = r.toJSON()
      r.scheduled = schedule.isScheduled(r._id)
      return r
    }))
    .then(data => ctx.body = data)
})

.use('/repositories/:name')
  /**
   * @api {get} /repositories/:name Get Repository
   * @apiName GetRepository
   * @apiGroup Repositories
   *
   * @apiParam {String} name Name of the Repository
   *
   * @apiSuccess {String} interval Task interval
   * @apiSuccess {String} image Name of the Docker image
   * @apiSuccess {String} storageDir Path to storage directory
   * @apiSuccess {Boolean} autoRotLog Whether rotates log automatically
   * @apiSuccess {Number} rotateCycle Number of the cycle versions to save
   * @apiSuccess {String[]} args Arguments passed to image
   * @apiSuccess {String[]} envs Environment variables
   * @apiSuccess {String[]} volumes Volumes to be mount
   * @apiSuccess {String} bindIp Local ip to be bound
   * @apiSuccess {String} user Owner of the storage directory
   *
   * @apiUse CommonErr
   */
  .get((ctx) => {
    const name = ctx.params.name
    const proj = ctx.state.isLoggedIn ? null : {
      interval: 1, image: 1
    }
    return Repo.findById(name, proj)
      .then((data) => {
        if (data !== null) {
          data = data.toJSON()
          data.scheduled = schedule.isScheduled(name)
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
  /**
   * @api {post} /repositories/:name Create Repository
   * @apiName CreateRepository
   * @apiGroup Repositories
   *
   * @apiUse AccessToken
   * @apiParam {String} name Name of the Repository
   * @apiParam {String} interval Task interval
   * @apiParam {String} image Name of the Docker image
   * @apiParam {String} storageDir Path to storage directory
   * @apiParam {Boolean} autoRotLog=true Whether rotates log automatically (Optional)
   * @apiParam {Number} rotateCycle=10 Number of the cycle versions to save (Optional)
   * @apiParam {String[]} args Arguments passed to image (Optional)
   * @apiParam {String[]} envs Environment variables (Optional)
   * @apiParam {String[]} volumes Volumes to be mount (Optional)
   * @apiParam {String} bindIp Local ip to be bound (Optional)
   * @apiParam {String} user Owner of the storage directory (Optional)
   *
   *
   * @apiUse CommonErr
   */
  .post(isLoggedIn, (ctx) => {
    const body = ctx.body
    body.name = ctx.params.name
    if (!dirExists(body.storageDir)) {
      setErrMsg(ctx, `no such directory: ${body.storageDir}`)
      logger.error(`Creating ${body.name}: no such directory: ${body.storageDir}`)
      return ctx.status = 400
    }
    return Repo.create(body)
      .then((repo) => {
        ctx.status = 201
        schedule.addJob(repo._id, repo.interval)
      }, (err) => {
        logger.error(`Creating ${body.name}: %s`, err)
        ctx.status = 400
        setErrMsg(ctx, err.message)
      })
  })
  /**
   * @api {put} /repositories/:name Update Repository
   * @apiName UpdateRepository
   * @apiGroup Repositories
   *
   * @apiUse AccessToken
   * @apiParam {String} name Name of the Repository
   * @apiParam {String} interval Task interval
   * @apiParam {String} image Name of the Docker image
   * @apiParam {String} storageDir Path to storage directory
   * @apiParam {Boolean} autoRotLog=true Whether rotates log automatically
   * @apiParam {Number} rotateCycle=10 Number of the cycle versions to save
   * @apiParam {String[]} args Arguments passed to image
   * @apiParam {String[]} envs Environment variables
   * @apiParam {String[]} volumes Volumes to be mount
   * @apiParam {String} bindIp Local ip to be bound
   * @apiParam {String} user Owner of the storage directory
   *
   * @apiSuccess (Success 204) {String} empty
   *
   * @apiUse CommonErr
   */
  .put(isLoggedIn, (ctx) => {
    const name = ctx.params.name
    return Repo.findByIdAndUpdate(name, ctx.body, {
      // return the modified doc
      new: true,
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
      setErrMsg(ctx, err.message)
    })
  })
  /**
   * @api {delete} /repositories/:name Delete Repository
   * @apiName DeleteRepository
   * @apiGroup Repositories
   *
   * @apiUse AccessToken
   * @apiParam {String} name Name of the Repository
   *
   * @apiSuccess (Success 204) {String} empty
   *
   * @apiUse CommonErr
   */
  .delete(isLoggedIn, (ctx) => {
    const name = ctx.params.name
    return Repo.findByIdAndRemove(name)
      .then(() => {
        schedule.cancelJob(name)
        ctx.status = 204
      })
      .catch(err => {
        logger.error(`Deleting ${name}: %s`, err)
        ctx.status = 500
        setErrMsg(ctx, err.message)
      })
  })

/**
 * @api {get} /repositories/:name/sync Got Logs of Repository
 * @apiName FetchRepositoryLogs
 * @apiGroup Repositories
 *
 * @apiUse AccessToken
 * @apiParam {String} name Name of the Repository
 * @apiParam {Number} n Nth log-file
 * @apiParam {Boolean} stats Only return matched log-files' names
 *
 * @apiSuccess (Success 200) {Stream} logs Log stream
 * @apiSuccess (Success 200) {Object} names Names of the log-files
 *
 * @apiUse CommonErr
 */
.get('/repositories/:name/logs', isLoggedIn, (ctx) => {
  const repo = ctx.params.name
  const nth = ctx.query.n || 0
  const stats = !!ctx.query.stats
  const tail = ctx.query.tail || 'all'
  const logdir = path.join(CONFIG.LOGDIR_ROOT, repo)
  if (!dirExists(logdir)) {
    setErrMsg(ctx, `no such repo ${repo}`)
    return ctx.status = 404
  }

  if (!/^(all|\d+)$/.test(tail)) {
    setErrMsg(ctx, `Invalid argument: tail: ${tail}`)
    return ctx.status = 401
  }

  if (stats) {
    return readDir(logdir)
      .then((files) => {
        ctx.body = files.filter(f => f.startsWith('result.log.'))
          .reduce((acc, f) => {
            try {
              acc.push(myStat(logdir, f))
            } finally {
            // eslint-disable-next-line no-unsafe-finally
              return acc
            }
          }, [])
          .sort((x, y) => x.mtime - y.mtime)
      })
  }

  return readDir(logdir)
    .then((files) => {
      const wantedName = `result.log.${nth}`
      for (const f of files) {
        if (f.startsWith(wantedName)) {
          if (+tail === 0) {
            return ctx.body = ''
          }
          const fp = path.join(logdir, f)
          let content = null
          switch (path.extname(f)) {
            case '.gz':
              content = fs.createReadStream(fp).pipe(createGunzip())
              break
            default:
              content = fs.createReadStream(fp)
              break
          }
          if (tail === 'all') {
            return ctx.body = content
          }
          return tailStream(+tail, content)
            .then((data) => ctx.body = data)
        }
      }
      setErrMsg(ctx, `${path.join(repo, wantedName)} cannot be found`)
      ctx.status = 404
    })
    .catch(e => {
      logger.error(`${repo} logs: %s`, e)
      setErrMsg(ctx, e.message)
      ctx.status = 500
    })
})

/**
 * @api {post} /repositories/:name/sync Sync Repository
 * @apiName SyncRepository
 * @apiGroup Repositories
 *
 * @apiUse AccessToken
 * @apiParam {String} name Name of the Repository
 * @apiParam {Boolean} debug Start the container in debug mode
 *
 * @apiSuccess (Success 204) {String} empty
 * @apiSuccess (Success 200) {Stream} logs Log stream
 *
 * @apiUse CommonErr
 */
.post('/repositories/:name/sync', isLoggedIn, async (ctx) => {
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
    ctx.status = 204
  } else {
    ctx.res.setTimeout(0)
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
        logger.error(`${name} sync: %s`, err)
        ctx.status = err.statusCode
        setErrMsg(ctx, err.reason)
      })
  }

})

/**
 * @api {get} /containers List containers
 * @apiName ListContainers
 * @apiGroup Containers
 *
 * @apiSuccess {Object[]} containers(virtual field) List of containers
 *
 * @apiUse CommonErr
 */
.get('/containers', (ctx) => {
  return docker.listContainers({
    all: true,
    filters: {
      label: {
        [LABEL]: true,
        'ustcmirror.images': true
      }
    }
  })
    .then((data) => {
      ctx.body = data
    })
})
/**
 * @api {get} /containers/:repo Inspect container
 * @apiName InspectContainer
 * @apiGroup Containers
 *
 * @apiUse AccessToken
 * @apiParam {String} repo Name of the Repository
 *
 * @apiUse CommonErr
 */
.get('/containers/:repo', isLoggedIn, (ctx) => {
  const ct = getContainer(ctx.params.repo)
  return ct.inspect()
    .then((data) => {
      ctx.body = data
    }, (err) => {
      logger.error('Inspect container: %s', err)
      ctx.status = err.statusCode
      ctx.message = err.reason
      ctx.body = err.json
    })
})
/**
 * @api {delete} /containers/:repo Delete container
 * @apiName ListContainers
 * @apiGroup Containers
 *
 * @apiUse AccessToken
 * @apiParam {String} repo Name of the Repository
 *
 * @apiSuccess {String} empty
 *
 * @apiUse CommonErr
 */
.delete(isLoggedIn, (ctx) => {
  const ct = getContainer(ctx.params.repo)
  return ct.remove({ v: true, force: true })
    .then(() => ctx.status = 204)
    .catch(err => {
      logger.error('Delete container: %s', err)
      ctx.body = err.json
      ctx.status = err.statusCode
    })
})
/**
 * @api {post} /containers/:repo/wait Await container stop
 * @apiName WaitForContainer
 * @apiGroup Containers
 *
 * @apiParam {String} repo Name of the Repository
 *
 * @apiSuccess {Object} StatusCode Exit code
 *
 * @apiUse CommonErr
 */
.post('/containers/:repo/wait', (ctx) => {
  const ct = getContainer(ctx.params.repo)
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
/**
 * @api {get} /containers/:repo/logs Fetch logs of container
 * @apiName GetLogsOfContainer
 * @apiGroup Containers
 *
 * @apiUse AccessToken
 * @apiParam {String} repo Name of the Repository
 * @apiParam {Boolean} follow Follow log output
 * @apiParam {String} tail=all Number of lines to show from the end of the logs
 *
 * @apiSuccess {Stream} log Log Stream
 * @apiUse CommonErr
 */
.get('/containers/:repo/logs', isLoggedIn, (ctx) => {
  const follow = !!ctx.query.follow
  const tail = ctx.query.tail || 'all'

  const repo = ctx.params.repo
  const ct = getContainer(repo)
  const opts = {
    stdout: true,
    stderr: true,
    tail,
    follow
  }
  ctx.res.setTimeout(0)
  return ct.logs(opts)
    .then(s => {
      const logStream = new stream.PassThrough()
      s.on('end', () => logStream.end())
      ct.modem.demuxStream(s, logStream, logStream)
      ctx.body = logStream
    })
    .catch(err => {
      logger.error(`${repo} logs: %s`, err)
      ctx.status = err.statusCode
      setErrMsg(ctx, err.reason)
    })
})

router.post('/containers/:repo/stop', isLoggedIn, ctx => {
  const ct = getContainer(ctx.params.repo)
  const t = ctx.query.t || 10 // timeout(sec)
  return ct.stop({ t })
    .then(() => ctx.status = 204)
    .catch(err => {
      logger.error('Stopping container: %s', err)
      ctx.status = err.statusCode
      ctx.message = err.reason
      ctx.body = err.json
    })
})

const actions = ['start', 'restart', 'pause', 'unpause']
actions.forEach(action => {
  router.post(`/containers/:repo/${action}`, isLoggedIn, ctx => {
    const ct = getContainer(ctx.params.repo)
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

routerProxy.post('/images/update', isLoggedIn, async (ctx) => {
  return updateImages()
    .catch(err => {
      logger.error('Update images: %s', err)
      ctx.status = err.statusCode
      ctx.message = err.reason
      ctx.body = err.json
    })
})

/**
 * @api {get} /users List Users
 * @apiName ListUsers
 * @apiGroup Users
 *
 * @apiUse AccessToken
 *
 * @apiSuccess {Object[]} users
 *
 * @apiUse CommonErr
 */
routerProxy.get('/users', isLoggedIn, async (ctx) => {
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
/**
 * @api {put} /users/:name Update User
 * @apiName UpdateUser
 * @apiGroup Users
 *
 * @apiUse AccessToken
 * @apiParam {String} name Name of the User
 *
 * @apiUse CommonErr
 */
.put('/users/:name', isLoggedIn, ctx => {
  const name = ctx.params.name
  if (!ctx.state.isAdmin) {
    if (name !== ctx.state.username || ctx.body.admin) {
      setErrMsg(ctx, 'operation not permitted')
      logger.warn(`<${ctx.state.username}> tried to update <${ctx.params.name}>`)
      return ctx.status = 401
    }
  }
  return User.findByIdAndUpdate(name, ctx.body, {
    runValidators: true
  })
  .then((data) => {
    if (data === null) {
      setErrMsg(ctx, `no such user: ${name}`)
      logger.warn(`<${ctx.state.username}> tried to update <${ctx.params.name}>`)
      return ctx.status = 404
    }
    ctx.status = 204
  })
  .catch(err => {
    logger.error(`Updating user <${name}>: %s`, err)
    ctx.status = 500
    setErrMsg(ctx, err.errmsg)
  })
})

.use('/users/:name', isLoggedIn, isAdmin)
/**
 * @api {get} /users/:name Get User
 * @apiName GetUser
 * @apiGroup Users
 *
 * @apiUse AccessToken
 * @apiPermission admin
 * @apiParam {String} name Name of the User
 *
 * @apiUse CommonErr
 */
.get(async ctx => {
  const name = ctx.params.name
  const user = await User.findById(name, { password: false })
  if (user === null) {
    ctx.status = 404
    setErrMsg(ctx, `no such user <${name}>`)
  } else {
    ctx.body = user
  }
})
/**
 * @api {post} /users/:name Create User
 * @apiName CreateUser
 * @apiGroup Users
 *
 * @apiUse AccessToken
 * @apiPermission admin
 * @apiParam {String} name Name of the User
 *
 * @apiUse CommonErr
 */
.post(ctx => {
  const body = ctx.body
  const newUser = {
    name: ctx.params.name,
    password: body.password,
    admin: !!body.admin
  }
  return User.create(newUser)
  .then(() => {
    ctx.status = 201
  }, err => {
    logger.error(`Creating user <${ctx.params.name}>: %s`, err)
    ctx.status = 400
    setErrMsg(ctx, err.message)
  })
})
/**
 * @api {delete} /users/:name Delete User
 * @apiName DeleteUser
 * @apiGroup Users
 *
 * @apiUse AccessToken
 * @apiPermission admin
 * @apiParam {String} name Name of the User
 *
 * @apiUse CommonErr
 */
.delete(ctx => {
  const name = ctx.params.name
  return User.findByIdAndRemove(name)
  .then((user) => {
    if (user !== null) {
      ctx.status = 204
    } else {
      ctx.status = 404
      logger.warn(`<${ctx.state.username}> tried to delete <${ctx.params.name}>`)
      setErrMsg(ctx, `no such user: <${name}>`)
    }
  }, err => {
    logger.error(`Removing user <${ctx.params.name}>: %s`, err)
    ctx.status = 500
    setErrMsg(ctx, err.message)
  })

})

routerProxy.use('/config', isLoggedIn)
/**
 * @api {get} /config Export config of Repos
 * @apiName ExportConfig
 * @apiGroup Config
 *
 * @apiUse AccessToken
 *
 * @apiUse CommonErr
 */
.get((ctx) => {
  const pretty = !!ctx.query.pretty
  return Repo.find()
    .sort({ _id: 1 }).exec()
    .then(docs => {
      docs = docs.map(d => d.toJSON({ versionKey: false, getters: false }))
      if (pretty) {
        ctx.body = JSON.stringify(docs, null, 2)
      } else {
        ctx.body = docs
      }
    })
    .catch(err => {
      logger.error('Export config: %s', err)
      setErrMsg(ctx, err.message)
      ctx.status = 500
    })
})
/**
 * @api {post} /config Import config
 * @apiName ImportConfig
 * @apiGroup Config
 *
 * @apiUse AccessToken
 * @apiPermission admin
 *
 * @apiUse CommonErr
 */
.post(isAdmin, (ctx) => {
  const repos = ctx.body
  return Repo.create(repos)
    .then(updateMeta)
    .then(() => {
      ctx.status = 200
    }, (err) => {
      logger.error('Import config: %s', err)
      setErrMsg(ctx, err.message)
      ctx.status = 500
    })
})

/**
 * @api {post} /reload Reload config
 * @apiName ReloadConfig
 * @apiGroup Config
 *
 * @apiUse AccessToken
 * @apiPermission admin
 *
 * @apiUse CommonErr
 */
.post('/reload', isLoggedIn, isAdmin, (ctx) => {
  return schedule.schedRepos()
    .catch(err => {
      logger.error('Reload config: %s', err)
      setErrMsg(ctx, err.message)
      ctx.status = 500
    })
})

export default router
