#!/usr/bin/node

'use strict'

import fs from 'fs'
import path from 'path'
import Promise from 'bluebird'
import split from 'split'
import moment from 'moment'
import docker from './docker'
import { Repository as Repo, Log, Meta } from './models'
import CONFIG from './config'
import plugins from './plugins'

const PREFIX = CONFIG.CT_NAME_PREFIX
const LABEL = CONFIG.CT_LABEL

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

let storage
switch (CONFIG.STORAGE_OPTS.fs) {
  case 'zfs':
    storage = new plugins.Zfs()
    break;

  case 'fs':
  default:
    storage = new plugins.Fs()
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

async function bringUp(cfg) {
  let ct
  try {
    ct = await docker.createContainer(cfg)
  } catch (err) {
    if (err.statusCode === 404) {
      await docker.pull(cfg.Image)
      return bringUp(cfg)
    } else {
      throw err
    }
  }
  await ct.start()

  const name = cfg.name.substring(PREFIX.length + 1)

  insertLog(name)
  .catch((err) => console.error('%s', err))

  return ct
}

async function queryOpts({ name, debug = false }) {
  const cfg = await Repo.findById(name)
  if (cfg === null) {
    return null
  }
  const logdir = path.join(CONFIG.LOGDIR_ROOT, name)
  const opts = {
    Image: cfg.image,
    Env: [],
    AttachStdin: false,
    AttachStdout: false,
    AttachStderr: false,
    Tty: false,
    OpenStdin: true,
    Labels: {
      [LABEL]: ''
    },
    HostConfig: {
      Binds: [],
      AutoRemove: true,
    },
    name: `${PREFIX}-${name}`,
  }
  opts.Env.push(
    `REPO=${name}`,
    `OWNER=${cfg.user || CONFIG.OWNER}`
  )
  for (const [k, v] of Object.entries(cfg.envs)) {
    opts.Env.push(`${k}=${v}`)
  }
  for (const [k, v] of Object.entries(cfg.volumes)) {
    opts.HostConfig.Binds.push(`${k}:${v}`)
  }
  if (!CONFIG.isTest) {
    opts.HostConfig.Binds.push(`${cfg.storageDir}:/data`, `${logdir}:/log`)
  }
  if (debug) {
    opts.Env.push('DEBUG=true')
  }
  const addr = cfg.bindIp || CONFIG.BIND_ADDRESS
  if (addr) {
    opts.HostConfig.NetworkMode = 'host'
    opts.Env.push(`BIND_ADDRESS=${addr}`)
  }
  if (cfg.logRotCycle) {
    opts.Env.push(`LOG_ROTATE_CYCLE=${cfg.logRotCycle}`)
  }
  return opts
}

function dirExists(path) {
  if (CONFIG.isTest) return true

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
  const time2stamp = (time) => Math.round(time / 1000)
  return {
    name,
    size: stats.size,
    atime: time2stamp(stats.atime.getTime()),
    mtime: time2stamp(stats.mtime.getTime()),
    ctime: time2stamp(stats.ctime.getTime()),
    birthtime: time2stamp(stats.birthtime.getTime())
  }
}

async function insertLog(repo) {
  const cfg = await Repo.findById(repo, { storageDir: 1 })
  return docker.getContainer(`${PREFIX}-${repo}`)
    .wait()
    .then((data) => Log.create({
      name: repo,
      exitCode: data.StatusCode
    }))
    .then((doc) => {
      const meta = {
        size: storage.getSize(cfg.storageDir),
        lastExitCode: doc.exitCode
      }
      if (doc.exitCode === 0) {
        meta.lastSuccess = Date.now()
      }
      return Meta.findByIdAndUpdate(repo, meta, { upsert: true })
    })
}

function createMeta(docs) {
  let data
  if (docs === undefined) {
    data = Repo.find(null, { _id: 1, storageDir: 1 })
      .then(repos => repos.map(r => r.toJSON()))
  } else {
    if (Array.isArray(docs)) {
      data = Promise.resolve(docs)
    } else {
      data = Promise.resolve([docs])
    }
  }
  return data
    .then(data =>
      data.map(doc =>
        Meta.findByIdAndUpdate(doc._id, {
          size: storage.getSize(doc.storageDir)
        }, { upsert: true })
      )
    )
    .then(Promise.all)
}

function cleanContainers() {
  const removing = ['created', 'exited', 'dead']
    .map((status) => {
      return docker.listContainers({
        all: true,
        filters: {
          label: {
            syncing: true,
            'ustcmirror.images': true,
          },
          status: {
            [status]: true
          }
        }
      })
      .then((cts) => {
        return Promise.all(
          cts.map((info) => {
            const ct = docker.getContainer(info.Id)
            return ct.remove({
              v: true,
              force: true
            })
          })
        )
      })
    })
  return Promise.all(removing)
}

function updateImages() {
  return Repo.distinct('image')
    .then(tags => tags.map((tag) => docker.pull(tag)))
    .then(ps => Promise.all(ps))
}

function cleanImages() {
  return docker.listImages({
    filters: {
      label: {
        'ustcmirror.images': true
      },
      dangling: {
        true: true
      }
    }
  })
    .then(images => Promise.all(images.map(info => {
      return docker.getImage(info.Id).remove()
    })))
}

function getLocalTime(date) {
  return moment(date)
    .local()
    .format('YYYY-MM-DD HH:mm:ss')
}

export default {
  bringUp,
  cleanContainers,
  cleanImages,
  dirExists,
  getLocalTime,
  makeDir,
  myStat,
  queryOpts,
  tailStream,
  updateImages,
  createMeta
}
