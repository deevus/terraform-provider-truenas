# Deprecate host_path Resource, Improve Dataset Schema

**Date:** 2026-01-13
**Status:** Approved

## Overview

Deprecate the `truenas_host_path` resource and improve the `truenas_dataset` schema to better represent TrueNAS concepts. Users should create nested datasets instead of arbitrary directories.

## Motivation

The `host_path` resource uses SFTP to create directories, but:
1. SSH user is not guaranteed to be root
2. SFTP runs as the SSH user without sudo
3. Only `sudo midclt call` (TrueNAS API) is guaranteed to work

`host_path` is not a real TrueNAS concept - it's an artificial abstraction for "a directory path for Docker mounts." The idiomatic TrueNAS approach is to use **datasets** for app storage, not arbitrary directories.

## Design Decisions

### Deprecate host_path Resource

- **Decision:** Mark `host_path` as deprecated, recommend nested datasets
- **Rationale:** Datasets are native TrueNAS objects created via API (`pool.dataset.create`) with no SFTP permission issues. They provide per-app ZFS properties (compression, quotas, snapshots) and follow TrueNAS best practices.

### Keep host_path for Edge Cases

- **Decision:** Keep `host_path` functional but deprecated
- **Rationale:** Some edge cases may genuinely need arbitrary directories:
  - Third-party apps that expect specific non-dataset paths
  - Temporary directories that shouldn't be full datasets
  - Legacy configurations during migration

### Consolidate name into path

- **Decision:** Rename `name` attribute to `path`, deprecate `name`
- **Rationale:** `path` is more flexible - it can be a simple name when used with `parent`, or a relative path when used with `pool`. Consolidating simplifies the schema.

### Rename mount_path to full_path

- **Decision:** Rename `mount_path` to `full_path`, deprecate `mount_path`
- **Rationale:** `full_path` is clearer - it's the full absolute filesystem path to the mounted dataset. Frees up `mount_path` name and is more explicit.

## Schema Changes

### Dataset Resource

**Before (current schema):**

| Attribute | Type | Description |
|-----------|------|-------------|
| `pool` | input | Pool name (use with `path`) |
| `path` | input | Relative path in pool |
| `parent` | input | Parent dataset ID (use with `name`) |
| `name` | input | Dataset name |
| `mount_path` | computed | Filesystem mount path |

**After (new schema):**

| Attribute | Type | Description |
|-----------|------|-------------|
| `pool` | input | Pool name (use with `path` for pool-relative) |
| `parent` | input | Parent dataset ID (use with `path` for child) |
| `path` | input | Dataset path (relative to pool OR child name) |
| `full_path` | computed | Filesystem mount path (`/mnt/...`) |

**Deprecations:**
- `name` → use `path` instead
- `mount_path` → use `full_path` instead

### host_path Resource

Add `DeprecationMessage` to schema directing users to use `truenas_dataset` instead.

## Migration Path

**Before (host_path + old schema):**
```hcl
resource "truenas_dataset" "tailscale" {
  parent = data.truenas_dataset.apps.id
  name   = "tailscale"  # DEPRECATED
}

resource "truenas_host_path" "tailscale_state" {  # DEPRECATED
  path = "/mnt/storage/apps/tailscale/state"
  mode = "700"
  uid  = 0
  gid  = 0
  force_destroy = true
  depends_on = [truenas_dataset.tailscale]
}
```

**After (datasets only + new schema):**
```hcl
resource "truenas_dataset" "tailscale" {
  parent = data.truenas_dataset.apps.id
  path   = "tailscale"  # Replaces 'name'
}

resource "truenas_dataset" "tailscale_state" {
  parent = truenas_dataset.tailscale.id
  path   = "state"
  mode   = "700"
  uid    = 0
  gid    = 0
  force_destroy = true
}

# Use truenas_dataset.tailscale_state.full_path in app config
```

## Implementation

### dataset.go Changes

1. Update model to include both old and new attributes
2. Add deprecation messages to `name` and `mount_path` attributes
3. Update `getFullName()` to prefer `path` over `name`
4. Sync both `full_path` and `mount_path` to same value for backwards compat

### host_path.go Changes

1. Add `DeprecationMessage` to schema

### Test Updates

1. Update tests to use `path` instead of `name`
2. Update tests to use `full_path` instead of `mount_path`
3. Add tests verifying deprecated attributes still work

## Backwards Compatibility

- Existing configs using `name` and `mount_path` continue to work
- Deprecation warnings appear in terraform plan/apply output
- Users have time to migrate to new attribute names
- Full removal planned for future major version

## Benefits of Nested Datasets

- **API-based creation:** No SFTP permission issues regardless of SSH user
- **Native management:** Proper TrueNAS objects with full lifecycle support
- **ZFS features:** Per-dataset compression, quotas, snapshots
- **Best practices:** Aligns with TrueNAS recommended app storage patterns

## References

- [AWS EFS Access Points](https://shisho.dev/dojo/providers/aws/Amazon_EFS/aws-efs-access-point/) - Similar pattern but constrained by TrueNAS lacking `filesystem.mkdir` API
- [HashiCorp Provider Design Principles](https://developer.hashicorp.com/terraform/plugin/best-practices/hashicorp-provider-design-principles) - Resources should represent single components
