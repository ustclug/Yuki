import docker from '../../docker'
import CONFIG from '../../config'
import { setBody } from '../../util'

const LABEL = CONFIG.get('CT_LABEL')

export default function register(router) {
  router
    .get('/containers', (ctx) => {
      return docker.listContainers({
        all: true,
        filters: {
          label: {
            [LABEL]: true,
            'ustcmirror.images': true
          }
        }
      })
        .then(setBody(ctx))
    })
}
