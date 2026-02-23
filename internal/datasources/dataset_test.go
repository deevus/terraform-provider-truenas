package datasources

import (
	"context"
	"errors"
	"testing"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/terraform-provider-truenas/internal/services"
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
	_ = datasource.DataSource(ds)
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

	svc := &services.TrueNASServices{}

	req := datasource.ConfigureRequest{
		ProviderData: svc,
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
		ProviderData: "not a services",
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
		services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					if id != "storage/apps" {
						t.Errorf("expected id 'storage/apps', got %q", id)
					}
					return &truenas.Dataset{
						ID:          "storage/apps",
						Name:        "storage/apps",
						Pool:        "storage",
						Mountpoint:  "/mnt/storage/apps",
						Compression: "lz4",
						Used:        1000000,
						Available:   9000000,
					}, nil
				},
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
		services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					return nil, nil
				},
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
		services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					return nil, errors.New("connection failed")
				},
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

func TestDatasetDataSource_Read_ConfigError(t *testing.T) {
	ds := &DatasetDataSource{
		services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{},
		},
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

func TestDatasetDataSource_Read_NestedPath(t *testing.T) {
	ds := &DatasetDataSource{
		services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					if id != "tank/data/apps/myapp" {
						t.Errorf("expected id 'tank/data/apps/myapp', got %q", id)
					}
					return &truenas.Dataset{
						ID:          "tank/data/apps/myapp",
						Name:        "tank/data/apps/myapp",
						Pool:        "tank",
						Mountpoint:  "/mnt/tank/data/apps/myapp",
						Compression: "zstd",
						Used:        5000000,
						Available:   50000000,
					}, nil
				},
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

	_ = datasource.DataSource(ds)
	_ = datasource.DataSourceWithConfigure(ds.(*DatasetDataSource))
}
