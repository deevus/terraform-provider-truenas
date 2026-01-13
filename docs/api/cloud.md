# Cloud Services API

Cloud sync and cloud backup operations.

## Cloud Sync Credentials

### cloudsync.credentials.query
Query cloud sync credentials.
```bash
midclt call cloudsync.credentials.query
```

Returns:
- `id` - Credential ID
- `name` - Credential name
- `provider` - Provider type
- `attributes` - Provider-specific attributes

### cloudsync.credentials.create
Create cloud sync credentials.

S3-compatible:
```bash
midclt call cloudsync.credentials.create '{
  "name": "AWS S3",
  "provider": "S3",
  "attributes": {
    "access_key_id": "AKIAXXXXXXXX",
    "secret_access_key": "secretkey",
    "endpoint": "",
    "region": "us-east-1"
  }
}'
```

Backblaze B2:
```bash
midclt call cloudsync.credentials.create '{
  "name": "Backblaze B2",
  "provider": "B2",
  "attributes": {
    "account": "accountid",
    "key": "applicationkey"
  }
}'
```

Google Cloud Storage:
```bash
midclt call cloudsync.credentials.create '{
  "name": "GCS",
  "provider": "GOOGLE_CLOUD_STORAGE",
  "attributes": {
    "service_account_credentials": "{...json...}"
  }
}'
```

Azure Blob:
```bash
midclt call cloudsync.credentials.create '{
  "name": "Azure",
  "provider": "AZUREBLOB",
  "attributes": {
    "account": "storageaccount",
    "key": "accountkey"
  }
}'
```

### cloudsync.credentials.update
Update cloud sync credentials.
```bash
midclt call cloudsync.credentials.update <cred_id> '{"name": "Updated name"}'
```

### cloudsync.credentials.delete
Delete cloud sync credentials.
```bash
midclt call cloudsync.credentials.delete <cred_id>
```

### cloudsync.credentials.verify
Verify credentials work.
```bash
midclt call cloudsync.credentials.verify '{
  "provider": "S3",
  "attributes": {
    "access_key_id": "AKIAXXXXXXXX",
    "secret_access_key": "secretkey",
    "region": "us-east-1"
  }
}'
```

## Cloud Sync Tasks

### cloudsync.query
Query cloud sync tasks.
```bash
midclt call cloudsync.query
```

Returns:
- `id` - Task ID
- `description` - Task description
- `path` - Local path
- `credentials` - Credential ID
- `attributes` - Provider-specific attributes
- `schedule` - Cron schedule
- `direction` - PUSH, PULL, or SYNC
- `transfer_mode` - SYNC, COPY, or MOVE
- `encryption` - Encryption enabled
- `enabled` - Task enabled
- `job` - Last job status

### cloudsync.create
Create a cloud sync task.
```bash
midclt call cloudsync.create '{
  "description": "Backup to S3",
  "path": "/mnt/tank/data",
  "credentials": <cred_id>,
  "attributes": {
    "bucket": "my-bucket",
    "folder": "backups"
  },
  "direction": "PUSH",
  "transfer_mode": "SYNC",
  "schedule": {
    "minute": "0",
    "hour": "2",
    "dom": "*",
    "month": "*",
    "dow": "*"
  },
  "enabled": true
}'
```

With encryption:
```bash
midclt call cloudsync.create '{
  "description": "Encrypted backup",
  "path": "/mnt/tank/sensitive",
  "credentials": <cred_id>,
  "attributes": {"bucket": "encrypted-bucket"},
  "direction": "PUSH",
  "transfer_mode": "SYNC",
  "encryption": true,
  "encryption_password": "strongpassword",
  "encryption_salt": "randomsalt",
  "enabled": true
}'
```

### cloudsync.update
Update a cloud sync task.
```bash
midclt call cloudsync.update <task_id> '{"enabled": false}'
```

