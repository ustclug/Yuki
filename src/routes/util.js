#!/usr/bin/node

'use strict'

import docker from './docker'
import Promise from 'bluebird'
import {Transform} from 'stream'

// All Transform streams are also Duplex Streams
function progresBar(stream) {
  return new Transform({
    writableObjectMode: true,

    transform(chunk, _, callback) {
      const data = JSON.parse(chunk.toString());
      stream.write(data.status + ': ')
      if (typeof data.progress === 'string') {
        stream.write(data.progress)
      }
      stream.write('\r')
      callback()
    }
  }).setEncoding('utf8')
}

function pullImage(image) {
  return new Promise((res, rej) => {
    docker.pull(image, (err, stream) => {
      if (err) {
        console.error(`pulling image ${image}: `, err)
        return rej(err)
      }
      // FIXME: One line log
      //stream.pipe(progresBar(process.stdout))
      stream.on('end', res)
    })
  })
}

async function bringUp(config) {
  try {
    const ct = await docker.createContainer(config)
    await ct.start()
      .catch(err => console.error(`startContainer ${config.name}: `, err))
  } catch (err) {
    if (err.statusCode === 404) {
      await pullImage(config.Image).catch(console.error)
      await bringUp(config)
    } else {
      console.error(`createContainer ${config.name}: `, err)
    }
  }
}

export {bringUp}
