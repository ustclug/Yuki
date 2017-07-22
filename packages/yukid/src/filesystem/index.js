#!/usr/bin/node

'use strict'

import Fs from './base'
import Zfs from './zfs'
import CONFIG from '../config'

let storage = null
switch (CONFIG.get('FILESYSTEM').type) {
  case 'zfs':
    storage = new Zfs()
    break;

  default:
    storage = new Fs()
}

export default storage
