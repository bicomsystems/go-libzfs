package zfs

// #include <stdlib.h>
// #include <libzfs.h>
// #include "common.h"
// #include "zpool.h"
// #include "zfs.h"
import "C"

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"
)

const (
	msgDatasetIsNil = "Dataset handle not initialized or its closed"
)

// DatasetProperties type is map of dataset or volume properties prop -> value
type DatasetProperties map[Prop]string

// DatasetType defines enum of dataset types
type DatasetType int32

const (
	// DatasetTypeFilesystem - file system dataset
	DatasetTypeFilesystem DatasetType = (1 << 0)
	// DatasetTypeSnapshot - snapshot of dataset
	DatasetTypeSnapshot = (1 << 1)
	// DatasetTypeVolume - volume (virtual block device) dataset
	DatasetTypeVolume = (1 << 2)
	// DatasetTypePool - pool dataset
	DatasetTypePool = (1 << 3)
	// DatasetTypeBookmark - bookmark dataset
	DatasetTypeBookmark = (1 << 4)
)

// Dataset - ZFS dataset object
type Dataset struct {
	list       C.dataset_list_ptr
	Type       DatasetType
	Properties map[Prop]Property
	Children   []Dataset
}

func (d *Dataset) openChildren() (err error) {
	d.Children = make([]Dataset, 0, 5)
	list := C.dataset_list_children(d.list)
	for list != nil {
		dataset := Dataset{list: list}
		dataset.Type = DatasetType(C.dataset_type(d.list))
		dataset.Properties = make(map[Prop]Property)
		err = dataset.ReloadProperties()
		if err != nil {
			return
		}
		d.Children = append(d.Children, dataset)
		list = C.dataset_next(list)
	}
	for ci := range d.Children {
		if err = d.Children[ci].openChildren(); err != nil {
			return
		}
	}
	return
}

// DatasetOpenAll recursive get handles to all available datasets on system
// (file-systems, volumes or snapshots).
func DatasetOpenAll() (datasets []Dataset, err error) {
	var dataset Dataset
	dataset.list = C.dataset_list_root()
	for dataset.list != nil {
		dataset.Type = DatasetType(C.dataset_type(dataset.list))
		err = dataset.ReloadProperties()
		if err != nil {
			return
		}
		datasets = append(datasets, dataset)
		dataset.list = C.dataset_next(dataset.list)
	}
	for ci := range datasets {
		if err = datasets[ci].openChildren(); err != nil {
			return
		}
	}
	return
}

// DatasetCloseAll close all datasets in slice and all of its recursive
// children datasets
func DatasetCloseAll(datasets []Dataset) {
	for _, d := range datasets {
		d.Close()
	}
}

// DatasetOpen open dataset and all of its recursive children datasets
func DatasetOpen(path string) (d Dataset, err error) {
	csPath := C.CString(path)
	d.list = C.dataset_open(csPath)
	C.free(unsafe.Pointer(csPath))

	if d.list == nil || d.list.zh == nil {
		err = LastError()
		if err == nil {
			err = newError(ENoent, "dataset not found.")
		}
		err = wrapError(err, fmt.Sprintf("%s - %s", err.Error(), path))
		return
	}
	d.Type = DatasetType(C.dataset_type(d.list))
	d.Properties = make(map[Prop]Property)
	err = d.ReloadProperties()
	if err != nil {
		return
	}
	err = d.openChildren()
	return
}

func datasetPropertiesTonvlist(props map[Prop]Property) (
	cprops C.nvlist_ptr, err error) {
	// convert properties to nvlist C type
	cprops = C.new_property_nvlist()
	if cprops == nil {
		err = errors.New("Failed to allocate properties")
		return
	}
	for prop, value := range props {
		csValue := C.CString(value.Value)
		r := C.property_nvlist_add(
			cprops, C.zfs_prop_to_name(C.zfs_prop_t(prop)), csValue)
		C.free(unsafe.Pointer(csValue))
		if r != 0 {
			err = errors.New("Failed to convert property")
			return
		}
	}
	return
}

// DatasetCreate create a new filesystem or volume on path representing
// pool/dataset or pool/parent/dataset
func DatasetCreate(path string, dtype DatasetType,
	props map[Prop]Property) (d Dataset, err error) {
	var cprops C.nvlist_ptr
	if cprops, err = datasetPropertiesTonvlist(props); err != nil {
		return
	}
	defer C.nvlist_free(cprops)

	csPath := C.CString(path)
	errcode := C.dataset_create(csPath, C.zfs_type_t(dtype), cprops)
	C.free(unsafe.Pointer(csPath))
	if errcode != 0 {
		err = LastError()
		return
	}
	return DatasetOpen(path)
}

