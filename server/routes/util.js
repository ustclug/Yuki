#!/usr/bin/node

'use strict'

import docker from './docker'
import Promise from 'bluebird'
import {Transform} from 'stream'

const createContainer = Promise.promisify(docker.createContainer, { context: docker })

// All Transform streams are also Duplex Streams
const progresBar = new Transform({
  writableObjectMode: true,

  transform(chunk, encoding, callback) {
    // Transform the chunk into something else.
    const data = JSON.parse(chunk.toString());

    // Push the data onto the readable queue.
    process.stdout.clearLine()

    process.stdout.write(data.status + ': ')
    if (typeof data.progress === 'string') {
      process.stdout.write(data.progress)
    }
    process.stdout.write('\r')
    callback()
  }
})
progresBar.setEncoding('utf8')

function pullImage(image) {
  return new Promise((res, rej) => {
    docker.pull(image, (err, stream) => {
      if (err) {
        console.log(`pulling image ${image}: `, err)
        return rej(err)
      }
      stream.pipe(progresBar)
      stream.on('end', res)
    })
  })
}

async function bringUp(config) {
  try {
    const ct = await createContainer(config)
    await Promise.promisify(ct.start, { context: ct })()
      .catch(err => console.error(`startContainer ${config.name}: `, err))
  } catch (err) {
    if (err.statusCode === 404) {
      await pullImage(config.Image)
      await bringUp(config)
    } else {
      console.error(`createContainer ${config.name}: `, err)
    }
  }
}

export {bringUp, progresBar}

//bringUp({
  //Image: 'debian:latest',
  //name: 'wtf',
  //Cmd: ['sleep', '5'],
//})


