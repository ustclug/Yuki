#!/usr/bin/node

'use strict'

import fs from 'fs'
import path from 'path'
import Promise from 'bluebird'
import split from 'split'
import { IS_TEST } from './globals'

class Queue {
  constructor(size) {
    this._size = size
    this._buffer = new Array()
  }
  push(...ele) {
    const after = this._buffer.length + ele.length
    if (after > this._size) {
      this.trimLeft(after - this._size)
    }
    return this._buffer.push.apply(this._buffer, ele)
  }
  join(sep) {
    return this._buffer.join(sep)
  }
  trimLeft(cnt) {
    for (; cnt > 0; cnt--) {
      this._buffer.shift()
    }
  }
}

function tailStream(cnt, stream) {
  return new Promise((res, rej) => {
    const q = new Queue(cnt)
    stream.pipe(split(/\r?\n(?=.)/))
      .on('data', q.push.bind(q))
      .on('close', () => res(q.join('\n')))
      .on('error', rej)
  })
}

function dirExists(path) {
  if (IS_TEST) return true

  let stat
  try {
    stat = fs.statSync(path)
  } catch (e) {
    return false
  }
  return stat.isDirectory()
}

function makeDir(path) {
  if (!dirExists(path)) {
    fs.mkdirSync(path)
  }
}

function myStat(dir, name) {
  const stats = fs.statSync(path.join(dir, name))
  return {
    name,
    size: stats.size,
    atime: stats.atime,
    mtime: stats.mtime,
    ctime: stats.ctime,
    birthtime: stats.birthtime
  }
}

export default {
  dirExists,
  makeDir,
  myStat,
  tailStream,
}
