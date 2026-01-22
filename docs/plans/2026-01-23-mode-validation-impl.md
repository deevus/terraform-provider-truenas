# Mode Validation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Terraform validation that requires `mode` when `uid` or `gid` is specified on datasets.

**Architecture:** Implement `ResourceWithValidateConfig` interface to catch invalid configurations during `terraform validate`/`terraform plan`. Remove dead code path (stripacl fallback) and update tests accordingly.

**Tech Stack:** Go, Terraform Plugin Framework

**Baseline Coverage:** 87.9% (internal/resources)

---

### Task 1: Add ValidateConfig Interface Assertion

**Files:**
- Modify: `internal/resources/dataset.go:18-20`

**Step 1: Add interface assertion**

Add after line 20:

```go
var _ resource.ResourceWithValidateConfig = &DatasetResource{}
```

**Step 2: Verify compilation fails**

Run: `go build ./...`
Expected: FAIL with "DatasetResource does not implement ResourceWithValidateConfig"

**Step 3: Commit checkpoint (failing build)**

Skip commit - we'll commit after implementing the method.

---

### Task 2: Implement ValidateConfig Method

**Files:**
- Modify: `internal/resources/dataset.go`

**Step 1: Add ValidateConfig method**

Add after the `Configure` method (around line 252):

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

**Step 2: Verify compilation passes**

Run: `go build ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/resources/dataset.go
git commit -m "feat(dataset): add ValidateConfig for mode requirement

Implements ResourceWithValidateConfig interface to validate that mode
is specified when uid or gid is set. This catches invalid configs early
during terraform validate/plan instead of failing at apply time.

Closes #9"
```

---

### Task 3: Write Validation Tests

**Files:**
- Modify: `internal/resources/dataset_test.go`

**Step 1: Write failing tests**

Add after `TestNewDatasetResource` (around line 26):

```go
func TestDatasetResource_ValidateConfig_ModeRequiredWithUID(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// uid set without mode - should fail validation
	configValue := createDatasetResourceModelWithPerms(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil, nil, int64(1000), nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected validation error when uid is set without mode")
	}

	// Verify error message
	found := false
	for _, diag := range resp.Diagnostics.Errors() {
		if diag.Summary() == "Mode Required with UID/GID" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error with summary 'Mode Required with UID/GID'")
	}
}

func TestDatasetResource_ValidateConfig_ModeRequiredWithGID(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// gid set without mode - should fail validation
	configValue := createDatasetResourceModelWithPerms(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, int64(1000))

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected validation error when gid is set without mode")
	}
}

func TestDatasetResource_ValidateConfig_ModeRequiredWithBoth(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// uid and gid set without mode - should fail validation
	configValue := createDatasetResourceModelWithPerms(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil, nil, int64(1000), int64(1000))

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected validation error when uid and gid are set without mode")
	}
}

func TestDatasetResource_ValidateConfig_ModeWithUID_OK(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// uid with mode - should pass validation
	configValue := createDatasetResourceModelWithPerms(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil, "755", int64(1000), nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected validation error: %v", resp.Diagnostics)
	}
}

func TestDatasetResource_ValidateConfig_ModeOnly_OK(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// mode alone - should pass validation
	configValue := createDatasetResourceModelWithPerms(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil, "755", nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected validation error: %v", resp.Diagnostics)
	}
}

func TestDatasetResource_ValidateConfig_NoPermissions_OK(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// no permissions - should pass validation
	configValue := createDatasetResourceModel(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected validation error: %v", resp.Diagnostics)
	}
}
```

**Step 2: Run tests to verify they pass**

Run: `mise run test`
Expected: All new tests PASS

**Step 3: Commit**

```bash
git add internal/resources/dataset_test.go
git commit -m "test(dataset): add ValidateConfig tests for mode requirement"
```

---

### Task 4: Update Interface Assertion in Test

**Files:**
- Modify: `internal/resources/dataset_test.go:22-26`

**Step 1: Update TestNewDatasetResource to verify new interface**

Find this block (around line 22-26):

```go
	// Verify it implements the required interfaces
	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*DatasetResource)
	var _ resource.ResourceWithImportState = r.(*DatasetResource)
```

