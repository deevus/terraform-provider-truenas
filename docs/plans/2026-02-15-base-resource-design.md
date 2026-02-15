# Extract BaseResource Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extract a `BaseResource` struct to eliminate duplicated `Configure()`, `ImportState()`, and `client` fields across all 13 resources.

**Architecture:** Create `BaseResource` with promoted `Configure()` and `ImportState()` methods. Each resource embeds `BaseResource` instead of declaring its own `client` field. Resources with custom `ImportState` override the promoted method. Go embedding promotes both the field and methods, so all existing `r.client` references work unchanged.

**Tech Stack:** Go, Terraform Plugin Framework

**Issue:** #39
**Branch:** `refactor/base-resource`
**Coverage baseline:** 90.2% (internal/resources)

---

### Task 1: Create BaseResource with tests

**Files:**
- Create: `internal/resources/base.go`
- Create: `internal/resources/base_test.go`

**Step 1: Write the base resource implementation**

Create `internal/resources/base.go`:

```go
package resources

import (
	"context"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// BaseResource provides shared Configure and ImportState behavior for all resources.
// Embed this in resource structs to inherit the client field and standard methods.
type BaseResource struct {
	client client.Client
}

func (b *BaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	b.client = c
}

func (b *BaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

**Step 2: Write the tests**

Create `internal/resources/base_test.go`:

```go
package resources

import (
	"context"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestBaseResource_Configure_NilProviderData(t *testing.T) {
	b := &BaseResource{}

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	b.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if b.client != nil {
		t.Error("expected client to remain nil")
	}
}

func TestBaseResource_Configure_WrongType(t *testing.T) {
	b := &BaseResource{}

	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}

	b.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}

	found := false
	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == "Unexpected Resource Configure Type" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Unexpected Resource Configure Type' diagnostic")
	}
}

func TestBaseResource_Configure_Success(t *testing.T) {
	b := &BaseResource{}

	mockClient := &client.MockClient{}

	req := resource.ConfigureRequest{
		ProviderData: mockClient,
	}
	resp := &resource.ConfigureResponse{}

	b.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if b.client != mockClient {
		t.Error("expected client to be set to mockClient")
	}
}

func TestBaseResource_ImportState(t *testing.T) {
	b := &BaseResource{}

	idType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id": tftypes.String,
		},
	}

	rawState := tftypes.NewValue(idType, map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, "test-id"),
	})

	req := resource.ImportStateRequest{
		ID: "imported-id",
	}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Raw:    rawState,
			Schema: importStateTestSchema(),
		},
	}

	b.ImportState(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}
