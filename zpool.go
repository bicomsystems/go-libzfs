package zfs

// #include <stdlib.h>
// #include <libzfs.h>
// #include "zpool.h"
// #include "zfs.h"
import "C"

import (
	"errors"
	"fmt"
	"strconv"
)

const (
	msgPoolIsNil = "Pool handle not initialized or its closed"
)

type PoolProperties map[PoolProp]string
type ZFSProperties map[ZFSProp]string

// Object represents handler to single ZFS pool
//
/* Pool.Properties map[string]Property
 */
// Map of all ZFS pool properties, changing any of this will not affect ZFS
// pool, for that use SetProperty( name, value string) method of the pool
// object. This map is initial loaded when ever you open or create pool to
// give easy access to listing all available properties. It can be refreshed
// with up to date values with call to (*Pool) ReloadProperties
type Pool struct {
	list       *C.zpool_list_t
	Properties []Property
	Features   map[string]string
}

// Open ZFS pool handler by name.
// Returns Pool object, requires Pool.Close() to be called explicitly
// for memory cleanup after object is not needed anymore.
func PoolOpen(name string) (pool Pool, err error) {
	pool.list = C.zpool_list_open(libzfs_handle, C.CString(name))
	if pool.list != nil {
		err = pool.ReloadProperties()
		return
	}
	err = LastError()
	return
}

// Given a list of directories to search, find and import pool with matching
// name stored on disk.
func PoolImport(name string, searchpaths []string) (pool Pool, err error) {
	errPoolList := errors.New("Failed to list pools")
	var elem *C.nvpair_t
	var config *C.nvlist_t
	numofp := len(searchpaths)
	cpaths := C.alloc_strings(C.int(numofp))
	for i, path := range searchpaths {
		C.strings_setat(cpaths, C.int(i), C.CString(path))
	}

	pools := C.zpool_find_import(libzfs_handle, C.int(numofp), cpaths)
	defer C.nvlist_free(pools)

	elem = C.nvlist_next_nvpair(pools, elem)
	for ; elem != nil; elem = C.nvlist_next_nvpair(pools, elem) {
		var cname *C.char
		var tconfig *C.nvlist_t
		retcode := C.nvpair_value_nvlist(elem, &tconfig)
		if retcode != 0 {
			err = errPoolList
			return
		}
		retcode = C.nvlist_lookup_string(tconfig,
			C.CString(C.ZPOOL_CONFIG_POOL_NAME), &cname)
		if retcode != 0 {
			err = errPoolList
			return
		}
		oname := C.GoString(cname)
		if name == oname {
			config = tconfig
			break
		}
	}
	if config == nil {
		err = errors.New("No pools to import found with name " + name)
		return
	}

	retcode := C.zpool_import(libzfs_handle, config, C.CString(name), nil)
	if retcode != 0 {
		err = LastError()
		return
	}
	pool, err = PoolOpen(name)
	return
}

// Open all active ZFS pools on current system.
// Returns array of Pool handlers, each have to be closed after not needed
// anymore. Call Pool.Close() method.
func PoolOpenAll() (pools []Pool, err error) {
	var pool Pool
	errcode := C.zpool_list(libzfs_handle, &pool.list)
	for pool.list != nil {
		err = pool.ReloadProperties()
		if err != nil {
			return
		}
		pools = append(pools, pool)
		pool.list = C.zpool_next(pool.list)
	}
	if errcode != 0 {
		err = LastError()
	}
	return
}

func PoolCloseAll(pools []Pool) {
	for _, p := range pools {
		p.Close()
	}
}

// Convert property to name
// ( returns built in string representation of property name).
// This is optional, you can represent each property with string
// name of choice.
func PoolPropertyToName(p PoolProp) (name string) {
	if p == PoolNumProps {
		return "numofprops"
	}
	prop := C.zpool_prop_t(p)
	name = C.GoString(C.zpool_prop_to_name(prop))
	return
}

// Re-read ZFS pool properties and features, refresh Pool.Properties and
// Pool.Features map
func (pool *Pool) ReloadProperties() (err error) {
	propList := C.read_zpool_properties(pool.list.zph)
	if propList == nil {
		err = LastError()
		return
	}

	pool.Properties = make([]Property, PoolNumProps+1)
	next := propList
	for next != nil {
		pool.Properties[next.property] = Property{Value: C.GoString(&(next.value[0])), Source: C.GoString(&(next.source[0]))}
		next = C.next_property(next)
	}
	C.free_properties(propList)

	// read features
	pool.Features = map[string]string{
		"async_destroy": "disabled",
		"empty_bpobj":   "disabled",
		"lz4_compress":  "disabled"}
	for name, _ := range pool.Features {
		pool.GetFeature(name)
	}
	return
}

