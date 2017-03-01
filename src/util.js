#!/usr/bin/node

'use strict'

import fs from 'fs'
import path from 'path'
import Promise from 'bluebird'
import docker from './docker'
import { Repository as Repo } from './models'
import CONFIG from './config'

const PREFIX = CONFIG.CT_NAME_PREFIX
const LABEL = CONFIG.CT_LABEL

if (typeof Object.entries !== 'function') {
  Object.entries = function entries(obj) {
    var entrys = []
    for (var key in obj) {
      if (obj.hasOwnProperty(key)) {
        entrys.push([key, obj[key]])
      }
    }
    return entrys
  }
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
    Cmd: cfg.cmd,
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
      RestartPolicy: {
        Name: 'on-failure',
        MaximumRetryCount: 2
      },
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

function autoRemove(ct) {
  return ct.wait()
  // FIXME
  // res: {
  // "StatusCode": 0
  // }
    .then((res) => ct.remove({ v: true, force: true }))
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

function cleanContainers() {
  return docker.listContainers({
    all: true,
    filters: {
      label: {
        syncing: true,
        'ustcmirror.images': true,
      }
    }
  })
  .then(cts => {
    cts.forEach(info => {
      const ct = docker.getContainer(info.Id)
      autoRemove(ct).catch(console.error)
    })
  })
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
  .then(images => {
    images.forEach(info => {
      docker.getImage(info.Id).remove().catch(console.error)
    })
  })
}

export default {
  autoRemove,
  bringUp,
  cleanContainers,
  cleanImages,
  dirExists,
  makeDir,
  myStat,
  queryOpts,
  updateImages
}
