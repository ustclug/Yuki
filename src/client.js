#!/usr/bin/node

'use strict'

import fs from 'fs'
import url from 'url'
import path from 'path'
import { createHash } from 'crypto'
import program from 'commander'
import { getLocalTime } from '../build/Release/addon.node'
import { API_ROOT, TOKEN_NAME } from './config'
import meta from '../package.json'
import request from './request'

const AUTH_RECORD = path.join(process.env['HOME'], '.ustcmirror', 'auth.json')
let auths
try {
  auths = require(AUTH_RECORD)
} catch (e) {
  if (e.code !== 'MODULE_NOT_FOUND') {
    throw e
  }
  auths = {}
}

url.join = function(...eles) {
  return eles.reduce((sum, ele) => url.resolve(sum, ele), '')
}

const req = function(path, data, method, apiroot = API_ROOT) {
  const api = url.join(apiroot, 'api/v1/', path)
  return request(api, data, method, {
    headers: { [TOKEN_NAME]: auths[apiroot] || '' }
  })
}

program
  .version(meta.version)

program
  .command('login <username> <password>')
  .description('log in to remote registry')
  .action((username, password) => {
    password = createHash('md5').update(password).digest('hex')
    req('auth', { username, password })
    .then(async (res) => {
      const content = await res.json()
      if (res.ok) {
        console.log('Login succeeded!')

        if (typeof auths[API_ROOT] === 'undefined' ||
            auths[API_ROOT] !== content.token)
        {
          auths[API_ROOT] = content.token
          return new Promise((ful, rej) => {
            fs.writeFile(AUTH_RECORD, JSON.stringify(auths, null, 4), err => {
              if (err) return rej(err)
              ful()
            })
          })
        }
      } else {
        console.log(`Failed to login: ${content.message}`)
      }
    })
    .catch(console.error)
  })

program
  .command('logout')
  .description('log out from remote registry')
  .action(() => {
    if (typeof auths[API_ROOT] === 'string') {
      delete auths[API_ROOT]
      fs.writeFile(AUTH_RECORD, JSON.stringify(auths, null, 4), err => {
        if (err) {
          return console.error(err)
        }
        console.log(`Remove token for ${API_ROOT}`)
      })
    } else {
      console.error(`Not logged in to ${API_ROOT}`)
    }
  })

program
  .command('rmuser <name>')
  .description('remove user')
  .action((name, options) => {
    req(`users/${name}`, null, 'DELETE')
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
  })

program
  .command('repos')
  .description('list all repositories')
  .action(() => {
    req('repositories')
    .then(res => {
      if (res.ok) {
        return res.json()
      } else {
        res.body.pipe(process.stderr)
        return []
      }
    })
    .then(repos => {
      for (const repo of repos) {
        console.log(`${repo.name}:`)
        console.log(`\timage: ${repo.image}`)
        console.log(`\tinterval: ${repo.interval}`)
      }
    })
    .catch(console.error)
  })

program
  .command('rmrepo <repo>')
  .description('manually remove repository')
  .action((repo) => {
    req(`repositories/${repo}`, null, 'DELETE')
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('containers')
  .description('list all containers')
  .action(() => {
    req('containers')
    .then(res => {
      if (res.ok) {
        return res.json()
      } else {
        res.body.pipe(process.stderr)
        return []
      }
    })
    .then(cts => {
      for (const ct of cts) {
        console.log(`${ct.Names[0]}:`)
        console.log(`\tCreated: ${getLocalTime(ct.Created)}`)
        console.log(`\tState: ${ct.State}`)
        console.log(`\tStatus: ${ct.Status}`)
      }
    })
    .catch(console.error)
  })

program
  .command('sync <repo>')
  .description('sync')
  .option('-v, --verbose', 'debug mode')
  .action((repo, options) => {
    const url = (options.verbose) ?
      `repositories/${repo}/sync?debug=true` :
      `repositories/${repo}/sync`

    req(url, null, 'POST')
    .then(async res => {
      if (res.ok) {
        res.body.pipe(process.stdout)
        res.body.on('end', () => {
          console.log('!!! Please manually remove the container !!!')
        })
      } else {
        const data = await res.json()
        console.error(data)
      }
    })
    .catch(console.error)
  })

program
  .command('logs <repo>')
  .description('capture container logs')
  .option('-f, --follow', 'follow log output')
  .action((repo, options) => {
    const url = options.follow ?
      `containers/${repo}/logs?follow=true` :
      `containers/${repo}/logs`

    req(url)
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('rmct <repo>')
  .description('manually remove container')
  .action((repo) => {
    req(`containers/${repo}`, null, 'DELETE')
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('export [file]')
  .description('export configuration')

program
  .command('import <file>')
  .description('import configuration')

program
  .command('*')
  .action(() => program.outputHelp())

program.parse(process.argv)
