# Cloud Sync Resources Design

Design for `truenas_cloudsync_credentials` and `truenas_cloudsync_task` resources.

## Overview

Two resources and one data source for managing TrueNAS cloud sync functionality:

- `truenas_cloudsync_credentials` - Manage cloud provider credentials
- `truenas_cloudsync_task` - Manage cloud sync backup tasks
- `data.truenas_cloudsync_credentials` - Look up existing credentials

## Supported Providers

Initial implementation supports four providers:

| Provider | Block | API Type |
|----------|-------|----------|
| S3-compatible | `s3 {}` | `S3` |
| Backblaze B2 | `b2 {}` | `B2` |
| Google Cloud Storage | `gcs {}` | `GOOGLE_CLOUD_STORAGE` |
| Azure Blob | `azure {}` | `AZUREBLOB` |

OAuth-based providers (Google Drive, OneDrive, Dropbox, Box) require interactive authentication and are out of scope. Users can create these via UI and reference by ID if needed.

## truenas_cloudsync_credentials Resource

### Schema

```hcl
resource "truenas_cloudsync_credentials" "example" {
  name = "Credential name"  # required

  # Exactly one provider block required
  s3 {
    access_key_id     = "..."  # required, sensitive
    secret_access_key = "..."  # required, sensitive
    endpoint          = "..."  # optional (for S3-compatible)
    region            = "..."  # optional
  }

  # OR
  b2 {
    account = "..."  # required, sensitive
    key     = "..."  # required, sensitive
  }

  # OR
  gcs {
    service_account_credentials = "..."  # required, sensitive (JSON)
  }

  # OR
  azure {
    account = "..."  # required, sensitive
    key     = "..."  # required, sensitive
  }
}
```

### Attributes

| Attribute | Type | Required | Sensitive | Description |
|-----------|------|----------|-----------|-------------|
| `id` | string | computed | no | Credential ID |
| `name` | string | yes | no | Display name |
| `s3` | block | one of | - | S3-compatible credentials |
| `s3.access_key_id` | string | yes | yes | Access key ID |
| `s3.secret_access_key` | string | yes | yes | Secret access key |
| `s3.endpoint` | string | no | no | Custom endpoint URL |
| `s3.region` | string | no | no | Region |
| `b2` | block | one of | - | Backblaze B2 credentials |
| `b2.account` | string | yes | yes | Account ID |
| `b2.key` | string | yes | yes | Application key |
| `gcs` | block | one of | - | Google Cloud Storage credentials |
| `gcs.service_account_credentials` | string | yes | yes | Service account JSON |
| `azure` | block | one of | - | Azure Blob credentials |
| `azure.account` | string | yes | yes | Storage account name |
| `azure.key` | string | yes | yes | Account key |

### API Mapping

- Create: `cloudsync.credentials.create`
- Read: `cloudsync.credentials.query` with filter `[["id", "=", <id>]]`
- Update: `cloudsync.credentials.update <id> {...}`
- Delete: `cloudsync.credentials.delete <id>`

### Examples

```hcl
# S3-compatible (Scaleway)
resource "truenas_cloudsync_credentials" "scaleway" {
  name = "Scaleway Amsterdam"

  s3 {
    access_key_id     = var.scaleway_access_key
    secret_access_key = var.scaleway_secret_key
    endpoint          = "s3.nl-ams.scw.cloud"
    region            = "nl-ams"
  }
}

# Backblaze B2
resource "truenas_cloudsync_credentials" "backblaze" {
  name = "Backblaze B2"

  b2 {
    account = var.b2_account
    key     = var.b2_key
  }
}

# Google Cloud Storage
resource "truenas_cloudsync_credentials" "google" {
  name = "GCS Production"

  gcs {
    service_account_credentials = file("gcs-sa.json")
  }
}

# Azure Blob
resource "truenas_cloudsync_credentials" "azure" {
  name = "Azure Blob"

  azure {
    account = var.azure_account
    key     = var.azure_key
  }
}
```

## truenas_cloudsync_task Resource

### Schema

