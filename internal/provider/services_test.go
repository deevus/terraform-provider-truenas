package provider

import (
	"testing"

	truenas "github.com/deevus/truenas-go"
)

func TestTrueNASServices_FieldTypes(t *testing.T) {
	// Verify TrueNASServices accepts interface types (compile-time check)
	_ = &TrueNASServices{
		App:        &truenas.MockAppService{},
		CloudSync:  &truenas.MockCloudSyncService{},
		Cron:       &truenas.MockCronService{},
		Dataset:    &truenas.MockDatasetService{},
		Filesystem: &truenas.MockFilesystemService{},
		Snapshot:   &truenas.MockSnapshotService{},
		Virt:       &truenas.MockVirtService{},
		VM:         &truenas.MockVMService{},
	}
}
