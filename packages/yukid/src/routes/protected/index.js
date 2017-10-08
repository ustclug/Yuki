import R from 'ramda'
import koarouter from 'koa-router'
import httpstatus from 'http-status'
import {
  JWTMiddleware,
  Unauthorized, ServerError, NoContent
} from '../lib'
import { TOKEN_NAME } from '../../globals'
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

  .post('/reload', (ctx) => {
    return scheduler.schedRepos()
      .then(NoContent(ctx))
      .catch((err) => ServerError(ctx, `Reload config: ${err.message}`))
  })

  .delete('/token', (ctx) => {
    ctx.cookies.set(TOKEN_NAME, '', {
      path: '/api/v1/',
      httpOnly: true
    })
    return NoContent(ctx)
  })

R.juxt([ctRoutes, repoRoutes, cfgRoutes, userRoutes])(router)

export default router