// Close close dataset and all its recursive children datasets (close handle
// and cleanup dataset object/s from memory)
func (d *Dataset) Close() {
	// path, _ := d.Path()
	C.dataset_list_close(d.list)
	d.list = nil
	for _, cd := range d.Children {
		cd.Close()
	}
}

// Destroy destroys the dataset.  The caller must make sure that the filesystem
// isn't mounted, and that there are no active dependents. Set Defer argument
// to true to defer destruction for when dataset is not in use. Call Close() to
// cleanup memory.
func (d *Dataset) Destroy(Defer bool) (err error) {
	if len(d.Children) > 0 {
		path, e := d.Path()
		if e != nil {
			return
		}
		dsType, e := d.GetProperty(DatasetPropType)
		if e != nil {
			dsType.Value = err.Error() // just put error (why it didn't fetch property type)
		}
		err = errors.New("Cannot destroy dataset " + path +
			": " + dsType.Value + " has children")
		return
	}
	if d.list != nil {
		if ec := C.dataset_destroy(d.list, booleanT(Defer)); ec != 0 {
			err = LastError()
		}
	} else {
		err = errors.New(msgDatasetIsNil)
	}
	return
}

// DestroyRecursive recursively destroy children of dataset and dataset.
func (d *Dataset) DestroyRecursive() (err error) {
	var path string
	if path, err = d.Path(); err != nil {
		return
	}
	if !strings.Contains(path, "@") { // not snapshot
		if len(d.Children) > 0 {
			for _, c := range d.Children {
				if err = c.DestroyRecursive(); err != nil {
					return
				}
				// close handle to destroyed child dataset
				c.Close()
			}
			// clear closed children array
			d.Children = make([]Dataset, 0)
		}
		err = d.Destroy(false)
	} else {
		var parent Dataset
		tmp := strings.Split(path, "@")
		ppath, snapname := tmp[0], tmp[1]
		if parent, err = DatasetOpen(ppath); err != nil {
			return
		}
		defer parent.Close()
		if len(parent.Children) > 0 {
			for _, c := range parent.Children {
				if path, err = c.Path(); err != nil {
					return
				}
				if strings.Contains(path, "@") {
					continue // skip other snapshots
				}
				if c, err = DatasetOpen(path + "@" + snapname); err != nil {
					continue
				}
				if err = c.DestroyRecursive(); err != nil {
					c.Close()
					return
				}
				c.Close()
			}
		}
		err = d.Destroy(false)
	}
	return
}

// Pool returns pool dataset belongs to
func (d *Dataset) Pool() (p Pool, err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	p.list = C.dataset_get_pool(d.list)
	if p.list != nil && p.list.zph != nil {
		err = p.ReloadProperties()
		return
	}
	err = LastError()
	return
}

// ReloadProperties re-read dataset's properties
func (d *Dataset) ReloadProperties() (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	d.Properties = make(map[Prop]Property)
	Global.Mtx.Lock()
	defer Global.Mtx.Unlock()
	for prop := DatasetPropType; prop < DatasetNumProps; prop++ {
		plist := C.read_dataset_property(d.list, C.int(prop))
		if plist == nil {
			continue
		}
		d.Properties[prop] = Property{Value: C.GoString(&(*plist).value[0]),
			Source: C.GoString(&(*plist).source[0])}
		C.free_properties(plist)
	}
	return
}

// GetProperty reload and return single specified property. This also reloads requested
// property in Properties map.
func (d *Dataset) GetProperty(p Prop) (prop Property, err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	plist := C.read_dataset_property(d.list, C.int(p))
	if plist == nil {
		err = LastError()
		return
	}
	defer C.free_properties(plist)
	prop = Property{Value: C.GoString(&(*plist).value[0]),
		Source: C.GoString(&(*plist).source[0])}
	d.Properties[p] = prop
	return
}

func (d *Dataset) GetUserProperty(p string) (prop Property, err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	csp := C.CString(p)
	defer C.free(unsafe.Pointer(csp))
	plist := C.read_user_property(d.list, csp)
	if plist == nil {
		err = LastError()
		return
	}
	defer C.free_properties(plist)
	prop = Property{Value: C.GoString(&(*plist).value[0]),
		Source: C.GoString(&(*plist).source[0])}
	return
}

// SetProperty set ZFS dataset property to value. Not all properties can be set,
// some can be set only at creation time and some are read only.
// Always check if returned error and its description.
func (d *Dataset) SetProperty(p Prop, value string) (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	csValue := C.CString(value)
	errcode := C.dataset_prop_set(d.list, C.zfs_prop_t(p), csValue)
	C.free(unsafe.Pointer(csValue))
	if errcode != 0 {
		err = LastError()
	}
	// Update Properties member with change made
	if _, err = d.GetProperty(p); err != nil {
		return
	}
	return
}

