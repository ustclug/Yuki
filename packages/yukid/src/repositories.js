import path from 'path'
import Promise from 'bluebird'
import { IS_TEST } from './globals'
import CONFIG from './config'
import fs from './filesystem'
import { Repository as Repo, Meta } from './models'

const PREFIX = CONFIG.get('CT_NAME_PREFIX')
const LABEL = CONFIG.get('CT_LABEL')

export async function queryOpts({ name, debug = false }) {
  const cfg = await Repo.findById(name)
  if (cfg === null) {
    return null
  }
  const logdir = path.join(CONFIG.get('LOGDIR_ROOT'), name)
  const opts = {
    Image: cfg.image,
    Env: [],
    AttachStdin: false,
    AttachStdout: false,
    AttachStderr: false,
    Tty: false,
    OpenStdin: true,
    Labels: {
      [LABEL]: ''
    },
    HostConfig: {
      Binds: [],
      AutoRemove: true,
    },
    name: `${PREFIX}-${name}`,
  }
  opts.Env.push(
    `REPO=${name}`,
    `OWNER=${cfg.user || CONFIG.get('OWNER')}`
  )
  for (const [k, v] of Object.entries(cfg.envs)) {
    opts.Env.push(`${k}=${v}`)
  }
  for (const [k, v] of Object.entries(cfg.volumes)) {
    opts.HostConfig.Binds.push(`${k}:${v}`)
  }
  if (!IS_TEST) {
    opts.HostConfig.Binds.push(`${cfg.storageDir}:/data/`, `${logdir}:/log/`)
  }
  if (debug) {
    opts.Env.push('DEBUG=true')
  }
  const addr = cfg.bindIp || CONFIG.get('BIND_ADDRESS')
  if (addr) {
    opts.HostConfig.NetworkMode = 'host'
    opts.Env.push(`BIND_ADDRESS=${addr}`)
  }
  if (cfg.logRotCycle) {
    opts.Env.push(`LOG_ROTATE_CYCLE=${cfg.logRotCycle}`)
  }
  return opts
}

export function createMeta(docs) {
  let data
  if (docs === undefined) {
    data = Repo.find(null, { storageDir: 1 })
      .then((repos) => repos.map((r) => r.toJSON()))
  } else {
    if (Array.isArray(docs)) {
      data = Promise.resolve(docs)
    } else {
      data = Promise.resolve([docs])
    }
  }
  return data
    .then((data) =>
      data.map((doc) =>
        Meta.findByIdAndUpdate(doc._id, {
          $set: {
            size: fs.getSize(doc.storageDir)
          },
          $setOnInsert: {
            updatedAt: new Date(),
          }
        }, { upsert: true })
      )
    )
    .then(Promise.all)
}
