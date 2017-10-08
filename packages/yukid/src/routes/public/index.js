import koarouter from 'koa-router'
import R from 'ramda'

import { verify } from '../../authenticator'
import logger from '../../logger'
import { jwtsign, NotFound, Created } from '../lib'
import { TOKEN_NAME } from '../../globals'

import container from './ct'
import meta from './meta'
import repo from './repo'

const router = new koarouter()

R.juxt([container, meta, repo])(router)

router
  .post('/token', async (ctx) => {
    const { auth } = ctx.$body
    const decoded = new Buffer(auth, 'base64').toString('utf8')
    const [ name, password ] = decoded.split(':')
    try {
      await verify(name, password)
    } catch (e) {
      return NotFound(ctx, 'Wrong name or password')
    }

    const token = jwtsign({ name })
    logger.info(`${name} login`)

    if (ctx.query.cookie === '1') {
      ctx.cookies.set(TOKEN_NAME, token, {
        path: '/api/v1/',
        httpOnly: true
      })
      ctx.body = ''
    } else {
      ctx.body = { token }
    }
    return Created(ctx)
  })

export default router