func (d *Dataset) SetUserProperty(prop, value string) (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	csValue := C.CString(value)
	csProp := C.CString(prop)
	errcode := C.dataset_user_prop_set(d.list, csProp, csValue)
	C.free(unsafe.Pointer(csValue))
	C.free(unsafe.Pointer(csProp))
	if errcode != 0 {
		err = LastError()
	}
	return
}

// Clone - clones the dataset.  The target must be of the same type as
// the source.
func (d *Dataset) Clone(target string, props map[Prop]Property) (rd Dataset, err error) {
	var cprops C.nvlist_ptr
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	if cprops, err = datasetPropertiesTonvlist(props); err != nil {
		return
	}
	defer C.nvlist_free(cprops)
	csTarget := C.CString(target)
	defer C.free(unsafe.Pointer(csTarget))
	if errc := C.dataset_clone(d.list, csTarget, cprops); errc != 0 {
		err = LastError()
		return
	}
	rd, err = DatasetOpen(target)
	return
}

// DatasetSnapshot create dataset snapshot. Set recur to true to snapshot child datasets.
func DatasetSnapshot(path string, recur bool, props map[Prop]Property) (rd Dataset, err error) {
	var cprops C.nvlist_ptr
	if cprops, err = datasetPropertiesTonvlist(props); err != nil {
		return
	}
	defer C.nvlist_free(cprops)
	csPath := C.CString(path)
	defer C.free(unsafe.Pointer(csPath))
	if errc := C.dataset_snapshot(csPath, booleanT(recur), cprops); errc != 0 {
		err = LastError()
		return
	}
	rd, err = DatasetOpen(path)
	return
}

// Path return zfs dataset path/name
func (d *Dataset) Path() (path string, err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	name := C.dataset_get_name(d.list)
	path = C.GoString(name)
	return
}

// Rollback rollabck's dataset snapshot
func (d *Dataset) Rollback(snap *Dataset, force bool) (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	if errc := C.dataset_rollback(d.list, snap.list, booleanT(force)); errc != 0 {
		err = LastError()
		return
	}
	d.ReloadProperties()
	return
}

// Promote promotes dataset clone
func (d *Dataset) Promote() (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	if errc := C.dataset_promote(d.list); errc != 0 {
		err = LastError()
		return
	}
	d.ReloadProperties()
	return
}

// Rename dataset
func (d *Dataset) Rename(newName string, recur,
	forceUnmount bool) (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	csNewName := C.CString(newName)
	defer C.free(unsafe.Pointer(csNewName))
	if errc := C.dataset_rename(d.list, csNewName,
		booleanT(recur), booleanT(forceUnmount)); errc != 0 {
		err = LastError()
		return
	}
	d.ReloadProperties()
	return
}

// IsMounted checks to see if the mount is active.  If the filesystem is mounted,
// sets in 'where' argument the current mountpoint, and returns true.  Otherwise,
// returns false.
func (d *Dataset) IsMounted() (mounted bool, where string) {
	if d.list == nil {
		return
	}
	mp := C.dataset_is_mounted(d.list)
	// defer C.free(mp)
	if mounted = (mp != nil); mounted {
		where = C.GoString(mp)
		C.free(unsafe.Pointer(mp))
	}
	return
}

// Mount the given filesystem.
func (d *Dataset) Mount(options string, flags int) (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	csOptions := C.CString(options)
	defer C.free(unsafe.Pointer(csOptions))
	if ec := C.dataset_mount(d.list, csOptions, C.int(flags)); ec != 0 {
		err = LastError()
	}
	return
}

// Unmount the given filesystem.
func (d *Dataset) Unmount(flags int) (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	if ec := C.dataset_unmount(d.list, C.int(flags)); ec != 0 {
		err = LastError()
	}
	return
}

// UnmountAll unmount this filesystem and any children inheriting the
// mountpoint property.
func (d *Dataset) UnmountAll(flags int) (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	// This is implemented recursive because zfs_unmountall() didn't work
	if len(d.Children) > 0 {
		for _, c := range d.Children {
			if err = c.UnmountAll(flags); err != nil {
				return
			}
		}
	}
	return d.Unmount(flags)
}

// DatasetPropertyToName convert property to name
// ( returns built in string representation of property name).
// This is optional, you can represent each property with string
// name of choice.
func DatasetPropertyToName(p Prop) (name string) {
	if p == DatasetNumProps {
		return "numofprops"
	}
	prop := C.zfs_prop_t(p)
	name = C.GoString(C.zfs_prop_to_name(prop))
	return
}
