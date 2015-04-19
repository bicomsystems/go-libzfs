// Implements basic manipulation of ZFS pools and data sets.
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
#include "zpool.h"
#include "zfs.h"
*/
import "C"

import (
	"errors"
)

type VDevType string

var libzfs_handle *C.struct_libzfs_handle

func init() {
	libzfs_handle = C.libzfs_init()
	return
}

// Types of Virtual Devices
const (
	VDevTypeRoot      VDevType = "root"
	VDevTypeMirror             = "mirror"
	VDevTypeReplacing          = "replacing"
	VDevTypeRaidz              = "raidz"
	VDevTypeDisk               = "disk"
	VDevTypeFile               = "file"
	VDevTypeMissing            = "missing"
	VDevTypeHole               = "hole"
	VDevTypeSpare              = "spare"
	VDevTypeLog                = "log"
	VDevTypeL2cache            = "l2cache"
)

type PoolProp int
type ZFSProp int
type PoolStatus int
type PoolState uint64

// Zfs pool or dataset property
type Property struct {
	Value  string
	Source string
}

// Pool status
const (
	/*
	 * The following correspond to faults as defined in the (fault.fs.zfs.*)
	 * event namespace.  Each is associated with a corresponding message ID.
	 */
	PoolStatusCorrupt_cache       PoolStatus = iota /* corrupt /kernel/drv/zpool.cache */
	PoolStatusMissing_dev_r                         /* missing device with replicas */
	PoolStatusMissing_dev_nr                        /* missing device with no replicas */
	PoolStatusCorrupt_label_r                       /* bad device label with replicas */
	PoolStatusCorrupt_label_nr                      /* bad device label with no replicas */
	PoolStatusBad_guid_sum                          /* sum of device guids didn't match */
	PoolStatusCorrupt_pool                          /* pool metadata is corrupted */
	PoolStatusCorrupt_data                          /* data errors in user (meta)data */
	PoolStatusFailing_dev                           /* device experiencing errors */
	PoolStatusVersion_newer                         /* newer on-disk version */
	PoolStatusHostid_mismatch                       /* last accessed by another system */
	PoolStatusIo_failure_wait                       /* failed I/O, failmode 'wait' */
	PoolStatusIo_failure_continue                   /* failed I/O, failmode 'continue' */
	PoolStatusBad_log                               /* cannot read log chain(s) */
	PoolStatusErrata                                /* informational errata available */

	/*
	 * If the pool has unsupported features but can still be opened in
	 * read-only mode, its status is ZPOOL_STATUS_UNSUP_FEAT_WRITE. If the
	 * pool has unsupported features but cannot be opened at all, its
	 * status is ZPOOL_STATUS_UNSUP_FEAT_READ.
	 */
	PoolStatusUnsup_feat_read  /* unsupported features for read */
	PoolStatusUnsup_feat_write /* unsupported features for write */

	/*
	 * These faults have no corresponding message ID.  At the time we are
	 * checking the status, the original reason for the FMA fault (I/O or
	 * checksum errors) has been lost.
	 */
	PoolStatusFaulted_dev_r  /* faulted device with replicas */
	PoolStatusFaulted_dev_nr /* faulted device with no replicas */

	/*
	 * The following are not faults per se, but still an error possibly
	 * requiring administrative attention.  There is no corresponding
	 * message ID.
	 */
	PoolStatusVersion_older /* older legacy on-disk version */
	PoolStatusFeat_disabled /* supported features are disabled */
	PoolStatusResilvering   /* device being resilvered */
	PoolStatusOffline_dev   /* device online */
	PoolStatusRemoved_dev   /* removed device */

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
	PoolPropName PoolProp = iota
	PoolPropSize
	PoolPropCapacity
	PoolPropAltroot
	PoolPropHealth
	PoolPropGuid
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
	PoolNumProps
)

/*
 * Dataset properties are identified by these constants and must be added to
 * the end of this list to ensure that external consumers are not affected
 * by the change. If you make any changes to this list, be sure to update
 * the property table in module/zcommon/zfs_prop.c.
 */
const (
	ZFSPropType ZFSProp = iota
	ZFSPropCreation
	ZFSPropUsed
	ZFSPropAvailable
	ZFSPropReferenced
	ZFSPropCompressratio
	ZFSPropMounted
	ZFSPropOrigin
	ZFSPropQuota
	ZFSPropReservation
	ZFSPropVolsize
	ZFSPropVolblocksize
	ZFSPropRecordsize
	ZFSPropMountpoint
	ZFSPropSharenfs
	ZFSPropChecksum
	ZFSPropCompression
	ZFSPropAtime
	ZFSPropDevices
	ZFSPropExec
	ZFSPropSetuid
	ZFSPropReadonly
	ZFSPropZoned
	ZFSPropSnapdir
	ZFSPropPrivate /* not exposed to user, temporary */
	ZFSPropAclinherit
	ZFSPropCreatetxg /* not exposed to the user */
	ZFSPropName      /* not exposed to the user */
	ZFSPropCanmount
	ZFSPropIscsioptions /* not exposed to the user */
	ZFSPropXattr
	ZFSPropNumclones /* not exposed to the user */
	ZFSPropCopies
	ZFSPropVersion
	ZFSPropUtf8only
	ZFSPropNormalize
	ZFSPropCase
	ZFSPropVscan
	ZFSPropNbmand
	ZFSPropSharesmb
	ZFSPropRefquota
	ZFSPropRefreservation
	ZFSPropGuid
	ZFSPropPrimarycache
	ZFSPropSecondarycache
	ZFSPropUsedsnap
	ZFSPropUsedds
	ZFSPropUsedchild
	ZFSPropUsedrefreserv
	ZFSPropUseraccounting /* not exposed to the user */
	ZFSPropStmf_shareinfo /* not exposed to the user */
	ZFSPropDefer_destroy
	ZFSPropUserrefs
	ZFSPropLogbias
	ZFSPropUnique   /* not exposed to the user */
	ZFSPropObjsetid /* not exposed to the user */
	ZFSPropDedup
	ZFSPropMlslabel
	ZFSPropSync
	ZFSPropRefratio
	ZFSPropWritten
	ZFSPropClones
	ZFSPropLogicalused
	ZFSPropLogicalreferenced
	ZFSPropInconsistent /* not exposed to the user */
	ZFSPropSnapdev
	ZFSPropAcltype
	ZFSPropSelinux_context
	ZFSPropSelinux_fscontext
	ZFSPropSelinux_defcontext
	ZFSPropSelinux_rootcontext
	ZFSPropRelatime
	ZFSPropRedundant_metadata
	ZFSNumProps
)

// Get last underlying libzfs error description if any
func LastError() (err error) {
	errno := C.libzfs_errno(libzfs_handle)
	if errno == 0 {
		return nil
	}
	return errors.New(C.GoString(C.libzfs_error_description(libzfs_handle)))
}

// Force clear of any last error set by undeliying libzfs
func ClearLastError() (err error) {
	err = LastError()
	C.clear_last_error(libzfs_handle)
	return
}

func boolean_t(b bool) (r C.boolean_t) {
	if b {
		return 1
	}
	return 0
}