// Reload and return single specified property. This also reloads requested
// property in Properties map.
func (pool *Pool) GetProperty(p PoolProp) (prop Property, err error) {
	if pool.list != nil {
		// First check if property exist at all
		if p < PoolPropName || p > PoolNumProps {
			err = errors.New(fmt.Sprint("Unknown zpool property: ",
				PoolPropertyToName(p)))
			return
		}
		var list C.property_list_t
		r := C.read_zpool_property(pool.list.zph, &list, C.int(p))
		if r != 0 {
			err = LastError()
		}
		prop.Value = C.GoString(&(list.value[0]))
		prop.Source = C.GoString(&(list.source[0]))
		pool.Properties[p] = prop
		return
	}
	return prop, errors.New(msgPoolIsNil)
}

// Reload and return single specified feature. This also reloads requested
// feature in Features map.
func (pool *Pool) GetFeature(name string) (value string, err error) {
	var fvalue [512]C.char
	sname := fmt.Sprint("feature@", name)
	r := C.zpool_prop_get_feature(pool.list.zph, C.CString(sname), &(fvalue[0]), 512)
	if r != 0 {
		err = errors.New(fmt.Sprint("Unknown zpool feature: ", name))
		return
	}
	value = C.GoString(&(fvalue[0]))
	pool.Features[name] = value
	return
}

// Set ZFS pool property to value. Not all properties can be set,
// some can be set only at creation time and some are read only.
// Always check if returned error and its description.
func (pool *Pool) SetProperty(p PoolProp, value string) (err error) {
	if pool.list != nil {
		// First check if property exist at all
		if p < PoolPropName || p > PoolNumProps {
			err = errors.New(fmt.Sprint("Unknown zpool property: ",
				PoolPropertyToName(p)))
			return
		}
		r := C.zpool_set_prop(pool.list.zph, C.CString(PoolPropertyToName(p)), C.CString(value))
		if r != 0 {
			err = LastError()
		} else {
			// Update Properties member with change made
			if _, err = pool.GetProperty(p); err != nil {
				return
			}
		}
		return
	}
	return errors.New(msgPoolIsNil)
}

// Close ZFS pool handler and release associated memory.
// Do not use Pool object after this.
func (pool *Pool) Close() {
	C.zpool_list_close(pool.list)
	pool.list = nil
}

// Get (re-read) ZFS pool name property
func (pool *Pool) Name() (name string, err error) {
	if pool.list == nil {
		err = errors.New(msgPoolIsNil)
	} else {
		name = C.GoString(C.zpool_get_name(pool.list.zph))
		pool.Properties[PoolPropName] = Property{Value: name, Source: "none"}
	}
	return
}

// Get ZFS pool state
// Return the state of the pool (ACTIVE or UNAVAILABLE)
func (pool *Pool) State() (state PoolState, err error) {
	if pool.list == nil {
		err = errors.New(msgPoolIsNil)
	} else {
		state = PoolState(C.zpool_read_state(pool.list.zph))
	}
	return
}

// ZFS virtual device specification
type VDevSpec struct {
	Type    VDevType
	Devices []VDevSpec // groups other devices (e.g. mirror)
	Parity  uint
	Path    string
}

func (self *VDevSpec) isGrouping() (grouping bool, mindevs, maxdevs int) {
	maxdevs = int(^uint(0) >> 1)
	if self.Type == VDevTypeRaidz {
		grouping = true
		if self.Parity == 0 {
			self.Parity = 1
		}
		if self.Parity > 254 {
			self.Parity = 254
		}
		mindevs = int(self.Parity) + 1
		maxdevs = 255
	} else if self.Type == VDevTypeMirror {
		grouping = true
		mindevs = 2
	} else if self.Type == VDevTypeLog || self.Type == VDevTypeSpare || self.Type == VDevTypeL2cache {
		grouping = true
		mindevs = 1
	}
	return
}

func (self *VDevSpec) isLog() (r C.uint64_t) {
	r = 0
	if self.Type == VDevTypeLog {
		r = 1
	}
	return
}

func toCPoolProperties(props PoolProperties) (cprops *C.nvlist_t) {
	cprops = nil
	for prop, value := range props {
		name := C.zpool_prop_to_name(C.zpool_prop_t(prop))
		r := C.add_prop_list(name, C.CString(value), &cprops, C.boolean_t(1))
		if r != 0 {
			if cprops != nil {
				C.nvlist_free(cprops)
				cprops = nil
			}
			return
		}
	}
	return
}

