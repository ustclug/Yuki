import { execSync } from 'child_process'
import { basename } from 'path'
import { EOL } from 'os'
import Fs from './base'

export default class Xfs extends Fs {
  constructor() {
    super()
    const prefix = (process.getuid() === 0) ? '' : 'sudo'
    this.cmd = `${prefix} xfs_quota -c 'quota -pN $repo'`
  }

  getSize(dir) {
    const cmd = this.cmd.replace('$repo', basename(dir))
    let lines = null
    try {
      lines = execSync(cmd, { encoding: 'utf8' }).split(EOL)
    } catch (e) {
      return -1
    }
    const got = lines[1]
    if (!got) {
      return -1
    }
    const size = +got.split(/\s+/)[1]

    return size * 1024
  }
}
