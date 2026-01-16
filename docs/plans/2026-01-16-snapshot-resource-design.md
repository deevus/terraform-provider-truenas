# Snapshot Resource Design

**Date:** 2026-01-16
**Status:** Draft

## Overview

Add ZFS snapshot support to enable pre-upgrade backups and point-in-time recovery for TrueNAS apps.

## Motivation

Users need to create backups before upgrading apps (e.g., Forgejo v9 → v14) with the ability to restore if the upgrade fails. The idiomatic Terraform pattern is a standalone snapshot resource combined with the ability to create datasets from snapshots.

## Design Decisions

### 1. Standalone snapshot resource (idiomatic)

Follows the AWS pattern (`aws_db_snapshot`, `aws_ebs_snapshot`) - snapshots are independent resources that users compose with `depends_on`.

### 2. No rollback capability on snapshot

Rollback is an imperative emergency operation, not a declarative state. Users handle rollback manually via TrueNAS UI/CLI, or by creating a new dataset from the snapshot.

### 3. Clone via `snapshot_id` on `truenas_dataset`

Matches AWS pattern where "restore" means creating a NEW resource from a snapshot:
- `aws_ebs_volume.snapshot_id`
- `aws_db_instance.snapshot_identifier`

### 4. Reference-based `dataset_id`

The snapshot resource accepts `dataset_id` which should be a reference to a `truenas_dataset` resource or data source - not a raw string. For unmanaged datasets, use the existing `truenas_dataset` data source.

---

## `truenas_snapshot` Resource

```hcl
resource "truenas_dataset" "forgejo_data" {
  pool = "tank"
  path = "apps/forgejo"
}

resource "truenas_snapshot" "pre_upgrade" {
  dataset_id = truenas_dataset.forgejo_data.id
  name       = "pre-v14-upgrade"
  hold       = true
  recursive  = false
}
```

### Schema

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `dataset_id` | string | Yes | Dataset ID (reference to resource or data source) |
| `name` | string | Yes | Snapshot name |
| `hold` | bool | No | Prevent automatic deletion (default: false) |
| `recursive` | bool | No | Include child datasets (default: false) |

### Computed Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | string | Full snapshot ID: `dataset@name` |
| `createtxg` | string | Transaction group when created |
| `used_bytes` | number | Space consumed by snapshot |
| `referenced_bytes` | number | Space referenced by snapshot |

### Operations

| Operation | API Method |
|-----------|------------|
| Create | `pool.snapshot.create` |
| Read | `pool.snapshot.query` |
| Update (hold) | `pool.snapshot.hold` / `pool.snapshot.release` |
| Delete | `pool.snapshot.delete` |

---

## `truenas_snapshots` Data Source

Query existing snapshots for a dataset.

```hcl
data "truenas_snapshots" "app_backups" {
  dataset_id = truenas_dataset.forgejo_data.id
  recursive    = true
  name_pattern = "pre-upgrade-*"
}

output "snapshots" {
  value = data.truenas_snapshots.app_backups.snapshots
}
```

### Inputs

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `dataset_id` | string | Yes | Dataset to query snapshots for |
| `recursive` | bool | No | Include child dataset snapshots (default: false) |
| `name_pattern` | string | No | Glob filter on snapshot name |

### Outputs

| Attribute | Type | Description |
|-----------|------|-------------|
| `snapshots` | list | List of snapshot objects |
| `snapshots[].id` | string | Full snapshot ID (`dataset@name`) |
| `snapshots[].name` | string | Snapshot name |
| `snapshots[].dataset_id` | string | Parent dataset |
| `snapshots[].used_bytes` | number | Space consumed |
| `snapshots[].referenced_bytes` | number | Space referenced |
| `snapshots[].hold` | bool | Whether held |

---

## `truenas_dataset` Modification

Add `snapshot_id` attribute for creating a dataset from a snapshot (clone).

```hcl
resource "truenas_dataset" "forgejo_data_restored" {
  pool        = "tank"
  path        = "apps/forgejo-restored"
  snapshot_id = truenas_snapshot.pre_upgrade.id
}
```

### New Attribute

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `snapshot_id` | string | No | Create dataset as clone of this snapshot |

### Behavior

