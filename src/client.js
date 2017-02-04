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

(function(fp) {
  const d = path.dirname(fp)
  try {
    fs.statSync(d)
    return
  } catch (e) {
    fs.mkdirSync(d)
  }
})(AUTH_RECORD)

url.join = function(...eles) {
  return eles.reduce((sum, ele) => url.resolve(sum, ele), '')
}

const req = function(apiroot, path, data, method) {
  if (!apiroot.startsWith('http')) apiroot = `http://${apiroot}`
  if (!apiroot.endsWith('/')) apiroot += '/'

  const api = url.join(apiroot, 'api/v1/', path)
  return request(api, data, method, {
    headers: { [TOKEN_NAME]: auths[apiroot] || '' }
  })
}

const md5hash = function(text) {
  return createHash('md5').update(text).digest('hex')
}

program
  .version(meta.version)
  .option('--apiroot <url>', `base url of remote registry (default ${API_ROOT})`, API_ROOT)

program
  .command('login <username> <password>')
  .description('log in to remote registry')
  .action((username, password, opts) => {
    password = md5hash(password)
    req(opts.parent.apiroot, 'auth', { username, password })
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
  .command('whoami')
  .description('print current user')
  .action((opts) => {
    req(opts.parent.apiroot, 'auth')
    .then(async (res) => {
      const data = await res.json()
      if (res.ok) {
        console.log(`name: ${data.name}`)
        console.log(`admin: ${data.admin}`)
        console.log(`registry: ${API_ROOT}`)
      } else {
        console.error(data.message)
      }
    })
    .catch(console.error)
  })

program
  .command('user-add')
  .option('-n --name <name>', 'username')
  .option('-p --pass <password>', 'password')
  .option('-r --role <role>', 'role of user [admin,normal]', /^(admin|normal)$/i, 'normal')
  .description('create user')
  .action((opts) => {
    if (!opts.name || !opts.pass) {
      console.error('Please tell me the username and password')
      return
    }
    req(opts.parent.apiroot, `users/${opts.name}`, {
      password: md5hash(opts.pass),
      admin: opts.role === 'admin'
    })
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('user-list [name]')
  .description('list user(s)')
  .action((name, opts) => {
    const u = name ? `users/${name}` : 'users'
    req(opts.parent.apiroot, u)
    .then(async (res) => {
      const data = await res.json()
      const output = (user) => {
        const token = user.token === undefined ? 'secret' : user.token
        const admin = user.admin === undefined ? 'secret' : user.admin
        console.log(`${user.name}:`)
        console.log(`\tToken: ${token}`)
        console.log(`\tAdministrator: ${admin}`)
      }
      if (res.ok) {
        Array.isArray(data) ? data.forEach(output) : output(data)
      } else {
        console.error(data.message)
      }
    })
    .catch(console.error)
  })

program
  .command('user-update <name>')
  .option('-p --pass <password>', 'password')
  .option('-r --role <role>', 'role of user [admin,normal]', /^(admin|normal)$/i)
  .description('update user profile')
  .action((name, opts) => {
    if (typeof opts.role === 'undefined' &&
        typeof opts.pass === 'undefined') {
      return console.error('Nothing changes')
    }
    const payload = {}
    if (opts.role) {
      payload.admin = opts.role === 'admin'
    }
    if (opts.pass) {
      payload.password = md5hash(opts.pass)
    }
    req(opts.parent.apiroot, `users/${name}`, payload, 'PUT')
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('user-rm <name>')
  .description('remove user')
  .action((name, opts) => {
    req(opts.parent.apiroot, `users/${name}`, null, 'DELETE')
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('repo-list [repo]')
  .description('list repository(s)')
  .action((repo, opts) => {
    const u = repo ? `repositories/${repo}` : 'repositories'
    req(opts.parent.apiroot, u)
    .then(async res => {
      if (res.ok) {
        const data = await res.json()

        let repos = null
        if (!Array.isArray(data)) repos = [data]
        else repos = data

        for (const repo of repos) {
          console.log(`${repo.name}:`)
          console.log(`\timage: ${repo.image}`)
          console.log(`\tinterval: ${repo.interval}`)
        }

      } else {
        res.body.pipe(process.stderr)
      }
    })
    .catch(console.error)
  })

program
  .command('repo-rm <repo>')
  .description('manually remove repository')
  .action((repo, opts) => {
    req(opts.parent.apiroot, `repositories/${repo}`, null, 'DELETE')
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('ct-list')
  .description('list all containers')
  .action((opts) => {
    req(opts.parent.apiroot, 'containers')
    .then(async (res) => {
      if (res.ok) {
        const data = await res.json()
        for (const ct of data) {
          console.log(`${ct.Names[0]}:`)
          console.log(`\tCreated: ${getLocalTime(ct.Created)}`)
          console.log(`\tState: ${ct.State}`)
          console.log(`\tStatus: ${ct.Status}`)
        }
      } else {
        res.body.pipe(process.stderr)
      }
    })
    .catch(console.error)
  })

program
  .command('ct-rm <repo>')
  .description('manually remove container')
  .action((repo, opts) => {
    req(opts.parent.apiroot, `containers/${repo}`, null, 'DELETE')
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('ct-logs <repo>')
  .description('capture container logs')
  .option('-f, --follow', 'follow log output')
  .option('--tail <num>', 'specify lines of logs at the end', /^(all|\d+)$/, 'all')
  .action((repo, opts) => {
    const tail = opts.tail
    let url = `containers/${repo}/logs?tail=${tail}`
    if (opts.follow) url += '&follow=true'

    req(opts.parent.apiroot, url)
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('images-update')
  .description('update ustcmirror images')
  .action((opts) => {
    req(opts.parent.apiroot, 'images/update', null, 'POST')
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('sync <repo>')
  .description('sync')
  .option('-v, --verbose', 'debug mode')
  .action((repo, opts) => {
    const url = (opts.verbose) ?
      `repositories/${repo}/sync?debug=true` :
      `repositories/${repo}/sync`

    req(opts.parent.apiroot, url, null, 'POST')
      .then(async res => {
        if (res.ok) {
          res.body.pipe(process.stdout)
          if (opts.verbose) {
            res.body.on('end', () => {
              console.log('!!! Please manually remove the container !!!')
            })
          }
        } else {
          const data = await res.json()
          console.error(data.message)
        }
      })
      .catch(console.error)
  })

program
  .command('export [file]')
  .description('export configuration')
  .action((file, opts) => {
    req(opts.parent.apiroot, 'config')
    .then(async (res) => {
      if (res.ok) {
        file = path.resolve(process.cwd(), file ? file : 'repositories.json')
        const fout = fs.createWriteStream(file)
        res.body.pipe(fout)
      } else {
        res.body.pipe(process.stderr)
      }
    })
    .catch(console.error)
  })

program
  .command('import <file>')
  .description('import configuration')
  .action((file, opts) => {
    file = path.resolve(file)
    const data = require(file)
    req(opts.parent.apiroot, 'config', data)
    .then(res => {
      res.body.pipe(res.ok ? process.stdout : process.stderr)
    })
    .catch(console.error)
  })

program
  .command('*')
  .action(() => program.outputHelp())

program.parse(process.argv)

if (process.argv.length === 2) {
  program.outputHelp()
}
