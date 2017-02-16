#!/usr/bin/node

'use strict'

import Router from 'koa-router'

const router = new Router({ prefix: '/home' })

router.get('/', (ctx) => {
  ctx.body = 'hello world!'
})

export default router
