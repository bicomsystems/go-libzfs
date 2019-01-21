package zfs

// #include <stdlib.h>
// #include <libzfs.h>
// #include "common.h"
// #include "zpool.h"
// #include "zfs.h"
// #include <memory.h>
// #include <string.h>
import "C"
import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	Parsable   bool
	Progress   bool
	LargeBlock bool
	EmbedData  bool
	Compress   bool
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
	cflags.parsable = to_boolean_t(flags.Parsable)
	cflags.progress = to_boolean_t(flags.Progress)
	cflags.largeblock = to_boolean_t(flags.LargeBlock)
	cflags.embed_data = to_boolean_t(flags.EmbedData)
	cflags.compress = to_boolean_t(flags.Compress)
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
	sendparams := strings.Split(dpath, "@")
	parent := sendparams[0]
	if len(FromName) > 0 {
		if FromName[0] == '@' {
			FromName = FromName[1:]
		} else if strings.Contains(FromName, "/") {
			from := strings.Split(FromName, "@")
			if len(from) > 0 {
				FromName = from[1]
			}
		}
		cfromname = C.CString(FromName)
		defer C.free(unsafe.Pointer(cfromname))
	}
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
	err = d.send("@"+from[1], outf, &flags)
	return
}

// SendSize - estimate snapshot size to transfer
func (d *Dataset) SendSize(FromName string, flags SendFlags) (size int64, err error) {
	var r, w *os.File
	errch := make(chan error)
	defer func() {
		select {
		case <-errch:
		default:
		}
		close(errch)
	}()
	flags.DryRun = true
	flags.Verbose = true
	flags.Progress = true
	flags.Parsable = true
	if r, w, err = os.Pipe(); err != nil {
		return
	}
	defer r.Close()
	go func() {
		var tmpe error
		saveOut := C.redirect_libzfs_stdout(C.int(w.Fd()))
		if saveOut < 0 {
			tmpe = fmt.Errorf("Redirection of zfslib stdout failed %d", saveOut)
		} else {
			tmpe = d.send(FromName, w, &flags)
			C.restore_libzfs_stdout(saveOut)
		}
		w.Close()
		errch <- tmpe
	}()

	r.SetReadDeadline(time.Now().Add(15 * time.Second))
	var data []byte
	if data, err = ioutil.ReadAll(r); err != nil {
		return
	}
	// parse size
	var sizeRe *regexp.Regexp
	if sizeRe, err = regexp.Compile("size[ \t]*([0-9]+)"); err != nil {
		return
	}
	matches := sizeRe.FindAllSubmatch(data, 3)
	if len(matches) > 0 && len(matches[0]) > 1 {
		if size, err = strconv.ParseInt(
			string(matches[0][1]), 10, 64); err != nil {
			return
		}
	}
	err = <-errch
	return
}

// Receive - receive snapshot stream
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
	ec := C.zfs_receive(C.libzfsHandle, dest, nil, cflags, C.int(inf.Fd()), nil)
	if ec != 0 {
		err = fmt.Errorf("ZFS receive of %s failed. %s", C.GoString(dest), LastError().Error())
	}
	return
}
