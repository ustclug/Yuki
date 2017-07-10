'use strict'

import logger from '../../logger'
import { User } from '../../models'
import { setErrMsg, isLoggedIn, isAdmin } from './lib'

export default function register(router) {

  /**
   * @api {get} /users List Users
   * @apiName ListUsers
   * @apiGroup Users
   *
   * @apiUse AccessToken
   *
   * @apiSuccess {Object[]} users
   *
   * @apiUse CommonErr
   */
  router.get('/users', isLoggedIn, async (ctx) => {
    let users = null
    if (ctx.state.isAdmin) {
      // only hide password
      users = await User.find({}, { password: false })
    } else {
      // only return username
      users = await User.find({}, { name: true })
    }
    return ctx.body = users
  })
  /**
   * @api {put} /users/:name Update User
   * @apiName UpdateUser
   * @apiGroup Users
   *
   * @apiUse AccessToken
   * @apiParam {String} name Name of the User
   *
   * @apiUse CommonErr
   */
    .put('/users/:name', isLoggedIn, ctx => {
      const name = ctx.params.name
      if (!ctx.state.isAdmin) {
        if (name !== ctx.state.username || ctx.$body.admin) {
          setErrMsg(ctx, 'operation not permitted')
          logger.warn(`<${ctx.state.username}> tried to update <${ctx.params.name}>`)
          return ctx.status = 401
        }
      }
      return User.findByIdAndUpdate(name, ctx.$body, {
        runValidators: true
      })
        .then((data) => {
          if (data === null) {
            setErrMsg(ctx, `no such user: ${name}`)
            logger.warn(`<${ctx.state.username}> tried to update <${ctx.params.name}>`)
            return ctx.status = 404
          }
          ctx.status = 204
        })
        .catch(err => {
          logger.error(`Updating user <${name}>: %s`, err)
          ctx.status = 500
          setErrMsg(ctx, err.errmsg)
        })
    })

    .use('/users/:name', isLoggedIn, isAdmin)
  /**
   * @api {get} /users/:name Get User
   * @apiName GetUser
   * @apiGroup Users
   *
   * @apiUse AccessToken
   * @apiPermission admin
   * @apiParam {String} name Name of the User
   *
   * @apiUse CommonErr
   */
    .get(async ctx => {
      const name = ctx.params.name
      const user = await User.findById(name, { password: false })
      if (user === null) {
        ctx.status = 404
        setErrMsg(ctx, `no such user <${name}>`)
      } else {
        ctx.body = user
      }
    })
  /**
   * @api {post} /users/:name Create User
   * @apiName CreateUser
   * @apiGroup Users
   *
   * @apiUse AccessToken
   * @apiPermission admin
   * @apiParam {String} name Name of the User
   *
   * @apiUse CommonErr
   */
    .post(ctx => {
      const body = ctx.$body
      const newUser = {
        name: ctx.params.name,
        password: body.password,
        admin: !!body.admin
      }
      return User.create(newUser)
        .then(() => {
          ctx.status = 201
        }, err => {
          logger.error(`Creating user <${ctx.params.name}>: %s`, err)
          ctx.status = 400
          setErrMsg(ctx, err.message)
        })
    })
  /**
   * @api {delete} /users/:name Delete User
   * @apiName DeleteUser
   * @apiGroup Users
   *
   * @apiUse AccessToken
   * @apiPermission admin
   * @apiParam {String} name Name of the User
   *
   * @apiUse CommonErr
   */
    .delete(ctx => {
      const name = ctx.params.name
      return User.findByIdAndRemove(name)
        .then((user) => {
          if (user !== null) {
            ctx.status = 204
          } else {
            ctx.status = 404
            logger.warn(`<${ctx.state.username}> tried to delete <${ctx.params.name}>`)
            setErrMsg(ctx, `no such user: <${name}>`)
          }
        }, err => {
          logger.error(`Removing user <${ctx.params.name}>: %s`, err)
          ctx.status = 500
          setErrMsg(ctx, err.message)
        })

    })

}
