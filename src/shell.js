#!/usr/bin/node

'use strict'

const pty = require('node-pty')
const TTY = process.binding('tty_wrap').TTY

export default function bash(socket) {
  const shell = pty.spawn('bash', ['-ri'], {
    cwd: process.env.HOME || '/',
  })
  const t = new TTY(shell.fd, true)
  t.setRawMode(true)
  shell.on('data', (data) => {
    socket.emit('shell-output', data)
  })
  socket.on('shell-input', (data) => {
    shell.write(data + '\r')
  })
  socket.on('disconnect', () => {
    shell.kill()
  })
}
