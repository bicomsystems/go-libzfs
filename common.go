// Package zfs implements basic manipulation of ZFS pools and data sets.
// Use libzfs C library instead CLI zfs tools, with goal
// to let using and manipulating OpenZFS form with in go project.
//
// TODO: Adding to the pool. (Add the given vdevs to the pool)
// TODO: Scan for pools.
//
//
package zfs

/*
#cgo CFLAGS: -I /usr/include/libzfs -I /usr/include/libspl -DHAVE_IOCTL_IN_SYS_IOCTL_H
#cgo LDFLAGS: -lzfs -lzpool -lnvpair

#include <stdlib.h>
#include <libzfs.h>
#include "common.h"
#include "zpool.h"
#include "zfs.h"
*/
import "C"

import (
	"sync"
)

// VDevType type of device in the pool
type VDevType string

func init() {
	C.go_libzfs_init()
	return
}

// Types of Virtual Devices
const (
	VDevTypeRoot      VDevType = "root"      // VDevTypeRoot root device in ZFS pool
	VDevTypeMirror             = "mirror"    // VDevTypeMirror mirror device in ZFS pool
	VDevTypeReplacing          = "replacing" // VDevTypeReplacing replacing
	VDevTypeRaidz              = "raidz"     // VDevTypeRaidz RAIDZ device
	VDevTypeDisk               = "disk"      // VDevTypeDisk device is disk
	VDevTypeFile               = "file"      // VDevTypeFile device is file
	VDevTypeMissing            = "missing"   // VDevTypeMissing missing device
	VDevTypeHole               = "hole"      // VDevTypeHole hole
	VDevTypeSpare              = "spare"     // VDevTypeSpare spare device
	VDevTypeLog                = "log"       // VDevTypeLog ZIL device
	VDevTypeL2cache            = "l2cache"   // VDevTypeL2cache cache device (disk)
)

// Prop type to enumerate all different properties suppoerted by ZFS
type Prop int

// PoolStatus type representing status of the pool
type PoolStatus int

// PoolState type representing pool state
type PoolState uint64

// VDevState - vdev states tye
type VDevState uint64

// VDevAux - vdev aux states
type VDevAux uint64

// Property ZFS pool or dataset property value
type Property struct {
	Value  string
	Source string
}

var Global struct {
	Mtx sync.Mutex
}

// Pool status
const (
	/*
	 * The following correspond to faults as defined in the (fault.fs.zfs.*)
	 * event namespace.  Each is associated with a corresponding message ID.
	 */
	PoolStatusCorruptCache      PoolStatus = iota /* corrupt /kernel/drv/zpool.cache */
	PoolStatusMissingDevR                         /* missing device with replicas */
	PoolStatusMissingDevNr                        /* missing device with no replicas */
	PoolStatusCorruptLabelR                       /* bad device label with replicas */
	PoolStatusCorruptLabelNr                      /* bad device label with no replicas */
	PoolStatusBadGUIDSum                          /* sum of device guids didn't match */
	PoolStatusCorruptPool                         /* pool metadata is corrupted */
	PoolStatusCorruptData                         /* data errors in user (meta)data */
	PoolStatusFailingDev                          /* device experiencing errors */
	PoolStatusVersionNewer                        /* newer on-disk version */
	PoolStatusHostidMismatch                      /* last accessed by another system */
	PoolStatusIoFailureWait                       /* failed I/O, failmode 'wait' */
	PoolStatusIoFailureContinue                   /* failed I/O, failmode 'continue' */
	PoolStatusBadLog                              /* cannot read log chain(s) */
	PoolStatusErrata                              /* informational errata available */

	/*
	 * If the pool has unsupported features but can still be opened in
	 * read-only mode, its status is ZPOOL_STATUS_UNSUP_FEAT_WRITE. If the
	 * pool has unsupported features but cannot be opened at all, its
	 * status is ZPOOL_STATUS_UNSUP_FEAT_READ.
	 */
	PoolStatusUnsupFeatRead  /* unsupported features for read */
	PoolStatusUnsupFeatWrite /* unsupported features for write */

	/*
	 * These faults have no corresponding message ID.  At the time we are
	 * checking the status, the original reason for the FMA fault (I/O or
	 * checksum errors) has been lost.
	 */
	PoolStatusFaultedDevR  /* faulted device with replicas */
	PoolStatusFaultedDevNr /* faulted device with no replicas */

	/*
	 * The following are not faults per se, but still an error possibly
	 * requiring administrative attention.  There is no corresponding
	 * message ID.
	 */
	PoolStatusVersionOlder /* older legacy on-disk version */
	PoolStatusFeatDisabled /* supported features are disabled */
	PoolStatusResilvering  /* device being resilvered */
	PoolStatusOfflineDev   /* device online */
	PoolStatusRemovedDev   /* removed device */

	/*
	 * Finally, the following indicates a healthy pool.
	 */
	PoolStatusOk
)

