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

func TestNewPoolDataSource(t *testing.T) {
	ds := NewPoolDataSource()
	if ds == nil {
		t.Fatal("expected non-nil data source")
	}

	// Verify it implements the required interfaces
	_ = datasource.DataSource(ds)
	_ = datasource.DataSourceWithConfigure(ds.(*PoolDataSource))
}

func TestPoolDataSource_Metadata(t *testing.T) {
	ds := NewPoolDataSource()

	req := datasource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_pool" {
		t.Errorf("expected TypeName 'truenas_pool', got %q", resp.TypeName)
	}
}

func TestPoolDataSource_Schema(t *testing.T) {
	ds := NewPoolDataSource()

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), req, resp)

	// Verify schema has description
	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify name attribute exists and is required
	nameAttr, ok := resp.Schema.Attributes["name"]
	if !ok {
		t.Fatal("expected 'name' attribute in schema")
	}
	if !nameAttr.IsRequired() {
		t.Error("expected 'name' attribute to be required")
	}

	// Verify id attribute exists and is computed
	idAttr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal("expected 'id' attribute in schema")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' attribute to be computed")
	}

	// Verify path attribute exists and is computed
	pathAttr, ok := resp.Schema.Attributes["path"]
	if !ok {
		t.Fatal("expected 'path' attribute in schema")
	}
	if !pathAttr.IsComputed() {
		t.Error("expected 'path' attribute to be computed")
	}

	// Verify status attribute exists and is computed
	statusAttr, ok := resp.Schema.Attributes["status"]
	if !ok {
		t.Fatal("expected 'status' attribute in schema")
	}
	if !statusAttr.IsComputed() {
		t.Error("expected 'status' attribute to be computed")
	}

	// Verify available_bytes attribute exists and is computed
	availableAttr, ok := resp.Schema.Attributes["available_bytes"]
	if !ok {
		t.Fatal("expected 'available_bytes' attribute in schema")
	}
	if !availableAttr.IsComputed() {
		t.Error("expected 'available_bytes' attribute to be computed")
	}

	// Verify used_bytes attribute exists and is computed
	usedAttr, ok := resp.Schema.Attributes["used_bytes"]
	if !ok {
		t.Fatal("expected 'used_bytes' attribute in schema")
	}
	if !usedAttr.IsComputed() {
		t.Error("expected 'used_bytes' attribute to be computed")
	}
}

func TestPoolDataSource_Configure_Success(t *testing.T) {
	ds := NewPoolDataSource().(*PoolDataSource)

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

func TestPoolDataSource_Configure_NilProviderData(t *testing.T) {
	ds := NewPoolDataSource().(*PoolDataSource)

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

func TestPoolDataSource_Configure_WrongType(t *testing.T) {
	ds := NewPoolDataSource().(*PoolDataSource)

	req := datasource.ConfigureRequest{
		ProviderData: "not a services",
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

// createTestReadRequest creates a datasource.ReadRequest with the given name
func createTestReadRequest(t *testing.T, name string) datasource.ReadRequest {
	t.Helper()

	// Get the schema
	ds := NewPoolDataSource()
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	// Build config value
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":              tftypes.String,
			"name":            tftypes.String,
			"path":            tftypes.String,
			"status":          tftypes.String,
			"available_bytes": tftypes.Number,
			"used_bytes":      tftypes.Number,
		},
	}, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"name":            tftypes.NewValue(tftypes.String, name),
		"path":            tftypes.NewValue(tftypes.String, nil),
		"status":          tftypes.NewValue(tftypes.String, nil),
		"available_bytes": tftypes.NewValue(tftypes.Number, nil),
		"used_bytes":      tftypes.NewValue(tftypes.Number, nil),
	})

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    configValue,
	}

	return datasource.ReadRequest{
		Config: config,
	}
}

