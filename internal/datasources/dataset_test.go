package datasources

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewDatasetDataSource(t *testing.T) {
	ds := NewDatasetDataSource()
	if ds == nil {
		t.Fatal("expected non-nil data source")
	}

	// Verify it implements the required interfaces
	var _ datasource.DataSource = ds
	var _ datasource.DataSourceWithConfigure = ds.(*DatasetDataSource)
}

func TestDatasetDataSource_Metadata(t *testing.T) {
	ds := NewDatasetDataSource()

	req := datasource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_dataset" {
		t.Errorf("expected TypeName 'truenas_dataset', got %q", resp.TypeName)
	}
}

func TestDatasetDataSource_Schema(t *testing.T) {
	ds := NewDatasetDataSource()

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), req, resp)

	// Verify schema has description
	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify pool attribute exists and is required
	poolAttr, ok := resp.Schema.Attributes["pool"]
	if !ok {
		t.Fatal("expected 'pool' attribute in schema")
	}
	if !poolAttr.IsRequired() {
		t.Error("expected 'pool' attribute to be required")
	}

	// Verify path attribute exists and is required
	pathAttr, ok := resp.Schema.Attributes["path"]
	if !ok {
		t.Fatal("expected 'path' attribute in schema")
	}
	if !pathAttr.IsRequired() {
		t.Error("expected 'path' attribute to be required")
	}

	// Verify id attribute exists and is computed
	idAttr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal("expected 'id' attribute in schema")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' attribute to be computed")
	}

	// Verify mount_path attribute exists and is computed
	mountPathAttr, ok := resp.Schema.Attributes["mount_path"]
	if !ok {
		t.Fatal("expected 'mount_path' attribute in schema")
	}
	if !mountPathAttr.IsComputed() {
		t.Error("expected 'mount_path' attribute to be computed")
	}

	// Verify compression attribute exists and is computed
	compressionAttr, ok := resp.Schema.Attributes["compression"]
	if !ok {
		t.Fatal("expected 'compression' attribute in schema")
	}
	if !compressionAttr.IsComputed() {
		t.Error("expected 'compression' attribute to be computed")
	}

	// Verify used_bytes attribute exists and is computed
	usedAttr, ok := resp.Schema.Attributes["used_bytes"]
	if !ok {
		t.Fatal("expected 'used_bytes' attribute in schema")
	}
	if !usedAttr.IsComputed() {
		t.Error("expected 'used_bytes' attribute to be computed")
	}

	// Verify available_bytes attribute exists and is computed
	availableAttr, ok := resp.Schema.Attributes["available_bytes"]
	if !ok {
		t.Fatal("expected 'available_bytes' attribute in schema")
	}
	if !availableAttr.IsComputed() {
		t.Error("expected 'available_bytes' attribute to be computed")
	}
}

func TestDatasetDataSource_Configure_Success(t *testing.T) {
	ds := NewDatasetDataSource().(*DatasetDataSource)

	mockClient := &client.MockClient{}

	req := datasource.ConfigureRequest{
		ProviderData: mockClient,
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestDatasetDataSource_Configure_NilProviderData(t *testing.T) {
	ds := NewDatasetDataSource().(*DatasetDataSource)

	req := datasource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	// Should not error - nil ProviderData is valid during schema validation
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestDatasetDataSource_Configure_WrongType(t *testing.T) {
	ds := NewDatasetDataSource().(*DatasetDataSource)

	req := datasource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

// createDatasetTestReadRequest creates a datasource.ReadRequest with the given pool and path
func createDatasetTestReadRequest(t *testing.T, pool, path string) datasource.ReadRequest {
	t.Helper()

	// Get the schema
	ds := NewDatasetDataSource()
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	// Build config value
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":              tftypes.String,
			"pool":            tftypes.String,
			"path":            tftypes.String,
			"mount_path":      tftypes.String,
			"compression":     tftypes.String,
			"used_bytes":      tftypes.Number,
			"available_bytes": tftypes.Number,
		},
	}, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"pool":            tftypes.NewValue(tftypes.String, pool),
		"path":            tftypes.NewValue(tftypes.String, path),
		"mount_path":      tftypes.NewValue(tftypes.String, nil),
		"compression":     tftypes.NewValue(tftypes.String, nil),
		"used_bytes":      tftypes.NewValue(tftypes.Number, nil),
		"available_bytes": tftypes.NewValue(tftypes.Number, nil),
	})

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    configValue,
	}

	return datasource.ReadRequest{
		Config: config,
	}
}

