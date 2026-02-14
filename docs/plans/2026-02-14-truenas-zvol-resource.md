# `truenas_zvol` Resource Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `truenas_zvol` resource that manages ZFS volumes (zvols) on TrueNAS, sharing code with the existing `truenas_dataset` resource.

**Architecture:** Zvols use the same `pool.dataset.*` API with `type=VOLUME`. Shared logic (identity resolution, query, delete, response types) is extracted into `pool_dataset_shared.go` so both resources use it. The zvol resource has its own schema with zvol-specific attributes (volsize, volblocksize, sparse) and omits filesystem attributes (mountpoint, mode, uid, gid, atime).

**Tech Stack:** Go, terraform-plugin-framework, `pool.dataset.*` TrueNAS API, `customtypes.SizeStringValue` for human-readable sizes

---

## Coverage Baseline

- `internal/resources`: 90.2%
- `internal/provider`: 89.4%

## Tasks (6 total)

---

### Task 1: Extract shared pool/dataset helpers from `dataset.go`

Extract code used by both `truenas_dataset` and `truenas_zvol` into a shared file. This is a pure refactor -- all existing dataset tests must continue to pass.

**Files:**
- Create: `internal/resources/pool_dataset_shared.go`
- Modify: `internal/resources/dataset.go`
- Test: existing `internal/resources/dataset_test.go` (must still pass)

**Step 1: Create `pool_dataset_shared.go` with extracted code**

```go
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// -- Shared response types for pool.dataset.query --

// propertyValueField represents a ZFS property with a string value (e.g., compression, atime).
// Query responses return these as {"value": "lz4"}.
type propertyValueField struct {
	Value string `json:"value"`
}

// sizePropertyField represents a ZFS size property with both raw and parsed values.
// Query responses return these as {"value": "10G", "parsed": 10737418240}.
type sizePropertyField struct {
	Parsed int64  `json:"parsed"`
	Value  string `json:"value"`
}

// -- Shared identity helpers --

// poolDatasetFullName builds the full dataset/zvol name from pool+path or parent+path.
// Returns "" if the configuration is invalid.
//
// Modes:
//   - pool + path: "tank" + "vms/disk0" -> "tank/vms/disk0"
//   - parent + path: "tank/vms" + "disk0" -> "tank/vms/disk0"
func poolDatasetFullName(pool, path, parent, name types.String) string {
	hasPool := !pool.IsNull() && !pool.IsUnknown() && pool.ValueString() != ""
	hasPath := !path.IsNull() && !path.IsUnknown() && path.ValueString() != ""

	if hasPool && hasPath {
		return fmt.Sprintf("%s/%s", pool.ValueString(), path.ValueString())
	}

	hasParent := !parent.IsNull() && !parent.IsUnknown() && parent.ValueString() != ""
	hasName := !name.IsNull() && !name.IsUnknown() && name.ValueString() != ""

	if hasParent {
		if hasPath {
			return fmt.Sprintf("%s/%s", parent.ValueString(), path.ValueString())
		}
		if hasName {
			return fmt.Sprintf("%s/%s", parent.ValueString(), name.ValueString())
		}
	}

	return ""
}

// poolDatasetIDToParts splits a dataset ID like "tank/vms/disk0" into pool="tank", path="vms/disk0".
func poolDatasetIDToParts(id string) (pool, path string) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return id, ""
}

// -- Shared API operations --

// queryPoolDataset queries a dataset/zvol by ID using pool.dataset.query.
// Returns nil, nil if not found (deleted outside Terraform).
func queryPoolDataset(ctx context.Context, c client.Client, datasetID string) (json.RawMessage, error) {
	filter := [][]any{{"id", "=", datasetID}}
	result, err := c.Call(ctx, "pool.dataset.query", filter)
	if err != nil {
		return nil, err
	}

	var results []json.RawMessage
	if err := json.Unmarshal(result, &results); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	return results[0], nil
}

// deletePoolDataset deletes a dataset/zvol by ID.
// If recursive is true, child datasets are also deleted.
func deletePoolDataset(ctx context.Context, c client.Client, datasetID string, recursive bool) error {
	var params any = datasetID
	if recursive {
		params = []any{datasetID, map[string]bool{"recursive": true}}
	}

	_, err := c.Call(ctx, "pool.dataset.delete", params)
	return err
}

// -- Shared schema attributes --

// poolDatasetIdentitySchema returns the common pool/path/parent identity attributes
// shared by both dataset and zvol resources.
func poolDatasetIdentitySchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "Dataset identifier (pool/path).",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"pool": schema.StringAttribute{
			Description: "Pool name. Use with 'path' attribute.",
			Optional:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"path": schema.StringAttribute{
			Description: "Path within the pool (e.g., 'vms/disk0').",
			Optional:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"parent": schema.StringAttribute{
			Description: "Parent dataset ID (e.g., 'tank/vms'). Use with 'path' attribute.",
			Optional:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
	}
}
```

**Step 2: Update `dataset.go` to use shared code**

