package zfs

// #include <stdlib.h>
// #include <libzfs.h>
// #include "zpool.h"
// #include "zfs.h"
import "C"

import (
	"errors"
)

const (
	msgDatasetIsNil = "Dataset handle not initialized or its closed"
)

type DatasetType int32

const (
	DatasetTypeFilesystem DatasetType = (1 << 0)
	DatasetTypeSnapshot               = (1 << 1)
	DatasetTypeVolume                 = (1 << 2)
	DatasetTypePool                   = (1 << 3)
	DatasetTypeBookmark               = (1 << 4)
)

type Dataset struct {
	list       *C.dataset_list_t
	Type       DatasetType
	Properties map[ZFSProp]Property
	Children   []Dataset
}

func (d *Dataset) openChildren() (err error) {
	var dataset Dataset
	d.Children = make([]Dataset, 0, 5)
	errcode := C.dataset_list_children(d.list.zh, &(dataset.list))
	for dataset.list != nil {
		dataset.Type = DatasetType(C.zfs_get_type(dataset.list.zh))
		dataset.Properties = make(map[ZFSProp]Property)
		err = dataset.ReloadProperties()
		if err != nil {
			return
		}
		d.Children = append(d.Children, dataset)
		dataset.list = C.dataset_next(dataset.list)
	}
	if errcode != 0 {
		err = LastError()
		return
	}
	for ci, _ := range d.Children {
		if err = d.Children[ci].openChildren(); err != nil {
			return
		}
	}
	return
}

// Recursive get handles to all available datasets on system
// (file-systems, volumes or snapshots).
func DatasetOpenAll() (datasets []Dataset, err error) {
	var dataset Dataset
	errcode := C.dataset_list_root(libzfs_handle, &dataset.list)
	for dataset.list != nil {
		dataset.Type = DatasetType(C.zfs_get_type(dataset.list.zh))
		err = dataset.ReloadProperties()
		if err != nil {
			return
		}
		datasets = append(datasets, dataset)
		dataset.list = C.dataset_next(dataset.list)
	}
	if errcode != 0 {
		err = LastError()
		return
	}
	for ci, _ := range datasets {
		if err = datasets[ci].openChildren(); err != nil {
			return
		}
	}
	return
}

// Close all datasets in slice and all of its recursive children datasets
func DatasetCloseAll(datasets []Dataset) {
	for _, d := range datasets {
		d.Close()
	}
}

// Open dataset and all of its recursive children datasets
func DatasetOpen(path string) (d Dataset, err error) {
	d.list = C.create_dataset_list_item()
	d.list.zh = C.zfs_open(libzfs_handle, C.CString(path), 0xF)

	if d.list.zh == nil {
		err = LastError()
		return
	}
	d.Type = DatasetType(C.zfs_get_type(d.list.zh))
	d.Properties = make(map[ZFSProp]Property)
	err = d.ReloadProperties()
	if err != nil {
		return
	}
	err = d.openChildren()
	return
}

func datasetPropertiesTo_nvlist(props map[ZFSProp]Property) (
	cprops *C.nvlist_t, err error) {
	// convert properties to nvlist C type
	r := C.nvlist_alloc(&cprops, C.NV_UNIQUE_NAME, 0)
	if r != 0 {
		err = errors.New("Failed to allocate properties")
		return
	}
	for prop, value := range props {
		r := C.nvlist_add_string(
			cprops, C.zfs_prop_to_name(
				C.zfs_prop_t(prop)), C.CString(value.Value))
		if r != 0 {
			err = errors.New("Failed to convert property")
			return
		}
	}
	return
}

// Create a new filesystem or volume on path representing pool/dataset or pool/parent/dataset
func DatasetCreate(path string, dtype DatasetType,
	props map[ZFSProp]Property) (d Dataset, err error) {
	var cprops *C.nvlist_t
	if cprops, err = datasetPropertiesTo_nvlist(props); err != nil {
		return
	}
	defer C.nvlist_free(cprops)

	errcode := C.zfs_create(libzfs_handle, C.CString(path),
		C.zfs_type_t(dtype), cprops)
	if errcode != 0 {
		err = LastError()
	}
	return
}

// Close dataset and all its recursive children datasets (close handle and cleanup dataset object/s from memory)
func (d *Dataset) Close() {
	if d.list != nil && d.list.zh != nil {
		C.dataset_list_close(d.list)
	}
	for _, cd := range d.Children {
		cd.Close()
	}
}

// Destroys the dataset.  The caller must make sure that the filesystem
// isn't mounted, and that there are no active dependents. Set Defer argument
// to true to defer destruction for when dataset is not in use.
func (d *Dataset) Destroy(Defer bool) (err error) {
	if len(d.Children) > 0 {
		path, e := d.Path()
		if e != nil {
			return
		}
		dsType, e := d.GetProperty(ZFSPropType)
		err = errors.New("Cannot destroy dataset " + path +
			": " + dsType.Value + " has children")
		return
	}
	if d.list != nil {
		if ec := C.zfs_destroy(d.list.zh, boolean_t(Defer)); ec != 0 {
			err = LastError()
		}
	} else {
		err = errors.New(msgDatasetIsNil)
	}
	return
}

// Recursively destroy children of dataset and dataset.
func (d *Dataset) DestroyRecursive() (err error) {
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
	return
}

