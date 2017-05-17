#!/usr/bin/node

'use strict'

import logger from '../logger'
import { User } from '../models'

import { spawn } from 'node-pty'

export default function bash(socket) {
  socket.on('shell-auth', async (data) => {
    const token = data.token || ''
    const user = await User.findOne({ token }, { admin: 1 })
    if (user === null) {
      socket.emit('shell-message', {
        ok: false,
        msg: 'Invalid token.'
      })
      return socket.disconnect(true)
    }
    if (!user.admin) {
      logger.warn(`shell: unauthorized access: <${user._id}>.`)
      socket.emit('shell-message', {
        ok: false,
        msg: 'Remote shell is only available to administrators.'
      })
      return socket.diconnect(true)
    }
    socket.emit('shell-message', { ok: true })
    const shell = spawn('bash', ['-i'], {
      cols: data.cols || 80,
      rows: data.rows || 24,
      cwd: process.env.HOME || '/',
    });
    shell.on('data', (data) => {
      socket.emit('shell-output', data)
    })
    socket.on('shell-input', (data) => {
      shell.write(data + '\r')
    })
    socket.on('disconnect', () => {
      shell.kill()
    })
  })
}