### cloudsync.delete
Delete a cloud sync task.
```bash
midclt call cloudsync.delete <task_id>
```

### cloudsync.sync
Run a cloud sync task immediately.
```bash
midclt call cloudsync.sync <task_id>
```

### cloudsync.sync_onetime
Run a one-time sync (no saved task).
```bash
midclt call cloudsync.sync_onetime '{
  "description": "One-time sync",
  "path": "/mnt/tank/data",
  "credentials": <cred_id>,
  "attributes": {"bucket": "my-bucket"},
  "direction": "PUSH",
  "transfer_mode": "COPY"
}'
```

### cloudsync.abort
Abort a running cloud sync.
```bash
midclt call cloudsync.abort <task_id>
```

### cloudsync.restore
Restore from cloud to local.
```bash
midclt call cloudsync.restore <task_id> '{"transfer_mode": "COPY"}'
```

### cloudsync.providers
Get available cloud providers.
```bash
midclt call cloudsync.providers
```

### cloudsync.list_buckets
List buckets for a credential.
```bash
midclt call cloudsync.list_buckets <cred_id>
```

### cloudsync.list_directory
List directory in cloud storage.
```bash
midclt call cloudsync.list_directory <cred_id> '{"bucket": "my-bucket", "folder": "path/to/folder"}'
```

### cloudsync.create_bucket
Create a bucket.
```bash
midclt call cloudsync.create_bucket <cred_id> '{"bucket": "new-bucket"}'
```

### cloudsync.onedrive_list_drives
List OneDrive drives.
```bash
midclt call cloudsync.onedrive_list_drives <cred_id>
```

## Cloud Backup

### cloud_backup.query
Query cloud backups.
```bash
midclt call cloud_backup.query
```

### cloud_backup.create
Create a cloud backup task.
```bash
midclt call cloud_backup.create '{
  "description": "Dataset backup",
  "path": "/mnt/tank/important",
  "credentials": <cred_id>,
  "attributes": {"bucket": "backup-bucket", "folder": "truenas"},
  "password": "encryption_password",
  "schedule": {"minute": "0", "hour": "3", "dom": "*", "month": "*", "dow": "*"},
  "enabled": true
}'
```

### cloud_backup.update
Update a cloud backup task.
```bash
midclt call cloud_backup.update <backup_id> '{"enabled": false}'
```

### cloud_backup.delete
Delete a cloud backup task.
```bash
midclt call cloud_backup.delete <backup_id>
```

### cloud_backup.sync
Run cloud backup immediately.
```bash
midclt call cloud_backup.sync <backup_id>
```

### cloud_backup.list_snapshots
List available snapshots.
```bash
midclt call cloud_backup.list_snapshots <backup_id>
```

### cloud_backup.delete_snapshot
Delete a cloud backup snapshot.
```bash
midclt call cloud_backup.delete_snapshot <backup_id> "<snapshot_id>"
```

### cloud_backup.restore
Restore from cloud backup.
```bash
midclt call cloud_backup.restore <backup_id> <snapshot_id> '{"path": "/mnt/tank/restored"}'
```

### cloud_backup.transfer_setting_choices
Get available transfer settings.
```bash
midclt call cloud_backup.transfer_setting_choices
```

## Supported Cloud Providers

| Provider | Type |
|----------|------|
| `S3` | Amazon S3 / S3-compatible |
| `B2` | Backblaze B2 |
| `GOOGLE_CLOUD_STORAGE` | Google Cloud Storage |
| `GOOGLE_DRIVE` | Google Drive |
| `AZUREBLOB` | Azure Blob Storage |
| `ONEDRIVE` | Microsoft OneDrive |
| `DROPBOX` | Dropbox |
| `BOX` | Box |
| `SFTP` | SFTP server |
| `FTP` | FTP server |
| `MEGA` | MEGA |
| `PCLOUD` | pCloud |
| `STORJ` | Storj |
| `WEBDAV` | WebDAV |
