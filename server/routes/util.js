#!/usr/bin/node

'use strict'

import docker from './docker'
import logger from '../logger'
import readline from 'readline'
import {Transform} from 'stream'

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

async function bringUp(config) {
  let ct
  try {
    ct = await docker.createContainer(config)
  } catch (err) {
    if (err.statusCode === 404) {
      await docker.pull(config.Image)
      await bringUp(config)
    } else {
      throw err
    }
  }
  await ct.start()
  return ct
}

export {bringUp}
