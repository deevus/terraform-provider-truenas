package provider

import truenas "github.com/deevus/truenas-go"

// TrueNASServices holds typed service instances for all TrueNAS API namespaces.
// Resources and datasources access services through this registry.
type TrueNASServices struct {
	App        truenas.AppServiceAPI
	CloudSync  truenas.CloudSyncServiceAPI
	Cron       truenas.CronServiceAPI
	Dataset    truenas.DatasetServiceAPI
	Filesystem truenas.FilesystemServiceAPI
	Snapshot   truenas.SnapshotServiceAPI
	Virt       truenas.VirtServiceAPI
	VM         truenas.VMServiceAPI
}
