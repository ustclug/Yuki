import koarouter from 'koa-router'
import R from 'ramda'

import logger from '../../logger'
import { User } from '../../models'
import { jwtsign, NotFound } from '../lib'
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
    const user = await User.findOne({ _id: name, password })

    if (user === null) {
      return NotFound(ctx, 'Wrong name or password')
    }

    const token = jwtsign({ name }, { noTimestamp: true })
    logger.info(`${name} login`)

    ctx.cookies.set(TOKEN_NAME, token, {
      path: '/api/v1/',
      httpOnly: true
    })
    ctx.body = { token }
  })

export default router