```hcl
resource "truenas_cloudsync_task" "example" {
  # Core attributes
  description = "Task description"              # required
  path        = "/mnt/pool/dataset"             # required
  credentials = truenas_cloudsync_credentials.x.id  # required
  direction   = "push"                          # required: push, pull
  transfer_mode = "sync"                        # optional: sync (default), copy, move
  enabled     = true                            # optional, default true

  # Provider block - must match credentials type
  s3 {
    bucket = "bucket-name"  # required
    folder = "/path/"       # optional
  }

  # Schedule block
  schedule {
    minute = "0"   # required
    hour   = "2"   # required
    dom    = "*"   # optional, default "*"
    month  = "*"   # optional, default "*"
    dow    = "*"   # optional, default "*"
  }

  # Encryption (optional block, presence enables encryption)
  encryption {
    password = "..."  # required, sensitive
    salt     = "..."  # optional, sensitive
  }

  # Options
  snapshot              = true                  # optional, default false
  transfers             = 8                     # optional, default 4
  bwlimit               = "10M"                 # optional
  exclude               = ["*.log", "*.tmp"]    # optional
  follow_symlinks       = false                 # optional, default false
  create_empty_src_dirs = true                  # optional, default false
  sync_on_change        = true                  # optional, default false
}
```

### Attributes

| Attribute | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `id` | string | computed | - | Task ID |
| `description` | string | yes | - | Task description |
| `path` | string | yes | - | Local filesystem path |
| `credentials` | number | yes | - | Credential ID reference |
| `direction` | string | yes | - | "push" or "pull" |
| `transfer_mode` | string | no | "sync" | "sync", "copy", or "move" |
| `enabled` | bool | no | true | Whether schedule is active |
| `snapshot` | bool | no | false | ZFS snapshot before sync |
| `transfers` | number | no | 4 | Parallel transfer count |
| `bwlimit` | string | no | - | Bandwidth limit (e.g., "10M") |
| `exclude` | list(string) | no | [] | Glob patterns to exclude |
| `follow_symlinks` | bool | no | false | Follow symbolic links |
| `create_empty_src_dirs` | bool | no | false | Create empty directories |
| `sync_on_change` | bool | no | false | Fire-and-forget sync after create/update |

### Provider Blocks

Each provider block contains bucket/folder attributes:

| Block | Attributes |
|-------|------------|
| `s3` | `bucket` (required), `folder` (optional) |
| `b2` | `bucket` (required), `folder` (optional) |
| `gcs` | `bucket` (required), `folder` (optional) |
| `azure` | `container` (required), `folder` (optional) |

### Schedule Block

| Attribute | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `minute` | string | yes | - | Cron minute field |
| `hour` | string | yes | - | Cron hour field |
| `dom` | string | no | "*" | Day of month |
| `month` | string | no | "*" | Month |
| `dow` | string | no | "*" | Day of week |

### Encryption Block

| Attribute | Type | Required | Sensitive | Description |
|-----------|------|----------|-----------|-------------|
| `password` | string | yes | yes | Encryption password |
| `salt` | string | no | yes | Encryption salt (auto-generated if omitted) |

### API Mapping

- Create: `cloudsync.create`
- Read: `cloudsync.query` with filter `[["id", "=", <id>]]`
- Update: `cloudsync.update <id> {...}`
- Delete: `cloudsync.delete <id>`
- Sync: `cloudsync.sync <id>` (fire-and-forget when `sync_on_change = true`)

### Direction and Transfer Mode

Values are lowercase in Terraform, transformed to uppercase for API:

| Terraform | API |
|-----------|-----|
| `"push"` | `"PUSH"` |
| `"pull"` | `"PULL"` |
| `"sync"` | `"SYNC"` |
| `"copy"` | `"COPY"` |
| `"move"` | `"MOVE"` |

### Examples

