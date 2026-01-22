# Design: Add Validation for Dataset Permission Configuration

**Issue:** #9 (related: #8)
**Date:** 2026-01-23
**Status:** Approved

## Problem

When users create datasets with `uid` or `gid` but without `mode`, the TrueNAS API returns a cryptic error:

```
filesystem_setperm: Value error, Payload must either explicitly
specify permissions or contain the stripacl option.
```

This error occurs at apply time after the dataset has already been created, leaving the resource in a partially configured state.

## Solution

Add Terraform validation that fails early during `terraform validate` / `terraform plan` with a clear error message.

## Implementation

### 1. Add ValidateConfig Interface

**File:** `internal/resources/dataset.go`

Add interface assertion:

```go
var _ resource.ResourceWithValidateConfig = &DatasetResource{}
```

### 2. Implement ValidateConfig Method

```go
func (r *DatasetResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
    var data DatasetResourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    hasMode := !data.Mode.IsNull() && !data.Mode.IsUnknown()
    hasUID := !data.UID.IsNull() && !data.UID.IsUnknown()
    hasGID := !data.GID.IsNull() && !data.GID.IsUnknown()

    if (hasUID || hasGID) && !hasMode {
        resp.Diagnostics.AddAttributeError(
            path.Root("mode"),
            "Mode Required with UID/GID",
            "The 'mode' attribute is required when 'uid' or 'gid' is specified. "+
                "TrueNAS requires explicit permissions when setting ownership.",
        )
    }
}
```

### 3. Remove Dead Code

Remove the `stripacl` fallback in `buildPermParams()` (lines 632-639). With validation in place, this branch is unreachable and the comment is misleading.

Before:
```go
if !data.Mode.IsNull() && !data.Mode.IsUnknown() {
    params["mode"] = data.Mode.ValueString()
} else {
    params["options"] = map[string]any{
        "stripacl": false,
    }
}
```

After:
```go
if !data.Mode.IsNull() && !data.Mode.IsUnknown() {
    params["mode"] = data.Mode.ValueString()
}
```

### 4. Add Tests

Test cases for `ValidateConfig`:
- `uid` without `mode` → error
- `gid` without `mode` → error
- `uid` + `gid` without `mode` → error
- `uid` + `mode` → no error
- `mode` alone → no error
- No permissions set → no error

### 5. Update Documentation

Add note to `docs/resources/dataset.md` under permission attributes:

> **Note:** The `mode` attribute is required when `uid` or `gid` is specified.

## Files to Modify

- `internal/resources/dataset.go` - Add validation, remove dead code
- `internal/resources/dataset_test.go` - Add validation tests
- `docs/resources/dataset.md` - Update attribute documentation

## User Experience

After this change, users see:

```
$ terraform validate
│ Error: Mode Required with UID/GID
│
│   with truenas_dataset.example,
│   on main.tf line 1:
│
│ The 'mode' attribute is required when 'uid' or 'gid' is specified.
│ TrueNAS requires explicit permissions when setting ownership.
```
