#!/usr/bin/node

'use strict'

import fs from 'fs'
import path from 'path'
import docker from './docker'
import models from './models'
import CONFIG from './config'
import readline from 'readline'
import {Transform} from 'stream'

const Repo = models.Repository
const PREFIX = CONFIG.CT_NAME_PREFIX
const LABEL = CONFIG.CT_LABEL

// All Transform streams are also Duplex Streams
function progresBar(stream) {
  return new Transform({
    writableObjectMode: true,

    transform(chunk, _, callback) {
      const data = JSON.parse(chunk.toString());

      // Sample data:
      // {"status":"Pulling from library/centos","id":"latest"}
      // {"status":"Pulling fs layer","progressDetail":{},"id":"8d30e94188e7"}
      // {"status":"Downloading","progressDetail":{"current":531313,"total":70591526},"progress":"[=> ] 531.3 kB/70.59 MB","id":"8d30e94188e7"} ]"
      // {"status":"Verifying Checksum","progressDetail":{},"id":"8d30e94188e7"}
      // {"status":"Download complete","progressDetail":{},"id":"8d30e94188e7"}
      // {"status":"Extracting","progressDetail":{"current":557056,"total":70591526},"progress":"[=> ] 557.1 kB/70.59 MB","id":"8d30e94188e7"} ]"
      // {"status":"Pull complete","progressDetail":{},"id":"8d30e94188e7"}
      // {"status":"Digest: sha256:2ae0d2c881c7123870114fb9cc7afabd1e31f9888dac8286884f6cf59373ed9b"}
      // {"status":"Status: Downloaded newer image for centos:latest"}
      //

      //stream.write(chunk)
      stream.write('\r')
      readline.clearLine(stream, 1) // clear to the right
      stream.write(data.status)
      if (data.hasOwnProperty('progress')) {
        stream.write(': ' + data.progress)
      }
      callback()
    }
  }).setEncoding('utf8')
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
  opts.Env.push(`REPO=${name}`, `BIND_ADDRESS=${CONFIG.BIND_ADDR}`)
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
