package datasources

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/deevus/truenas-go/client"
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
		ProviderData: "not a client",
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
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method != "pool.query" {
					t.Errorf("expected method 'pool.query', got %q", method)
				}
				// Return a pool response
				return json.RawMessage(`[{
					"id": 1,
					"name": "tank",
					"path": "/mnt/tank",
					"status": "ONLINE",
					"size": 1000000000,
					"allocated": 400000000,
					"free": 600000000
				}]`), nil
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
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// Return empty array - pool not found
				return json.RawMessage(`[]`), nil
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
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection failed")
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

func TestPoolDataSource_Read_InvalidJSON(t *testing.T) {
	ds := &PoolDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
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
		t.Fatal("expected error for invalid JSON")
	}
}

func TestPoolDataSource_Read_ConfigError(t *testing.T) {
	ds := &PoolDataSource{
		client: &client.MockClient{},
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

func TestPoolDataSource_Read_VerifyFilterParams(t *testing.T) {
	var capturedParams any

	ds := &PoolDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`[{
					"id": 1,
					"name": "mypool",
					"path": "/mnt/mypool",
					"status": "ONLINE",
					"size": 1000000000,
					"allocated": 400000000,
					"free": 600000000
				}]`), nil
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

	// Verify the filter params were passed correctly
	// Expected format: [["name", "=", "mypool"]]
	filters, ok := capturedParams.([][]string)
	if !ok {
		t.Fatalf("expected params to be [][]string, got %T", capturedParams)
	}

	if len(filters) != 1 {
		t.Fatalf("expected 1 filter, got %d", len(filters))
	}

	filter := filters[0]
	if len(filter) != 3 {
		t.Fatalf("expected 3 filter parts, got %d", len(filter))
	}

	if filter[0] != "name" || filter[1] != "=" || filter[2] != "mypool" {
		t.Errorf("expected filter ['name', '=', 'mypool'], got %v", filter)
	}
}

// Test that PoolDataSource implements the DataSource interface
func TestPoolDataSource_ImplementsInterfaces(t *testing.T) {
	ds := NewPoolDataSource()

	_ = datasource.DataSource(ds)
	_ = datasource.DataSourceWithConfigure(ds.(*PoolDataSource))
}
