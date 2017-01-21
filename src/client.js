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
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('logs [repo]')
  .description('capture container logs')
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
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('rmct [repo]')
  .description('manually remove container')
  .action((repo) => {
    if (typeof repo === 'undefined') {
      return console.error('Need to specify repo')
    }

    const url = `${API}/containers/${repo}`

    request(url, null, 'DELETE')
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('rmrepo [repo]')
  .description('manually remove repository')
  .action((repo) => {
    if (typeof repo === 'undefined') {
      return console.error('Need to specify repo')
    }

    const url = `${API}/repositories/${repo}`

    request(url, null, 'DELETE')
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program.parse(process.argv)
