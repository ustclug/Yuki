'use strict'

import fs from 'fs'
import path from 'path'
import stream from 'stream'
import Promise from 'bluebird'
import { createGunzip } from 'zlib'
import CONFIG from '../../config'
import logger from '../../logger'
import scheduler from '../../scheduler'
import { Repository as Repo } from '../../models'
import { setErrMsg, isLoggedIn } from './lib'
import { dirExists, makeDir, myStat, tailStream } from '../../util'
import { bringUp } from '../../containers'
import { queryOpts } from '../../repositories'

export default function register(router) {

  const readDir = Promise.promisify(fs.readdir)

  /**
   * @api {get} /repositories List repositories
   * @apiName ListRepositories
   * @apiGroup Repositories
   *
   * @apiParam {String} type Basename of the Docker image
   *
   * @apiSuccess {Object[]} repos(virtual field) List of repositories
   * @apiSuccess {String} .interval Task interval
   * @apiSuccess {String} .image Name of the Docker image
   * @apiSuccess {String} .storageDir Path to storage directory
   * @apiSuccess {Boolean} .autoRotLog Whether rotates log automatically
   * @apiSuccess {Number} .rotateCycle Number of the cycle versions to save
   * @apiSuccess {String[]} .args Arguments passed to image
   * @apiSuccess {String[]} .envs Environment variables
   * @apiSuccess {String[]} .volumes Volumes to be mount
   * @apiSuccess {String} .bindIp Local ip to be bound
   * @apiSuccess {String} .user Owner of the storage directory
   *
   * @apiUse CommonErr
   */
  router.get('/repositories', (ctx) => {
    const type = ctx.query.type
    const query = type ? {
      'image': {
        '$regex': `ustcmirror/${type}`
      }
    } : null
    return Repo.find(query, { image: 1, interval: 1 })
      .sort({ _id: 1 })
      .exec()
      .then(data => data.map(r => {
        r = r.toJSON()
        r.scheduled = scheduler.isScheduled(r._id)
        return r
      }))
      .then(data => ctx.body = data)
  })

    .use('/repositories/:name')
  /**
   * @api {get} /repositories/:name Get Repository
   * @apiName GetRepository
   * @apiGroup Repositories
   *
   * @apiParam {String} name Name of the Repository
   *
   * @apiSuccess {String} interval Task interval
   * @apiSuccess {String} image Name of the Docker image
   * @apiSuccess {String} storageDir Path to storage directory
   * @apiSuccess {Boolean} autoRotLog Whether rotates log automatically
   * @apiSuccess {Number} rotateCycle Number of the cycle versions to save
   * @apiSuccess {String[]} args Arguments passed to image
   * @apiSuccess {String[]} envs Environment variables
   * @apiSuccess {String[]} volumes Volumes to be mount
   * @apiSuccess {String} bindIp Local ip to be bound
   * @apiSuccess {String} user Owner of the storage directory
   *
   * @apiUse CommonErr
   */
    .get((ctx) => {
      const name = ctx.params.name
      const proj = ctx.state.isLoggedIn ? null : {
        interval: 1, image: 1
      }
      return Repo.find({ _id: { $regex: name } }, proj)
        .then((data) => {
          if (data.length === 0) {
            setErrMsg(ctx, `no such repository: ${ctx.params.name}`)
            return ctx.status = 404
          }
          data = data.map((d) => {
            d = d.toJSON()
            d.scheduled = scheduler.isScheduled(d._id)
            return d
          })
          const matched = data.find((e) => e._id === name)
          if (matched) {
            return ctx.body = [matched]
          }
          ctx.body = data.map((d, _, arr) => {
            if (arr.length > 1) {
              return {
                _id: d._id,
                image: d.image,
                interval: d.interval,
                scheduled: d.scheduled
              }
            }
            return d
          })
        })
        .catch(err => {
          logger.error(`Get repo ${name}: %s`, err)
          ctx.status = 500
          setErrMsg(ctx, err)
        })
    })
  /**
   * @api {post} /repositories/:name Create Repository
   * @apiName CreateRepository
   * @apiGroup Repositories
   *
   * @apiUse AccessToken
   * @apiParam {String} name Name of the Repository
   * @apiParam {String} interval Task interval
   * @apiParam {String} image Name of the Docker image
   * @apiParam {String} storageDir Path to storage directory
   * @apiParam {Boolean} autoRotLog=true Whether rotates log automatically (Optional)
   * @apiParam {Number} rotateCycle=10 Number of the cycle versions to save (Optional)
   * @apiParam {String[]} args Arguments passed to image (Optional)
   * @apiParam {String[]} envs Environment variables (Optional)
   * @apiParam {String[]} volumes Volumes to be mount (Optional)
   * @apiParam {String} bindIp Local ip to be bound (Optional)
   * @apiParam {String} user Owner of the storage directory (Optional)
   *
   *
   * @apiUse CommonErr
   */
    .post(isLoggedIn, (ctx) => {
      const body = ctx.$body
      body.name = ctx.params.name
      if (!dirExists(body.storageDir)) {
        setErrMsg(ctx, `no such directory: ${body.storageDir}`)
        logger.error(`Creating ${body.name}: no such directory: ${body.storageDir}`)
        return ctx.status = 400
      }
      return Repo.create(body)
        .then((repo) => {
          ctx.status = 201
          scheduler.addJob(repo._id, repo.interval)
        }, (err) => {
          logger.error(`Creating ${body.name}: %s`, err)
          ctx.status = 400
          setErrMsg(ctx, err.message)
        })
    })
  /**
   * @api {put} /repositories/:name Update Repository
   * @apiName UpdateRepository
   * @apiGroup Repositories
   *
   * @apiUse AccessToken
   * @apiParam {String} name Name of the Repository
   * @apiParam {String} interval Task interval
   * @apiParam {String} image Name of the Docker image
   * @apiParam {String} storageDir Path to storage directory
   * @apiParam {Boolean} autoRotLog=true Whether rotates log automatically
   * @apiParam {Number} rotateCycle=10 Number of the cycle versions to save
   * @apiParam {String[]} args Arguments passed to image
   * @apiParam {String[]} envs Environment variables
   * @apiParam {String[]} volumes Volumes to be mount
   * @apiParam {String} bindIp Local ip to be bound
   * @apiParam {String} user Owner of the storage directory
   *
   * @apiSuccess (Success 204) {String} empty
   *
   * @apiUse CommonErr
   */
    .put(isLoggedIn, (ctx) => {
      const name = ctx.params.name
      return Repo.findByIdAndUpdate(name, ctx.$body, {
        // return the modified doc
        new: true,
        runValidators: true
      })
        .then((repo) => {
          // would automatically cancel the existing job
          scheduler.addJob(repo._id /* name */, repo.interval)
          ctx.status = 204
        })
        .catch(err => {
          logger.error(`Updating ${name}: %s`, err)
          ctx.status = 500
          setErrMsg(ctx, err.message)
        })
    })
  /**
   * @api {delete} /repositories/:name Delete Repository
   * @apiName DeleteRepository
   * @apiGroup Repositories
   *
   * @apiUse AccessToken
   * @apiParam {String} name Name of the Repository
   *
   * @apiSuccess (Success 204) {String} empty
   *
   * @apiUse CommonErr
   */
    .delete(isLoggedIn, (ctx) => {
      const name = ctx.params.name
      return Repo.findByIdAndRemove(name)
        .then(() => {
          scheduler.cancelJob(name)
          ctx.status = 204
        })
        .catch(err => {
          logger.error(`Deleting ${name}: %s`, err)
          ctx.status = 500
          setErrMsg(ctx, err.message)
        })
    })

  /**
   * @api {get} /repositories/:name/sync Got Logs of Repository
   * @apiName FetchRepositoryLogs
   * @apiGroup Repositories
   *
   * @apiUse AccessToken
   * @apiParam {String} name Name of the Repository
   * @apiParam {Number} n Nth log-file
   * @apiParam {Boolean} stats Only return matched log-files' names
   *
   * @apiSuccess (Success 200) {Stream} logs Log stream
   * @apiSuccess (Success 200) {Object} names Names of the log-files
   *
   * @apiUse CommonErr
   */
    .get('/repositories/:name/logs', isLoggedIn, (ctx) => {
      const repo = ctx.params.name
      const nth = ctx.query.n || 0
      const stats = !!ctx.query.stats
      const tail = ctx.query.tail || 'all'
      const logdir = path.join(CONFIG.LOGDIR_ROOT, repo)
      if (!dirExists(logdir)) {
        setErrMsg(ctx, `no such repo ${repo}`)
        return ctx.status = 404
      }

      if (!/^(all|\d+)$/.test(tail)) {
        setErrMsg(ctx, `Invalid argument: tail: ${tail}`)
        return ctx.status = 401
      }

      if (stats) {
        return readDir(logdir)
          .then((files) => {
            ctx.body = files.filter(f => f.startsWith('result.log.'))
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
                .then((data) => ctx.body = data)
            }
          }
          setErrMsg(ctx, `${path.join(repo, wantedName)} cannot be found`)
          ctx.status = 404
        })
        .catch(e => {
          logger.error(`${repo} logs: %s`, e)
          setErrMsg(ctx, e.message)
          ctx.status = 500
        })
    })

  /**
   * @api {post} /repositories/:name/sync Sync Repository
   * @apiName SyncRepository
   * @apiGroup Repositories
   *
   * @apiUse AccessToken
   * @apiParam {String} name Name of the Repository
   * @apiParam {Boolean} debug Start the container in debug mode
   *
   * @apiSuccess (Success 204) {String} empty
   * @apiSuccess (Success 200) {Stream} logs Log stream
   *
   * @apiUse CommonErr
   */
    .post('/repositories/:name/sync', isLoggedIn, async (ctx) => {
      const name = ctx.params.name
      const debug = !!ctx.query.debug // dirty hack, convert to boolean

      const cfg = await Repo.findById(name)
      if (cfg === null) {
        setErrMsg(ctx, `no such repository: ${name}`)
        return ctx.status = 404
      }

      if (!dirExists(cfg.storageDir)) {
        setErrMsg(ctx, `no such directory: ${cfg.storageDir}`)
        logger.warn(`Sync ${name}: no such directory: ${cfg.storageDir}`)
        return ctx.status = 404
      }

      const logdir = path.join(CONFIG.LOGDIR_ROOT, name)
      try {
        makeDir(logdir)
      } catch (e) {
        setErrMsg(ctx, e.message)
        logger.error(`Sync ${name}: ${e.message}`)
        return ctx.status = 404
      }

      const opts = await queryOpts({ name, debug })

      let ct
      try {
        ct = await bringUp(opts)
      } catch (e) {
        logger.error(`bringUp ${name}: %s`, e)
        ctx.body = e.json
        return ctx.status = e.statusCode
      }

      if (!debug) {
        ctx.status = 204
      } else {
        ctx.res.setTimeout(0)
        return ct.logs({
          stdout: true,
          stderr: true,
          follow: true
        })
          .then(s => {
            const logStream = new stream.PassThrough()
            s.on('end', logStream.end.bind(logStream))
            ctx.req.on('close', s.destroy.bind(s))
            ct.modem.demuxStream(s, logStream, logStream)
            ctx.body = logStream
          })
          .catch(err => {
            logger.error(`${name} sync: %s`, err)
            ctx.status = err.statusCode
            setErrMsg(ctx, err.reason)
          })
      }

    })

}
