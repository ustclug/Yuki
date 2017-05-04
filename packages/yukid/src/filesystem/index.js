#!/usr/bin/node

'use strict'

import Fs from './base'
import Zfs from './zfs'
import { STORAGE } from '../config'

let storage = null
switch (STORAGE.fs) {
  case 'zfs':
    storage = new Zfs()
    break;

  default:
    storage = new Fs()
}

export default storage
