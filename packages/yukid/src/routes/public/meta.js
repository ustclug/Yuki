import R from 'ramda'
import { Meta } from '../../models'
import { NotFound, InvalidParams } from '../lib'
import { setBody, invoke } from '../../util'

export default function register(router) {
  router
    .get('/meta', (ctx) => {
      const availKeys = ['name', 'size', 'lastSuccess']
      let key = ctx.query.key || 'name'
      if (availKeys.indexOf(key) < 0) {
        return InvalidParams(ctx, `Invalid key: ${key}`)
      }
      const order = +ctx.query.order || 1
      if (key === 'name') key = '_id'
      return Meta.find()
        .populate('upstream')
        .sort({ [key]: order })
        .then(R.map(invoke('toJSON')))
        .then(setBody(ctx))
    })
    .get('/meta/:name', (ctx) => {
      const name = ctx.params.name
      return Meta.findById(name)
        .populate('upstream')
        .then((r) => {
          if (r === null) {
            return NotFound(ctx, `No such repository: ${name}`)
          }
          ctx.body = r.toJSON()
        })
    })
}
