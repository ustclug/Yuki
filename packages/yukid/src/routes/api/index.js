#!/usr/bin/node

'use strict'

import koarouter from 'koa-router'
import { Repository as Repo, User, Meta } from '../../models'
import logger from '../../logger'
import scheduler from '../../scheduler'
import { updateImages } from '../../containers'
import { createMeta } from '../../repositories'
import { setErrMsg, isLoggedIn, isAdmin } from './lib'

import containerRoutes from './containers'
import repoRoutes from './repos'
import userRoutes from './users'

const router = new koarouter({ prefix: '/api/v1' })

const routerProxy = { router, url: '/' }

;['get', 'put', 'post', 'delete', 'use'].forEach(m => {
  routerProxy[m] = function(url, ...rest) {
    if (typeof url === 'string') {
      this.url = url
      this.router[m].call(this.router, url, ...rest)
    } else {
      this.router[m].call(this.router, this.url, url, ...rest)
    }
    return this
  }
})

containerRoutes(routerProxy)
repoRoutes(routerProxy)
userRoutes(routerProxy)

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
    const name = ctx.$body.username
    const pwHash = ctx.$body.password
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

  .post('/images/update', isLoggedIn, async (ctx) => {
    return updateImages()
      .then(() => {
        ctx.status = 200
      }, (err) => {
        logger.error('Update images: %s', err)
        ctx.status = err.statusCode
        ctx.message = err.reason
        ctx.body = err.json
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
    const repos = ctx.$body
    return Repo.create(repos)
      .then(createMeta)
      .then(() => {
        ctx.status = 200
        ctx.body = {
          message: 'Data has been successfully imported.'
        }
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
    return scheduler.schedRepos()
      .then(() => {
        ctx.status = 200
      }, (err) => {
        logger.error('Reload config: %s', err)
        setErrMsg(ctx, err.message)
        ctx.status = 500
      })
  })

export default router