Remove from `dataset.go`:
- `propertyValueField` struct (lines 69-71) -- now in shared file
- `sizePropertyField` struct (lines 73-77) -- now in shared file
- `getFullName` function (lines 657-682) -- replaced by `poolDatasetFullName`

Update `dataset.go`:
- Replace `getFullName(&data)` call with `poolDatasetFullName(data.Pool, data.Path, data.Parent, data.Name)`
- Replace `queryDataset` to use `queryPoolDataset` internally
- Replace the delete body to use `deletePoolDataset`
- Remove the `strings` import if no longer used after `getFullName` removal

The `queryDataset` method stays on `DatasetResource` as a thin wrapper that unmarshals into `datasetQueryResponse`:

```go
func (r *DatasetResource) queryDataset(ctx context.Context, datasetID string) (*datasetQueryResponse, error) {
	raw, err := queryPoolDataset(ctx, r.client, datasetID)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	var ds datasetQueryResponse
	if err := json.Unmarshal(raw, &ds); err != nil {
		return nil, fmt.Errorf("parse dataset: %w", err)
	}
	return &ds, nil
}
```

The `Delete` method becomes:

```go
func (r *DatasetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DatasetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	datasetID := data.ID.ValueString()
	recursive := !data.ForceDestroy.IsNull() && data.ForceDestroy.ValueBool()

	if err := deletePoolDataset(ctx, r.client, datasetID, recursive); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Dataset",
			fmt.Sprintf("Unable to delete dataset %q: %s", datasetID, err.Error()),
		)
	}
}
```

**Step 3: Run all dataset tests**

Run: `mise run test -- -run TestDataset -v 2>&1 | tail -5`
Expected: all pass, no failures

**Step 4: Run full test suite**

Run: `mise run test`
Expected: all pass

**Step 5: Commit**

```bash
git add internal/resources/pool_dataset_shared.go internal/resources/dataset.go
git commit -m "refactor: extract shared pool/dataset helpers for zvol reuse"
```

---

### Task 2: Scaffold `truenas_zvol` resource -- model, schema, provider registration

**Files:**
- Create: `internal/resources/zvol.go`
- Create: `internal/resources/zvol_test.go`
- Modify: `internal/provider/provider.go:388-403`

**Step 1: Write the test for schema and provider registration**

In `internal/resources/zvol_test.go`:

```go
package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewZvolResource(t *testing.T) {
	r := NewZvolResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(r.(*ZvolResource))
	_ = resource.ResourceWithImportState(r.(*ZvolResource))
}

func TestZvolResource_Metadata(t *testing.T) {
	r := NewZvolResource().(*ZvolResource)
	req := resource.MetadataRequest{ProviderTypeName: "truenas"}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_zvol" {
		t.Errorf("expected type name 'truenas_zvol', got %q", resp.TypeName)
	}
}

func TestZvolResource_Schema(t *testing.T) {
	schemaResp := getZvolResourceSchema(t)

	expectedAttrs := []string{
		"id", "pool", "path", "parent",
		"volsize", "volblocksize", "sparse", "force_size",
		"compression", "comments",
		"force_destroy",
	}

	for _, attr := range expectedAttrs {
		if _, ok := schemaResp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing expected attribute %q", attr)
		}
	}
}

// -- Test helpers --

func getZvolResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewZvolResource().(*ZvolResource)
	resp := resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema error: %v", resp.Diagnostics)
	}
	return resp
}

// zvolObjectType returns the tftypes.Object for constructing test values.
func zvolObjectType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":           tftypes.String,
			"pool":         tftypes.String,
			"path":         tftypes.String,
			"parent":       tftypes.String,
			"volsize":      tftypes.String,
			"volblocksize": tftypes.String,
			"sparse":       tftypes.Bool,
			"force_size":   tftypes.Bool,
			"compression":  tftypes.String,
			"comments":     tftypes.String,
			"force_destroy": tftypes.Bool,
		},
	}
}

type zvolModelParams struct {
	ID           *string
	Pool         *string
	Path         *string
	Parent       *string
	Volsize      *string
	Volblocksize *string
	Sparse       *bool
	ForceSize    *bool
	Compression  *string
	Comments     *string
	ForceDestroy *bool
}

func createZvolModelValue(p zvolModelParams) tftypes.Value {
	strVal := func(s *string) tftypes.Value {
		if s == nil {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, *s)
	}
	boolVal := func(b *bool) tftypes.Value {
		if b == nil {
			return tftypes.NewValue(tftypes.Bool, nil)
		}
		return tftypes.NewValue(tftypes.Bool, *b)
	}

	return tftypes.NewValue(zvolObjectType(), map[string]tftypes.Value{
		"id":            strVal(p.ID),
		"pool":          strVal(p.Pool),
		"path":          strVal(p.Path),
		"parent":        strVal(p.Parent),
		"volsize":       strVal(p.Volsize),
		"volblocksize":  strVal(p.Volblocksize),
		"sparse":        boolVal(p.Sparse),
		"force_size":    boolVal(p.ForceSize),
		"compression":   strVal(p.Compression),
		"comments":      strVal(p.Comments),
		"force_destroy": boolVal(p.ForceDestroy),
	})
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func defaultZvolPlanParams() zvolModelParams {
	return zvolModelParams{
		Pool:    strPtr("tank"),
		Path:    strPtr("myvol"),
		Volsize: strPtr("10737418240"),
	}
}

// mockZvolQueryResponse returns a mock pool.dataset.query response for a zvol.
func mockZvolQueryResponse(id, compression, comments string, volsize int64, volblocksize string, sparse bool) string {
	return fmt.Sprintf(`[{
		"id": %q,
		"type": "VOLUME",
		"name": %q,
		"pool": "tank",
		"volsize": {"value": "%d", "parsed": %d},
		"volblocksize": {"value": %q, "parsed": 0},
		"sparse": {"value": "%t", "parsed": %t},
		"compression": {"value": %q},
		"comments": {"value": %q}
	}]`, id, id, volsize, volsize, volblocksize, sparse, sparse, compression, comments)
}
```

