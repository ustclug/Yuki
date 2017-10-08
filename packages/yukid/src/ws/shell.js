import jwt from 'jsonwebtoken'
import { spawn } from 'node-pty'

import logger from '../logger'
import { User } from '../models'
import CONFIG from '../config'

const secret = CONFIG.get('JWT_SECRET')

export default function bash(socket) {
  socket.on('shell-auth', async (data) => {
    const token = data.token || ''
    let decoded
    try {
      decoded = jwt.verify(token, secret)
    } catch (e) {
      socket.emit('shell-message', {
        ok: false,
        msg: 'Invalid token.'
      })
      return socket.disconnect(true)
    }
    const user = await User.findById(decoded.name)
    if (user === null) {
      socket.emit('shell-message', {
        ok: false,
        msg: 'Cannot find the user.'
      })
      return socket.disconnect(true)
    }
    if (!user.admin) {
      logger.warn(`Shell: unauthorized access from: <${user._id}>.`)
      socket.emit('shell-message', {
        ok: false,
        msg: 'Remote shell is only available to administrators.'
      })
      return socket.disconnect(true)
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
