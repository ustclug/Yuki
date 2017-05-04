#!/usr/bin/node

'use strict'

import Promise from 'bluebird'
import { Repository as Repo, Log, Meta } from './models'
import { CT_NAME_PREFIX as PREFIX } from './config'
import docker from './docker'
import fs from './filesystem'

const imgTag = 'ustcmirror.images'

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
        size: fs.getSize(cfg.storageDir),
        lastExitCode: doc.exitCode
      }
      if (doc.exitCode === 0) {
        meta.lastSuccess = Date.now()
      }
      return Meta.findByIdAndUpdate(repo, meta, { upsert: true })
    })
}

function cleanContainers() {
  const removing = ['created', 'exited', 'dead']
    .map((status) => {
      return docker.listContainers({
        all: true,
        filters: {
          label: {
            syncing: true,
            [imgTag]: true,
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
        [imgTag]: true
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

export default {
  bringUp,
  cleanContainers,
  cleanImages,
  insertLog,
  updateImages,
}