Add after line 25:

```go
	var _ resource.ResourceWithValidateConfig = r.(*DatasetResource)
```

**Step 2: Run test to verify**

Run: `go test -run TestNewDatasetResource ./internal/resources/ -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/resources/dataset_test.go
git commit -m "test(dataset): verify ValidateConfig interface in TestNewDatasetResource"
```

---

### Task 5: Remove Dead Code (stripacl fallback)

**Files:**
- Modify: `internal/resources/dataset.go:630-639`

**Step 1: Remove the else branch in buildPermParams**

Find this code (lines 630-639):

```go
	if !data.Mode.IsNull() && !data.Mode.IsUnknown() {
		params["mode"] = data.Mode.ValueString()
	} else {
		// TrueNAS API requires either 'mode' or 'options.stripacl' to be set.
		// When only changing ownership (uid/gid), we set stripacl=false to
		// preserve existing permissions and ACLs.
		params["options"] = map[string]any{
			"stripacl": false,
		}
	}
```

Replace with:

```go
	if !data.Mode.IsNull() && !data.Mode.IsUnknown() {
		params["mode"] = data.Mode.ValueString()
	}
```

**Step 2: Verify tests fail**

Run: `go test -run TestDatasetResource_Create_WithUIDGIDOnly ./internal/resources/ -v`
Expected: FAIL - test expects stripacl behavior that no longer exists

**Step 3: Commit (code change only)**

```bash
git add internal/resources/dataset.go
git commit -m "refactor(dataset): remove unreachable stripacl fallback

With ValidateConfig ensuring mode is always present when uid/gid is set,
the else branch in buildPermParams is unreachable. The comment was also
misleading - stripacl=false doesn't actually work (that's issue #8)."
```

---

### Task 6: Fix Broken Test

**Files:**
- Modify: `internal/resources/dataset_test.go:2222-2302`

**Step 1: Delete the test that expects stripacl behavior**

Find and delete `TestDatasetResource_Create_WithUIDGIDOnly` (lines 2222-2302). This test is now invalid because:
1. The configuration (uid/gid without mode) is no longer valid
2. The stripacl fallback code has been removed

**Step 2: Run all tests to verify**

Run: `mise run test`
Expected: All tests PASS

**Step 3: Commit**

```bash
git add internal/resources/dataset_test.go
git commit -m "test(dataset): remove obsolete stripacl test

TestDatasetResource_Create_WithUIDGIDOnly tested the stripacl fallback
which has been removed. The scenario it tested (uid/gid without mode)
now fails validation, so this test is no longer valid."
```

---

### Task 7: Update Documentation

**Files:**
- Modify: `docs/resources/dataset.md`

**Step 1: Add note about mode requirement**

Find the `mode`, `uid`, and `gid` attribute descriptions (lines 23-24, 32) and update them.

Replace lines 23-24 and 32 with:

```markdown
- `gid` (Number) Owner group ID for the dataset mountpoint. Requires `mode` to be set.
- `mode` (String) Unix mode for the dataset mountpoint (e.g., '755'). Required when `uid` or `gid` is specified.
```

And for uid (line 32):

```markdown
- `uid` (Number) Owner user ID for the dataset mountpoint. Requires `mode` to be set.
```

**Step 2: Commit**

```bash
git add docs/resources/dataset.md
git commit -m "docs(dataset): document mode requirement with uid/gid"
```

---

### Task 8: Final Verification

**Step 1: Run full test suite**

Run: `mise run test`
Expected: All tests PASS

**Step 2: Check coverage**

Run: `mise run coverage`
Expected: Coverage >= 87.9% (baseline)

**Step 3: Verify build**

Run: `go build ./...`
Expected: PASS

**Step 4: Run linter (if available)**

Run: `mise run lint` (if task exists)
Expected: PASS or task not found

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Add interface assertion | dataset.go |
| 2 | Implement ValidateConfig | dataset.go |
| 3 | Write validation tests | dataset_test.go |
| 4 | Update interface test | dataset_test.go |
| 5 | Remove dead code | dataset.go |
| 6 | Fix broken test | dataset_test.go |
| 7 | Update documentation | dataset.md |
| 8 | Final verification | - |