func (d *Dataset) Pool() (p Pool, err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	p.list = C.create_zpool_list_item()
	p.list.zph = C.zfs_get_pool_handle(d.list.zh)
	if p.list != nil {
		err = p.ReloadProperties()
		return
	}
	err = LastError()
	return
}

func (d *Dataset) ReloadProperties() (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	var plist *C.property_list_t
	plist = C.new_property_list()
	defer C.free_properties(plist)
	d.Properties = make(map[ZFSProp]Property)
	for prop := ZFSPropType; prop < ZFSNumProps; prop++ {
		errcode := C.read_dataset_property(d.list.zh, plist, C.int(prop))
		if errcode != 0 {
			continue
		}
		d.Properties[prop] = Property{Value: C.GoString(&(*plist).value[0]),
			Source: C.GoString(&(*plist).source[0])}
	}
	return
}

// Reload and return single specified property. This also reloads requested
// property in Properties map.
func (d *Dataset) GetProperty(p ZFSProp) (prop Property, err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	var plist *C.property_list_t
	plist = C.new_property_list()
	defer C.free_properties(plist)
	errcode := C.read_dataset_property(d.list.zh, plist, C.int(p))
	if errcode != 0 {
		err = LastError()
		return
	}
	prop = Property{Value: C.GoString(&(*plist).value[0]),
		Source: C.GoString(&(*plist).source[0])}
	d.Properties[p] = prop
	return
}

// Set ZFS dataset property to value. Not all properties can be set,
// some can be set only at creation time and some are read only.
// Always check if returned error and its description.
func (d *Dataset) SetProperty(p ZFSProp, value string) (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	errcode := C.zfs_prop_set(d.list.zh, C.zfs_prop_to_name(
		C.zfs_prop_t(p)), C.CString(value))
	if errcode != 0 {
		err = LastError()
	}
	return
}

// Clones the dataset.  The target must be of the same type as
// the source.
func (d *Dataset) Clone(target string, props map[ZFSProp]Property) (rd Dataset, err error) {
	var cprops *C.nvlist_t
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	if cprops, err = datasetPropertiesTo_nvlist(props); err != nil {
		return
	}
	defer C.nvlist_free(cprops)
	if errc := C.zfs_clone(d.list.zh, C.CString(target), cprops); errc != 0 {
		err = LastError()
		return
	}
	rd, err = DatasetOpen(target)
	return
}

// Create dataset snapshot. Set recur to true to snapshot child datasets.
func DatasetSnapshot(path string, recur bool, props map[ZFSProp]Property) (rd Dataset, err error) {
	var cprops *C.nvlist_t
	if cprops, err = datasetPropertiesTo_nvlist(props); err != nil {
		return
	}
	defer C.nvlist_free(cprops)
	if errc := C.zfs_snapshot(libzfs_handle, C.CString(path), boolean_t(recur), cprops); errc != 0 {
		err = LastError()
		return
	}
	rd, err = DatasetOpen(path)
	return
}

// Return zfs dataset path/name
func (d *Dataset) Path() (path string, err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	name := C.zfs_get_name(d.list.zh)
	path = C.GoString(name)
	return
}

// Rollabck dataset snapshot
func (d *Dataset) Rollback(snap *Dataset, force bool) (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	if errc := C.zfs_rollback(d.list.zh,
		snap.list.zh, boolean_t(force)); errc != 0 {
		err = LastError()
	}
	return
}

// Rename dataset
func (d *Dataset) Rename(newname string, recur,
	force_umount bool) (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	if errc := C.zfs_rename(d.list.zh, C.CString(newname),
		boolean_t(recur), boolean_t(force_umount)); errc != 0 {
		err = LastError()
	}
	return
}

// Checks to see if the mount is active.  If the filesystem is mounted, fills
// in 'where' with the current mountpoint, and returns true.  Otherwise,
// returns false.
func (d *Dataset) IsMounted() (mounted bool, where string) {
	var cw *C.char
	if d.list == nil {
		return false, ""
	}
	m := C.zfs_is_mounted(d.list.zh, &cw)
	defer C.free_cstring(cw)
	if m != 0 {
		return true, C.GoString(cw)
	}
	return false, ""
}

// Mount the given filesystem.
func (d *Dataset) Mount(options string, flags int) (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	if ec := C.zfs_mount(d.list.zh, C.CString(options), C.int(flags)); ec != 0 {
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
	if ec := C.zfs_unmount(d.list.zh, nil, C.int(flags)); ec != 0 {
		err = LastError()
	}
	return
}

// Unmount this filesystem and any children inheriting the mountpoint property.
func (d *Dataset) UnmountAll(flags int) (err error) {
	if d.list == nil {
		err = errors.New(msgDatasetIsNil)
		return
	}
	if ec := C.zfs_unmountall(d.list.zh, C.int(flags)); ec != 0 {
		err = LastError()
	}
	return
}

// Convert property to name
// ( returns built in string representation of property name).
// This is optional, you can represent each property with string
// name of choice.
func (d *Dataset) PropertyToName(p ZFSProp) (name string) {
	if p == ZFSNumProps {
		return "numofprops"
	}
	prop := C.zfs_prop_t(p)
	name = C.GoString(C.zfs_prop_to_name(prop))
	return
}
