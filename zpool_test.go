package zfs_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/bicomsystems/go-libzfs"
)

/* ------------------------------------------------------------------------- */
// HELPERS:

var TSTPoolName = "TESTPOOL"
var TSTPoolGUID string

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

var s1path, s2path, s3path string

// This will create sparse files in tmp directory,
// for purpose of creating test pool.
func createTestpoolVdisks() (err error) {
	if s1path, err = CreateTmpSparse("zfs_test_", 0x140000000); err != nil {
		return
	}
	if s2path, err = CreateTmpSparse("zfs_test_", 0x140000000); err != nil {
		// try cleanup
		os.Remove(s1path)
		return
	}
	if s3path, err = CreateTmpSparse("zfs_test_", 0x140000000); err != nil {
		// try cleanup
		os.Remove(s1path)
		os.Remove(s2path)
		return
	}
	return
}

// Cleanup sparse files used for tests
func cleanupVDisks() {
	// try cleanup
	os.Remove(s1path)
	os.Remove(s2path)
	os.Remove(s3path)
}

/* ------------------------------------------------------------------------- */
// TESTS:

// Create 3 sparse file in /tmp directory each 5G size, and use them to create
// mirror TESTPOOL with one spare "disk"
func zpoolTestPoolCreate(t *testing.T) {
	println("TEST PoolCreate ... ")
	// first check if pool with same name already exist
	// we don't want conflict
	for {
		p, err := zfs.PoolOpen(TSTPoolName)
		if err != nil {
			break
		}
		p.Close()
		TSTPoolName += "0"
	}
	var err error

	if err = createTestpoolVdisks(); err != nil {
		t.Error(err)
		return
	}

	disks := [2]string{s1path, s2path}

	var vdevs, mdevs, sdevs []zfs.VDevTree
	for _, d := range disks {
		mdevs = append(mdevs,
			zfs.VDevTree{Type: zfs.VDevTypeFile, Path: d})
	}
	sdevs = []zfs.VDevTree{
		{Type: zfs.VDevTypeFile, Path: s3path}}
	vdevs = []zfs.VDevTree{
		zfs.VDevTree{Type: zfs.VDevTypeMirror, Devices: mdevs},
		zfs.VDevTree{Type: zfs.VDevTypeSpare, Devices: sdevs},
	}

	props := make(map[zfs.Prop]string)
	fsprops := make(map[zfs.Prop]string)
	features := make(map[string]string)
	fsprops[zfs.DatasetPropMountpoint] = "none"
	features["async_destroy"] = "enabled"
	features["empty_bpobj"] = "enabled"
	features["lz4_compress"] = "enabled"

	pool, err := zfs.PoolCreate(TSTPoolName, vdevs, features, props, fsprops)
	if err != nil {
		t.Error(err)
		// try cleanup
		os.Remove(s1path)
		os.Remove(s2path)
		os.Remove(s3path)
		return
	}
	defer pool.Close()

	pguid, _ := pool.GetProperty(zfs.PoolPropGUID)
	TSTPoolGUID = pguid.Value

	print("PASS\n\n")
}

// Open and list all pools and them state on the system
// Then list properties of last pool in the list
func zpoolTestPoolOpenAll(t *testing.T) {
	println("TEST PoolOpenAll() ... ")
	var pname string
	pools, err := zfs.PoolOpenAll()
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
	print("PASS\n\n")
}

func zpoolTestPoolDestroy(t *testing.T) {
	println("TEST POOL Destroy( ", TSTPoolName, " ) ... ")
	p, err := zfs.PoolOpen(TSTPoolName)
	if err != nil {
		t.Error(err)
		return
	}
	defer p.Close()
	if err = p.Destroy("Test of pool destroy (" + TSTPoolName + ")"); err != nil {
		t.Error(err.Error())
		return
	}
	print("PASS\n\n")
}

func zpoolTestFailPoolOpen(t *testing.T) {
	println("TEST open of non existing pool ... ")
	pname := "fail to open this pool"
	p, err := zfs.PoolOpen(pname)
	if err != nil {
		print("PASS\n\n")
		return
	}
	t.Error("PoolOpen pass when it should fail")
	p.Close()
}

func zpoolTestExport(t *testing.T) {
	println("TEST POOL Export( ", TSTPoolName, " ) ... ")
	p, err := zfs.PoolOpen(TSTPoolName)
	if err != nil {
		t.Error(err)
		return
	}
	p.Export(false, "Test exporting pool")
	defer p.Close()
	print("PASS\n\n")
}

