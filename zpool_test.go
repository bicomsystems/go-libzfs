package zfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
)

const (
	TST_POOL_NAME    = "TESTPOOL"
	TST_DATASET_PATH = "TESTPOOL/DATASET"
)

func CreateTmpSparse(prefix string, size int64) (path string, err error) {
	sf, err := ioutil.TempFile("/tmp", prefix)
	if err != nil {
		return
	}
	defer sf.Close()
	if err = sf.Truncate(size); err != nil {
		return
	}
	path = sf.Name()
	return
}

// Create 3 sparse file 5G in /tmp directory each 5G size, and use them to create mirror TESTPOOL with one spare "disk"
func TestPoolCreate(t *testing.T) {
	print("TEST PoolCreate ... ")
	var s1path, s2path, s3path string
	var err error
	if s1path, err = CreateTmpSparse("zfs_test_", 0x140000000); err != nil {
		t.Error(err)
		return
	}
	if s2path, err = CreateTmpSparse("zfs_test_", 0x140000000); err != nil {
		// try cleanup
		os.Remove(s1path)
		t.Error(err)
		return
	}
	if s3path, err = CreateTmpSparse("zfs_test_", 0x140000000); err != nil {
		// try cleanup
		os.Remove(s1path)
		os.Remove(s2path)
		t.Error(err)
		return
	}
	disks := [2]string{s1path, s2path}

	var vdevs, mdevs, sdevs []VDevSpec
	for _, d := range disks {
		mdevs = append(mdevs,
			VDevSpec{Type: VDevTypeFile, Path: d})
	}
	sdevs = []VDevSpec{
		{Type: VDevTypeFile, Path: s3path}}
	vdevs = []VDevSpec{
		VDevSpec{Type: VDevTypeMirror, Devices: mdevs},
		VDevSpec{Type: VDevTypeSpare, Devices: sdevs},
	}

	props := make(map[PoolProp]string)
	fsprops := make(map[ZFSProp]string)
	features := make(map[string]string)
	fsprops[ZFSPropMountpoint] = "none"
	features["async_destroy"] = "enabled"
	features["empty_bpobj"] = "enabled"
	features["lz4_compress"] = "enabled"

	pool, err := PoolCreate(TST_POOL_NAME, vdevs, features, props, fsprops)
	if err != nil {
		t.Error(err)
		// try cleanup
		os.Remove(s1path)
		os.Remove(s2path)
		os.Remove(s3path)
		return
	}
	defer pool.Close()
	// try cleanup
	os.Remove(s1path)
	os.Remove(s2path)
	os.Remove(s3path)
	println("PASS")
}

// Open and list all pools and them state on the system
// Then list properties of last pool in the list
func TestPoolOpenAll(t *testing.T) {
	println("TEST PoolOpenAll() ... ")
	var pname string
	pools, err := PoolOpenAll()
	if err != nil {
		t.Error(err)
		return
	}
	println("\tThere is ", len(pools), " ZFS pools.")
	for _, p := range pools {
		pname, err = p.Name()
		if err != nil {
			t.Error(err)
			p.Close()
			return
		}
		pstate, err := p.State()
		if err != nil {
			t.Error(err)
			p.Close()
			return
		}
		println("\tPool: ", pname, " state: ", pstate)
		p.Close()
	}
	if len(pname) > 0 {
		// test open on last pool
		println("\tTry to open pool ", pname)
		p, err := PoolOpen(pname)
		if err != nil {
			t.Error(err)
			return
		}
		println("\tOpen pool: ", pname, " success")
		println("\t", pname, " PROPERTIES:")

		pc, _ := strconv.Atoi(p.Properties[PoolNumProps].Value)
		if len(p.Properties) != (pc + 1) {
			p.Close()
			t.Error(fmt.Sprint("Number of zpool properties does not match ",
				len(p.Properties), " != ", pc+1))
			return
		}
		for key, value := range p.Properties {
			pkey := PoolProp(key)
			println("\t\t", p.PropertyToName(pkey), " = ", value.Value, " <- ", value.Source)
		}
		for key, value := range p.Features {
			fmt.Printf("\t feature@%s = %s <- local\n", key, value)
		}
		if p.Properties[PoolPropListsnaps].Value == "off" {
			println("\tlistsnapshots to on")
			if err = p.SetProperty(PoolPropListsnaps, "on"); err != nil {
				t.Error(err)
			}
		} else {
			println("\tlistsnapshots to off")
			if err = p.SetProperty(PoolPropListsnaps, "off"); err != nil {
				t.Error(err)
			}
		}
		if err == nil {
			println("\tlistsnapshots", "is changed to ",
				p.Properties[PoolPropListsnaps].Value, " <- ",
				p.Properties[PoolPropListsnaps].Source)
		}
		p.Close()
	}
	println("PASS")
}

