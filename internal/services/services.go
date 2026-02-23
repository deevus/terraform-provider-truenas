package services

import (
	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/truenas-go/client"
)

// TrueNASServices holds typed service instances for all TrueNAS API namespaces.
// Resources and datasources access services through this registry.
type TrueNASServices struct {
	// Client provides backward-compatible access to the raw client.Client
	// for resources that haven't been migrated to typed services yet.
	// Remove this field once all resources use typed service methods.
	Client client.Client

	App        truenas.AppServiceAPI
	CloudSync  truenas.CloudSyncServiceAPI
	Cron       truenas.CronServiceAPI
	Dataset    truenas.DatasetServiceAPI
	Filesystem truenas.FilesystemServiceAPI
	Snapshot   truenas.SnapshotServiceAPI
	Virt       truenas.VirtServiceAPI
	VM         truenas.VMServiceAPI
}