func toCZFSProperties(props ZFSProperties) (cprops *C.nvlist_t) {
	cprops = nil
	for prop, value := range props {
		name := C.zfs_prop_to_name(C.zfs_prop_t(prop))
		r := C.add_prop_list(name, C.CString(value), &cprops, C.boolean_t(0))
		if r != 0 {
			if cprops != nil {
				C.nvlist_free(cprops)
				cprops = nil
			}
			return
		}
	}
	return
}

func buildVDevSpec(root *C.nvlist_t, rtype VDevType, vdevs []VDevSpec,
	props PoolProperties) (err error) {
	count := len(vdevs)
	if count == 0 {
		return
	}
	childrens := C.nvlist_alloc_array(C.int(count))
	if childrens == nil {
		err = errors.New("No enough memory")
		return
	}
	defer C.nvlist_free_array(childrens)
	spares := C.nvlist_alloc_array(C.int(count))
	if childrens == nil {
		err = errors.New("No enough memory")
		return
	}
	nspares := 0
	defer C.nvlist_free_array(spares)
	l2cache := C.nvlist_alloc_array(C.int(count))
	if childrens == nil {
		err = errors.New("No enough memory")
		return
	}
	nl2cache := 0
	defer C.nvlist_free_array(l2cache)
	for i, vdev := range vdevs {
		grouping, mindevs, maxdevs := vdev.isGrouping()
		var child *C.nvlist_t = nil
		// fmt.Println(vdev.Type)
		if r := C.nvlist_alloc(&child, C.NV_UNIQUE_NAME, 0); r != 0 {
			err = errors.New("Failed to allocate vdev")
			return
		}
		vcount := len(vdev.Devices)
		if vcount < mindevs || vcount > maxdevs {
			err = errors.New(fmt.Sprintf(
				"Invalid vdev specification: %s supports no less than %d or more than %d devices", vdev.Type, mindevs, maxdevs))
			return
		}
		if r := C.nvlist_add_string(child, C.CString(C.ZPOOL_CONFIG_TYPE),
			C.CString(string(vdev.Type))); r != 0 {
			err = errors.New("Failed to set vdev type")
			return
		}
		if r := C.nvlist_add_uint64(child, C.CString(C.ZPOOL_CONFIG_IS_LOG),
			vdev.isLog()); r != 0 {
			err = errors.New("Failed to allocate vdev (is_log)")
			return
		}
		if grouping {
			if vdev.Type == VDevTypeRaidz {
				r := C.nvlist_add_uint64(child,
					C.CString(C.ZPOOL_CONFIG_NPARITY),
					C.uint64_t(mindevs-1))
				if r != 0 {
					err = errors.New("Failed to allocate vdev (parity)")
					return
				}
			}
			if err = buildVDevSpec(child, vdev.Type, vdev.Devices,
				props); err != nil {
				return
			}
		} else {
			// if vdev.Type == VDevTypeDisk {
			if r := C.nvlist_add_uint64(child,
				C.CString(C.ZPOOL_CONFIG_WHOLE_DISK), 1); r != 0 {
				err = errors.New("Failed to allocate vdev child (whdisk)")
				return
			}
			// }
			if len(vdev.Path) > 0 {
				if r := C.nvlist_add_string(
					child, C.CString(C.ZPOOL_CONFIG_PATH),
					C.CString(vdev.Path)); r != 0 {
					err = errors.New("Failed to allocate vdev child (type)")
					return
				}
				ashift, _ := strconv.Atoi(props[PoolPropAshift])
				if ashift > 0 {
					if r := C.nvlist_add_uint64(child,
						C.CString(C.ZPOOL_CONFIG_ASHIFT),
						C.uint64_t(ashift)); r != 0 {
						err = errors.New("Failed to allocate vdev child (ashift)")
						return
					}
				}
			}
			if vdev.Type == VDevTypeSpare {
				C.nvlist_array_set(spares, C.int(nspares), child)
				nspares++
				count--
				continue
			} else if vdev.Type == VDevTypeL2cache {
				C.nvlist_array_set(l2cache, C.int(nl2cache), child)
				nl2cache++
				count--
				continue
			}
		}
		C.nvlist_array_set(childrens, C.int(i), child)
	}
	if count > 0 {
		if r := C.nvlist_add_nvlist_array(root,
			C.CString(C.ZPOOL_CONFIG_CHILDREN), childrens,
			C.uint_t(count)); r != 0 {
			err = errors.New("Failed to allocate vdev children")
			return
		}
		// fmt.Println("childs", root, count, rtype)
		// debug.PrintStack()
	}
	if nl2cache > 0 {
		if r := C.nvlist_add_nvlist_array(root,
			C.CString(C.ZPOOL_CONFIG_L2CACHE), l2cache,
			C.uint_t(nl2cache)); r != 0 {
			err = errors.New("Failed to allocate vdev cache")
			return
		}
	}
	if nspares > 0 {
		if r := C.nvlist_add_nvlist_array(root,
			C.CString(C.ZPOOL_CONFIG_SPARES), spares,
			C.uint_t(nspares)); r != 0 {
			err = errors.New("Failed to allocate vdev spare")
			return
		}
		// fmt.Println("spares", root, count)
	}
	return
}