func TestDatasetCreate(t *testing.T) {
	print("TEST DatasetCreate(", TST_DATASET_PATH, ") ... ")
	props := make(map[ZFSProp]Property)
	d, err := DatasetCreate(TST_DATASET_PATH, DatasetTypeFilesystem, props)
	if err != nil {
		t.Error(err)
		return
	}
	d.Close()
	println("PASS")
}

func TestDatasetOpen(t *testing.T) {
	print("TEST DatasetOpen(", TST_DATASET_PATH, ") ... ")
	d, err := DatasetOpen(TST_DATASET_PATH)
	if err != nil {
		t.Error(err)
		return
	}
	d.Close()
	println("PASS")
}

func printDatasets(ds []Dataset) error {
	for _, d := range ds {
		path, err := d.Path()
		if err != nil {
			return err
		}
		println("\t", path)
		if len(d.Children) > 0 {
			printDatasets(d.Children)
		}
	}
	return nil
}

func TestDatasetOpenAll(t *testing.T) {
	println("TEST DatasetOpenAll()/DatasetCloseAll() ... ")
	ds, err := DatasetOpenAll()
	if err != nil {
		t.Error(err)
		return
	}
	if err = printDatasets(ds); err != nil {
		DatasetCloseAll(ds)
		t.Error(err)
		return
	}
	DatasetCloseAll(ds)
	println("PASS")
}

func TestDatasetDestroy(t *testing.T) {
	print("TEST DATASET Destroy()", TST_DATASET_PATH, " ... ")
	d, err := DatasetOpen(TST_DATASET_PATH)
	if err != nil {
		t.Error(err)
		return
	}
	defer d.Close()
	if err = d.Destroy(false); err != nil {
		t.Error(err)
		return
	}
	println("PASS")
}

func TestPoolDestroy(t *testing.T) {
	print("TEST POOL Destroy()", TST_POOL_NAME, " ... ")
	p, err := PoolOpen(TST_POOL_NAME)
	if err != nil {
		t.Error(err)
		return
	}
	defer p.Close()
	if err = p.Destroy("Test of pool destroy (" + TST_POOL_NAME + ")"); err != nil {
		t.Error(err.Error())
		return
	}
	println("PASS")
}

func TestFailPoolOpen(t *testing.T) {
	print("TEST failing to open pool ... ")
	pname := "fail to open this pool"
	p, err := PoolOpen(pname)
	if err != nil {
		println("PASS")
		return
	}
	t.Error("PoolOpen pass when it should fail")
	p.Close()
}

func ExamplePoolProp() {
	if pool, err := PoolOpen("SSD"); err == nil {
		print("Pool size is: ", pool.Properties[PoolPropSize].Value)
		// Turn on snapshot listing for pool
		pool.SetProperty(PoolPropListsnaps, "on")
	} else {
		print("Error: ", err)
	}
}