func zpoolTestExportForce(t *testing.T) {
	println("TEST POOL ExportForce( ", TSTPoolName, " ) ... ")
	p, err := zfs.PoolOpen(TSTPoolName)
	if err != nil {
		t.Error(err)
		return
	}
	p.ExportForce("Test force exporting pool")
	defer p.Close()
	print("PASS\n\n")
}

func zpoolTestImport(t *testing.T) {
	println("TEST POOL Import( ", TSTPoolName, " ) ... ")
	p, err := zfs.PoolImport(TSTPoolName, []string{"/tmp"})
	if err != nil {
		t.Error(err)
		return
	}
	defer p.Close()
	print("PASS\n\n")
}

func zpoolTestImportByGUID(t *testing.T) {
	println("TEST POOL ImportByGUID( ", TSTPoolGUID, " ) ... ")
	p, err := zfs.PoolImportByGUID(TSTPoolGUID, []string{"/tmp"})
	if err != nil {
		t.Error(err)
		return
	}
	defer p.Close()
	print("PASS\n\n")
}

func printVDevTree(vt zfs.VDevTree, pref string) {
	first := pref + vt.Name
	fmt.Printf("%-30s | %-10s | %-10s | %s\n", first, vt.Type,
		vt.Stat.State.String(), vt.Path)
	for _, v := range vt.Devices {
		printVDevTree(v, "  "+pref)
	}
}

func zpoolTestPoolImportSearch(t *testing.T) {
	println("TEST PoolImportSearch")
	pools, err := zfs.PoolImportSearch([]string{"/tmp"})
	if err != nil {
		t.Error(err.Error())
		return
	}
	for _, p := range pools {
		println()
		println("---------------------------------------------------------------")
		println("pool: ", p.Name)
		println("guid: ", p.GUID)
		println("state: ", p.State.String())
		fmt.Printf("%-30s | %-10s | %-10s | %s\n", "NAME", "TYPE", "STATE", "PATH")
		println("---------------------------------------------------------------")
		printVDevTree(p.VDevs, "")

	}
	print("PASS\n\n")
}

func zpoolTestPoolProp(t *testing.T) {
	println("TEST PoolProp on ", TSTPoolName, " ... ")
	if pool, err := zfs.PoolOpen(TSTPoolName); err == nil {
		defer pool.Close()
		// Turn on snapshot listing for pool
		pool.SetProperty(zfs.PoolPropListsnaps, "on")
		// Verify change is succesfull
		if pool.Properties[zfs.PoolPropListsnaps].Value != "on" {
			t.Error(fmt.Errorf("Update of pool property failed"))
			return
		}

		// Test fetching property
		propHealth, err := pool.GetProperty(zfs.PoolPropHealth)
		if err != nil {
			t.Error(err)
			return
		}
		println("Pool property health: ", propHealth.Value)

		propGUID, err := pool.GetProperty(zfs.PoolPropGUID)
		if err != nil {
			t.Error(err)
			return
		}
		println("Pool property GUID: ", propGUID.Value)

		// this test pool should not be bootable
		prop, err := pool.GetProperty(zfs.PoolPropBootfs)
		if err != nil {
			t.Error(err)
			return
		}
		if prop.Value != "-" {
			t.Errorf("Failed at bootable fs property evaluation")
			return
		}

		// fetch all properties
		if err = pool.ReloadProperties(); err != nil {
			t.Error(err)
			return
		}
	} else {
		t.Error(err)
		return
	}
	print("PASS\n\n")
}

func zpoolTestPoolStatusAndState(t *testing.T) {
	println("TEST pool Status/State ( ", TSTPoolName, " ) ... ")
	pool, err := zfs.PoolOpen(TSTPoolName)
	if err != nil {
		t.Error(err.Error())
		return
	}
	defer pool.Close()

	if _, err = pool.Status(); err != nil {
		t.Error(err.Error())
		return
	}

	var pstate zfs.PoolState
	if pstate, err = pool.State(); err != nil {
		t.Error(err.Error())
		return
	}
	println("POOL", TSTPoolName, "state:", zfs.PoolStateToName(pstate))

	print("PASS\n\n")
}

func zpoolTestPoolVDevTree(t *testing.T) {
	var vdevs zfs.VDevTree
	println("TEST pool VDevTree ( ", TSTPoolName, " ) ... ")
	pool, err := zfs.PoolOpen(TSTPoolName)
	if err != nil {
		t.Error(err.Error())
		return
	}
	defer pool.Close()
	vdevs, err = pool.VDevTree()
	if err != nil {
		t.Error(err.Error())
		return
	}
	fmt.Printf("%-30s | %-10s | %-10s | %s\n", "NAME", "TYPE", "STATE", "PATH")
	println("---------------------------------------------------------------")
	printVDevTree(vdevs, "")
	print("PASS\n\n")
}

