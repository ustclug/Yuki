import httpstatus from 'http-status'
import jwt from 'jsonwebtoken'
import koajwt from 'koa-jwt'
import CONFIG from '../config'
import { User } from '../models'
import { IS_TEST, TOKEN_NAME } from '../globals'

export function InvalidParams(ctx, msg) {
  ctx.throw(httpstatus.UNPROCESSABLE_ENTITY, msg)
}

export function NotFound(ctx, msg) {
  ctx.throw(httpstatus.NOT_FOUND, msg)
}

export function Created(ctx) {
  ctx.status = httpstatus.CREATED
}

export function NoContent(ctx) {
  ctx.status = httpstatus.NO_CONTENT
}

export function Unauthorized(ctx, msg) {
  ctx.throw(httpstatus.UNAUTHORIZED, msg)
}

export function ServerError(ctx, msg) {
  ctx.throw(httpstatus.INTERNAL_SERVER_ERROR, msg)
}

const secret = IS_TEST ? 'iamasecret' : CONFIG.get('JWT_SECRET')

export const JWTMiddleware = koajwt({
  secret,
  cookie: TOKEN_NAME,
  debug: IS_TEST
})

export function jwtsign(obj, opts) {
  return jwt.sign(obj, secret, opts)
}

function checkAdmin({ force = false }) {
  return async (ctx, next) => {
    const data = ctx.state.user
    if (!data) {
      return ServerError(ctx, 'Missing JWT')
    }
    const { name } = data
    const user = await User.findById(name)
    if (user === null) {
      return NotFound(ctx, `No such user: ${name}`)
    }
    ctx.state.isAdmin = user.admin
    if (force && !user.admin) {
      return Unauthorized(ctx, 'Operation not permitted. Please concat administrator.')
    }
    return next()
  }
}

export const requireAdmin = checkAdmin({ force: true })
export const isAdmin = checkAdmin({ force: false })