Note: The `mockZvolQueryResponse` helper and the `fmt` import will be needed -- add `"fmt"` to the import block.

**Step 2: Run tests to verify they fail**

Run: `mise run test -- -run TestZvol -v 2>&1 | tail -5`
Expected: FAIL (ZvolResource not defined)

**Step 3: Write the minimal implementation in `zvol.go`**

```go
package resources

import (
	"context"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	customtypes "github.com/deevus/terraform-provider-truenas/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ZvolResource{}
var _ resource.ResourceWithConfigure = &ZvolResource{}
var _ resource.ResourceWithImportState = &ZvolResource{}

type ZvolResource struct {
	client client.Client
}

type ZvolResourceModel struct {
	ID           types.String                `tfsdk:"id"`
	Pool         types.String                `tfsdk:"pool"`
	Path         types.String                `tfsdk:"path"`
	Parent       types.String                `tfsdk:"parent"`
	Volsize      customtypes.SizeStringValue `tfsdk:"volsize"`
	Volblocksize types.String                `tfsdk:"volblocksize"`
	Sparse       types.Bool                  `tfsdk:"sparse"`
	ForceSize    types.Bool                  `tfsdk:"force_size"`
	Compression  types.String                `tfsdk:"compression"`
	Comments     types.String                `tfsdk:"comments"`
	ForceDestroy types.Bool                  `tfsdk:"force_destroy"`
}

// zvolQueryResponse represents the JSON response from pool.dataset.query for a zvol.
type zvolQueryResponse struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Pool         string             `json:"pool"`
	Volsize      sizePropertyField  `json:"volsize"`
	Volblocksize propertyValueField `json:"volblocksize"`
	Sparse       propertyValueField `json:"sparse"`
	Compression  propertyValueField `json:"compression"`
	Comments     propertyValueField `json:"comments"`
}

func NewZvolResource() resource.Resource {
	return &ZvolResource{}
}

func (r *ZvolResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zvol"
}

func (r *ZvolResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := poolDatasetIdentitySchema()

	// Zvol-specific attributes
	attrs["volsize"] = schema.StringAttribute{
		CustomType:  customtypes.SizeStringType{},
		Description: "Volume size. Accepts human-readable sizes (e.g., '10G', '500M', '1T') or bytes. Must be a multiple of volblocksize.",
		Required:    true,
	}
	attrs["volblocksize"] = schema.StringAttribute{
		Description: "Volume block size. Cannot be changed after creation. Options: 512, 512B, 1K, 2K, 4K, 8K, 16K, 32K, 64K, 128K.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
			stringplanmodifier.RequiresReplace(),
		},
	}
	attrs["sparse"] = schema.BoolAttribute{
		Description: "Create a sparse (thin-provisioned) volume. Defaults to false.",
		Optional:    true,
	}
	attrs["force_size"] = schema.BoolAttribute{
		Description: "Allow setting volsize that is not a multiple of volblocksize, or allow shrinking. Not stored in state.",
		Optional:    true,
	}
	attrs["compression"] = schema.StringAttribute{
		Description: "Compression algorithm (e.g., 'LZ4', 'ZSTD', 'OFF').",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	attrs["comments"] = schema.StringAttribute{
		Description: "Comments / description for this volume.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	attrs["force_destroy"] = schema.BoolAttribute{
		Description: "Force destroy including child datasets. Defaults to false.",
		Optional:    true,
	}

	resp.Schema = schema.Schema{
		Description: "Manages a ZFS volume (zvol) on TrueNAS. Zvols are block devices backed by ZFS, commonly used as VM disks or iSCSI targets.",
		Attributes:  attrs,
	}
}

func (r *ZvolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *ZvolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("Not Implemented", "Create is not yet implemented")
}

func (r *ZvolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("Not Implemented", "Read is not yet implemented")
}

func (r *ZvolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Not Implemented", "Update is not yet implemented")
}

func (r *ZvolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("Not Implemented", "Delete is not yet implemented")
}

func (r *ZvolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

**Step 4: Register in provider**

In `internal/provider/provider.go`, add `resources.NewZvolResource,` to the `Resources()` return list (after `NewVMResource`).

**Step 5: Run tests**

Run: `mise run test -- -run TestZvol -v 2>&1 | tail -10`
Expected: all 3 tests pass

Run: `mise run test`
Expected: full suite passes

**Step 6: Commit**

```bash
git add internal/resources/zvol.go internal/resources/zvol_test.go internal/provider/provider.go
git commit -m "feat(zvol): scaffold truenas_zvol resource with schema and model types"
```

---

### Task 3: Implement Create + Read

**Files:**
- Modify: `internal/resources/zvol.go`
- Modify: `internal/resources/zvol_test.go`

**Step 1: Write tests for Create and Read**

Add to `zvol_test.go`:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestZvolResource_Create_Basic(t *testing.T) {
	var createCalled bool
	var createParams map[string]any

	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.dataset.create" {
					createCalled = true
					createParams = params.(map[string]any)
					return json.RawMessage(`{"id":"tank/myvol"}`), nil
				}
				if method == "pool.dataset.query" {
					return json.RawMessage(mockZvolQueryResponse("tank/myvol", "lz4", "", 10737418240, "16K", false)), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	planValue := createZvolModelValue(defaultZvolPlanParams())

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if !createCalled {
		t.Fatal("expected pool.dataset.create to be called")
	}
	if createParams["name"] != "tank/myvol" {
		t.Errorf("expected name 'tank/myvol', got %v", createParams["name"])
	}
	if createParams["type"] != "VOLUME" {
		t.Errorf("expected type 'VOLUME', got %v", createParams["type"])
	}
	if createParams["volsize"] != int64(10737418240) {
		t.Errorf("expected volsize 10737418240, got %v", createParams["volsize"])
	}

	var model ZvolResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.ID.ValueString() != "tank/myvol" {
		t.Errorf("expected ID 'tank/myvol', got %q", model.ID.ValueString())
	}
}

func TestZvolResource_Create_WithOptionalFields(t *testing.T) {
	var createParams map[string]any

	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.dataset.create" {
					createParams = params.(map[string]any)
					return json.RawMessage(`{"id":"tank/myvol"}`), nil
				}
				if method == "pool.dataset.query" {
					return json.RawMessage(mockZvolQueryResponse("tank/myvol", "zstd", "test vol", 10737418240, "64K", true)), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.Volblocksize = strPtr("64K")
	p.Sparse = boolPtr(true)
	p.Compression = strPtr("zstd")
	p.Comments = strPtr("test vol")
	planValue := createZvolModelValue(p)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if createParams["volblocksize"] != "64K" {
		t.Errorf("expected volblocksize '64K', got %v", createParams["volblocksize"])
	}
	if createParams["sparse"] != true {
		t.Errorf("expected sparse true, got %v", createParams["sparse"])
	}
	if createParams["compression"] != "zstd" {
		t.Errorf("expected compression 'zstd', got %v", createParams["compression"])
	}
	if createParams["comments"] != "test vol" {
		t.Errorf("expected comments 'test vol', got %v", createParams["comments"])
	}
}

func TestZvolResource_Create_InvalidName(t *testing.T) {
	r := &ZvolResource{client: &client.MockClient{}}

	schemaResp := getZvolResourceSchema(t)
	p := zvolModelParams{Volsize: strPtr("10G")} // no pool/path
	planValue := createZvolModelValue(p)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid name")
	}
}

func TestZvolResource_Create_APIError(t *testing.T) {
	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("pool not found")
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	planValue := createZvolModelValue(defaultZvolPlanParams())

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API failure")
	}
}

func TestZvolResource_Read_Basic(t *testing.T) {
	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(mockZvolQueryResponse("tank/myvol", "lz4", "", 10737418240, "16K", false)), nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	stateValue := createZvolModelValue(p)

	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model ZvolResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.ID.ValueString() != "tank/myvol" {
		t.Errorf("expected ID 'tank/myvol', got %q", model.ID.ValueString())
	}
	if model.Compression.ValueString() != "lz4" {
		t.Errorf("expected compression 'lz4', got %q", model.Compression.ValueString())
	}
}

func TestZvolResource_Read_NotFound(t *testing.T) {
	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/deleted")
	stateValue := createZvolModelValue(p)

	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// State should be removed (resource deleted outside Terraform)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed for deleted zvol")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `mise run test -- -run "TestZvolResource_Create|TestZvolResource_Read" -v 2>&1 | tail -5`
Expected: FAIL (Create/Read return "Not Implemented")

**Step 3: Implement Create and Read**

Replace the stub Create in `zvol.go`:

```go
func (r *ZvolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ZvolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fullName := poolDatasetFullName(data.Pool, data.Path, data.Parent, types.StringNull())
	if fullName == "" {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Either 'pool' with 'path', or 'parent' with 'path' must be provided.",
		)
		return
	}

	// Parse volsize
	volsizeBytes, err := api.ParseSize(data.Volsize.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Volsize", fmt.Sprintf("Unable to parse volsize %q: %s", data.Volsize.ValueString(), err.Error()))
		return
	}

	params := map[string]any{
		"name":    fullName,
		"type":    "VOLUME",
		"volsize": volsizeBytes,
	}

	if !data.Volblocksize.IsNull() && !data.Volblocksize.IsUnknown() {
		params["volblocksize"] = data.Volblocksize.ValueString()
	}
	if !data.Sparse.IsNull() && !data.Sparse.IsUnknown() {
		params["sparse"] = data.Sparse.ValueBool()
	}
	if !data.ForceSize.IsNull() && !data.ForceSize.IsUnknown() {
		params["force_size"] = data.ForceSize.ValueBool()
	}
	if !data.Compression.IsNull() && !data.Compression.IsUnknown() {
		params["compression"] = data.Compression.ValueString()
	}
	if !data.Comments.IsNull() && !data.Comments.IsUnknown() {
		params["comments"] = data.Comments.ValueString()
	}

	result, err := r.client.Call(ctx, "pool.dataset.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Zvol",
			fmt.Sprintf("Unable to create zvol %q: %s", fullName, err.Error()),
		)
		return
	}

	// Parse create response to get ID
	var createResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(result, &createResp); err != nil {
		resp.Diagnostics.AddError("Unable to Parse Response", fmt.Sprintf("Unable to parse create response: %s", err.Error()))
		return
	}

	// Query to get all computed attributes
	r.readZvol(ctx, createResp.ID, &data, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZvolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ZvolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zvolID := data.ID.ValueString()

	raw, err := queryPoolDataset(ctx, r.client, zvolID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Zvol", fmt.Sprintf("Unable to read zvol %q: %s", zvolID, err.Error()))
		return
	}

	if raw == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var zvol zvolQueryResponse
	if err := json.Unmarshal(raw, &zvol); err != nil {
		resp.Diagnostics.AddError("Unable to Parse Response", fmt.Sprintf("Unable to parse zvol response: %s", err.Error()))
		return
	}

	mapZvolToModel(&zvol, &data)

	// Populate pool/path from ID if not set (e.g., after import)
	if data.Pool.IsNull() && data.Path.IsNull() && data.Parent.IsNull() {
		pool, path := poolDatasetIDToParts(zvol.ID)
		data.Pool = types.StringValue(pool)
		data.Path = types.StringValue(path)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// readZvol queries a zvol and maps it into the model.
// Used by Create (after create) and shared read logic.
func (r *ZvolResource) readZvol(ctx context.Context, zvolID string, data *ZvolResourceModel, resp *resource.CreateResponse) {
	raw, err := queryPoolDataset(ctx, r.client, zvolID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Zvol After Create", fmt.Sprintf("Zvol was created but unable to read it: %s", err.Error()))
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Zvol Not Found After Create", fmt.Sprintf("Zvol %q was created but could not be found", zvolID))
		return
	}

	var zvol zvolQueryResponse
	if err := json.Unmarshal(raw, &zvol); err != nil {
		resp.Diagnostics.AddError("Unable to Parse Response", fmt.Sprintf("Unable to parse zvol response: %s", err.Error()))
		return
	}

	mapZvolToModel(&zvol, data)
}

// mapZvolToModel maps a query response to the resource model.
func mapZvolToModel(zvol *zvolQueryResponse, data *ZvolResourceModel) {
	data.ID = types.StringValue(zvol.ID)
	data.Volsize = customtypes.NewSizeStringValue(fmt.Sprintf("%d", zvol.Volsize.Parsed))
	data.Volblocksize = types.StringValue(zvol.Volblocksize.Value)
	data.Compression = types.StringValue(zvol.Compression.Value)

	if zvol.Comments.Value != "" {
		data.Comments = types.StringValue(zvol.Comments.Value)
	} else {
		data.Comments = types.StringNull()
	}
}
```

Add to imports in `zvol.go`:

```go
"encoding/json"

"github.com/deevus/terraform-provider-truenas/internal/api"
```

**Step 4: Run tests**

Run: `mise run test -- -run "TestZvolResource_Create|TestZvolResource_Read" -v 2>&1 | tail -10`
Expected: all pass

**Step 5: Commit**

```bash
git add internal/resources/zvol.go internal/resources/zvol_test.go
git commit -m "feat(zvol): implement Create and Read operations"
```

---

### Task 4: Implement Update + Delete

**Files:**
- Modify: `internal/resources/zvol.go`
- Modify: `internal/resources/zvol_test.go`

**Step 1: Write tests for Update and Delete**

Add to `zvol_test.go`:

```go
func TestZvolResource_Update_Volsize(t *testing.T) {
	var updateID string
	var updateParams map[string]any

	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.dataset.update" {
					args := params.([]any)
					updateID = args[0].(string)
					updateParams = args[1].(map[string]any)
					return json.RawMessage(`{"id":"tank/myvol"}`), nil
				}
				if method == "pool.dataset.query" {
					return json.RawMessage(mockZvolQueryResponse("tank/myvol", "lz4", "", 21474836480, "16K", false)), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)

	stateP := defaultZvolPlanParams()
	stateP.ID = strPtr("tank/myvol")
	stateP.Volblocksize = strPtr("16K")
	stateP.Compression = strPtr("lz4")
	stateValue := createZvolModelValue(stateP)

	planP := defaultZvolPlanParams()
	planP.ID = strPtr("tank/myvol")
	planP.Volsize = strPtr("21474836480") // doubled
	planP.Volblocksize = strPtr("16K")
	planP.Compression = strPtr("lz4")
	planValue := createZvolModelValue(planP)

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if updateID != "tank/myvol" {
		t.Errorf("expected update ID 'tank/myvol', got %q", updateID)
	}
	if updateParams["volsize"] != int64(21474836480) {
		t.Errorf("expected volsize 21474836480, got %v", updateParams["volsize"])
	}
}

func TestZvolResource_Update_NoChanges(t *testing.T) {
	var updateCalled bool

	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.dataset.update" {
					updateCalled = true
					return json.RawMessage(`{"id":"tank/myvol"}`), nil
				}
				if method == "pool.dataset.query" {
					return json.RawMessage(mockZvolQueryResponse("tank/myvol", "lz4", "", 10737418240, "16K", false)), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)

	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	p.Volblocksize = strPtr("16K")
	p.Compression = strPtr("lz4")
	value := createZvolModelValue(p)

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: value},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: value},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if updateCalled {
		t.Error("expected pool.dataset.update to NOT be called when nothing changed")
	}
}