func TestDatasetDataSource_Read_Success(t *testing.T) {
	ds := &DatasetDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method != "pool.dataset.query" {
					t.Errorf("expected method 'pool.dataset.query', got %q", method)
				}
				// Return a dataset response
				return json.RawMessage(`[{
					"id": "storage/apps",
					"name": "storage/apps",
					"pool": "storage",
					"mountpoint": "/mnt/storage/apps",
					"compression": {"value": "lz4"},
					"used": {"parsed": 1000000},
					"available": {"parsed": 9000000}
				}]`), nil
			},
		},
	}

	req := createDatasetTestReadRequest(t, "storage", "apps")

	// Get the schema for the state
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify the state was set correctly
	var model DatasetDataSourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "storage/apps" {
		t.Errorf("expected ID 'storage/apps', got %q", model.ID.ValueString())
	}
	if model.Pool.ValueString() != "storage" {
		t.Errorf("expected Pool 'storage', got %q", model.Pool.ValueString())
	}
	if model.Path.ValueString() != "apps" {
		t.Errorf("expected Path 'apps', got %q", model.Path.ValueString())
	}
	if model.MountPath.ValueString() != "/mnt/storage/apps" {
		t.Errorf("expected MountPath '/mnt/storage/apps', got %q", model.MountPath.ValueString())
	}
	if model.Compression.ValueString() != "lz4" {
		t.Errorf("expected Compression 'lz4', got %q", model.Compression.ValueString())
	}
	if model.UsedBytes.ValueInt64() != 1000000 {
		t.Errorf("expected UsedBytes 1000000, got %d", model.UsedBytes.ValueInt64())
	}
	if model.AvailableBytes.ValueInt64() != 9000000 {
		t.Errorf("expected AvailableBytes 9000000, got %d", model.AvailableBytes.ValueInt64())
	}
}

func TestDatasetDataSource_Read_DatasetNotFound(t *testing.T) {
	ds := &DatasetDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// Return empty array - dataset not found
				return json.RawMessage(`[]`), nil
			},
		},
	}

	req := createDatasetTestReadRequest(t, "storage", "nonexistent")

	// Get the schema for the state
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for dataset not found")
	}
}

func TestDatasetDataSource_Read_APIError(t *testing.T) {
	ds := &DatasetDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection failed")
			},
		},
	}

	req := createDatasetTestReadRequest(t, "storage", "apps")

	// Get the schema for the state
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestDatasetDataSource_Read_InvalidJSON(t *testing.T) {
	ds := &DatasetDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		},
	}

	req := createDatasetTestReadRequest(t, "storage", "apps")

	// Get the schema for the state
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestDatasetDataSource_Read_ConfigError(t *testing.T) {
	ds := &DatasetDataSource{
		client: &client.MockClient{},
	}

	// Get the schema
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	// Create an invalid config value with wrong type for pool
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":              tftypes.String,
			"pool":            tftypes.Number, // Wrong type!
			"path":            tftypes.String,
			"mount_path":      tftypes.String,
			"compression":     tftypes.String,
			"used_bytes":      tftypes.Number,
			"available_bytes": tftypes.Number,
		},
	}, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"pool":            tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"path":            tftypes.NewValue(tftypes.String, "apps"),
		"mount_path":      tftypes.NewValue(tftypes.String, nil),
		"compression":     tftypes.NewValue(tftypes.String, nil),
		"used_bytes":      tftypes.NewValue(tftypes.Number, nil),
		"available_bytes": tftypes.NewValue(tftypes.Number, nil),
	})

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    configValue,
	}

	req := datasource.ReadRequest{
		Config: config,
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for config parse error")
	}
}

func TestDatasetDataSource_Read_VerifyFilterParams(t *testing.T) {
	var capturedParams any

	ds := &DatasetDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`[{
					"id": "storage/apps",
					"name": "storage/apps",
					"pool": "storage",
					"mountpoint": "/mnt/storage/apps",
					"compression": {"value": "lz4"},
					"used": {"parsed": 1000000},
					"available": {"parsed": 9000000}
				}]`), nil
			},
		},
	}

	req := createDatasetTestReadRequest(t, "storage", "apps")

	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify the filter params were passed correctly
	// Expected format: [["id", "=", "storage/apps"]]
	filters, ok := capturedParams.([][]any)
	if !ok {
		t.Fatalf("expected params to be [][]any, got %T", capturedParams)
	}

	if len(filters) != 1 {
		t.Fatalf("expected 1 filter, got %d", len(filters))
	}

	filter := filters[0]
	if len(filter) != 3 {
		t.Fatalf("expected 3 filter parts, got %d", len(filter))
	}

	if filter[0] != "id" || filter[1] != "=" || filter[2] != "storage/apps" {
		t.Errorf("expected filter ['id', '=', 'storage/apps'], got %v", filter)
	}
}

func TestDatasetDataSource_Read_NestedPath(t *testing.T) {
	ds := &DatasetDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// Return a dataset with nested path
				return json.RawMessage(`[{
					"id": "tank/data/apps/myapp",
					"name": "tank/data/apps/myapp",
					"pool": "tank",
					"mountpoint": "/mnt/tank/data/apps/myapp",
					"compression": {"value": "zstd"},
					"used": {"parsed": 5000000},
					"available": {"parsed": 50000000}
				}]`), nil
			},
		},
	}

	req := createDatasetTestReadRequest(t, "tank", "data/apps/myapp")

	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model DatasetDataSourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "tank/data/apps/myapp" {
		t.Errorf("expected ID 'tank/data/apps/myapp', got %q", model.ID.ValueString())
	}
	if model.MountPath.ValueString() != "/mnt/tank/data/apps/myapp" {
		t.Errorf("expected MountPath '/mnt/tank/data/apps/myapp', got %q", model.MountPath.ValueString())
	}
	if model.Compression.ValueString() != "zstd" {
		t.Errorf("expected Compression 'zstd', got %q", model.Compression.ValueString())
	}
}

// Test that DatasetDataSource implements the DataSource interface
func TestDatasetDataSource_ImplementsInterfaces(t *testing.T) {
	ds := NewDatasetDataSource()

	var _ datasource.DataSource = ds
	var _ datasource.DataSourceWithConfigure = ds.(*DatasetDataSource)
}