// Create ZFS pool per specs, features and properties of pool and root dataset
func PoolCreate(name string, vdevs []VDevSpec, features map[string]string,
	props PoolProperties, fsprops ZFSProperties) (pool Pool, err error) {
	// create root vdev nvroot
	var nvroot *C.nvlist_t = nil
	if r := C.nvlist_alloc(&nvroot, C.NV_UNIQUE_NAME, 0); r != 0 {
		err = errors.New("Failed to allocate root vdev")
		return
	}
	if r := C.nvlist_add_string(nvroot, C.CString(C.ZPOOL_CONFIG_TYPE),
		C.CString(string(VDevTypeRoot))); r != 0 {
		err = errors.New("Failed to allocate root vdev")
		return
	}
	defer C.nvlist_free(nvroot)

	// Now we need to build specs (vdev hierarchy)
	if err = buildVDevSpec(nvroot, VDevTypeRoot, vdevs, props); err != nil {
		return
	}

	// convert properties
	cprops := toCPoolProperties(props)
	if cprops != nil {
		defer C.nvlist_free(cprops)
	} else if len(props) > 0 {
		err = errors.New("Failed to allocate pool properties")
		return
	}
	cfsprops := toCZFSProperties(fsprops)
	if cfsprops != nil {
		defer C.nvlist_free(cfsprops)
	} else if len(fsprops) > 0 {
		err = errors.New("Failed to allocate FS properties")
		return
	}
	for fname, fval := range features {
		sfname := fmt.Sprintf("feature@%s", fname)
		r := C.add_prop_list(C.CString(sfname), C.CString(fval), &cprops,
			C.boolean_t(1))
		if r != 0 {
			if cprops != nil {
				C.nvlist_free(cprops)
				cprops = nil
			}
			return
		}
	}

	// Create actual pool then open
	if r := C.zpool_create(libzfs_handle, C.CString(name), nvroot,
		cprops, cfsprops); r != 0 {
		err = LastError()
		return
	}
	pool, err = PoolOpen(name)
	return
}

// Get pool status. Let you check if pool healthy.
func (pool *Pool) Status() (status PoolStatus, err error) {
	var msgid *C.char
	var reason C.zpool_status_t
	var errata C.zpool_errata_t
	if pool.list == nil {
		err = errors.New(msgPoolIsNil)
		return
	}
	reason = C.zpool_get_status(pool.list.zph, &msgid, &errata)
	status = PoolStatus(reason)
	return
}

// Destroy the pool.  It is up to the caller to ensure that there are no
// datasets left in the pool. logStr is optional if specified it is
// appended to ZFS history
func (pool *Pool) Destroy(logStr string) (err error) {
	if pool.list == nil {
		err = errors.New(msgPoolIsNil)
		return
	}
	retcode := C.zpool_destroy(pool.list.zph, C.CString(logStr))
	if retcode != 0 {
		err = LastError()
	}
	return
}

// Exports the pool from the system.
// Before exporting the pool, all datasets within the pool are unmounted.
// A pool can not be exported if it has a shared spare that is currently
// being used.
func (pool *Pool) Export(force bool, log string) (err error) {
	var force_t C.boolean_t = 0
	if force {
		force_t = 1
	}
	if rc := C.zpool_export(pool.list.zph, force_t, C.CString(log)); rc != 0 {
		err = LastError()
	}
	return
}

// Hard force
func (pool *Pool) ExportForce(log string) (err error) {
	if rc := C.zpool_export_force(pool.list.zph, C.CString(log)); rc != 0 {
		err = LastError()
	}
	return
}
