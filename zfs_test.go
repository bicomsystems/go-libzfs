package zfs_test

import (
	"fmt"
	"testing"

	"github.com/lorenz/go-libzfs"
)

/* ------------------------------------------------------------------------- */
// HELPERS:
var TSTDatasetPath = TSTPoolName + "/DATASET"
var TSTVolumePath = TSTDatasetPath + "/VOLUME"
var TSTDatasetPathSnap = TSTDatasetPath + "@test"

func printDatasets(ds []zfs.Dataset) error {
	for _, d := range ds {

		path, err := d.Path()
		if err != nil {
			return err
		}
		p, err := d.GetProperty(zfs.DatasetPropType)
		if err != nil {
			return err
		}
		fmt.Printf(" %30s | %10s\n", path, p.Value)
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
	TSTDatasetPath = TSTPoolName + "/DATASET"
	TSTVolumePath = TSTDatasetPath + "/VOLUME"
	TSTDatasetPathSnap = TSTDatasetPath + "@test"

	println("TEST DatasetCreate(", TSTDatasetPath, ") (filesystem) ... ")
	props := make(map[zfs.Prop]zfs.Property)
	d, err := zfs.DatasetCreate(TSTDatasetPath, zfs.DatasetTypeFilesystem, props)
	if err != nil {
		t.Error(err)
		return
	}
	d.Close()
	print("PASS\n\n")

	strSize := "536870912" // 512M

	println("TEST DatasetCreate(", TSTVolumePath, ") (volume) ... ")
	props[zfs.DatasetPropVolsize] = zfs.Property{Value: strSize}
	// In addition I explicitly choose some more properties to be set.
	props[zfs.DatasetPropVolblocksize] = zfs.Property{Value: "4096"}
	props[zfs.DatasetPropReservation] = zfs.Property{Value: strSize}
	d, err = zfs.DatasetCreate(TSTVolumePath, zfs.DatasetTypeVolume, props)
	if err != nil {
		t.Error(err)
		return
	}
	d.Close()
	print("PASS\n\n")
}

func zfsTestDatasetOpen(t *testing.T) {
	println("TEST DatasetOpen(", TSTDatasetPath, ") ... ")
	d, err := zfs.DatasetOpen(TSTDatasetPath)
	if err != nil {
		t.Error(err)
		return
	}
	defer d.Close()
	print("PASS\n\n")

	println("TEST Set/GetUserProperty(prop, value string) ... ")
	var p zfs.Property
	// Test set/get user property
	if err = d.SetUserProperty("go-libzfs:test", "yes"); err != nil {
		t.Error(err)
		return
	}
	if p, err = d.GetUserProperty("go-libzfs:test"); err != nil {
		t.Error(err)
		return
	}
	println("go-libzfs:test", " = ",
		p.Value)
	print("PASS\n\n")
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
	print("PASS\n\n")
}

func zfsTestDatasetSnapshot(t *testing.T) {
	println("TEST DatasetSnapshot(", TSTDatasetPath, ", true, ...) ... ")
	props := make(map[zfs.Prop]zfs.Property)
	d, err := zfs.DatasetSnapshot(TSTDatasetPathSnap, true, props)
	if err != nil {
		t.Error(err)
		return
	}
	defer d.Close()
	print("PASS\n\n")
}

func zfsTestDatasetDestroy(t *testing.T) {
	println("TEST DATASET Destroy( ", TSTDatasetPath, " ) ... ")
	d, err := zfs.DatasetOpen(TSTDatasetPath)
	if err != nil {
		t.Error(err)
		return
	}
	defer d.Close()
	if err = d.DestroyRecursive(); err != nil {
		t.Error(err)
		return
	}
	print("PASS\n\n")
}

/* ------------------------------------------------------------------------- */
// EXAMPLES:

// Example of creating ZFS volume
func ExampleDatasetCreate() {
	// Create map to represent ZFS dataset properties. This is equivalent to
	// list of properties you can get from ZFS CLI tool, and some more
	// internally used by libzfs.
	props := make(map[zfs.Prop]zfs.Property)

	// I choose to create (block) volume 1GiB in size. Size is just ZFS dataset
	// property and this is done as map of strings. So, You have to either
	// specify size as base 10 number in string, or use strconv package or
	// similar to convert in to string (base 10) from numeric type.
	strSize := "1073741824"

	props[zfs.DatasetPropVolsize] = zfs.Property{Value: strSize}
	// In addition I explicitly choose some more properties to be set.
	props[zfs.DatasetPropVolblocksize] = zfs.Property{Value: "4096"}
	props[zfs.DatasetPropReservation] = zfs.Property{Value: strSize}

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
	}
	defer d.Close()
	var p zfs.Property
	if p, err = d.GetProperty(zfs.DatasetPropAvailable); err != nil {
		panic(err.Error())
	}
	println(zfs.DatasetPropertyToName(zfs.DatasetPropAvailable), " = ",
		p.Value)
}

func ExampleDatasetOpenAll() {
	datasets, err := zfs.DatasetOpenAll()
	if err != nil {
		panic(err.Error())
	}
	defer zfs.DatasetCloseAll(datasets)

	// Print out path and type of root datasets
	for _, d := range datasets {
		path, err := d.Path()
		if err != nil {
			panic(err.Error())
		}
		p, err := d.GetProperty(zfs.DatasetPropType)
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("%30s | %10s\n", path, p.Value)
	}

}
