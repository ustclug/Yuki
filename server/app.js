#!/usr/bin/node

'use strict'

import Koa from 'koa'
import routes from './routes'
import serve from 'koa-static'

const app = new Koa()

app.use(routes)
app.on('error', (err) => {
  console.error('Uncaught error: ', err)
})

module.exports = app
