# truenas_cloudsync_task Resource

Manages cloud sync tasks for scheduled or on-demand backups to cloud storage.

## Example Usage

### Basic S3 Backup

```hcl
resource "truenas_cloudsync_credentials" "aws" {
  name = "aws-backup"

  s3 {
    access_key_id     = var.aws_access_key
    secret_access_key = var.aws_secret_key
    region            = "us-east-1"
  }
}

resource "truenas_cloudsync_task" "daily_backup" {
  description   = "Daily backup to S3"
  path          = "/mnt/tank/data"
  credentials   = truenas_cloudsync_credentials.aws.id
  direction     = "push"
  transfer_mode = "sync"

  schedule {
    minute = "0"
    hour   = "2"
  }

  s3 {
    bucket = "my-backup-bucket"
    folder = "truenas-backup"
  }
}
```

### Encrypted Backup with Bandwidth Limit

```hcl
resource "truenas_cloudsync_task" "encrypted_backup" {
  description   = "Encrypted backup to B2"
  path          = "/mnt/tank/sensitive"
  credentials   = truenas_cloudsync_credentials.b2.id
  direction     = "push"
  transfer_mode = "copy"

  schedule {
    minute = "0"
    hour   = "3"
  }

  b2 {
    bucket = "my-b2-bucket"
    folder = "encrypted-backup"
  }

  encryption {
    password = var.encryption_password
    salt     = var.encryption_salt
  }

  bwlimit   = "10M"
  transfers = 4
  exclude   = ["*.tmp", "*.log"]
}
```

### Sync Specific Directories

```hcl
resource "truenas_cloudsync_task" "selective_backup" {
  description   = "Backup specific folders"
  path          = "/mnt/tank/data"
  credentials   = truenas_cloudsync_credentials.s3.id
  direction     = "push"
  transfer_mode = "sync"

  schedule {
    minute = "0"
    hour   = "3"
  }

  s3 {
    bucket = "my-backup-bucket"
    folder = "selective"
  }

  # Only sync photos and documents directories
  include = ["/photos/**", "/documents/**"]

  # But exclude temporary files within those
  exclude = ["*.tmp", "*.cache"]
}
```

### Sync on Change

```hcl
resource "truenas_cloudsync_task" "immediate_sync" {
  description    = "Sync documents to GCS"
  path           = "/mnt/tank/documents"
  credentials    = truenas_cloudsync_credentials.gcs.id
  direction      = "push"
  transfer_mode  = "sync"
  sync_on_change = true

  schedule {
    minute = "0"
    hour   = "0"
  }

  gcs {
    bucket = "my-gcs-bucket"
    folder = "documents"
  }
}
```

## Argument Reference

* `description` - (Required) Task description.
* `path` - (Required) Local path to sync.
* `credentials` - (Required) Credentials ID. Reference a truenas_cloudsync_credentials resource.
* `direction` - (Required) Sync direction. Valid values: `push`, `pull`.
* `transfer_mode` - (Required) Transfer mode. Valid values: `sync`, `copy`, `move`.
* `schedule` - (Required) Schedule block. See Schedule below.
* `enabled` - (Optional) Enable the task. Default: true.
* `snapshot` - (Optional) Create a snapshot before sync. Default: false.
* `follow_symlinks` - (Optional) Follow symbolic links. Default: false.
* `create_empty_src_dirs` - (Optional) Create empty source directories at destination. Default: false.
* `sync_on_change` - (Optional) Trigger sync immediately after create or update. Default: false.
* `transfers` - (Optional) Number of concurrent transfers.
* `bwlimit` - (Optional) Bandwidth limit (e.g., "10M" for 10 MB/s).
* `exclude` - (Optional) List of exclude patterns.
* `include` - (Optional) List of include patterns. Supports glob patterns like `/folder/**` or `*.jpg`. When specified, only matching files are synced. Include patterns are applied before exclude patterns.

Exactly one of the following provider-specific blocks is required:

### s3 Block

* `bucket` - (Required) S3 bucket name.
* `folder` - (Optional) Folder path within the bucket.

### b2 Block

* `bucket` - (Required) B2 bucket name.
* `folder` - (Optional) Folder path within the bucket.

### gcs Block

* `bucket` - (Required) GCS bucket name.
* `folder` - (Optional) Folder path within the bucket.

### azure Block

* `container` - (Required) Azure blob container name.
* `folder` - (Optional) Folder path within the container.

### schedule Block

* `minute` - (Required) Minute (0-59).
* `hour` - (Required) Hour (0-23).
* `dom` - (Optional) Day of month (1-31). Default: "*".
* `month` - (Optional) Month (1-12). Default: "*".
* `dow` - (Optional) Day of week (0-6, 0=Sunday). Default: "*".

### encryption Block

* `password` - (Required) Encryption password.
* `salt` - (Optional) Encryption salt.

## Attribute Reference

* `id` - Task ID.

## Import

Cloud sync tasks can be imported using the task ID:

```bash
terraform import truenas_cloudsync_task.example 1
```
