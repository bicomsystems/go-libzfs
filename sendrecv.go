package zfs

// #include <stdlib.h>
// #include <libzfs.h>
// #include "common.h"
// #include "zpool.h"
// #include "zfs.h"
import "C"
import (
	"fmt"
	"os"
	"path"
	"strings"
	"syscall"
	"unsafe"
)

type SendFlags struct {
	Verbose    bool
	Replicate  bool
	DoAll      bool
	FromOrigin bool
	Dedup      bool
	Props      bool
	DryRun     bool
	// Parsable   bool
	// Progress   bool
	LargeBlock bool
	EmbedData  bool
	// Compress   bool
}

type RecvFlags struct {
	Verbose     bool
	IsPrefix    bool
	IsTail      bool
	DryRun      bool
	Force       bool
	CanmountOff bool
	Resumable   bool
	ByteSwap    bool
	NoMount     bool
}

func to_boolean_t(a bool) C.boolean_t {
	if a {
		return 1
	}
	return 0
}

func to_sendflags_t(flags *SendFlags) (cflags *C.sendflags_t) {
	cflags = C.alloc_sendflags()
	cflags.verbose = to_boolean_t(flags.Verbose)
	cflags.replicate = to_boolean_t(flags.Replicate)
	cflags.doall = to_boolean_t(flags.DoAll)
	cflags.fromorigin = to_boolean_t(flags.FromOrigin)
	cflags.dedup = to_boolean_t(flags.Dedup)
	cflags.props = to_boolean_t(flags.Props)
	cflags.dryrun = to_boolean_t(flags.DryRun)
	// cflags.parsable = to_boolean_t(flags.Parsable)
	// cflags.progress = to_boolean_t(flags.Progress)
	cflags.largeblock = to_boolean_t(flags.LargeBlock)
	cflags.embed_data = to_boolean_t(flags.EmbedData)
	// cflags.compress = to_boolean_t(flags.Compress)
	return
}

func to_recvflags_t(flags *RecvFlags) (cflags *C.recvflags_t) {
	cflags = C.alloc_recvflags()
	cflags.verbose = to_boolean_t(flags.Verbose)
	cflags.isprefix = to_boolean_t(flags.IsPrefix)
	cflags.istail = to_boolean_t(flags.IsTail)
	cflags.dryrun = to_boolean_t(flags.DryRun)
	cflags.force = to_boolean_t(flags.Force)
	cflags.canmountoff = to_boolean_t(flags.CanmountOff)
	// cflags.resumable = to_boolean_t(flags.Resumable)
	cflags.byteswap = to_boolean_t(flags.ByteSwap)
	cflags.nomount = to_boolean_t(flags.NoMount)
	return
}

func (d *Dataset) send(FromName string, outf *os.File, flags *SendFlags) (err error) {
	var cfromname, ctoname *C.char
	var dpath string
	var pd Dataset

	if d.Type != DatasetTypeSnapshot || (len(FromName) > 0 && strings.Contains(FromName, "#")) {
		err = fmt.Errorf(
			"Unsupported method on filesystem or bookmark. Use func SendOne() for that purpose.")
		return
	}

	cflags := to_sendflags_t(flags)
	defer C.free(unsafe.Pointer(cflags))
	if dpath, err = d.Path(); err != nil {
		return
	}
	if len(FromName) > 0 {
		if FromName[0] == '#' || FromName[0] == '@' {
			FromName = dpath + FromName
		}
		cfromname = C.CString(FromName)
		defer C.free(unsafe.Pointer(cfromname))
	}
	sendparams := strings.Split(dpath, "@")
	parent := sendparams[0]
	ctoname = C.CString(sendparams[1])
	defer C.free(unsafe.Pointer(ctoname))
	if pd, err = DatasetOpen(parent); err != nil {
		return
	}
	defer pd.Close()
	cerr := C.zfs_send(pd.list.zh, cfromname, ctoname, cflags, C.int(outf.Fd()), nil, nil, nil)
	if cerr != 0 {
		err = LastError()
	}
	return
}

