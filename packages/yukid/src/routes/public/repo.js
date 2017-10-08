import R from 'ramda'

import scheduler from '../../scheduler'
import { Repository as Repo } from '../../models'
import { setBody } from '../../util'

export default function register(router) {
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
      .then(R.map((r) => {
        r = r.toJSON()
        r.scheduled = scheduler.isScheduled(r._id)
        return r
      }))
      .then(setBody(ctx))
  })
}
