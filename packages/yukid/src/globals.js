//
// ONLY CONSTANTS ARE ALLOWED
// BE AWARE OF NOT INTRODUCING ANY OTHER SUBMODULES!
//
const EventEmitter = require('events')

if (!process.env.NODE_ENV) {
  process.env.NODE_ENV = 'production'
}

export const EMITTER = new EventEmitter()
export const IS_DEV = process.env.NODE_ENV.startsWith('dev')
export const IS_TEST = process.env.NODE_ENV.startsWith('test')
export const IS_PROD = process.env.NODE_ENV.startsWith('prod')
export const TOKEN_NAME = 'x-mirror-token'