/* ------------------------------------------------------------------------- */
// EXAMPLES:

func ExamplePoolProp() {
	if pool, err := zfs.PoolOpen("SSD"); err == nil {
		print("Pool size is: ", pool.Properties[zfs.PoolPropSize].Value)
		// Turn on snapshot listing for pool
		pool.SetProperty(zfs.PoolPropListsnaps, "on")
		println("Changed property",
			zfs.PoolPropertyToName(zfs.PoolPropListsnaps), "to value:",
			pool.Properties[zfs.PoolPropListsnaps].Value)

		prop, err := pool.GetProperty(zfs.PoolPropHealth)
		if err != nil {
			panic(err)
		}
		println("Update and print out pool health:", prop.Value)
	} else {
		print("Error: ", err)
	}
}

// Open and list all pools on system with them properties
func ExamplePoolOpenAll() {
	// Lets open handles to all active pools on system
	pools, err := zfs.PoolOpenAll()
	if err != nil {
		println(err)
	}

	// Print each pool name and properties
	for _, p := range pools {
		// Print fancy header
		fmt.Printf("\n -----------------------------------------------------------\n")
		fmt.Printf("   POOL: %49s   \n", p.Properties[zfs.PoolPropName].Value)
		fmt.Printf("|-----------------------------------------------------------|\n")
		fmt.Printf("|  PROPERTY      |  VALUE                |  SOURCE          |\n")
		fmt.Printf("|-----------------------------------------------------------|\n")

		// Iterate pool properties and print name, value and source
		for key, prop := range p.Properties {
			pkey := zfs.Prop(key)
			if pkey == zfs.PoolPropName {
				continue // Skip name its already printed above
			}
			fmt.Printf("|%14s  | %20s  | %15s  |\n",
				zfs.PoolPropertyToName(pkey),
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

	var vdevs, mdevs, sdevs []zfs.VDevTree

	// build mirror devices specs
	for _, d := range disks {
		mdevs = append(mdevs,
			zfs.VDevTree{Type: zfs.VDevTypeDisk, Path: d})
	}

	// spare device specs
	sdevs = []zfs.VDevTree{
		{Type: zfs.VDevTypeDisk, Path: "/dev/disk/by-id/ATA-789"}}

	// pool specs
	vdevs = []zfs.VDevTree{
		zfs.VDevTree{Type: zfs.VDevTypeMirror, Devices: mdevs},
		zfs.VDevTree{Type: zfs.VDevTypeSpare, Devices: sdevs},
	}

	// pool properties
	props := make(map[zfs.Prop]string)
	// root dataset filesystem properties
	fsprops := make(map[zfs.Prop]string)
	// pool features
	features := make(map[string]string)

	// Turn off auto mounting by ZFS
	fsprops[zfs.DatasetPropMountpoint] = "none"

	// Enable some features
	features["async_destroy"] = "enabled"
	features["empty_bpobj"] = "enabled"
	features["lz4_compress"] = "enabled"

	// Based on specs formed above create test pool as 2 disk mirror and
	// one spare disk
	pool, err := zfs.PoolCreate("TESTPOOL", vdevs, features, props, fsprops)
	if err != nil {
		println("Error: ", err.Error())
		return
	}
	defer pool.Close()
}

func ExamplePool_Destroy() {
	pname := "TESTPOOL"

	// Need handle to pool at first place
	p, err := zfs.PoolOpen(pname)
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

func ExamplePoolImport() {
	p, err := zfs.PoolImport("TESTPOOL", []string{"/dev/disk/by-id"})
	if err != nil {
		panic(err)
	}
	p.Close()
}

func ExamplePool_Export() {
	p, err := zfs.PoolOpen("TESTPOOL")
	if err != nil {
		panic(err)
	}
	defer p.Close()
	if err = p.Export(false, "Example exporting pool"); err != nil {
		panic(err)
	}
}

func ExamplePool_ExportForce() {
	p, err := zfs.PoolOpen("TESTPOOL")
	if err != nil {
		panic(err)
	}
	defer p.Close()
	if err = p.ExportForce("Example exporting pool"); err != nil {
		panic(err)
	}
}

func ExamplePool_State() {
	p, err := zfs.PoolOpen("TESTPOOL")
	if err != nil {
		panic(err)
	}
	defer p.Close()
	pstate, err := p.State()
	if err != nil {
		panic(err)
	}
	println("POOL TESTPOOL state:", zfs.PoolStateToName(pstate))
}
