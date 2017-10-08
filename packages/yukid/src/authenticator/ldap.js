import ldap from 'ldapjs'
import fs from 'fs'

export default class Ldap {
  constructor({ base, url, ca = '' }) {
    const opts = { url }
    if (ca) {
      opts.tlsOptions = {
        ca: [fs.readFileSync(ca)]
      }
    }
    this.searchBase = base
    this.client = ldap.createClient(opts)
  }

  _search(name, searchBase) {
    return new Promise((ful, rej) => {
      this.client.search(searchBase, {
        filter: `(uid=${name})`,
        scope: 'sub',
      }, (err, res) => {
        let found = null
        res.on('searchEntry', (entry) => {
          found = entry.object
        })
        res.on('error', rej)
        res.on('end', () => {
          if (!found) {
            rej(new Error(`No such user: ${name}`))
          }
          return ful(found)
        })
      })
    })
  }

  verify(name, passwd) {
    return this._search(name, this.searchBase)
      .then((user) => new Promise((ful, rej) => {
        this.client.bind(user.dn, passwd, (err) => {
          if (err) return rej(err)
          return ful(user)
        })
      }))
  }

  close() {
    return new Promise((ful, rej) => {
      this.client.unbind((err) => {
        if (err) return rej(err)
        return ful(null)
      })
    })
  }
}
