#!/usr/bin/node

'use strict'

import { execSync } from 'child_process'
import path from 'path'
import Fs from './fs'

export default class Zfs extends Fs {
  constructor({ root = 'pool0/' } = {}) {
    super()
    this.root = root
    const prefix = (process.getuid() === 0) ? '' : 'sudo'
    this.cmd = `${prefix} zfs list -H -o used `
  }

  getSize(repo) {
    let cmd = this.cmd
    const fp = path.join(this.root, repo)
    cmd += fp
    return execSync(cmd, { encoding: 'utf8' }).trim()
  }
}