func TestZvolResource_Update_APIError(t *testing.T) {
	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.dataset.update" {
					return nil, errors.New("update failed")
				}
				return nil, nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)

	stateP := defaultZvolPlanParams()
	stateP.ID = strPtr("tank/myvol")
	stateP.Volblocksize = strPtr("16K")
	stateP.Compression = strPtr("lz4")
	stateValue := createZvolModelValue(stateP)

	planP := defaultZvolPlanParams()
	planP.ID = strPtr("tank/myvol")
	planP.Volsize = strPtr("21474836480")
	planP.Volblocksize = strPtr("16K")
	planP.Compression = strPtr("lz4")
	planValue := createZvolModelValue(planP)

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for update API failure")
	}
}

func TestZvolResource_Delete_Basic(t *testing.T) {
	var deleteCalled bool
	var deleteID string

	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.dataset.delete" {
					deleteCalled = true
					deleteID = params.(string)
					return json.RawMessage(`true`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	stateValue := createZvolModelValue(p)

	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if !deleteCalled {
		t.Fatal("expected pool.dataset.delete to be called")
	}
	if deleteID != "tank/myvol" {
		t.Errorf("expected delete ID 'tank/myvol', got %q", deleteID)
	}
}

func TestZvolResource_Delete_ForceDestroy(t *testing.T) {
	var deleteParams []any

	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.dataset.delete" {
					deleteParams = params.([]any)
					return json.RawMessage(`true`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	p.ForceDestroy = boolPtr(true)
	stateValue := createZvolModelValue(p)

	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if len(deleteParams) != 2 {
		t.Fatalf("expected 2 delete params (id + options), got %d", len(deleteParams))
	}
	opts := deleteParams[1].(map[string]bool)
	if !opts["recursive"] {
		t.Error("expected recursive=true for force_destroy")
	}
}

func TestZvolResource_Delete_APIError(t *testing.T) {
	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("zvol is busy")
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	stateValue := createZvolModelValue(p)

	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for delete API failure")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `mise run test -- -run "TestZvolResource_Update|TestZvolResource_Delete" -v 2>&1 | tail -5`
Expected: FAIL (returns "Not Implemented")

**Step 3: Implement Update and Delete**

Replace stubs in `zvol.go`:

```go
func (r *ZvolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ZvolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateParams := map[string]any{}

	// Check volsize change
	if !plan.Volsize.Equal(state.Volsize) {
		volsizeBytes, err := api.ParseSize(plan.Volsize.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid Volsize", fmt.Sprintf("Unable to parse volsize %q: %s", plan.Volsize.ValueString(), err.Error()))
			return
		}
		updateParams["volsize"] = volsizeBytes
	}

	if !plan.Compression.Equal(state.Compression) && !plan.Compression.IsNull() {
		updateParams["compression"] = plan.Compression.ValueString()
	}

	if !plan.Comments.Equal(state.Comments) {
		if plan.Comments.IsNull() {
			updateParams["comments"] = ""
		} else {
			updateParams["comments"] = plan.Comments.ValueString()
		}
	}

	if !plan.ForceSize.IsNull() && !plan.ForceSize.IsUnknown() && plan.ForceSize.ValueBool() {
		updateParams["force_size"] = true
	}

	zvolID := state.ID.ValueString()

	if len(updateParams) > 0 {
		_, err := r.client.Call(ctx, "pool.dataset.update", []any{zvolID, updateParams})
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Update Zvol",
				fmt.Sprintf("Unable to update zvol %q: %s", zvolID, err.Error()),
			)
			return
		}
	}

	// Re-read to get current state
	raw, err := queryPoolDataset(ctx, r.client, zvolID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Zvol After Update", fmt.Sprintf("Unable to read zvol %q: %s", zvolID, err.Error()))
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Zvol Not Found After Update", fmt.Sprintf("Zvol %q not found after update", zvolID))
		return
	}

	var zvol zvolQueryResponse
	if err := json.Unmarshal(raw, &zvol); err != nil {
		resp.Diagnostics.AddError("Unable to Parse Response", fmt.Sprintf("Unable to parse zvol response: %s", err.Error()))
		return
	}

	mapZvolToModel(&zvol, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZvolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ZvolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zvolID := data.ID.ValueString()
	recursive := !data.ForceDestroy.IsNull() && data.ForceDestroy.ValueBool()

	if err := deletePoolDataset(ctx, r.client, zvolID, recursive); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Zvol",
			fmt.Sprintf("Unable to delete zvol %q: %s", zvolID, err.Error()),
		)
	}
}
```

**Step 4: Run tests**

Run: `mise run test -- -run TestZvolResource -v 2>&1 | tail -10`
Expected: all pass

Run: `mise run test`
Expected: full suite passes

**Step 5: Commit**

```bash
git add internal/resources/zvol.go internal/resources/zvol_test.go
git commit -m "feat(zvol): implement Update and Delete operations"
```

---

### Task 5: Import + edge case tests + comprehensive coverage

**Files:**
- Modify: `internal/resources/zvol_test.go`

**Step 1: Write import and edge case tests**

Add to `zvol_test.go`:

```go
func TestZvolResource_ImportState(t *testing.T) {
	r := NewZvolResource().(*ZvolResource)
	schemaResp := getZvolResourceSchema(t)

	emptyState := createZvolModelValue(defaultZvolPlanParams())

	req := resource.ImportStateRequest{ID: "tank/myvol"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: emptyState},
	}

	r.ImportState(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model ZvolResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.ID.ValueString() != "tank/myvol" {
		t.Errorf("expected ID 'tank/myvol', got %q", model.ID.ValueString())
	}
}

func TestZvolResource_Read_PopulatesPoolPath_AfterImport(t *testing.T) {
	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(mockZvolQueryResponse("tank/vms/disk0", "lz4", "", 10737418240, "16K", false)), nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	// After import, only ID is set -- pool/path/parent are null
	p := zvolModelParams{
		ID:      strPtr("tank/vms/disk0"),
		Volsize: strPtr("10737418240"),
	}
	stateValue := createZvolModelValue(p)

	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model ZvolResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.Pool.ValueString() != "tank" {
		t.Errorf("expected pool 'tank', got %q", model.Pool.ValueString())
	}
	if model.Path.ValueString() != "vms/disk0" {
		t.Errorf("expected path 'vms/disk0', got %q", model.Path.ValueString())
	}
}

func TestZvolResource_Read_APIError(t *testing.T) {
	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection failed")
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	stateValue := createZvolModelValue(p)

	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for read API failure")
	}
}

func TestZvolResource_Create_QueryAfterCreateFails(t *testing.T) {
	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.dataset.create" {
					return json.RawMessage(`{"id":"tank/myvol"}`), nil
				}
				if method == "pool.dataset.query" {
					return nil, errors.New("query failed")
				}
				return nil, nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	planValue := createZvolModelValue(defaultZvolPlanParams())

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when query after create fails")
	}
}

func TestZvolResource_Configure_NilProviderData(t *testing.T) {
	r := NewZvolResource().(*ZvolResource)

	req := resource.ConfigureRequest{ProviderData: nil}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestZvolResource_Configure_WrongType(t *testing.T) {
	r := NewZvolResource().(*ZvolResource)

	req := resource.ConfigureRequest{ProviderData: "not a client"}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong provider data type")
	}
}

// -- Shared helper tests --

func TestPoolDatasetFullName(t *testing.T) {
	tests := []struct {
		name     string
		pool     string
		path     string
		parent   string
		nameAttr string
		want     string
	}{
		{"pool+path", "tank", "vms/disk0", "", "", "tank/vms/disk0"},
		{"parent+path", "", "disk0", "tank/vms", "", "tank/vms/disk0"},
		{"parent+name", "", "", "tank/vms", "disk0", "tank/vms/disk0"},
		{"nothing", "", "", "", "", ""},
		{"pool only", "tank", "", "", "", ""},
		{"path only", "", "disk0", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toStr := func(s string) types.String {
				if s == "" {
					return types.StringNull()
				}
				return types.StringValue(s)
			}
			got := poolDatasetFullName(toStr(tt.pool), toStr(tt.path), toStr(tt.parent), toStr(tt.nameAttr))
			if got != tt.want {
				t.Errorf("poolDatasetFullName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPoolDatasetIDToParts(t *testing.T) {
	tests := []struct {
		id       string
		wantPool string
		wantPath string
	}{
		{"tank/vms/disk0", "tank", "vms/disk0"},
		{"tank/disk0", "tank", "disk0"},
		{"tank", "tank", ""},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			pool, path := poolDatasetIDToParts(tt.id)
			if pool != tt.wantPool {
				t.Errorf("pool = %q, want %q", pool, tt.wantPool)
			}
			if path != tt.wantPath {
				t.Errorf("path = %q, want %q", path, tt.wantPath)
			}
		})
	}
}
```

Add `"github.com/hashicorp/terraform-plugin-framework/types"` to the import block in `zvol_test.go`.

**Step 2: Run tests**

Run: `mise run test -- -run TestZvolResource -v 2>&1 | tail -10`
Expected: all pass

Run: `mise run test -- -run "TestPoolDataset" -v 2>&1 | tail -10`
Expected: all pass

**Step 3: Run full suite + coverage**

Run: `mise run test`
Expected: all pass

Run: `mise run coverage`
Expected: `internal/resources` >= 90.2%

**Step 4: Commit**

```bash
git add internal/resources/zvol_test.go
git commit -m "test(zvol): add import, edge case, and shared helper tests"
```

---

### Task 6: Examples + docs + final verification

**Files:**
- Create: `examples/resources/zvol/resource.tf`
- Create: `docs/resources/zvol.md`
- Delete: `docs/plans/2026-02-14-truenas-zvol-resource.md` (this plan)

**Step 1: Create example**

`examples/resources/zvol/resource.tf`:

```hcl
# Basic zvol for VM disk
resource "truenas_zvol" "vm_disk" {
  pool    = "tank"
  path    = "vms/my-vm-disk0"
  volsize = "50G"
}

# Zvol with all options
resource "truenas_zvol" "data_volume" {
  pool         = "tank"
  path         = "iscsi/target0-lun0"
  volsize      = "100G"
  volblocksize = "16K"
  sparse       = true
  compression  = "LZ4"
  comments     = "iSCSI target LUN"
}

# Using parent instead of pool
resource "truenas_zvol" "child_volume" {
  parent  = "tank/vms"
  path    = "disk1"
  volsize = "20G"
}
```

**Step 2: Create docs**

`docs/resources/zvol.md`:

```markdown
---
page_title: "truenas_zvol Resource - terraform-provider-truenas"
subcategory: ""
description: |-
  Manages a ZFS volume (zvol) on TrueNAS.
---

# truenas_zvol (Resource)

Manages a ZFS volume (zvol) on TrueNAS. Zvols are block devices backed by ZFS, commonly used as VM disks or iSCSI targets.

## Example Usage

### Basic VM Disk

` ``terraform
resource "truenas_zvol" "vm_disk" {
  pool    = "tank"
  path    = "vms/my-vm-disk0"
  volsize = "50G"
}
` ``

### Zvol with All Options

` ``terraform
resource "truenas_zvol" "data_volume" {
  pool         = "tank"
  path         = "iscsi/target0-lun0"
  volsize      = "100G"
  volblocksize = "16K"
  sparse       = true
  compression  = "LZ4"
  comments     = "iSCSI target LUN"
}
` ``

## Import

Zvols can be imported using the dataset ID (pool/path):

` ``shell
terraform import truenas_zvol.example tank/vms/my-vm-disk0
` ``

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `volsize` (String) Volume size. Accepts human-readable sizes (e.g., '10G', '500M', '1T') or bytes. Must be a multiple of volblocksize.

### Optional

- `pool` (String) Pool name. Use with 'path' attribute.
- `path` (String) Path within the pool (e.g., 'vms/disk0').
- `parent` (String) Parent dataset ID (e.g., 'tank/vms'). Use with 'path' attribute.
- `volblocksize` (String) Volume block size. Cannot be changed after creation. Options: 512, 512B, 1K, 2K, 4K, 8K, 16K, 32K, 64K, 128K.
- `sparse` (Boolean) Create a sparse (thin-provisioned) volume. Defaults to false.
- `force_size` (Boolean) Allow setting volsize that is not a multiple of volblocksize, or allow shrinking.
- `compression` (String) Compression algorithm (e.g., 'LZ4', 'ZSTD', 'OFF').
- `comments` (String) Comments / description for this volume.
- `force_destroy` (Boolean) Force destroy including child datasets. Defaults to false.

### Read-Only

- `id` (String) Dataset identifier (pool/path).
```

Note: Fix the triple backtick escaping above -- they should be proper markdown code fences (the backslash-space is only to avoid breaking this plan's markdown).

**Step 3: Run full test suite**

Run: `mise run test`
Expected: all pass

**Step 4: Run coverage**

Run: `mise run coverage`
Expected: `internal/resources` >= 90.2%, `internal/provider` >= 89.4%

**Step 5: Clean up plan file**

Delete `docs/plans/2026-02-14-truenas-zvol-resource.md`.

**Step 6: Commit**

```bash
git add examples/resources/zvol/resource.tf docs/resources/zvol.md
git rm docs/plans/2026-02-14-truenas-zvol-resource.md
git commit -m "docs: add truenas_zvol resource example and documentation"
```

**Step 7: Use finishing-a-development-branch skill**

Follow `superpowers:finishing-a-development-branch` to verify tests, present completion options, and handle merge/PR/cleanup.