// Possible ZFS pool states
const (
	PoolStateActive            PoolState = iota /* In active use		*/
	PoolStateExported                           /* Explicitly exported		*/
	PoolStateDestroyed                          /* Explicitly destroyed		*/
	PoolStateSpare                              /* Reserved for hot spare use	*/
	PoolStateL2cache                            /* Level 2 ARC device		*/
	PoolStateUninitialized                      /* Internal spa_t state		*/
	PoolStateUnavail                            /* Internal libzfs state	*/
	PoolStatePotentiallyActive                  /* Internal libzfs state	*/
)

// Pool properties. Enumerates available ZFS pool properties. Use it to access
// pool properties either to read or set soecific property.
const (
	PoolPropName Prop = iota
	PoolPropSize
	PoolPropCapacity
	PoolPropAltroot
	PoolPropHealth
	PoolPropGUID
	PoolPropVersion
	PoolPropBootfs
	PoolPropDelegation
	PoolPropAutoreplace
	PoolPropCachefile
	PoolPropFailuremode
	PoolPropListsnaps
	PoolPropAutoexpand
	PoolPropDedupditto
	PoolPropDedupratio
	PoolPropFree
	PoolPropAllocated
	PoolPropReadonly
	PoolPropAshift
	PoolPropComment
	PoolPropExpandsz
	PoolPropFreeing
	PoolPropFragmentaion
	PoolPropLeaked
	PoolPropMaxBlockSize
	PoolPropTName
	PoolPropMaxNodeSize
	PoolPropMultihost
	PoolPropCheckpoint
	PoolPropLoadGUID
	PoolNumProps
)

/*
 * Dataset properties are identified by these constants and must be added to
 * the end of this list to ensure that external consumers are not affected
 * by the change. If you make any changes to this list, be sure to update
 * the property table in module/zcommon/zfs_prop.c.
 */
