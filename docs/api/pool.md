# Pool & Dataset API

ZFS pool and dataset management operations.

## Pool Operations

### pool.query
Query pool information.
```bash
midclt call pool.query
midclt call pool.query '[[["name", "=", "tank"]]]'
midclt call pool.query '[]' '{"extra": {"is_upgraded": true}}'
```

### pool.create
Create a new pool.
```bash
midclt call pool.create '{
  "name": "tank",
  "topology": {
    "data": [{"type": "STRIPE", "disks": ["sda", "sdb"]}]
  }
}'
```

### pool.update
Update pool properties.
```bash
midclt call pool.update <pool_id> '{"autotrim": "ON"}'
```

### pool.export
Export a pool.
```bash
midclt call pool.export <pool_id>
midclt call pool.export <pool_id> '{"destroy": true}'
```

### pool.import_find
Find pools available for import.
```bash
midclt call pool.import_find
```

### pool.import_pool
Import a pool.
```bash
midclt call pool.import_pool '{"guid": "<pool_guid>", "name": "tank"}'
```

### pool.attach
Attach a disk to a vdev (mirror).
```bash
midclt call pool.attach <pool_id> '{
  "target_vdev": "<vdev_guid>",
  "new_disk": "sdc"
}'
```

### pool.detach
Detach a disk from a mirror.
```bash
midclt call pool.detach <pool_id> '{"label": "<disk_label>"}'
```

### pool.replace
Replace a disk in a pool.
```bash
midclt call pool.replace <pool_id> '{
  "label": "<old_disk_label>",
  "disk": "sdd"
}'
```

### pool.remove
Remove a vdev from a pool.
```bash
midclt call pool.remove <pool_id> '{"label": "<vdev_label>"}'
```

### pool.offline
Take a disk offline.
```bash
midclt call pool.offline <pool_id> '{"label": "<disk_label>"}'
```

### pool.online
Bring a disk online.
```bash
midclt call pool.online <pool_id> '{"label": "<disk_label>"}'
```

### pool.expand
Expand a pool after disk replacement.
```bash
midclt call pool.expand <pool_id>
```

### pool.upgrade
Upgrade pool to latest ZFS features.
```bash
midclt call pool.upgrade <pool_id>
```

### pool.scan
Start a scrub or resilver.
```bash
midclt call pool.scan <pool_id> 'SCRUB'
midclt call pool.scan <pool_id> 'SCRUB' 'PAUSE'
midclt call pool.scan <pool_id> 'SCRUB' 'STOP'
```

### pool.processes
Get processes using a pool.
```bash
midclt call pool.processes <pool_id>
```

### pool.attachments
Get attachments using a pool.
```bash
midclt call pool.attachments <pool_id>
```

### pool.validate_name
Validate a pool name.
```bash
midclt call pool.validate_name "poolname"
```

### pool.filesystem_choices
Get available filesystem choices.
```bash
midclt call pool.filesystem_choices
```

### pool.reimport
Reimport a pool.
```bash
midclt call pool.reimport <pool_id>
```

### pool.ddt_prune
Prune deduplication table.
```bash
midclt call pool.ddt_prune <pool_id>
```

## Dataset Operations

### pool.dataset.query
Query datasets.
```bash
midclt call pool.dataset.query
midclt call pool.dataset.query '[[["id", "=", "tank/data"]]]'
midclt call pool.dataset.query '[]' '{"extra": {"properties": ["used", "available"]}}'
```

### pool.dataset.create
Create a dataset.
```bash
midclt call pool.dataset.create '{
  "name": "tank/newdataset",
  "type": "FILESYSTEM",
  "compression": "LZ4",
  "atime": "OFF"
}'
```

Create a zvol:
```bash
midclt call pool.dataset.create '{
  "name": "tank/zvol1",
  "type": "VOLUME",
  "volsize": 10737418240,
  "volblocksize": "16K"
}'
```

### pool.dataset.update
Update dataset properties.
```bash
midclt call pool.dataset.update "tank/data" '{
  "compression": "ZSTD",
  "quota": 107374182400
}'
```

### pool.dataset.delete
Delete a dataset.
```bash
midclt call pool.dataset.delete "tank/data"
midclt call pool.dataset.delete "tank/data" '{"recursive": true}'
```

### pool.dataset.promote
Promote a clone to a standalone dataset.
```bash
midclt call pool.dataset.promote "tank/clone"
```

### pool.dataset.details
Get detailed dataset information.
```bash
midclt call pool.dataset.details
```

### pool.dataset.attachments
Get attachments using a dataset.
```bash
midclt call pool.dataset.attachments "tank/data"
```

### pool.dataset.processes
Get processes using a dataset.
```bash
midclt call pool.dataset.processes "tank/data"
```

### pool.dataset.compression_choices
Get available compression algorithms.
```bash
midclt call pool.dataset.compression_choices
```

### pool.dataset.checksum_choices
Get available checksum algorithms.
```bash
midclt call pool.dataset.checksum_choices
```