- If `snapshot_id` set: uses `pool.snapshot.clone` instead of `pool.dataset.create`
- Immutable after creation (changing requires replace)

---

## Complete Workflow Example

Forgejo upgrade from v9 to v14 with pre-upgrade backup:

```hcl
# Dataset for app data
resource "truenas_dataset" "forgejo_data" {
  pool = "tank"
  path = "apps/forgejo"
}

# Pre-upgrade snapshot
resource "truenas_snapshot" "forgejo_pre_upgrade" {
  dataset_id = truenas_dataset.forgejo_data.id
  name       = "pre-v14-upgrade"
  hold       = true
}

# App with explicit dependency on snapshot
resource "truenas_app" "forgejo" {
  name           = "forgejo"
  custom_app     = true
  desired_state  = "RUNNING"
  compose_config = file("forgejo-v14.yaml")

  depends_on = [truenas_snapshot.forgejo_pre_upgrade]
}
```

**Apply order:** Dataset exists → Snapshot created → App updated

### Recovery (if upgrade fails)

1. Note the snapshot exists
2. Update Terraform config to clone from snapshot:

```hcl
resource "truenas_dataset" "forgejo_data_restored" {
  pool        = "tank"
  path        = "apps/forgejo-v9"
  snapshot_id = truenas_snapshot.forgejo_pre_upgrade.id
}
```

3. Revert compose config to v9
4. `terraform apply`

---

## Implementation

### Deliverables

| Item | Type | Effort |
|------|------|--------|
| `truenas_snapshot` | New resource | Medium |
| `truenas_snapshots` | New data source | Low |
| `truenas_dataset` + `snapshot_id` | Modify existing | Low |

### Implementation Order

1. `truenas_snapshot` resource - core functionality
2. `truenas_snapshots` data source - query existing
3. `truenas_dataset` modification - add `snapshot_id` for clone

### Files

```
internal/
├── resources/
│   └── snapshot.go          # NEW
├── datasources/
│   └── snapshots.go         # NEW (plural)
│   └── dataset.go           # EXISTS (modify for snapshot_id awareness)
```

### API Methods

| Operation | Method |
|-----------|--------|
| Create snapshot | `pool.snapshot.create` |
| Query snapshots | `pool.snapshot.query` |
| Hold snapshot | `pool.snapshot.hold` |
| Release snapshot | `pool.snapshot.release` |
| Delete snapshot | `pool.snapshot.delete` |
| Clone to dataset | `pool.snapshot.clone` |

---

## Go Code Style: Parameter Objects

**Rule:** Maximum 3 parameters per function. Beyond that, extract to a struct.

When adding parameters to an existing method, evaluate its current parameter count first. If adding the parameter would exceed 3, refactor to use a parameter object.

### Pattern

```go
// BAD: too many params
func (c *Client) CreateSnapshot(ctx context.Context, dataset, name string, hold, recursive bool) error

// GOOD: params struct
type CreateSnapshotParams struct {
    Dataset   string
    Name      string
    Hold      bool
    Recursive bool
}

func (c *Client) CreateSnapshot(ctx context.Context, params CreateSnapshotParams) error
```

### When to Extract

- More than 3 parameters
- Multiple boolean parameters (hard to read at call site)
- Parameters that logically group together
- Adding a parameter to an existing method would exceed 3

This pattern should be applied to new snapshot operations and evaluated for existing methods when modifications are needed.

---

## Testing Strategy

**Goal:** 100% test coverage.

Uses the established codebase patterns:
- Go stdlib `testing` (no external frameworks)
- Hand-rolled `MockClient` with injectable functions
- Helper functions per resource: `getXResourceSchema()`, `createXResourceModelValue()`
- Naming convention: `Test{Resource}_{Method}_{Scenario}`

### `truenas_snapshot` Resource Tests

**Schema:**
- `TestSnapshotResource_Schema` - attributes exist, types correct, computed flags set

**Create:**
- `TestSnapshotResource_Create_Success` - verify `pool.snapshot.create` params
- `TestSnapshotResource_Create_WithHold` - verify `pool.snapshot.hold` called after create
- `TestSnapshotResource_Create_WithRecursive` - verify recursive param passed
- `TestSnapshotResource_Create_APIError` - error propagated to diagnostics
- `TestSnapshotResource_Create_InvalidJSONResponse` - handles malformed response

