# truenas_snapshot Resource

Manages a ZFS snapshot for pre-upgrade backups and point-in-time recovery.

## Example Usage

```hcl
resource "truenas_dataset" "app_data" {
  pool = "tank"
  path = "apps/myapp"
}

resource "truenas_snapshot" "pre_upgrade" {
  dataset_id = truenas_dataset.app_data.id
  name       = "pre-v2-upgrade"
  hold       = true
}
```

## Argument Reference

* `dataset_id` - (Required) Dataset ID to snapshot. Reference a truenas_dataset resource or data source.
* `name` - (Required) Snapshot name.
* `hold` - (Optional) Prevent automatic deletion. Default: false.
* `recursive` - (Optional) Include child datasets. Default: false.

## Attribute Reference

* `id` - Snapshot identifier (dataset@name).
* `createtxg` - Transaction group when created.
* `used_bytes` - Space consumed by snapshot.
* `referenced_bytes` - Space referenced by snapshot.

## Import

Snapshots can be imported using the snapshot ID:

```bash
terraform import truenas_snapshot.example "tank/data@snap1"
```