// Open and list all pools on system with them properties
func ExamplePoolOpenAll() {
	// Lets open handles to all active pools on system
	pools, err := PoolOpenAll()
	if err != nil {
		println(err)
	}

	// Print each pool name and properties
	for _, p := range pools {
		// Print fancy header
		fmt.Printf("\n -----------------------------------------------------------\n")
		fmt.Printf("   POOL: %49s   \n", p.Properties[PoolPropName].Value)
		fmt.Printf("|-----------------------------------------------------------|\n")
		fmt.Printf("|  PROPERTY      |  VALUE                |  SOURCE          |\n")
		fmt.Printf("|-----------------------------------------------------------|\n")

		// Iterate pool properties and print name, value and source
		for key, prop := range p.Properties {
			pkey := PoolProp(key)
			if pkey == PoolPropName {
				continue // Skip name its already printed above
			}
			fmt.Printf("|%14s  | %20s  | %15s  |\n", p.PropertyToName(pkey),
				prop.Value, prop.Source)
			println("")
		}
		println("")

		// Close pool handle and free memory, since it will not be used anymore
		p.Close()
	}
}

func ExamplePoolCreate() {
	disks := [2]string{"/dev/disk/by-id/ATA-123", "/dev/disk/by-id/ATA-456"}

	var vdevs, mdevs, sdevs []VDevSpec

	// build mirror devices specs
	for _, d := range disks {
		mdevs = append(mdevs,
			VDevSpec{Type: VDevTypeDisk, Path: d})
	}

	// spare device specs
	sdevs = []VDevSpec{
		{Type: VDevTypeDisk, Path: "/dev/disk/by-id/ATA-789"}}

	// pool specs
	vdevs = []VDevSpec{
		VDevSpec{Type: VDevTypeMirror, Devices: mdevs},
		VDevSpec{Type: VDevTypeSpare, Devices: sdevs},
	}

	// pool properties
	props := make(map[PoolProp]string)
	// root dataset filesystem properties
	fsprops := make(map[ZFSProp]string)
	// pool features
	features := make(map[string]string)

	// Turn off auto mounting by ZFS
	fsprops[ZFSPropMountpoint] = "none"

	// Enable some features
	features["async_destroy"] = "enabled"
	features["empty_bpobj"] = "enabled"
	features["lz4_compress"] = "enabled"

	// Based on specs formed above create test pool as 2 disk mirror and
	// one spare disk
	pool, err := PoolCreate("TESTPOOL", vdevs, features, props, fsprops)
	if err != nil {
		println("Error: ", err.Error())
		return
	}
	defer pool.Close()
}

func ExamplePool_Destroy() {
	pname := "TESTPOOL"

	// Need handle to pool at first place
	p, err := PoolOpen(pname)
	if err != nil {
		println("Error: ", err.Error())
		return
	}

	// Make sure pool handle is free after we are done here
	defer p.Close()

	if err = p.Destroy("Example of pool destroy (TESTPOOL)"); err != nil {
		println("Error: ", err.Error())
		return
	}
}

// Example of creating ZFS volume
func ExampleDatasetCreate() {
	// Create map to represent ZFS dataset properties. This is equivalent to
	// list of properties you can get from ZFS CLI tool, and some more
	// internally used by libzfs.
	props := make(map[ZFSProp]Property)

	// I choose to create (block) volume 1GiB in size. Size is just ZFS dataset
	// property and this is done as map of strings. So, You have to either
	// specify size as base 10 number in string, or use strconv package or
	// similar to convert in to string (base 10) from numeric type.
	strSize := "1073741824"

	props[ZFSPropVolsize] = Property{Value: strSize}
	// In addition I explicitly choose some more properties to be set.
	props[ZFSPropVolblocksize] = Property{Value: "4096"}
	props[ZFSPropReservation] = Property{Value: strSize}

	// Lets create desired volume
	d, err := DatasetCreate("TESTPOOL/VOLUME1", DatasetTypeVolume, props)
	if err != nil {
		println(err.Error())
		return
	}
	// Dataset have to be closed for memory cleanup
	defer d.Close()

	println("Created zfs volume TESTPOOL/VOLUME1")
}