func TestPoolDataSource_Read_Success(t *testing.T) {
	ds := &PoolDataSource{
		services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				ListPoolsFunc: func(ctx context.Context) ([]truenas.Pool, error) {
					return []truenas.Pool{
						{
							ID:        1,
							Name:      "tank",
							Path:      "/mnt/tank",
							Status:    "ONLINE",
							Size:      1000000000,
							Allocated: 400000000,
							Free:      600000000,
						},
					}, nil
				},
			},
		},
	}

	req := createTestReadRequest(t, "tank")

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
	var model PoolDataSourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "1" {
		t.Errorf("expected ID '1', got %q", model.ID.ValueString())
	}
	if model.Name.ValueString() != "tank" {
		t.Errorf("expected Name 'tank', got %q", model.Name.ValueString())
	}
	if model.Path.ValueString() != "/mnt/tank" {
		t.Errorf("expected Path '/mnt/tank', got %q", model.Path.ValueString())
	}
	if model.Status.ValueString() != "ONLINE" {
		t.Errorf("expected Status 'ONLINE', got %q", model.Status.ValueString())
	}
	if model.AvailableBytes.ValueInt64() != 600000000 {
		t.Errorf("expected AvailableBytes 600000000, got %d", model.AvailableBytes.ValueInt64())
	}
	if model.UsedBytes.ValueInt64() != 400000000 {
		t.Errorf("expected UsedBytes 400000000, got %d", model.UsedBytes.ValueInt64())
	}
}

func TestPoolDataSource_Read_PoolNotFound(t *testing.T) {
	ds := &PoolDataSource{
		services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				ListPoolsFunc: func(ctx context.Context) ([]truenas.Pool, error) {
					return []truenas.Pool{}, nil
				},
			},
		},
	}

	req := createTestReadRequest(t, "nonexistent")

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
		t.Fatal("expected error for pool not found")
	}
}

func TestPoolDataSource_Read_APIError(t *testing.T) {
	ds := &PoolDataSource{
		services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				ListPoolsFunc: func(ctx context.Context) ([]truenas.Pool, error) {
					return nil, errors.New("connection failed")
				},
			},
		},
	}

	req := createTestReadRequest(t, "tank")

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

func TestPoolDataSource_Read_ConfigError(t *testing.T) {
	ds := &PoolDataSource{
		services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{},
		},
	}

	// Get the schema
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	// Create an invalid config value with wrong type for name
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":              tftypes.String,
			"name":            tftypes.Number, // Wrong type!
			"path":            tftypes.String,
			"status":          tftypes.String,
			"available_bytes": tftypes.Number,
			"used_bytes":      tftypes.Number,
		},
	}, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"name":            tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"path":            tftypes.NewValue(tftypes.String, nil),
		"status":          tftypes.NewValue(tftypes.String, nil),
		"available_bytes": tftypes.NewValue(tftypes.Number, nil),
		"used_bytes":      tftypes.NewValue(tftypes.Number, nil),
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

func TestPoolDataSource_Read_MultiplePoolsFindsMatch(t *testing.T) {
	ds := &PoolDataSource{
		services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				ListPoolsFunc: func(ctx context.Context) ([]truenas.Pool, error) {
					return []truenas.Pool{
						{ID: 1, Name: "boot-pool", Path: "/mnt/boot-pool", Status: "ONLINE", Allocated: 100, Free: 900},
						{ID: 2, Name: "mypool", Path: "/mnt/mypool", Status: "ONLINE", Allocated: 400000000, Free: 600000000},
						{ID: 3, Name: "archive", Path: "/mnt/archive", Status: "DEGRADED", Allocated: 200, Free: 800},
					}, nil
				},
			},
		},
	}

	req := createTestReadRequest(t, "mypool")

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

	var model PoolDataSourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "2" {
		t.Errorf("expected ID '2', got %q", model.ID.ValueString())
	}
	if model.Name.ValueString() != "mypool" {
		t.Errorf("expected Name 'mypool', got %q", model.Name.ValueString())
	}
}

// Test that PoolDataSource implements the DataSource interface
func TestPoolDataSource_ImplementsInterfaces(t *testing.T) {
	ds := NewPoolDataSource()

	_ = datasource.DataSource(ds)
	_ = datasource.DataSourceWithConfigure(ds.(*PoolDataSource))
}
