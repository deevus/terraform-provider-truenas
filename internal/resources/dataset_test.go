package resources

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewDatasetResource(t *testing.T) {
	r := NewDatasetResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	// Verify it implements the required interfaces
	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*DatasetResource)
	var _ resource.ResourceWithImportState = r.(*DatasetResource)
}

func TestDatasetResource_Metadata(t *testing.T) {
	r := NewDatasetResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_dataset" {
		t.Errorf("expected TypeName 'truenas_dataset', got %q", resp.TypeName)
	}
}

func TestDatasetResource_Schema(t *testing.T) {
	r := NewDatasetResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify schema has description
	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify id attribute exists and is computed
	idAttr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal("expected 'id' attribute in schema")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' attribute to be computed")
	}

	// Verify pool attribute exists and is optional
	poolAttr, ok := resp.Schema.Attributes["pool"]
	if !ok {
		t.Fatal("expected 'pool' attribute in schema")
	}
	if !poolAttr.IsOptional() {
		t.Error("expected 'pool' attribute to be optional")
	}

	// Verify path attribute exists and is optional
	pathAttr, ok := resp.Schema.Attributes["path"]
	if !ok {
		t.Fatal("expected 'path' attribute in schema")
	}
	if !pathAttr.IsOptional() {
		t.Error("expected 'path' attribute to be optional")
	}

	// Verify parent attribute exists and is optional
	parentAttr, ok := resp.Schema.Attributes["parent"]
	if !ok {
		t.Fatal("expected 'parent' attribute in schema")
	}
	if !parentAttr.IsOptional() {
		t.Error("expected 'parent' attribute to be optional")
	}

	// Verify name attribute exists and is optional
	nameAttr, ok := resp.Schema.Attributes["name"]
	if !ok {
		t.Fatal("expected 'name' attribute in schema")
	}
	if !nameAttr.IsOptional() {
		t.Error("expected 'name' attribute to be optional")
	}

	// Verify mount_path attribute exists and is computed
	mountPathAttr, ok := resp.Schema.Attributes["mount_path"]
	if !ok {
		t.Fatal("expected 'mount_path' attribute in schema")
	}
	if !mountPathAttr.IsComputed() {
		t.Error("expected 'mount_path' attribute to be computed")
	}

	// Verify compression attribute exists and is optional
	compressionAttr, ok := resp.Schema.Attributes["compression"]
	if !ok {
		t.Fatal("expected 'compression' attribute in schema")
	}
	if !compressionAttr.IsOptional() {
		t.Error("expected 'compression' attribute to be optional")
	}

	// Verify quota attribute exists and is optional
	quotaAttr, ok := resp.Schema.Attributes["quota"]
	if !ok {
		t.Fatal("expected 'quota' attribute in schema")
	}
	if !quotaAttr.IsOptional() {
		t.Error("expected 'quota' attribute to be optional")
	}

	// Verify refquota attribute exists and is optional
	refquotaAttr, ok := resp.Schema.Attributes["refquota"]
	if !ok {
		t.Fatal("expected 'refquota' attribute in schema")
	}
	if !refquotaAttr.IsOptional() {
		t.Error("expected 'refquota' attribute to be optional")
	}

	// Verify atime attribute exists and is optional
	atimeAttr, ok := resp.Schema.Attributes["atime"]
	if !ok {
		t.Fatal("expected 'atime' attribute in schema")
	}
	if !atimeAttr.IsOptional() {
		t.Error("expected 'atime' attribute to be optional")
	}

	// Verify force_destroy attribute exists and is optional
	forceDestroyAttr, ok := resp.Schema.Attributes["force_destroy"]
	if !ok {
		t.Fatal("expected 'force_destroy' attribute in schema")
	}
	if !forceDestroyAttr.IsOptional() {
		t.Error("expected 'force_destroy' attribute to be optional")
	}
}

