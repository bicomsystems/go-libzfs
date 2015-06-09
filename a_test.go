package zfs_test

import (
	"testing"
)

/* ------------------------------------------------------------------------- */
// TESTS ARE DEPENDED AND MUST RUN IN DEPENDENT ORDER

func Test(t *testing.T) {
	zpoolTestPoolCreate(t)
	zpoolTestExport(t)
	zpoolTestImport(t)
	zpoolTestExportForce(t)
	zpoolTestImport(t)
	zpoolTestPoolProp(t)
	zpoolTestPoolStatusAndState(t)
	zpoolTestPoolOpenAll(t)
	zpoolTestFailPoolOpen(t)

	zfsTestDatasetCreate(t)
	zfsTestDatasetOpen(t)
	zfsTestDatasetSnapshot(t)
	zfsTestDatasetOpenAll(t)

	zfsTestDatasetDestroy(t)

	zpoolTestPoolDestroy(t)

	cleanupVDisks()
}
