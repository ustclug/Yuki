'use strict'

import logger from '../../logger'

export function setErrMsg(ctx, msg) {
  ctx.body = { message: msg }
}

export function isLoggedIn(ctx, next) {
  if (!ctx.state.isLoggedIn) {
    ctx.body = { message: 'unauthenticated access' }
    logger.warn(`Unauthenticated: ${ctx.method} ${ctx.request.url}`)
    return ctx.status = 401
  }
  return next()
}

export function isAdmin(ctx, next) {
  if (!ctx.state.isAdmin) {
    ctx.body = { message: 'Operation not permitted. Please concat administrator.' }
    logger.warn(`Unauthorized: ${ctx.state.username} ${ctx.method} ${ctx.request.url}`)
    return ctx.status = 401
  }
  return next()
}