```

Note: `importStateTestSchema()` — check if a similar helper already exists in the test files. If not, add a minimal one:

```go
func importStateTestSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}
```

Check existing test files for how `ImportState` tests set up `tfsdk.State` — match that pattern exactly.

**Step 3: Run tests to verify they pass**

Run: `go test ./internal/resources/ -run TestBaseResource -v`
Expected: 4 PASS

**Step 4: Verify 100% coverage on base.go**

Run: `go test ./internal/resources/ -run TestBaseResource -coverprofile=base_coverage.out && go tool cover -func=base_coverage.out | grep base.go`
Expected: 100.0%

**Step 5: Commit**

```bash
git add internal/resources/base.go internal/resources/base_test.go
git commit -m "refactor: add BaseResource with Configure and ImportState"
```

---

### Task 2: Migrate Group A resources (delete both Configure and ImportState)

9 resources that use standard `Configure()` + standard `ImportStatePassthroughID`:

1. `internal/resources/app_registry.go` — struct line 37, Configure lines 90-105, ImportState lines 285-287
2. `internal/resources/cloudsync_credentials.go` — struct line 59, Configure lines 156-171, ImportState lines 481-483
3. `internal/resources/cloudsync_task.go` — struct line 94, Configure lines 295-310, ImportState lines 738-740
4. `internal/resources/cron_job.go` — struct line 40, Configure lines 133-148, ImportState lines 328-330
5. `internal/resources/dataset.go` — struct line 25, Configure lines 235-251, ImportState lines 621-623
6. `internal/resources/file.go` — struct line 28, Configure lines 114-129, ImportState lines 438-440
7. `internal/resources/snapshot.go` — struct line 26, Configure lines 117-132, ImportState lines 373-375
8. `internal/resources/vm.go` — struct line 176, Configure lines 504-519, ImportState lines 796-798
9. `internal/resources/zvol.go` — struct line 23, Configure lines 114-129, ImportState lines 329-331

**For each resource, apply these changes:**

**Step 1: Change struct to embed BaseResource**

Before:
```go
type AppRegistryResource struct {
	client client.Client
}
```

After:
```go
type AppRegistryResource struct {
	BaseResource
}
```

**Step 2: Delete the Configure() method entirely**

Remove the full `func (r *XxxResource) Configure(...)` method.

**Step 3: Delete the ImportState() method entirely**

Remove the full `func (r *XxxResource) ImportState(...)` method.

**Step 4: Remove unused imports**

After deleting `Configure()`, the `"fmt"` import may become unused. After deleting `ImportState()`, the `"github.com/hashicorp/terraform-plugin-framework/path"` import may become unused. Remove any unused imports.

Keep the compile-time interface checks (`var _ resource.ResourceWithConfigure = ...` and `var _ resource.ResourceWithImportState = ...`) — these verify embedding promotes the methods correctly.

**Step 5: Remove duplicate Configure tests from each resource's test file**

Delete these test functions from each resource's test file:
- `TestXxxResource_Configure_Success`
- `TestXxxResource_Configure_NilProviderData`
- `TestXxxResource_Configure_WrongType`

These are now tested via `TestBaseResource_Configure_*` in `base_test.go`.

Keep the `ImportState` tests in each resource's test file — they verify the embedding works end-to-end.

Keep the interface checks in test `NewXxxResource` functions (`_ = resource.ResourceWithConfigure(...)`, `_ = resource.ResourceWithImportState(...)`).

**Step 6: Run tests**

Run: `go test ./internal/resources/ -v`
Expected: All PASS

**Step 7: Commit**

```bash
git add internal/resources/app_registry.go internal/resources/app_registry_test.go \
       internal/resources/cloudsync_credentials.go internal/resources/cloudsync_credentials_test.go \
       internal/resources/cloudsync_task.go internal/resources/cloudsync_task_test.go \
       internal/resources/cron_job.go internal/resources/cron_job_test.go \
       internal/resources/dataset.go internal/resources/dataset_test.go \
       internal/resources/file.go internal/resources/file_test.go \
       internal/resources/snapshot.go internal/resources/snapshot_test.go \
       internal/resources/vm.go internal/resources/vm_test.go \
       internal/resources/zvol.go internal/resources/zvol_test.go
git commit -m "refactor: migrate 9 resources to embed BaseResource"
```

---

### Task 3: Migrate Group B resources (delete Configure only, keep custom ImportState)

4 resources with custom `ImportState()`:

1. `internal/resources/app.go` — struct line 29, Configure lines 135-151. ImportState (lines 531-535) sets both `id` and `name`. **Keep ImportState.**
2. `internal/resources/host_path.go` — struct line 25, Configure lines 106-122. ImportState (lines 373-377) sets both `id` and `path`. **Keep ImportState.**
3. `internal/resources/virt_config.go` — struct line 41, Configure lines 85-100. ImportState (lines 226-237) validates singleton ID. **Keep ImportState.**
4. `internal/resources/virt_instance.go` — struct line 144, Configure lines 354-369. ImportState (lines 756-759) sets `name` from ID. **Keep ImportState.**

**For each resource, apply these changes:**

**Step 1: Change struct to embed BaseResource**

Before:
```go
type AppResource struct {
	client client.Client
}
```

After:
```go
type AppResource struct {
	BaseResource
}
```

**Step 2: Delete the Configure() method entirely**

**Step 3: Keep the ImportState() method — it overrides the embedded default**

**Step 4: Remove unused imports**

`"fmt"` may become unused after removing Configure. Remove if so.

**Step 5: Remove duplicate Configure tests**

Delete `TestXxxResource_Configure_Success`, `TestXxxResource_Configure_NilProviderData`, `TestXxxResource_Configure_WrongType` from each resource's test file.

Keep the custom `ImportState` tests — they test the override behavior.

**Step 6: Run tests**

Run: `go test ./internal/resources/ -v`
Expected: All PASS

**Step 7: Commit**

```bash
git add internal/resources/app.go internal/resources/app_test.go \
       internal/resources/host_path.go internal/resources/host_path_test.go \
       internal/resources/virt_config.go internal/resources/virt_config_test.go \
       internal/resources/virt_instance.go internal/resources/virt_instance_test.go
git commit -m "refactor: migrate 4 custom-ImportState resources to embed BaseResource"
```

---

### Task 4: Final verification

**Step 1: Run full test suite**

Run: `mise run test`
Expected: All PASS

**Step 2: Verify coverage**

Run: `mise run coverage`
Expected: `internal/resources` coverage >= 90.2%

**Step 3: Commit plan cleanup**

```bash
rm docs/plans/2026-02-15-base-resource-design.md
rmdir docs/plans 2>/dev/null || true
git add -A docs/plans/
git commit -m "chore: clean up plan docs"
```
