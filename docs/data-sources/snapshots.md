# truenas_snapshots Data Source

Retrieves snapshots for a dataset.

## Example Usage

```hcl
data "truenas_snapshots" "backups" {
  dataset_id   = truenas_dataset.app_data.id
  name_pattern = "pre-*"
}

output "backup_count" {
  value = length(data.truenas_snapshots.backups.snapshots)
}
```

## Argument Reference

* `dataset_id` - (Required) Dataset ID to query snapshots for.
* `recursive` - (Optional) Include child dataset snapshots. Default: false.
* `name_pattern` - (Optional) Glob pattern to filter snapshot names.

## Attribute Reference

* `snapshots` - List of snapshot objects with:
  * `id` - Snapshot ID (dataset@name).
  * `name` - Snapshot name.
  * `dataset_id` - Parent dataset ID.
  * `used_bytes` - Space consumed.
  * `referenced_bytes` - Space referenced.
  * `hold` - Whether held.
