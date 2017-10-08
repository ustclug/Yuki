import R from 'ramda'
import koarouter from 'koa-router'
import httpstatus from 'http-status'
import {
  JWTMiddleware,
  requireAdmin,
  Unauthorized, ServerError, NoContent
} from '../lib'

import scheduler from '../../scheduler'

import ctRoutes from './ct'
import repoRoutes from './repo'
import cfgRoutes from './config'
import userRoutes from './user'

const router = new koarouter()

router
  .use((ctx, next) => {
    return next().catch((err) => {
      if (err.status === httpstatus.UNAUTHORIZED) {
        return Unauthorized(ctx, err.originalError ? err.originalError.message : err.message)
      } else {
        throw err
      }
    })
  }, JWTMiddleware)

  .post('/reload', requireAdmin, (ctx) => {
    return scheduler.schedRepos()
      .then(NoContent(ctx))
      .catch((err) => ServerError(ctx, `Reload config: ${err.message}`))
  })

R.juxt([ctRoutes, repoRoutes, cfgRoutes, userRoutes])(router)

export default router
