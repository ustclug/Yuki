#!/usr/bin/node

'use strict'

import Promise from 'bluebird'
import { Repository as Repo } from './models'
import { EMITTER } from './globals'
import CONFIG from './config'
import docker from './docker'

const imgTag = 'ustcmirror.images'
const PREFIX = CONFIG.get('CT_NAME_PREFIX')

export async function bringUp(cfg) {
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
  postSync(cfg.name)
  return ct
}

function postSync(ctname) {
  return docker.getContainer(ctname).wait()
    .then(async (data) => {
      const name = ctname.substring(PREFIX.length + 1)
      const repo = await Repo.findById(name, { storageDir: 1 })
      EMITTER.emit('sync-end', {
        exitCode: data.StatusCode,
        name: repo._id,
        storageDir: repo.storageDir
      })
    })
}

export function observeRunningContainers() {
  return docker.listContainers({
    filters: {
      label: {
        syncing: true,
        [imgTag]: true,
      },
      status: {
        running: true
      }
    }
  })
    .then((infos) => infos.forEach((info) => postSync(info.Names[0].substr(1))))
}

export function cleanContainers() {
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

export function updateImages() {
  return Repo.distinct('image')
    .then((tags) => tags.map((tag) => docker.pull(tag)))
    .then((ps) => Promise.all(ps))
}

export function cleanImages() {
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
    .then((images) => Promise.all(images.map((info) => {
      return docker.getImage(info.Id).remove()
    })))
}
