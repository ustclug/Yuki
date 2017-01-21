#!/usr/bin/node

'use strict'

import { API_PORT } from './config'
import request from './request'
import meta from '../package.json'
import program from 'commander'

const API = `http://127.0.0.1:${API_PORT}/api/v1`

program
  .version(meta.version)

program
  .command('list')
  .description('list all repositories')
  .action(() => {
    request(`${API}/repositories`)
    .then(res => {
      if (res.ok) {
        return res.json()
      } else {
        res.body.pipe(process.stderr)
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
  .command('sync [repo]')
  .description('sync')
  .option('-v, --verbose', 'debug mode')
  .action((repo, options) => {
    if (typeof repo === 'undefined') {
      return console.error('Need to specify repo')
    }

    const url = (options.verbose) ?
      `${API}/repositories/${repo}/sync?debug=true` :
      `${API}/repositories/${repo}/sync`

    request(url, null, 'POST')
    .then(res => {
      if (res.ok) {
        return res.json()
      } else {
        throw (new Error(`${res.status} - unknown error`))
      }
    })
    .then(console.log)
  })

program
  .command('logs [repo]')
  .description('disp status')
  .option('-f, --follow', 'follow log output')
  .action((repo, options) => {
    if (typeof repo === 'undefined') {
      return console.error('Need to specify repo')
    }

    const url = options.follow ?
      `${API}/containers/${repo}/logs?follow=true` :
      `${API}/containers/${repo}/logs`

    request(url)
    .then(res => {
      res.body.pipe(process.stdout)
    })
  })

program.parse(process.argv)
