package zfs_test

import (
	"testing"
)

/* ------------------------------------------------------------------------- */
// TESTS ARE DEPENDED AND MUST RUN IN DEPENDENT ORDER

func Test(t *testing.T) {
	zpoolTestPoolCreate(t)
	zpoolTestPoolVDevTree(t)
	zpoolTestExport(t)
	zpoolTestPoolImportSearch(t)
	zpoolTestImport(t)
	zpoolTestExportForce(t)
	zpoolTestImportByGUID(t)
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
