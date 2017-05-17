#!/usr/bin/node

'use strict'

import Fs from './base'
import Zfs from './zfs'
import { FILESYSTEM } from '../config'

let storage = null
switch (FILESYSTEM.type) {
  case 'zfs':
    storage = new Zfs()
    break;

  default:
    storage = new Fs()
}

export default storage
