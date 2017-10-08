import { Created, ServerError, requireAdmin } from '../lib'
import { createMeta } from '../../repositories'
import { Repository as Repo } from '../../models'
import { invoke } from '../../util'

export default function register(router) {
  router
    .get('/config', (ctx) => {
      const pretty = !!ctx.query.pretty
      return Repo.find()
        .sort({ _id: 1 }).exec()
        .then((docs) => {
          docs = docs.map(invoke('toJSON', { versionKey: false, getters: false }))
          ctx.body = pretty ? JSON.stringify(docs, null, 2) : docs
        })
        .catch((err) => ServerError(ctx, `Export config: ${err.message}`))
    })
    .post('/config', requireAdmin, (ctx) => {
      const repos = ctx.$body
      return Repo.create(repos)
        .then(createMeta)
        .then(Created(ctx))
        .catch((err) => ServerError(ctx, `Import config: ${err.message}`))
    })
}
