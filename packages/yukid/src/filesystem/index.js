import FS from './base'
import ZFS from './zfs'
import XFS from './xfs'
import CONFIG from '../config'

let storage = null
switch (CONFIG.get('FILESYSTEM').type) {
  case 'zfs':
    storage = new ZFS()
    break;

  case 'xfs':
    storage = new XFS()
    break;

  default:
    storage = new FS()
}

export default storage
