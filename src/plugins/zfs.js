#!/usr/bin/node

'use strict'

import { execSync } from 'child_process'
import Fs from './fs'

export default class Zfs extends Fs {
  constructor() {
    super()
    const prefix = (process.getuid() === 0) ? '' : 'sudo'
    this.cmd = `${prefix} zfs list -H -o used `
  }

  getSize(dir) {
    const cmd = this.cmd + dir
    return execSync(cmd, { encoding: 'utf8' }).trim()
  }
}
