#!/usr/bin/node

'use strict'

import { isTest, TOKEN_NAME } from '../config'
import { User } from '../models'

export default async function(ctx, next) {
  if (isTest) {
    ctx.state.authorized = true
    return next()
  }
  const token = ctx.header[TOKEN_NAME] || ''
  const user = await User.findOne({ token })
  if (user === null) {
    ctx.state.authorized = false
  } else {
    ctx.state.authorized = true
    ctx.state.isAdmin = user.admin
    ctx.state.username = user.name
  }
  return next()
}
