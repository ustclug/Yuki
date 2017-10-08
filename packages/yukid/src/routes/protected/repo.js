import fs from 'fs'
import R from 'ramda'
import path from 'path'
import httpstatus from 'http-status'
import Promise from 'bluebird'
import { createGunzip } from 'zlib'
import { ServerError, NoContent, NotFound, InvalidParams } from '../lib'
import scheduler from '../../scheduler'
import CONFIG from '../../config'
import { Meta, Repository as Repo } from '../../models'
import { setBody, myStat, dirExists, tailStream } from '../../util'

export default function register(router) {

  const readDir = Promise.promisify(fs.readdir)

  router
    .get('/repositories/:name', (ctx) => {
      const name = ctx.params.name
      return Repo.find({ _id: { $regex: name } })
        .then((data) => {
          if (data.length === 0) {
            return NotFound(ctx, `No such repository: ${ctx.params.name}`)
          }
          data = data.map((d) => {
            d = d.toJSON()
            d.scheduled = scheduler.isScheduled(d._id)
            return d
          })
          const matched = R.find(R.propEq('_id', name))(data)
          if (matched) {
            return ctx.body = [matched]
          }
          ctx.body = R.ifElse(
            R.compose(R.equals(1), R.length),
            R.identity,
            R.map(R.pick(['_id', 'image', 'interval', 'scheduled']))
          )(data)
        })
    })
    .post('/repositories/:name', (ctx) => {
      const body = ctx.$body
      body.name = ctx.params.name
      if (!dirExists(body.storageDir)) {
        return NotFound(ctx, `Creating ${body.name}: No such directory: ${body.storageDir}`)
      }
      return Repo.create(body)
        .then((repo) => {
          ctx.status = httpstatus.CREATED
          scheduler.addJob(repo._id, repo.interval)
        })
        .catch((err) => {
          return ServerError(ctx, `Creating ${body.name}: %s`, err.message)
        })
    })
    .put('/repositories/:name', (ctx) => {
      const name = ctx.params.name
      return Repo.findByIdAndUpdate(name, ctx.$body, {
        // return the modified doc
        new: true,
        runValidators: true
      })
        .then((repo) => {
          // would automatically cancel the existing job
          scheduler.addJob(repo._id, repo.interval)
          return NoContent(ctx)
        })
        .catch((err) => {
          return ServerError(ctx, `Updating ${name}: %s`, err.message)
        })
    })
    .delete('/repositories/:name', (ctx) => {
      const name = ctx.params.name
      return Promise.all([
        Repo.findByIdAndRemove(name),
        Meta.findByIdAndRemove(name),
      ])
        .then(() => {
          scheduler.cancelJob(name)
          return NoContent(ctx)
        })
        .catch((err) => {
          return ServerError(ctx, `Deleting ${name}: %s`, err.message)
        })
    })

    .get('/repositories/:name/logs', (ctx) => {
      const repo = ctx.params.name
      const nth = ctx.query.n || 0
      const stats = !!ctx.query.stats
      const tail = ctx.query.tail || 'all'
      const logdir = path.join(CONFIG.get('LOGDIR_ROOT'), repo)
      if (!dirExists(logdir)) {
        return NotFound(ctx, `No such directory: ${logdir}`)
      }

      if (!/^(all|\d+)$/.test(tail)) {
        return InvalidParams(ctx, `Invalid argument: tail: ${tail}`)
      }

      if (stats) {
        return readDir(logdir)
          .then((files) => {
            ctx.body = files.filter((f) => f.startsWith('result.log.'))
              .reduce((acc, f) => {
                try {
                  acc.push(myStat(logdir, f))
                } finally {
                  // eslint-disable-next-line no-unsafe-finally
                  return acc
                }
              }, [])
              .sort((x, y) => x.mtime - y.mtime)
          })
      }

      return readDir(logdir)
        .then((files) => {
          const wantedName = `result.log.${nth}`
          for (const f of files) {
            if (f.startsWith(wantedName)) {
              if (+tail === 0) {
                return ctx.body = ''
              }
              const fp = path.join(logdir, f)
              let content = null
              switch (path.extname(f)) {
                case '.gz':
                  content = fs.createReadStream(fp).pipe(createGunzip())
                  break
                default:
                  content = fs.createReadStream(fp)
                  break
              }
              if (tail === 'all') {
                return ctx.body = content
              }
              return tailStream(+tail, content)
                .then(setBody(ctx))
            }
          }
          return NotFound(ctx, `No such file: ${path.join(repo, wantedName)}`)
        })
    })
}