```hcl
# Basic backup task
resource "truenas_cloudsync_task" "authelia_config" {
  description = "Backup: authelia/config"
  path        = "/mnt/storage/apps/authelia/config"
  credentials = truenas_cloudsync_credentials.scaleway.id
  direction   = "push"

  s3 {
    bucket = "shartcher-glacial-backups"
    folder = "/solaris-1/apps/authelia/config/"
  }

  schedule {
    minute = "0"
    hour   = "2"
  }
}

# Full-featured task
resource "truenas_cloudsync_task" "sensitive_data" {
  description = "Backup: sensitive data"
  path        = "/mnt/storage/sensitive"
  credentials = truenas_cloudsync_credentials.scaleway.id
  direction   = "push"

  s3 {
    bucket = "shartcher-glacial-backups"
    folder = "/solaris-1/sensitive/"
  }

  schedule {
    minute = "0"
    hour   = "2"
  }

  encryption {
    password = var.encryption_password
  }

  snapshot              = true
  transfers             = 8
  exclude               = ["*.log", "*.tmp", ".cache/"]
  sync_on_change        = true
}

# Programmatic generation with for_each
locals {
  backup_datasets = {
    "authelia-config" = { path = "apps/authelia/config", folder = "/authelia/config/" }
    "authelia-data"   = { path = "apps/authelia/data", folder = "/authelia/data/" }
    "caddy-config"    = { path = "apps/caddy/config", folder = "/caddy/config/" }
  }
}

resource "truenas_cloudsync_task" "backups" {
  for_each = local.backup_datasets

  description = "Backup: ${each.key}"
  path        = "/mnt/storage/${each.value.path}"
  credentials = truenas_cloudsync_credentials.scaleway.id
  direction   = "push"
  snapshot    = true
  transfers   = 8

  s3 {
    bucket = "shartcher-glacial-backups"
    folder = "/solaris-1${each.value.folder}"
  }

  schedule {
    minute = "0"
    hour   = "2"
  }

  exclude = ["*.log", "*.tmp"]
}
```

## truenas_cloudsync_credentials Data Source

### Schema

```hcl
data "truenas_cloudsync_credentials" "existing" {
  name = "Credential name"  # required
}
```

### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Credential name to look up (required) |
| `id` | number | Credential ID (computed) |
| `provider` | string | Provider type (computed) |

### Constraints

- Only returns credentials with supported provider types (S3, B2, GOOGLE_CLOUD_STORAGE, AZUREBLOB)
- Error if credential not found or provider not supported

### API Mapping

- Query: `cloudsync.credentials.query` with filter `[["name", "=", <name>]]`

### Example

```hcl
data "truenas_cloudsync_credentials" "existing_s3" {
  name = "My S3 Creds"
}

resource "truenas_cloudsync_task" "using_existing" {
  description = "Using existing creds"
  path        = "/mnt/storage/data"
  credentials = data.truenas_cloudsync_credentials.existing_s3.id
  direction   = "push"

  s3 {
    bucket = "my-bucket"
    folder = "/"
  }

  schedule {
    minute = "0"
    hour   = "4"
  }
}
```

## Import Support

Both resources support `terraform import` by TrueNAS ID:

```bash
terraform import truenas_cloudsync_credentials.mine 5
terraform import truenas_cloudsync_task.backup 12
```

Import reads the resource and populates all attributes. For credentials, sensitive fields (keys, passwords) will be empty after import and must be set in configuration.

## Implementation Notes

### Provider Block Validation

- Exactly one provider block required on both credentials and tasks
- Task provider block must match credential provider type
- Validation at plan time where possible

### Sensitive Fields

All credential secrets and encryption passwords marked `Sensitive: true`:
- `s3.access_key_id`, `s3.secret_access_key`
- `b2.account`, `b2.key`
- `gcs.service_account_credentials`
- `azure.account`, `azure.key`
- `encryption.password`, `encryption.salt`

### sync_on_change Behavior

When `sync_on_change = true`:
1. Create/update the task normally
2. Call `cloudsync.sync <task_id>`
3. Return immediately (fire-and-forget)
4. Sync runs asynchronously in TrueNAS

### Snapshot Validation

The `snapshot = true` option requires the path to be a leaf dataset (no child datasets). Validation is deferred to the API - TrueNAS returns an error if the constraint is violated.

## Future Enhancements

- Additional provider blocks: SFTP, FTP, WebDAV, STORJ, MEGA, PCLOUD
- OAuth provider support (requires interactive flow design)
- `truenas_cloudsync_task` data source
- Client-side leaf dataset validation for `snapshot = true`