func TestDatasetResource_Configure_Success(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	mockClient := &client.MockClient{}

	req := resource.ConfigureRequest{
		ProviderData: mockClient,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestDatasetResource_Configure_NilProviderData(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	// Should not error - nil ProviderData is valid during schema validation
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestDatasetResource_Configure_WrongType(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

// getDatasetResourceSchema returns the schema for the dataset resource
func getDatasetResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewDatasetResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	return *schemaResp
}

// createDatasetResourceModel creates a tftypes.Value for the dataset resource model
func createDatasetResourceModel(id, pool, path, parent, name, mountPath, compression, quota, refquota, atime, forceDestroy interface{}) tftypes.Value {
	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"pool":          tftypes.String,
			"path":          tftypes.String,
			"parent":        tftypes.String,
			"name":          tftypes.String,
			"mount_path":    tftypes.String,
			"compression":   tftypes.String,
			"quota":         tftypes.String,
			"refquota":      tftypes.String,
			"atime":         tftypes.String,
			"force_destroy": tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, id),
		"pool":          tftypes.NewValue(tftypes.String, pool),
		"path":          tftypes.NewValue(tftypes.String, path),
		"parent":        tftypes.NewValue(tftypes.String, parent),
		"name":          tftypes.NewValue(tftypes.String, name),
		"mount_path":    tftypes.NewValue(tftypes.String, mountPath),
		"compression":   tftypes.NewValue(tftypes.String, compression),
		"quota":         tftypes.NewValue(tftypes.String, quota),
		"refquota":      tftypes.NewValue(tftypes.String, refquota),
		"atime":         tftypes.NewValue(tftypes.String, atime),
		"force_destroy": tftypes.NewValue(tftypes.Bool, forceDestroy),
	})
}

func TestDatasetResource_Create_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedParams = params
				return json.RawMessage(`{
					"id": "storage/apps",
					"name": "storage/apps",
					"mountpoint": "/mnt/storage/apps"
				}`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	planValue := createDatasetResourceModel(nil, "storage", "apps", nil, nil, nil, "lz4", nil, nil, nil, nil)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify the API was called correctly
	if capturedMethod != "pool.dataset.create" {
		t.Errorf("expected method 'pool.dataset.create', got %q", capturedMethod)
	}

	// Verify params include the full dataset name
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["name"] != "storage/apps" {
		t.Errorf("expected name 'storage/apps', got %v", params["name"])
	}

	// Verify state was set
	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "storage/apps" {
		t.Errorf("expected ID 'storage/apps', got %q", model.ID.ValueString())
	}
	if model.MountPath.ValueString() != "/mnt/storage/apps" {
		t.Errorf("expected MountPath '/mnt/storage/apps', got %q", model.MountPath.ValueString())
	}
}

func TestDatasetResource_Create_InvalidConfig(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Neither pool/path nor parent/name provided
	planValue := createDatasetResourceModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid config")
	}
}

func TestDatasetResource_Create_WithParentName(t *testing.T) {
	var capturedParams any

	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`{
					"id": "tank/data/apps",
					"name": "tank/data/apps",
					"mountpoint": "/mnt/tank/data/apps"
				}`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Use parent/name mode instead of pool/path
	planValue := createDatasetResourceModel(nil, nil, nil, "tank/data", "apps", nil, nil, nil, nil, nil, nil)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify params include the full dataset name
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["name"] != "tank/data/apps" {
		t.Errorf("expected name 'tank/data/apps', got %v", params["name"])
	}

	// Verify state was set
	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "tank/data/apps" {
		t.Errorf("expected ID 'tank/data/apps', got %q", model.ID.ValueString())
	}
}

