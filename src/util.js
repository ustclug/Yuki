#!/usr/bin/node

'use strict'

import fs from 'fs'
import path from 'path'
import docker from './docker'
import { Repository as Repo } from './models'
import CONFIG from './config'

const PREFIX = CONFIG.CT_NAME_PREFIX
const LABEL = CONFIG.CT_LABEL

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
  if (!CONFIG.isTest) {
    cfg.volumes.push(`${cfg.storageDir}:/data`, `${logdir}:/log`)
  }
  const opts = {
    Image: cfg.image,
    Cmd: cfg.command,
    Env: cfg.envs,
    AttachStdin: false,
    AttachStdout: false,
    AttachStderr: false,
    Tty: false,
    OpenStdin: true,
    Labels: {
      [LABEL]: ''
    },
    HostConfig: {
      Binds: cfg.volumes,
      NetworkMode: 'host',
      RestartPolicy: {
        Name: 'on-failure',
        MaximumRetryCount: 2
      },
    },
    name: `${PREFIX}-${name}`,
  }
  opts.Env.push(
    `REPO=${name}`,
    `BIND_ADDRESS=${CONFIG.BIND_ADDRESS}`,
    `OWNER=${cfg.user || CONFIG.OWNER}`
  )
  if (debug) {
    opts.Env.push('DEBUG=true')
  }
  if (cfg.autoRotLog) {
    opts.Env.push('AUTO_ROTATE_LOG=true', `ROTATE_CYCLE=${cfg.rotateCycle}`)
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

function cleanContainers(status = {running: true}) {
  return docker.listContainers({
    all: true,
    filters: {
      label: {
        syncing: true,
        'ustcmirror.images': true,
      },
      status
    }
  })
  .then(cts => {
    cts.forEach(info => {
      const ct = docker.getContainer(info.Id)
      autoRemove(ct).catch(console.error)
    })
  })
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
  queryOpts
}