**Read:**
- `TestSnapshotResource_Read_Success` - state populated from API response
- `TestSnapshotResource_Read_NotFound` - removes from state (no error)
- `TestSnapshotResource_Read_APIError` - error propagated to diagnostics
- `TestSnapshotResource_Read_InvalidJSONResponse` - handles malformed response

**Update:**
- `TestSnapshotResource_Update_HoldToRelease` - calls `pool.snapshot.release`
- `TestSnapshotResource_Update_ReleaseToHold` - calls `pool.snapshot.hold`
- `TestSnapshotResource_Update_NoChange` - no API calls
- `TestSnapshotResource_Update_HoldAPIError` - error propagated
- `TestSnapshotResource_Update_ReleaseAPIError` - error propagated

**Delete:**
- `TestSnapshotResource_Delete_Success` - calls `pool.snapshot.delete`
- `TestSnapshotResource_Delete_WithHold` - releases hold first, then deletes
- `TestSnapshotResource_Delete_APIError` - error propagated
- `TestSnapshotResource_Delete_NotFound` - succeeds (already gone)

**Import:**
- `TestSnapshotResource_ImportState` - sets ID from import string
- `TestSnapshotResource_ImportState_InvalidFormat` - error for malformed ID

**Validation:**
- `TestSnapshotResource_ValidateConfig_MissingDatasetId` - required attribute
- `TestSnapshotResource_ValidateConfig_MissingName` - required attribute
- `TestSnapshotResource_ValidateConfig_InvalidSnapshotName` - name validation (if any)

### `truenas_snapshots` Data Source Tests

**Schema:**
- `TestSnapshotsDataSource_Schema` - attributes exist, types correct

**Read:**
- `TestSnapshotsDataSource_Read_Success` - returns snapshot list
- `TestSnapshotsDataSource_Read_Empty` - returns empty list (no error)
- `TestSnapshotsDataSource_Read_WithRecursive` - includes child dataset snapshots
- `TestSnapshotsDataSource_Read_WithNamePattern` - filters by glob pattern
- `TestSnapshotsDataSource_Read_WithNamePatternNoMatch` - returns empty list
- `TestSnapshotsDataSource_Read_APIError` - error propagated
- `TestSnapshotsDataSource_Read_InvalidJSONResponse` - handles malformed response

**Configure:**
- `TestSnapshotsDataSource_Configure_Success` - client set
- `TestSnapshotsDataSource_Configure_NilClient` - error handling

### `truenas_dataset` Clone Tests

**Create with snapshot_id:**
- `TestDatasetResource_Create_WithSnapshotId` - uses `pool.snapshot.clone` API
- `TestDatasetResource_Create_WithSnapshotId_APIError` - error propagated
- `TestDatasetResource_Create_WithSnapshotId_VerifyParams` - clone params correct

**Update with snapshot_id:**
- `TestDatasetResource_Update_SnapshotIdChanged` - forces replacement (RequiresReplace)
- `TestDatasetResource_Update_SnapshotIdRemoved` - forces replacement

**Plan modifiers:**
- `TestDatasetResource_PlanModifier_SnapshotIdRequiresReplace` - plan shows replacement

### Test Helpers

```go
// internal/resources/snapshot_test.go

func getSnapshotResourceSchema(t *testing.T) resource.SchemaResponse {
    t.Helper()
    // ...
}

func createSnapshotResourceModelValue(
    id, datasetId, name interface{},
    hold, recursive interface{},
    createtxg, usedBytes, referencedBytes interface{},
) tftypes.Value {
    // ...
}
```

```go
// internal/datasources/snapshots_test.go

func getSnapshotsDataSourceSchema(t *testing.T) datasource.SchemaResponse {
    t.Helper()
    // ...
}

func createSnapshotsDataSourceModelValue(
    datasetId interface{},
    recursive, namePattern interface{},
    snapshots interface{},
) tftypes.Value {
    // ...
}
```

---

## Future Considerations

- **Snapshot tasks:** Scheduled automatic snapshots (`pool.snapshottask.*`) - separate resource
- **Bulk operations:** Creating snapshots across multiple datasets
- **Retention policies:** Automatic cleanup of old snapshots