func TestDatasetResource_Create_APIError(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("dataset already exists")
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	planValue := createDatasetResourceModel(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestDatasetResource_Read_Success(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method != "pool.dataset.query" {
					t.Errorf("expected method 'pool.dataset.query', got %q", method)
				}
				return json.RawMessage(`[{
					"id": "storage/apps",
					"name": "storage/apps",
					"mountpoint": "/mnt/storage/apps",
					"compression": {"value": "lz4"},
					"quota": {"value": "10G"},
					"refquota": {"value": "5G"},
					"atime": {"value": "on"}
				}]`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	// State has compression, quota, refquota, and atime set (user specified them) - they should sync from API
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", "5G", "2G", "off", nil)

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify state was updated from API
	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "storage/apps" {
		t.Errorf("expected ID 'storage/apps', got %q", model.ID.ValueString())
	}
	if model.MountPath.ValueString() != "/mnt/storage/apps" {
		t.Errorf("expected MountPath '/mnt/storage/apps', got %q", model.MountPath.ValueString())
	}
	// Compression was set in state, so it syncs from API
	if model.Compression.ValueString() != "lz4" {
		t.Errorf("expected Compression 'lz4', got %q", model.Compression.ValueString())
	}
	if model.Quota.ValueString() != "10G" {
		t.Errorf("expected Quota '10G', got %q", model.Quota.ValueString())
	}
	if model.RefQuota.ValueString() != "5G" {
		t.Errorf("expected RefQuota '5G', got %q", model.RefQuota.ValueString())
	}
	// Atime was set in state to "off", API returns "on", so it syncs to "on"
	if model.Atime.ValueString() != "on" {
		t.Errorf("expected Atime 'on', got %q", model.Atime.ValueString())
	}
}

func TestDatasetResource_Read_DatasetNotFound(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// Return empty array - dataset not found
				return json.RawMessage(`[]`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	// Should NOT have errors - just remove from state
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// State should be empty (removed)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed (null) when dataset not found")
	}
}

func TestDatasetResource_Read_APIError(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection failed")
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestDatasetResource_Update_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedParams = params
				return json.RawMessage(`{
					"id": "storage/apps",
					"name": "storage/apps",
					"mountpoint": "/mnt/storage/apps",
					"compression": {"value": "zstd"},
					"quota": {"value": "10G"},
					"refquota": {"value": "5G"},
					"atime": {"value": "off"}
				}`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Current state has lz4 compression
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

	// Plan has zstd compression (changed)
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "zstd", nil, nil, nil, nil)

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify the API was called correctly
	if capturedMethod != "pool.dataset.update" {
		t.Errorf("expected method 'pool.dataset.update', got %q", capturedMethod)
	}

	// Verify params include the ID and update object
	params, ok := capturedParams.([]any)
	if !ok {
		t.Fatalf("expected params to be []any, got %T", capturedParams)
	}

	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}

	if params[0] != "storage/apps" {
		t.Errorf("expected ID 'storage/apps', got %v", params[0])
	}

	updateParams, ok := params[1].(map[string]any)
	if !ok {
		t.Fatalf("expected update params to be map[string]any, got %T", params[1])
	}

	if updateParams["compression"] != "zstd" {
		t.Errorf("expected compression 'zstd', got %v", updateParams["compression"])
	}

	// Verify state was updated from API response
	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.Quota.ValueString() != "10G" {
		t.Errorf("expected Quota '10G', got %q", model.Quota.ValueString())
	}
	if model.RefQuota.ValueString() != "5G" {
		t.Errorf("expected RefQuota '5G', got %q", model.RefQuota.ValueString())
	}
	if model.Atime.ValueString() != "off" {
		t.Errorf("expected Atime 'off', got %q", model.Atime.ValueString())
	}
}

func TestDatasetResource_Update_NoChanges(t *testing.T) {
	apiCalled := false

	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				apiCalled = true
				return nil, nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Same state and plan (no changes)
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// API should NOT be called when there are no changes
	if apiCalled {
		t.Error("expected API not to be called when there are no changes")
	}
}

func TestDatasetResource_Update_APIError(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("update failed")
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "zstd", nil, nil, nil, nil)

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestDatasetResource_Delete_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedParams = params
				return json.RawMessage(`null`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify the API was called correctly
	if capturedMethod != "pool.dataset.delete" {
		t.Errorf("expected method 'pool.dataset.delete', got %q", capturedMethod)
	}

	// Verify the ID was passed
	if capturedParams != "storage/apps" {
		t.Errorf("expected params 'storage/apps', got %v", capturedParams)
	}
}

func TestDatasetResource_Delete_APIError(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("dataset is busy")
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Delete(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestDatasetResource_Delete_WithForceDestroy(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedParams = params
				return json.RawMessage(`null`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	// State with force_destroy = true
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, true)

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify the API was called correctly
	if capturedMethod != "pool.dataset.delete" {
		t.Errorf("expected method 'pool.dataset.delete', got %q", capturedMethod)
	}

	// Verify the params include recursive option
	params, ok := capturedParams.([]any)
	if !ok {
		t.Fatalf("expected params to be []any, got %T", capturedParams)
	}
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	if params[0] != "storage/apps" {
		t.Errorf("expected first param 'storage/apps', got %v", params[0])
	}
	opts, ok := params[1].(map[string]bool)
	if !ok {
		t.Fatalf("expected second param to be map[string]bool, got %T", params[1])
	}
	if !opts["recursive"] {
		t.Error("expected recursive option to be true")
	}
}

func TestDatasetResource_ImportState(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// Initialize state with empty values (null)
	emptyStateValue := createDatasetResourceModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	req := resource.ImportStateRequest{
		ID: "storage/apps",
	}

	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    emptyStateValue,
		},
	}

	r.ImportState(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify the ID was set in state
	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "storage/apps" {
		t.Errorf("expected ID 'storage/apps', got %q", model.ID.ValueString())
	}
}

func TestGetFullName(t *testing.T) {
	tests := []struct {
		name           string
		model          DatasetResourceModel
		expectedResult string
	}{
		{
			name: "pool and path mode",
			model: DatasetResourceModel{
				Pool: stringValue("tank"),
				Path: stringValue("data/apps"),
			},
			expectedResult: "tank/data/apps",
		},
		{
			name: "parent and name mode",
			model: DatasetResourceModel{
				Parent: stringValue("tank/data"),
				Name:   stringValue("apps"),
			},
			expectedResult: "tank/data/apps",
		},
		{
			name: "pool only (invalid)",
			model: DatasetResourceModel{
				Pool: stringValue("tank"),
			},
			expectedResult: "",
		},
		{
			name: "path only (invalid)",
			model: DatasetResourceModel{
				Path: stringValue("data"),
			},
			expectedResult: "",
		},
		{
			name: "parent only (invalid)",
			model: DatasetResourceModel{
				Parent: stringValue("tank"),
			},
			expectedResult: "",
		},
		{
			name: "name only (invalid)",
			model: DatasetResourceModel{
				Name: stringValue("apps"),
			},
			expectedResult: "",
		},
		{
			name:           "all empty (invalid)",
			model:          DatasetResourceModel{},
			expectedResult: "",
		},
		{
			name: "both modes provided (pool/path takes precedence)",
			model: DatasetResourceModel{
				Pool:   stringValue("tank"),
				Path:   stringValue("data"),
				Parent: stringValue("other"),
				Name:   stringValue("name"),
			},
			expectedResult: "tank/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFullName(&tt.model)
			if result != tt.expectedResult {
				t.Errorf("expected %q, got %q", tt.expectedResult, result)
			}
		})
	}
}

// stringValue is a helper to create a types.String with a value
func stringValue(s string) types.String {
	return types.StringValue(s)
}

// Test interface compliance
func TestDatasetResource_ImplementsInterfaces(t *testing.T) {
	r := NewDatasetResource()

	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*DatasetResource)
	var _ resource.ResourceWithImportState = r.(*DatasetResource)
}

// Additional test for Create with all optional parameters
func TestDatasetResource_Create_AllOptions(t *testing.T) {
	var capturedParams any

	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`{
					"id": "storage/apps",
					"name": "storage/apps",
					"mountpoint": "/mnt/storage/apps"
				}`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	planValue := createDatasetResourceModel(nil, "storage", "apps", nil, nil, nil, "zstd", "10G", "5G", "on", nil)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify params include all options
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["compression"] != "zstd" {
		t.Errorf("expected compression 'zstd', got %v", params["compression"])
	}

	if params["quota"] != "10G" {
		t.Errorf("expected quota '10G', got %v", params["quota"])
	}

	if params["refquota"] != "5G" {
		t.Errorf("expected refquota '5G', got %v", params["refquota"])
	}

	if params["atime"] != "on" {
		t.Errorf("expected atime 'on', got %v", params["atime"])
	}
}

// Test Read with parent/name mode
func TestDatasetResource_Read_WithParentName(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": "tank/data/apps",
					"name": "tank/data/apps",
					"mountpoint": "/mnt/tank/data/apps",
					"compression": {"value": "lz4"},
					"quota": {"value": "0"},
					"refquota": {"value": "0"},
					"atime": {"value": "on"}
				}]`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("tank/data/apps", nil, nil, "tank/data", "apps", "/mnt/tank/data/apps", "lz4", nil, nil, nil, nil)

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "tank/data/apps" {
		t.Errorf("expected ID 'tank/data/apps', got %q", model.ID.ValueString())
	}
}

// Test Create with invalid JSON response
func TestDatasetResource_Create_InvalidJSONResponse(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	planValue := createDatasetResourceModel(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid JSON response")
	}
}

// Test Read with invalid JSON response
func TestDatasetResource_Read_InvalidJSONResponse(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid JSON response")
	}
}

// Test Update with invalid JSON response
func TestDatasetResource_Update_InvalidJSONResponse(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "zstd", nil, nil, nil, nil)

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid JSON response")
	}
}

// Test Update with quota change
func TestDatasetResource_Update_QuotaChange(t *testing.T) {
	var capturedParams any

	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`{
					"id": "storage/apps",
					"name": "storage/apps",
					"mountpoint": "/mnt/storage/apps",
					"compression": {"value": "lz4"},
					"quota": {"value": "10G"},
					"refquota": {"value": "0"},
					"atime": {"value": "on"}
				}`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Current state has no quota
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

	// Plan adds quota
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", "10G", nil, nil, nil)

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify params include the quota
	params, ok := capturedParams.([]any)
	if !ok {
		t.Fatalf("expected params to be []any, got %T", capturedParams)
	}

	updateParams, ok := params[1].(map[string]any)
	if !ok {
		t.Fatalf("expected update params to be map[string]any, got %T", params[1])
	}

	if updateParams["quota"] != "10G" {
		t.Errorf("expected quota '10G', got %v", updateParams["quota"])
	}
}

// Test Update with refquota change
func TestDatasetResource_Update_RefQuotaChange(t *testing.T) {
	var capturedParams any

	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`{
					"id": "storage/apps",
					"name": "storage/apps",
					"mountpoint": "/mnt/storage/apps",
					"compression": {"value": "lz4"},
					"quota": {"value": "0"},
					"refquota": {"value": "5G"},
					"atime": {"value": "on"}
				}`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, "5G", nil, nil)

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	params, ok := capturedParams.([]any)
	if !ok {
		t.Fatalf("expected params to be []any, got %T", capturedParams)
	}

	updateParams, ok := params[1].(map[string]any)
	if !ok {
		t.Fatalf("expected update params to be map[string]any, got %T", params[1])
	}

	if updateParams["refquota"] != "5G" {
		t.Errorf("expected refquota '5G', got %v", updateParams["refquota"])
	}
}

// Test Update with atime change
func TestDatasetResource_Update_AtimeChange(t *testing.T) {
	var capturedParams any

	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`{
					"id": "storage/apps",
					"name": "storage/apps",
					"mountpoint": "/mnt/storage/apps",
					"compression": {"value": "lz4"},
					"quota": {"value": "0"},
					"refquota": {"value": "0"},
					"atime": {"value": "off"}
				}`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, "off", nil)

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	params, ok := capturedParams.([]any)
	if !ok {
		t.Fatalf("expected params to be []any, got %T", capturedParams)
	}

	updateParams, ok := params[1].(map[string]any)
	if !ok {
		t.Fatalf("expected update params to be map[string]any, got %T", params[1])
	}

	if updateParams["atime"] != "off" {
		t.Errorf("expected atime 'off', got %v", updateParams["atime"])
	}
}

// Test Create with plan parsing error
func TestDatasetResource_Create_PlanParseError(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Create an invalid plan value with wrong type
	planValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":          tftypes.String,
			"pool":        tftypes.Number, // Wrong type!
			"path":        tftypes.String,
			"parent":      tftypes.String,
			"name":        tftypes.String,
			"mount_path":  tftypes.String,
			"compression": tftypes.String,
			"quota":       tftypes.String,
			"refquota":    tftypes.String,
			"atime":       tftypes.String,
		},
	}, map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, nil),
		"pool":        tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"path":        tftypes.NewValue(tftypes.String, "apps"),
		"parent":      tftypes.NewValue(tftypes.String, nil),
		"name":        tftypes.NewValue(tftypes.String, nil),
		"mount_path":  tftypes.NewValue(tftypes.String, nil),
		"compression": tftypes.NewValue(tftypes.String, nil),
		"quota":       tftypes.NewValue(tftypes.String, nil),
		"refquota":    tftypes.NewValue(tftypes.String, nil),
		"atime":       tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for plan parse error")
	}
}

// Test Read with state parsing error
func TestDatasetResource_Read_StateParseError(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Create an invalid state value with wrong type
	stateValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":          tftypes.Number, // Wrong type!
			"pool":        tftypes.String,
			"path":        tftypes.String,
			"parent":      tftypes.String,
			"name":        tftypes.String,
			"mount_path":  tftypes.String,
			"compression": tftypes.String,
			"quota":       tftypes.String,
			"refquota":    tftypes.String,
			"atime":       tftypes.String,
		},
	}, map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"pool":        tftypes.NewValue(tftypes.String, "storage"),
		"path":        tftypes.NewValue(tftypes.String, "apps"),
		"parent":      tftypes.NewValue(tftypes.String, nil),
		"name":        tftypes.NewValue(tftypes.String, nil),
		"mount_path":  tftypes.NewValue(tftypes.String, "/mnt/storage/apps"),
		"compression": tftypes.NewValue(tftypes.String, "lz4"),
		"quota":       tftypes.NewValue(tftypes.String, nil),
		"refquota":    tftypes.NewValue(tftypes.String, nil),
		"atime":       tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for state parse error")
	}
}

// Test Update with plan parsing error
func TestDatasetResource_Update_PlanParseError(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Valid state
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

	// Invalid plan with wrong type
	planValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"pool":          tftypes.Number, // Wrong type!
			"path":          tftypes.String,
			"parent":        tftypes.String,
			"name":          tftypes.String,
			"mount_path":    tftypes.String,
			"compression":   tftypes.String,
			"quota":         tftypes.String,
			"refquota":      tftypes.String,
			"atime":         tftypes.String,
			"force_destroy": tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, "storage/apps"),
		"pool":          tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"path":          tftypes.NewValue(tftypes.String, "apps"),
		"parent":        tftypes.NewValue(tftypes.String, nil),
		"name":          tftypes.NewValue(tftypes.String, nil),
		"mount_path":    tftypes.NewValue(tftypes.String, "/mnt/storage/apps"),
		"compression":   tftypes.NewValue(tftypes.String, "zstd"),
		"quota":         tftypes.NewValue(tftypes.String, nil),
		"refquota":      tftypes.NewValue(tftypes.String, nil),
		"atime":         tftypes.NewValue(tftypes.String, nil),
		"force_destroy": tftypes.NewValue(tftypes.Bool, nil),
	})

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for plan parse error")
	}
}

// Test Update with state parsing error
func TestDatasetResource_Update_StateParseError(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Invalid state with wrong type
	stateValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.Number, // Wrong type!
			"pool":          tftypes.String,
			"path":          tftypes.String,
			"parent":        tftypes.String,
			"name":          tftypes.String,
			"mount_path":    tftypes.String,
			"compression":   tftypes.String,
			"quota":         tftypes.String,
			"refquota":      tftypes.String,
			"atime":         tftypes.String,
			"force_destroy": tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"pool":          tftypes.NewValue(tftypes.String, "storage"),
		"path":          tftypes.NewValue(tftypes.String, "apps"),
		"parent":        tftypes.NewValue(tftypes.String, nil),
		"name":          tftypes.NewValue(tftypes.String, nil),
		"mount_path":    tftypes.NewValue(tftypes.String, "/mnt/storage/apps"),
		"compression":   tftypes.NewValue(tftypes.String, "lz4"),
		"quota":         tftypes.NewValue(tftypes.String, nil),
		"refquota":      tftypes.NewValue(tftypes.String, nil),
		"atime":         tftypes.NewValue(tftypes.String, nil),
		"force_destroy": tftypes.NewValue(tftypes.Bool, nil),
	})

	// Valid plan
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "zstd", nil, nil, nil, nil)

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for state parse error")
	}
}

// Test that compression attribute is Optional+Computed with UseStateForUnknown
func TestDatasetResource_Schema_CompressionIsComputed(t *testing.T) {
	r := NewDatasetResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify compression attribute is computed (required for UseStateForUnknown)
	compressionAttr, ok := resp.Schema.Attributes["compression"]
	if !ok {
		t.Fatal("expected 'compression' attribute in schema")
	}
	if !compressionAttr.IsComputed() {
		t.Error("expected 'compression' attribute to be computed")
	}
}

// Test that atime attribute is Optional+Computed with UseStateForUnknown
func TestDatasetResource_Schema_AtimeIsComputed(t *testing.T) {
	r := NewDatasetResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify atime attribute is computed (required for UseStateForUnknown)
	atimeAttr, ok := resp.Schema.Attributes["atime"]
	if !ok {
		t.Fatal("expected 'atime' attribute in schema")
	}
	if !atimeAttr.IsComputed() {
		t.Error("expected 'atime' attribute to be computed")
	}
}

// Test that Read preserves null compression when not set in config
func TestDatasetResource_Read_PreservesNullCompression(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// API returns default values even when user didn't set them
				return json.RawMessage(`[{
					"id": "storage/apps",
					"name": "storage/apps",
					"mountpoint": "/mnt/storage/apps",
					"compression": {"value": "LZ4"},
					"quota": {"value": "0"},
					"refquota": {"value": "0"},
					"atime": {"value": "OFF"}
				}]`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	// State has null compression (user never specified it)
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", nil, nil, nil, nil, nil)

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// Compression should remain null since user never specified it
	// (This is the drift-prevention behavior we want)
	if !model.Compression.IsNull() {
		t.Errorf("expected compression to remain null when not specified in config, got %q", model.Compression.ValueString())
	}
}

// Test that Read preserves null atime when not set in config
func TestDatasetResource_Read_PreservesNullAtime(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": "storage/apps",
					"name": "storage/apps",
					"mountpoint": "/mnt/storage/apps",
					"compression": {"value": "LZ4"},
					"quota": {"value": "0"},
					"refquota": {"value": "0"},
					"atime": {"value": "OFF"}
				}]`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)

	// State has null atime (user never specified it)
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", nil, nil, nil, nil, nil)

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// Atime should remain null since user never specified it
	if !model.Atime.IsNull() {
		t.Errorf("expected atime to remain null when not specified in config, got %q", model.Atime.ValueString())
	}
}

// Test Delete with state parsing error
func TestDatasetResource_Delete_StateParseError(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Invalid state with wrong type
	stateValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.Number, // Wrong type!
			"pool":          tftypes.String,
			"path":          tftypes.String,
			"parent":        tftypes.String,
			"name":          tftypes.String,
			"mount_path":    tftypes.String,
			"compression":   tftypes.String,
			"quota":         tftypes.String,
			"refquota":      tftypes.String,
			"atime":         tftypes.String,
			"force_destroy": tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"pool":          tftypes.NewValue(tftypes.String, "storage"),
		"path":          tftypes.NewValue(tftypes.String, "apps"),
		"parent":        tftypes.NewValue(tftypes.String, nil),
		"name":          tftypes.NewValue(tftypes.String, nil),
		"mount_path":    tftypes.NewValue(tftypes.String, "/mnt/storage/apps"),
		"compression":   tftypes.NewValue(tftypes.String, "lz4"),
		"quota":         tftypes.NewValue(tftypes.String, nil),
		"refquota":      tftypes.NewValue(tftypes.String, nil),
		"atime":         tftypes.NewValue(tftypes.String, nil),
		"force_destroy": tftypes.NewValue(tftypes.Bool, nil),
	})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Delete(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for state parse error")
	}
}
