#!/usr/bin/node

'use strict'

import Router from 'koa-router'
import bodyParser from 'koa-bodyparser'
import api from './api'
import { IS_DEV, IS_TEST, TOKEN_NAME } from '../globals'
import { User } from '../models'
import logger from '../logger'

const router = new Router()

if (IS_DEV) {
  router.use(async (ctx, next) => {
    await next()
    logger.debug(ctx.request.method, ctx.status, ctx.request.url)
  })
}

router.use(async function auth(ctx, next) {
  const token = ctx.header[TOKEN_NAME] || ''
  const user = await User.findOne({ token })
  if (user === null) {
    ctx.state.isLoggedIn = IS_TEST
  } else {
    ctx.state.isLoggedIn = true
    ctx.state.isAdmin = user.admin
    ctx.state.username = user.name
  }
  return next()
})
.use(bodyParser({
  onerror: (err, ctx) => {
    logger.warn('Parsing body: %s', err)
    ctx.body = {
      message: 'invalid json'
    }
    ctx.status = 400
  }
}), (ctx, next) => {
  if (ctx.request.body) {
    ctx.body$ = ctx.request.body
  }
  return next()
})

.use(api.routes(), api.allowedMethods())

export default router.routes()
