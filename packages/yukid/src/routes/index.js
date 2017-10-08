import koarouter from 'koa-router'
import koabody from 'koa-body'
import httpstatus from 'http-status'

import publicRouter from './public'
import privateRouter from './protected'
import { IS_DEV } from '../globals'
import logger from '../logger'

const router = new koarouter()

if (IS_DEV) {
  router.use(async (ctx, next) => {
    await next()
    const { name } = ctx.state.user || { name: '$public' }
    logger.debug(`(${name})`, ctx.request.method, ctx.status, ctx.request.url)
  })
}

router
  .use((ctx, next) => {
    return next().catch((e) => {
      ctx.status = e.status || httpstatus.INTERNAL_SERVER_ERROR
      ctx.body = {
        message: e.message
      }
      logger.warn('Unexpected Error: %s', e.message)
    })
  }, koabody({
    onError: (err, ctx) => {
      ctx.throw(httpstatus.BAD_REQUEST, `Invalid JSON: ${err.message}`)
    }
  }), (ctx, next) => {
    if (ctx.request.body) {
      ctx.$body = ctx.request.body
    }
    return next()
  })

  .use(publicRouter.routes(), publicRouter.allowedMethods())
  .use(privateRouter.routes(), privateRouter.allowedMethods())

export default router.routes()
