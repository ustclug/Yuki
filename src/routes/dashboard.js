#!/usr/bin/node

'use strict'

import Router from 'koa-router'

const router = new Router({ prefix: '/dashboard' })

router.get('/', (ctx) => {
  ctx.body = 'hello world!'
})

export default router
