#!/usr/bin/node

'use strict'

import fs from 'fs'
import url from 'url'
import path from 'path'
import program from 'commander'
import { getLocalTime } from '../build/Release/addon.node'
import { API_URL } from './config'
import meta from '../package.json'
import request from './request'

const API = url.resolve(API_URL, 'api/v1')

program
  .version(meta.version)

program
  .command('login <username>')
  .description('log in to remote registry')
  .action((username) => {
    request(`${API}/auth`, { username })
    .then(async (res) => {
      const content = await res.json()
      if (res.ok) {
        console.log('Login succeeded!')
        const fp = path.join(process.env['HOME'], '.ustcmirror', 'config.json')
        let modified = false
        let userCfg
        try {
          userCfg = require(fp)
        } catch (e) {
          if (e.code !== 'ENOENT') {
            throw e
          }
          userCfg = {}
        }
        if (typeof userCfg.auths === 'undefined') {
          userCfg['auths'] =  {
            [API_URL]: {
              auth: content.token
            }
          }
          modified = true
        } else if (typeof userCfg.auths[API_URL] === 'undefined') {
          userCfg.auths[API_URL] = {
            auth: content.token
          }
          modified = true
        } else if (userCfg.auths[API_URL].auth !== content.token) {
          userCfg.auths[API_URL].auth = content.token
          modified = true
        }
        if (modified) {
          return new Promise((ful, rej) => {
            fs.writeFile(fp, JSON.stringify(userCfg, null, 4), err => {
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
    const fp = path.join(process.env['HOME'], '.ustcmirror', 'config.json')
    let userCfg = null
    try {
      userCfg = require(fp)
    } catch (e) {
      if (e.code !== 'ENOENT') {
        throw e
      }
      return
    }
    if (typeof userCfg.auths === 'object' &&
        typeof userCfg.auths[API_URL] === 'object') {
      delete userCfg.auths[API_URL]
    }
    fs.writeFile(fp, JSON.stringify(userCfg, null, 4), err => {
      if (err) {
        return console.error(err)
      }
      console.log(`Remove token for ${API_URL}`)
    })
  })

program
  .command('repos')
  .description('list all repositories')
  .action(() => {
    request(`${API}/repositories`)
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
  .command('containers')
  .description('list all containers')
  .action(() => {
    request(`${API}/containers`)
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
      `${API}/repositories/${repo}/sync?debug=true` :
      `${API}/repositories/${repo}/sync`

    request(url, null, 'POST')
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
      if (options.verbose) {
        res.body.on('end', () => {
          console.log('!!! Please manually remove the container !!!')
        })
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
      `${API}/containers/${repo}/logs?follow=true` :
      `${API}/containers/${repo}/logs`

    request(url)
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('rmct <repo>')
  .description('manually remove container')
  .action((repo) => {
    const url = `${API}/containers/${repo}`

    request(url, null, 'DELETE')
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('rmrepo <repo>')
  .description('manually remove repository')
  .action((repo) => {
    const url = `${API}/repositories/${repo}`

    request(url, null, 'DELETE')
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

program.parse(process.argv)
