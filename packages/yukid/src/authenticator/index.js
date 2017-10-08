import CONFIG from '../config'
import { User } from '../models'
import Ldap from './ldap'

const ldapCfg = CONFIG.get('LDAP')
if (ldapCfg.enabled) {
  const c = new Ldap({
    url: ldapCfg.url,
    base: ldapCfg.searchBase,
    ca: ldapCfg.ca,
  })
  exports.default = {
    verify: c.verify.bind(c)
  }
} else {
  exports.default = {
    verify(name, password) {
      return User.findOne({ _id: name, password })
        .then((u) => {
          if (u === null) {
            throw new Error('Wrong name or password')
          }
          return u
        })
    }
  }
}

module.exports = exports['default']