const (
	DatasetPropType Prop = iota
	DatasetPropCreation
	DatasetPropUsed
	DatasetPropAvailable
	DatasetPropReferenced
	DatasetPropCompressratio
	DatasetPropMounted
	DatasetPropOrigin
	DatasetPropQuota
	DatasetPropReservation
	DatasetPropVolsize
	DatasetPropVolblocksize
	DatasetPropRecordsize
	DatasetPropMountpoint
	DatasetPropSharenfs
	DatasetPropChecksum
	DatasetPropCompression
	DatasetPropAtime
	DatasetPropDevices
	DatasetPropExec
	DatasetPropSetuid
	DatasetPropReadonly
	DatasetPropZoned
	DatasetPropSnapdir
	DatasetPropPrivate /* not exposed to user, temporary */
	DatasetPropAclinherit
	DatasetPropCreatetxg /* not exposed to the user */
	DatasetPropName      /* not exposed to the user */
	DatasetPropCanmount
	DatasetPropIscsioptions /* not exposed to the user */
	DatasetPropXattr
	DatasetPropNumclones /* not exposed to the user */
	DatasetPropCopies
	DatasetPropVersion
	DatasetPropUtf8only
	DatasetPropNormalize
	DatasetPropCase
	DatasetPropVscan
	DatasetPropNbmand
	DatasetPropSharesmb
	DatasetPropRefquota
	DatasetPropRefreservation
	DatasetPropGUID
	DatasetPropPrimarycache
	DatasetPropSecondarycache
	DatasetPropUsedsnap
	DatasetPropUsedds
	DatasetPropUsedchild
	DatasetPropUsedrefreserv
	DatasetPropUseraccounting /* not exposed to the user */
	DatasetPropStmfShareinfo  /* not exposed to the user */
	DatasetPropDeferDestroy
	DatasetPropUserrefs
	DatasetPropLogbias
	DatasetPropUnique   /* not exposed to the user */
	DatasetPropObjsetid /* not exposed to the user */
	DatasetPropDedup
	DatasetPropMlslabel
	DatasetPropSync
	DatasetPropRefratio
	DatasetPropWritten
	DatasetPropClones
	DatasetPropLogicalused
	DatasetPropLogicalreferenced
	DatasetPropInconsistent /* not exposed to the user */
	DatasetPropSnapdev
	DatasetPropAcltype
	DatasetPropSelinuxContext
	DatasetPropSelinuxFsContext
	DatasetPropSelinuxDefContext
	DatasetPropSelinuxRootContext
	DatasetPropRelatime
	DatasetPropRedundantMetadata
	DatasetPropOverlay
	DatasetNumProps
)

// Error represents a ZFS error with message and errno
type Error struct {
	errno   int
	message string
}

func newError(errno int, message string) error {
	return &Error{
		errno:   errno,
		message: message,
	}
}

func wrapError(e error, message string) error {
	switch refinedErr := e.(type) {
	case *Error:
		return newError(refinedErr.Errno(), message)
	default:
		return newError(EUnknown, message)
	}
}

func (z *Error) Error() string {
	return z.message
}

func (z *Error) Errno() int {
	return z.errno
}

// LastError get last underlying libzfs error description if any
func LastError() (err error) {
	return &Error{
		errno:   int(C.libzfs_last_error()),
		message: C.GoString(C.libzfs_last_error_str()),
	}
}

// ClearLastError force clear of any last error set by undeliying libzfs
func ClearLastError() (err error) {
	err = LastError()
	C.libzfs_clear_last_error()
	return
}

func booleanT(b bool) (r C.boolean_t) {
	if b {
		return 1
	}
	return 0
}

