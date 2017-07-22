#!/usr/bin/node

'use strict'

import stream from 'stream'
import docker from '../../docker'
import CONFIG from '../../config'
import logger from '../../logger'
import { setErrMsg, isLoggedIn } from './lib'

export default function register(router) {

  const PREFIX = CONFIG.get('CT_NAME_PREFIX')
  const LABEL = CONFIG.get('CT_LABEL')

  function getContainer(repo) {
    let spec = ''
    if (repo.startsWith('id:')) {
      spec = repo.slice(3)
    } else {
      spec = `${PREFIX}-${repo}`
    }
    return docker.getContainer(spec)
  }

  /**
   * @api {get} /containers List containers
   * @apiName ListContainers
   * @apiGroup Containers
   *
   * @apiSuccess {Object[]} containers(virtual field) List of containers
   *
   * @apiUse CommonErr
   */
  router.get('/containers', (ctx) => {
    return docker.listContainers({
      all: true,
      filters: {
        label: {
          [LABEL]: true,
          'ustcmirror.images': true
        }
      }
    })
      .then((data) => {
        ctx.body = data
      })
  })
  /**
   * @api {get} /containers/:repo Inspect container
   * @apiName InspectContainer
   * @apiGroup Containers
   *
   * @apiUse AccessToken
   * @apiParam {String} repo Name of the Repository
   *
   * @apiUse CommonErr
   */
    .get('/containers/:repo', isLoggedIn, (ctx) => {
      const ct = getContainer(ctx.params.repo)
      return ct.inspect()
        .then((data) => {
          ctx.body = data
        }, (err) => {
          logger.error('Inspect container: %s', err)
          ctx.status = err.statusCode
          ctx.message = err.reason
          ctx.body = err.json
        })
    })
  /**
   * @api {delete} /containers/:repo Delete container
   * @apiName ListContainers
   * @apiGroup Containers
   *
   * @apiUse AccessToken
   * @apiParam {String} repo Name of the Repository
   *
   * @apiSuccess {String} empty
   *
   * @apiUse CommonErr
   */
    .delete(isLoggedIn, (ctx) => {
      const ct = getContainer(ctx.params.repo)
      return ct.remove({ v: true, force: true })
        .then(() => ctx.status = 204)
        .catch(err => {
          logger.error('Delete container: %s', err)
          ctx.body = err.json
          ctx.status = err.statusCode
        })
    })
  /**
   * @api {post} /containers/:repo/wait Await container stop
   * @apiName WaitForContainer
   * @apiGroup Containers
   *
   * @apiParam {String} repo Name of the Repository
   *
   * @apiSuccess {Object} StatusCode Exit code
   *
   * @apiUse CommonErr
   */
    .post('/containers/:repo/wait', (ctx) => {
      const ct = getContainer(ctx.params.repo)
      return ct.wait()
        .then((res) => {
          ctx.status = 200
          ctx.body = res
        })
        .catch(err => {
          ctx.status = err.statusCode
          ctx.message = err.reason
          ctx.body = err.json
        })
    })
  /**
   * @api {get} /containers/:repo/logs Fetch logs of container
   * @apiName GetLogsOfContainer
   * @apiGroup Containers
   *
   * @apiUse AccessToken
   * @apiParam {String} repo Name of the Repository
   * @apiParam {Boolean} follow Follow log output
   * @apiParam {String} tail=all Number of lines to show from the end of the logs
   *
   * @apiSuccess {Stream} log Log Stream
   * @apiUse CommonErr
   */
    .get('/containers/:repo/logs', isLoggedIn, (ctx) => {
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
        .then(s => {
          const logStream = new stream.PassThrough()
          s.on('end', logStream.end.bind(logStream))
          ctx.req.on('close', s.destroy.bind(s))
          ct.modem.demuxStream(s, logStream, logStream)
          ctx.body = logStream
        })
        .catch(err => {
          logger.error(`${repo} logs: %s`, err)
          ctx.status = err.statusCode
          setErrMsg(ctx, err.reason)
        })
    })

    .post('/containers/:repo/stop', isLoggedIn, ctx => {
      const ct = getContainer(ctx.params.repo)
      const t = ctx.query.t || 10 // timeout(sec)
      return ct.stop({ t })
        .then(() => ctx.status = 204)
        .catch(err => {
          logger.error('Stopping container: %s', err)
          ctx.status = err.statusCode
          ctx.message = err.reason
          ctx.body = err.json
        })
    })

  const actions = ['start', 'restart', 'pause', 'unpause']
  actions.forEach(action => {
    router.post(`/containers/:repo/${action}`, isLoggedIn, ctx => {
      const ct = getContainer(ctx.params.repo)
      return ct[action]()
        .then(() => ctx.status = 204)
        .catch(err => {
          logger.error(`${action} container: %s`, err)
          ctx.status = err.statusCode
          ctx.message = err.reason
          ctx.body = err.json
        })
    })
  })
}