### pool.dataset.recordsize_choices
Get available record sizes.
```bash
midclt call pool.dataset.recordsize_choices
```

### pool.dataset.encryption_algorithm_choices
Get available encryption algorithms.
```bash
midclt call pool.dataset.encryption_algorithm_choices
```

### pool.dataset.recommended_zvol_blocksize
Get recommended zvol block size for a pool.
```bash
midclt call pool.dataset.recommended_zvol_blocksize "tank"
```

## Dataset Encryption

### pool.dataset.encryption_summary
Get encryption summary for a dataset.
```bash
midclt call pool.dataset.encryption_summary "tank/encrypted"
```

### pool.dataset.lock
Lock an encrypted dataset.
```bash
midclt call pool.dataset.lock "tank/encrypted"
```

### pool.dataset.unlock
Unlock an encrypted dataset.
```bash
midclt call pool.dataset.unlock "tank/encrypted" '{
  "datasets": [{"name": "tank/encrypted", "passphrase": "secret"}]
}'
```

### pool.dataset.change_key
Change encryption key.
```bash
midclt call pool.dataset.change_key "tank/encrypted" '{
  "passphrase": "newpassphrase"
}'
```

### pool.dataset.export_key
Export encryption key.
```bash
midclt call pool.dataset.export_key "tank/encrypted"
```

### pool.dataset.inherit_parent_encryption_properties
Inherit encryption from parent.
```bash
midclt call pool.dataset.inherit_parent_encryption_properties "tank/child"
```

## Dataset Quotas

### pool.dataset.get_quota
Get quotas for a dataset.
```bash
midclt call pool.dataset.get_quota "tank/data" "USER"
midclt call pool.dataset.get_quota "tank/data" "GROUP"
midclt call pool.dataset.get_quota "tank/data" "DATASET"
midclt call pool.dataset.get_quota "tank/data" "PROJECT"
```

### pool.dataset.set_quota
Set quotas for a dataset.
```bash
midclt call pool.dataset.set_quota "tank/data" '[{
  "quota_type": "USER",
  "id": "1000",
  "quota_value": 10737418240
}]'
```

## Snapshots

### pool.snapshot.query
Query snapshots.
```bash
midclt call pool.snapshot.query
midclt call pool.snapshot.query '[[["dataset", "=", "tank/data"]]]'
```

### pool.snapshot.create
Create a snapshot.
```bash
midclt call pool.snapshot.create '{
  "dataset": "tank/data",
  "name": "snap1"
}'
```

### pool.snapshot.delete
Delete a snapshot.
```bash
midclt call pool.snapshot.delete "tank/data@snap1"
```

### pool.snapshot.clone
Clone a snapshot.
```bash
midclt call pool.snapshot.clone '{
  "snapshot": "tank/data@snap1",
  "dataset_dst": "tank/clone"
}'
```

### pool.snapshot.rollback
Rollback to a snapshot.
```bash
midclt call pool.snapshot.rollback "tank/data@snap1"
midclt call pool.snapshot.rollback "tank/data@snap1" '{"recursive": true}'
```

### pool.snapshot.hold
Place a hold on a snapshot.
```bash
midclt call pool.snapshot.hold "tank/data@snap1"
```

### pool.snapshot.release
Release a hold on a snapshot.
```bash
midclt call pool.snapshot.release "tank/data@snap1"
```

## Snapshot Tasks

### pool.snapshottask.query
Query snapshot tasks.
```bash
midclt call pool.snapshottask.query
```

### pool.snapshottask.create
Create a snapshot task.
```bash
midclt call pool.snapshottask.create '{
  "dataset": "tank/data",
  "recursive": true,
  "lifetime_value": 2,
  "lifetime_unit": "WEEK",
  "naming_schema": "auto-%Y-%m-%d_%H-%M",
  "schedule": {"minute": "0", "hour": "0", "dom": "*", "month": "*", "dow": "*"}
}'
```

### pool.snapshottask.update
Update a snapshot task.
```bash
midclt call pool.snapshottask.update <task_id> '{"enabled": false}'
```

### pool.snapshottask.delete
Delete a snapshot task.
```bash
midclt call pool.snapshottask.delete <task_id>
```

## Scrub Tasks

### pool.scrub.query
Query scrub tasks.
```bash
midclt call pool.scrub.query
```

### pool.scrub.create
Create a scrub task.
```bash
midclt call pool.scrub.create '{
  "pool": 1,
  "threshold": 35,
  "schedule": {"minute": "0", "hour": "0", "dom": "1", "month": "*", "dow": "*"}
}'
```

### pool.scrub.update
Update a scrub task.
```bash
midclt call pool.scrub.update <task_id> '{"enabled": false}'
```

## Resilver Configuration

### pool.resilver.config
Get resilver configuration.
```bash
midclt call pool.resilver.config
```

### pool.resilver.update
Update resilver configuration.
```bash
midclt call pool.resilver.update '{
  "enabled": true,
  "begin": "18:00",
  "end": "09:00",
  "weekday": [1, 2, 3, 4, 5]
}'
```