// ZFS errors
const (
	ESuccess            = 0               /* no error -- success */
	ENomem              = 2000 + iota - 1 /* out of memory */
	EBadprop                              /* invalid property value */
	EPropreadonly                         /* cannot set readonly property */
	EProptype                             /* property does not apply to dataset type */
	EPropnoninherit                       /* property is not inheritable */
	EPropspace                            /* bad quota or reservation */
	EBadtype                              /* dataset is not of appropriate type */
	EBusy                                 /* pool or dataset is busy */
	EExists                               /* pool or dataset already exists */
	ENoent                                /* no such pool or dataset */
	EBadstream                            /* bad backup stream */
	EDsreadonly                           /* dataset is readonly */
	EVoltoobig                            /* volume is too large for 32-bit system */
	EInvalidname                          /* invalid dataset name */
	EBadrestore                           /* unable to restore to destination */
	EBadbackup                            /* backup failed */
	EBadtarget                            /* bad attach/detach/replace target */
	ENodevice                             /* no such device in pool */
	EBaddev                               /* invalid device to add */
	ENoreplicas                           /* no valid replicas */
	EResilvering                          /* currently resilvering */
	EBadversion                           /* unsupported version */
	EPoolunavail                          /* pool is currently unavailable */
	EDevoverflow                          /* too many devices in one vdev */
	EBadpath                              /* must be an absolute path */
	ECrosstarget                          /* rename or clone across pool or dataset */
	EZoned                                /* used improperly in local zone */
	EMountfailed                          /* failed to mount dataset */
	EUmountfailed                         /* failed to unmount dataset */
	EUnsharenfsfailed                     /* unshare(1M) failed */
	ESharenfsfailed                       /* share(1M) failed */
	EPerm                                 /* permission denied */
	ENospc                                /* out of space */
	EFault                                /* bad address */
	EIo                                   /* I/O error */
	EIntr                                 /* signal received */
	EIsspare                              /* device is a hot spare */
	EInvalconfig                          /* invalid vdev configuration */
	ERecursive                            /* recursive dependency */
	ENohistory                            /* no history object */
	EPoolprops                            /* couldn't retrieve pool props */
	EPoolNotsup                           /* ops not supported for this type of pool */
	EPoolInvalarg                         /* invalid argument for this pool operation */
	ENametoolong                          /* dataset name is too long */
	EOpenfailed                           /* open of device failed */
	ENocap                                /* couldn't get capacity */
	ELabelfailed                          /* write of label failed */
	EBadwho                               /* invalid permission who */
	EBadperm                              /* invalid permission */
	EBadpermset                           /* invalid permission set name */
	ENodelegation                         /* delegated administration is disabled */
	EUnsharesmbfailed                     /* failed to unshare over smb */
	ESharesmbfailed                       /* failed to share over smb */
	EBadcache                             /* bad cache file */
	EIsl2CACHE                            /* device is for the level 2 ARC */
	EVdevnotsup                           /* unsupported vdev type */
	ENotsup                               /* ops not supported on this dataset */
	EActiveSpare                          /* pool has active shared spare devices */
	EUnplayedLogs                         /* log device has unplayed logs */
	EReftagRele                           /* snapshot release: tag not found */
	EReftagHold                           /* snapshot hold: tag already exists */
	ETagtoolong                           /* snapshot hold/rele: tag too long */
	EPipefailed                           /* pipe create failed */
	EThreadcreatefailed                   /* thread create failed */
	EPostsplitOnline                      /* onlining a disk after splitting it */
	EScrubbing                            /* currently scrubbing */
	ENoScrub                              /* no active scrub */
	EDiff                                 /* general failure of zfs diff */
	EDiffdata                             /* bad zfs diff data */
	EPoolreadonly                         /* pool is in read-only mode */
	EUnknown
)

// vdev states are ordered from least to most healthy.
// A vdev that's VDevStateCantOpen or below is considered unusable.
const (
	VDevStateUnknown  VDevState = iota // Uninitialized vdev
	VDevStateClosed                    // Not currently open
	VDevStateOffline                   // Not allowed to open
	VDevStateRemoved                   // Explicitly removed from system
	VDevStateCantOpen                  // Tried to open, but failed
	VDevStateFaulted                   // External request to fault device
	VDevStateDegraded                  // Replicated vdev with unhealthy kids
	VDevStateHealthy                   // Presumed good
)

// vdev aux states.  When a vdev is in the VDevStateCantOpen state, the aux field
// of the vdev stats structure uses these constants to distinguish why.
const (
	VDevAuxNone         VDevAux = iota // no error
	VDevAuxOpenFailed                  // ldi_open_*() or vn_open() failed
	VDevAuxCorruptData                 // bad label or disk contents
	VDevAuxNoReplicas                  // insufficient number of replicas
	VDevAuxBadGUIDSum                  // vdev guid sum doesn't match
	VDevAuxTooSmall                    // vdev size is too small
	VDevAuxBadLabel                    // the label is OK but invalid
	VDevAuxVersionNewer                // on-disk version is too new
	VDevAuxVersionOlder                // on-disk version is too old
	VDevAuxUnsupFeat                   // unsupported features
	VDevAuxSpared                      // hot spare used in another pool
	VDevAuxErrExceeded                 // too many errors
	VDevAuxIOFailure                   // experienced I/O failure
	VDevAuxBadLog                      // cannot read log chain(s)
	VDevAuxExternal                    // external diagnosis
	VDevAuxSplitPool                   // vdev was split off into another pool
)
