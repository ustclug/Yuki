import path from 'path'
import stream from 'stream'
import httpstatus from 'http-status'
import { NoContent, NotFound } from '../lib'
import { setBody } from '../../util'
import docker from '../../docker'
import CONFIG from '../../config'
import { Repository as Repo } from '../../models'
import { dirExists, makeDir } from '../../util'
import { queryOpts } from '../../repositories'
import { bringUp } from '../../containers'
import logger from '../../logger'

export default function register(router) {

  const PREFIX = CONFIG.get('CT_NAME_PREFIX')

  function getContainer(repo) {
    let spec = ''
    if (repo.startsWith('id:')) {
      spec = repo.slice(3)
    } else {
      spec = `${PREFIX}-${repo}`
    }
    return docker.getContainer(spec)
  }

  router
    .get('/containers/:repo', (ctx) => {
      const repo = ctx.params.repo
      const ct = getContainer(repo)
      return ct.inspect()
        .then(setBody(ctx))
        .catch((err) => {
          const msg = err.json.message
          logger.error(`docker.inspect ${repo}: %s`, msg)
          ctx.throw(err.statusCode, msg)
        })
    })
    .post('/containers/:repo', async (ctx) => {
      const name = ctx.params.repo
      const debug = !!ctx.query.debug // dirty hack, convert to boolean
      const cfg = await Repo.findById(name)
      if (cfg === null) {
        return NotFound(ctx, `No such repository: ${name}`)
      }

      if (!dirExists(cfg.storageDir)) {
        return NotFound(ctx, `No such directory: ${cfg.storageDir}`)
      }

      const logdir = path.join(CONFIG.get('LOGDIR_ROOT'), name)

      // May throw Error
      makeDir(logdir)

      const opts = await queryOpts({ name, debug })

      let ct
      try {
        ct = await bringUp(opts)
      } catch (e) {
        return ctx.throw(e.statusCode, e.json.message)
      }

      if (!debug) {
        ctx.status = httpstatus.CREATED
      } else {
        ctx.res.setTimeout(0)
        return ct.logs({
          stdout: true,
          stderr: true,
          follow: true
        })
          .then((s) => {
            const logStream = new stream.PassThrough()
            s.on('end', logStream.end.bind(logStream))
            ctx.req.on('close', s.destroy.bind(s))
            ct.modem.demuxStream(s, logStream, logStream)
            ctx.body = logStream
          })
          .catch((err) => {
            ctx.throw(err.statusCode, err.json)
          })
      }
    })
    .delete('/containers/:repo', (ctx) => {
      const repo = ctx.params.repo
      const ct = getContainer(repo)
      return ct.stop()
        .then(() => ct.remove({ v: true, force: true }))
        .then(NoContent(ctx))
        .catch((err) => {
          const msg = err.json.message
          logger.error(`docker.remove ${repo}: %s`, msg)
          ctx.throw(err.statusCode, msg)
        })
    })
    .post('/containers/:repo/wait', (ctx) => {
      const repo = ctx.params.repo
      const ct = getContainer(repo)
      return ct.wait()
        .then(setBody(ctx))
        .catch((err) => {
          const msg = err.json.message
          logger.error(`docker.wait ${repo}: %s`, msg)
          ctx.throw(err.statusCode, msg)
        })
    })
    .get('/containers/:repo/logs', (ctx) => {
      const follow = !!ctx.query.follow
      const tail = ctx.query.tail || 'all'

      const repo = ctx.params.repo
      const ct = getContainer(repo)
      const opts = {
        stdout: true,
        stderr: true,
        tail,
        follow
      }
      ctx.res.setTimeout(0)
      return ct.logs(opts)
        .then((s) => {
          const logStream = new stream.PassThrough()
          s.on('end', logStream.end.bind(logStream))
          ctx.req.on('close', s.destroy.bind(s))
          ct.modem.demuxStream(s, logStream, logStream)
          ctx.body = logStream
        })
        .catch((err) => {
          const msg = err.json.message
          logger.error(`docker.logs ${repo}: %s`, msg)
          ctx.throw(err.statusCode, msg)
        })
    })

    .post('/containers/:repo/stop', (ctx) => {
      const repo = ctx.params.repo
      const ct = getContainer(repo)
      const t = ctx.query.t || 10 // timeout(sec)
      return ct.stop({ t })
        .then(NoContent(ctx))
        .catch((err) => {
          const msg = err.json.message
          logger.error(`docker.stop ${repo}: %s`, msg)
          ctx.throw(err.statusCode, msg)
        })
    })

  const actions = ['start', 'restart', 'pause', 'unpause']
  actions.forEach((action) => {
    router.post(`/containers/:repo/${action}`, (ctx) => {
      const repo = ctx.params.repo
      const ct = getContainer(repo)
      return ct[action]()
        .then(NoContent(ctx))
        .catch((err) => {
          const msg = err.json.message
          logger.error(`docker.${action} ${repo}: %s`, msg)
          ctx.throw(err.statusCode, msg)
        })
    })
  })
}
