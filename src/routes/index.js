#!/usr/bin/node

'use strict'

import Router from 'koa-router'
import api from './api'
import { isDev, isTest, TOKEN_NAME } from '../config'
import { User } from '../models'
import logger from '../logger'

const router = new Router()

if (isDev) {
  router.use(async (ctx, next) => {
    await next()
    logger.debug(ctx.request.method, ctx.status, ctx.request.url)
  })
}

router.use(async function auth(ctx, next) {
  const token = ctx.header[TOKEN_NAME] || ''
  const user = await User.findOne({ token })
  if (user === null) {
    ctx.state.isLoggedIn = isTest
  } else {
    ctx.state.isLoggedIn = true
    ctx.state.isAdmin = user.admin
    ctx.state.username = user.name
  }
  return next()
})

.use(api.routes(), api.allowedMethods())

export default router.routes()
