import R from 'ramda'
import logger from '../../logger'
import CONFIG from '../../config'
import { IS_TEST } from '../../globals'
import {
  isAdmin, requireAdmin,
  Unauthorized, NotFound, NoContent, Created
} from '../lib'
import { User } from '../../models'

const ldapCfg = CONFIG.get('LDAP')

export default function register(router) {
  if (!IS_TEST && ldapCfg.enabled) {
    return
  }
  router
    .get('/me', (ctx) => {
      const { name } = ctx.state.user
      return User.findById(name, { password: 0 })
        .then((u) => {
          if (u === null) {
            return NotFound(ctx, `No such user: ${name}`)
          }
          ctx.body = u
        })
    })

    .get('/users', isAdmin, async (ctx) => {
      let users = null
      if (ctx.state.isAdmin) {
        // only hide password
        users = await User.find({}, { password: 0 })
      } else {
        // only return username
        users = await User.find({}, { name: 1 })
      }
      return ctx.body = users
    })

    .put('/users/:name', isAdmin, (ctx) => {
      const dstName = ctx.params.name
      const srcName = ctx.state.user.name
      if (!ctx.state.isAdmin) {
        if (dstName !== srcName || ctx.$body.admin) {
          logger.warn(`<${srcName}> tried to update <${dstName}>`)
          return Unauthorized(ctx, `${srcName} is not admin.`)
        }
      }
      return User.findByIdAndUpdate(dstName,
        R.pick(['password', 'admin'], ctx.$body), {
          runValidators: true
        })
        .then((data) => {
          if (data === null) {
            logger.warn(`<${srcName}> tried to update <${dstName}>`)
            return NotFound(ctx, `No such user: ${dstName}`)
          }
          return NoContent(ctx)
        })
    })

    .use('/users/:name', requireAdmin)
    .get('/users/:name', async (ctx) => {
      const dstName = ctx.params.name
      const srcName = ctx.state.user.name
      const user = await User.findById(dstName, { password: 0 })
      if (user === null) {
        logger.warn(`<${srcName}> tried to get <${dstName}>`)
        return NotFound(ctx, `No such user: ${dstName}`)
      } else {
        ctx.body = user
      }
    })
    .post('/users/:name', (ctx) => {
      const body = ctx.$body
      const newUser = {
        name: ctx.params.name,
        password: body.password,
        admin: !!body.admin
      }
      return User.create(newUser)
        .then(Created(ctx))
    })
    .delete('/users/:name', (ctx) => {
      const dstName = ctx.params.name
      const srcName = ctx.state.user.name
      return User.findByIdAndRemove(dstName)
        .then((user) => {
          if (user !== null) {
            return NoContent(ctx)
          } else {
            logger.warn(`<${srcName}> tried to delete <${dstName}>`)
            return NotFound(ctx, `No such user: ${dstName}`)
          }
        })
    })
}
