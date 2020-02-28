package zfs_test

import (
	"testing"

	zfs "github.com/bicomsystems/go-libzfs"
)

func TestDataset_DestroyPromote(t *testing.T) {
	zpoolTestPoolCreate(t)
	// defer zpoolTestPoolDestroy(t)
	var c1, c2 zfs.Dataset

	props := make(map[zfs.Prop]zfs.Property)

	d, err := zfs.DatasetCreate(TSTPoolName+"/original",
		zfs.DatasetTypeFilesystem, make(map[zfs.Prop]zfs.Property))
	if err != nil {
		t.Errorf("DatasetCreate(\"%s/original\") error: %v", TSTPoolName, err)
		return
	}

	s1, _ := zfs.DatasetSnapshot(d.Properties[zfs.DatasetPropName].Value+"@snap2", false, props, nil)
	s2, _ := zfs.DatasetSnapshot(d.Properties[zfs.DatasetPropName].Value+"@snap1", false, props, nil)

	c1, err = s1.Clone(TSTPoolName+"/clone1", nil)
	if err != nil {
		t.Errorf("d.Clone(\"%s/clone1\", props)) error: %v", TSTPoolName, err)
		d.Close()
		return
	}

	zfs.DatasetSnapshot(c1.Properties[zfs.DatasetPropName].Value+"@snap1", false, props, nil)

	c2, err = s2.Clone(TSTPoolName+"/clone2", nil)
	if err != nil {
		t.Errorf("c1.Clone(\"%s/clone1\", props)) error: %v", TSTPoolName, err)
		d.Close()
		c1.Close()
		return
	}
	s2.Close()

	zfs.DatasetSnapshot(c2.Properties[zfs.DatasetPropName].Value+"@snap0", false, props, nil)
	c1.Close()
	c2.Close()

	// reopen pool
	d.Close()
	if d, err = zfs.DatasetOpen(TSTPoolName + "/original"); err != nil {
		t.Error("zfs.DatasetOpen")
		return
	}

	if err = d.DestroyPromote(); err != nil {
		t.Errorf("DestroyPromote error: %v", err)
		d.Close()
		return
	}
	t.Log("Destroy promote completed with success")
	d.Close()
	zpoolTestPoolDestroy(t)
}
