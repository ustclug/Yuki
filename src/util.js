#!/usr/bin/node

'use strict'

import fs from 'fs'
import path from 'path'
import docker from './docker'
import models from './models'
import CONFIG from './config'

const Repo = models.Repository
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
    User: cfg.user || CONFIG.OWNER,
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
  opts.Env.push(`REPO=${name}`, `BIND_ADDRESS=${CONFIG.BIND_ADDRESS}`)
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

export default {
  bringUp,
  autoRemove,
  dirExists,
  makeDir,
  queryOpts
}
