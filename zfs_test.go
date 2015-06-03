package zfs_test

import (
	"fmt"
	"github.com/bicomsystems/go-libzfs"
	"testing"
)

/* ------------------------------------------------------------------------- */
// HELPERS:
var TST_DATASET_PATH = TST_POOL_NAME + "/DATASET"
var TST_VOLUME_PATH = TST_DATASET_PATH + "/VOLUME"
var TST_DATASET_PATH_SNAP = TST_DATASET_PATH + "@test"

func printDatasets(ds []zfs.Dataset) error {
	for _, d := range ds {

		path, err := d.Path()
		if err != nil {
			return err
		}
		p, err := d.GetProperty(zfs.ZFSPropType)
		if err != nil {
			return err
		}
		fmt.Printf(" %30s | %10s\n", path,
			p.Value)
		if len(d.Children) > 0 {
			printDatasets(d.Children)
		}
	}
	return nil
}

/* ------------------------------------------------------------------------- */
// TESTS:

func zfsTestDatasetCreate(t *testing.T) {
	// reinit names used in case TESTPOOL was in conflict
	TST_DATASET_PATH = TST_POOL_NAME + "/DATASET"
	TST_VOLUME_PATH = TST_DATASET_PATH + "/VOLUME"
	TST_DATASET_PATH_SNAP = TST_DATASET_PATH + "@test"

	println("TEST DatasetCreate(", TST_DATASET_PATH, ") (filesystem) ... ")
	props := make(map[zfs.ZFSProp]zfs.Property)
	d, err := zfs.DatasetCreate(TST_DATASET_PATH, zfs.DatasetTypeFilesystem, props)
	if err != nil {
		t.Error(err)
		return
	}
	d.Close()
	println("PASS\n")

	strSize := "536870912" // 512M

	println("TEST DatasetCreate(", TST_VOLUME_PATH, ") (volume) ... ")
	props[zfs.ZFSPropVolsize] = zfs.Property{Value: strSize}
	// In addition I explicitly choose some more properties to be set.
	props[zfs.ZFSPropVolblocksize] = zfs.Property{Value: "4096"}
	props[zfs.ZFSPropReservation] = zfs.Property{Value: strSize}
	d, err = zfs.DatasetCreate(TST_VOLUME_PATH, zfs.DatasetTypeVolume, props)
	if err != nil {
		t.Error(err)
		return
	}
	d.Close()
	println("PASS\n")
}

func zfsTestDatasetOpen(t *testing.T) {
	println("TEST DatasetOpen(", TST_DATASET_PATH, ") ... ")
	d, err := zfs.DatasetOpen(TST_DATASET_PATH)
	if err != nil {
		t.Error(err)
		return
	}
	d.Close()
	println("PASS\n")
}

func zfsTestDatasetOpenAll(t *testing.T) {
	println("TEST DatasetOpenAll()/DatasetCloseAll() ... ")
	ds, err := zfs.DatasetOpenAll()
	if err != nil {
		t.Error(err)
		return
	}
	if err = printDatasets(ds); err != nil {
		zfs.DatasetCloseAll(ds)
		t.Error(err)
		return
	}
	zfs.DatasetCloseAll(ds)
	println("PASS\n")
}

func zfsTestDatasetSnapshot(t *testing.T) {
	println("TEST DatasetSnapshot(", TST_DATASET_PATH, ", true, ...) ... ")
	props := make(map[zfs.ZFSProp]zfs.Property)
	d, err := zfs.DatasetSnapshot(TST_DATASET_PATH_SNAP, true, props)
	if err != nil {
		t.Error(err)
		return
	}
	defer d.Close()
	println("PASS\n")
}

func zfsTestDatasetDestroy(t *testing.T) {
	println("TEST DATASET Destroy( ", TST_DATASET_PATH, " ) ... ")
	d, err := zfs.DatasetOpen(TST_DATASET_PATH)
	if err != nil {
		t.Error(err)
		return
	}
	defer d.Close()
	if err = d.DestroyRecursive(); err != nil {
		t.Error(err)
		return
	}
	println("PASS\n")
}

/* ------------------------------------------------------------------------- */
// EXAMPLES:

// Example of creating ZFS volume
func ExampleDatasetCreate() {
	// Create map to represent ZFS dataset properties. This is equivalent to
	// list of properties you can get from ZFS CLI tool, and some more
	// internally used by libzfs.
	props := make(map[zfs.ZFSProp]zfs.Property)

	// I choose to create (block) volume 1GiB in size. Size is just ZFS dataset
	// property and this is done as map of strings. So, You have to either
	// specify size as base 10 number in string, or use strconv package or
	// similar to convert in to string (base 10) from numeric type.
	strSize := "1073741824"

	props[zfs.ZFSPropVolsize] = zfs.Property{Value: strSize}
	// In addition I explicitly choose some more properties to be set.
	props[zfs.ZFSPropVolblocksize] = zfs.Property{Value: "4096"}
	props[zfs.ZFSPropReservation] = zfs.Property{Value: strSize}

	// Lets create desired volume
	d, err := zfs.DatasetCreate("TESTPOOL/VOLUME1", zfs.DatasetTypeVolume, props)
	if err != nil {
		println(err.Error())
		return
	}
	// Dataset have to be closed for memory cleanup
	defer d.Close()

	println("Created zfs volume TESTPOOL/VOLUME1")
}

func ExampleDatasetOpen() {
	// Open dataset and read its available space
	d, err := zfs.DatasetOpen("TESTPOOL/DATASET1")
	if err != nil {
		panic(err.Error())
		return
	}
	defer d.Close()
	var p zfs.Property
	if p, err = d.GetProperty(zfs.ZFSPropAvailable); err != nil {
		panic(err.Error())
		return
	}
	println(d.PropertyToName(zfs.ZFSPropAvailable), " = ", p.Value)
}