func (d *Dataset) SendOne(FromName string, outf *os.File, flags *SendFlags) (err error) {
	var cfromname, ctoname *C.char
	var dpath string
	var lzc_send_flags uint32

	if d.Type == DatasetTypeSnapshot || (len(FromName) > 0 && !strings.Contains(FromName, "#")) {
		err = fmt.Errorf(
			"Unsupported with snapshot. Use func Send() for that purpose.")
		return
	}
	if flags.Replicate || flags.DoAll || flags.Props || flags.Dedup || flags.DryRun {
		err = fmt.Errorf("Unsupported flag with filesystem or bookmark.")
		return
	}

	if flags.LargeBlock {
		lzc_send_flags |= C.LZC_SEND_FLAG_LARGE_BLOCK
	}
	if flags.EmbedData {
		lzc_send_flags |= C.LZC_SEND_FLAG_EMBED_DATA
	}
	// if (flags.Compress)
	// 	lzc_send_flags |= LZC_SEND_FLAG_COMPRESS;
	if dpath, err = d.Path(); err != nil {
		return
	}
	if len(FromName) > 0 {
		if FromName[0] == '#' || FromName[0] == '@' {
			FromName = dpath + FromName
		}
		cfromname = C.CString(FromName)
		defer C.free(unsafe.Pointer(cfromname))
	}
	ctoname = C.CString(path.Base(dpath))
	defer C.free(unsafe.Pointer(ctoname))
	cerr := C.zfs_send_one(d.list.zh, cfromname, C.int(outf.Fd()), lzc_send_flags)
	if cerr != 0 {
		err = LastError()
	}
	return
}

func (d *Dataset) Send(outf *os.File, flags SendFlags) (err error) {
	if flags.Replicate {
		flags.DoAll = true
	}
	err = d.send("", outf, &flags)
	return
}

func (d *Dataset) SendFrom(FromName string, outf *os.File, flags SendFlags) (err error) {
	var porigin Property
	var from, dest []string
	if err = d.ReloadProperties(); err != nil {
		return
	}
	porigin, _ = d.GetProperty(DatasetPropOrigin)
	if len(porigin.Value) > 0 && porigin.Value == FromName {
		FromName = ""
		flags.FromOrigin = true
	} else {
		var dpath string
		if dpath, err = d.Path(); err != nil {
			return
		}
		dest = strings.Split(dpath, "@")
		from = strings.Split(FromName, "@")

		if len(from[0]) > 0 && from[0] != dest[0] {
			err = fmt.Errorf("Incremental source must be in same filesystem.")
			return
		}
		if len(from) < 2 || strings.Contains(from[1], "@") || strings.Contains(from[1], "/") {
			err = fmt.Errorf("Invalid incremental source.")
			return
		}
	}
	err = d.send(from[1], outf, &flags)
	return
}

func (d *Dataset) SendSize(FromName string, flags SendFlags) (size uint64, err error) {
	var porigin Property
	var from Dataset
	var dpath string
	if dpath, err = d.Path(); err != nil {
		return
	}
	zc := C.new_zfs_cmd()
	defer C.free(unsafe.Pointer(zc))
	dpath = strings.Split(dpath, "@")[0]
	if len(FromName) > 0 {

		if FromName[0] == '#' || FromName[0] == '@' {
			FromName = dpath + FromName
		}
		porigin, _ = d.GetProperty(DatasetPropOrigin)
		if len(porigin.Value) > 0 && porigin.Value == FromName {
			FromName = ""
			flags.FromOrigin = true
		}
		if from, err = DatasetOpen(FromName); err != nil {
			return
		}
		zc.zc_fromobj = C.zfs_prop_get_int(from.list.zh, C.ZFS_PROP_OBJSETID)
		from.Close()
	} else {
		zc.zc_fromobj = 0
	}
	zc.zc_obj = C.uint64_t(to_boolean_t(flags.FromOrigin))
	zc.zc_sendobj = C.zfs_prop_get_int(d.list.zh, C.ZFS_PROP_OBJSETID)
	zc.zc_guid = 1
	zc.zc_flags = 0
	if flags.LargeBlock {
		zc.zc_flags |= C.LZC_SEND_FLAG_LARGE_BLOCK
	}
	if flags.EmbedData {
		zc.zc_flags |= C.LZC_SEND_FLAG_EMBED_DATA
	}

	// C.estimate_ioctl(d.list.zhp, prevsnap_obj, to_boolean_t(flags.FromOrigin), lzc_send_flags, unsafe.Pointer(&size))
	if ec, e := C.estimate_send_size(zc); ec != 0 {
		err = fmt.Errorf("Failed to estimate send size. %s %d", e.Error(), e.(syscall.Errno))
	}
	size = uint64(zc.zc_objset_type)
	return
}

func (d *Dataset) Receive(inf *os.File, flags RecvFlags) (err error) {
	var dpath string
	if dpath, err = d.Path(); err != nil {
		return
	}
	props := C.new_property_nvlist()
	if props == nil {
		err = fmt.Errorf("Out of memory func (d *Dataset) Recv()")
		return
	}
	defer C.nvlist_free(props)
	cflags := to_recvflags_t(&flags)
	defer C.free(unsafe.Pointer(cflags))
	dest := C.CString(dpath)
	defer C.free(unsafe.Pointer(dest))
	ec := C.zfs_receive(C.libzfsHandle, dest, props, cflags, C.int(inf.Fd()), nil)
	if ec != 0 {
		err = fmt.Errorf("ZFS receive of %s failed. %s", C.GoString(dest), LastError().Error())
	}
	return
}
