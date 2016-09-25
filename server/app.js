#!/usr/bin/node

'use strict'

import Koa from 'koa'
import routes from './routes'

const app = new Koa()

app.use(routes)
app.on('error', (err) => {
  console.log(err)
})

module.exports = app
