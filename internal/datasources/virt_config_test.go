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

func TestNewVirtConfigDataSource(t *testing.T) {
	ds := NewVirtConfigDataSource()
	if ds == nil {
		t.Fatal("expected non-nil data source")
	}

	// Verify it implements the required interfaces
	_ = datasource.DataSource(ds)
	_ = datasource.DataSourceWithConfigure(ds.(*VirtConfigDataSource))
}

func TestVirtConfigDataSource_Metadata(t *testing.T) {
	ds := NewVirtConfigDataSource()

	req := datasource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_virt_config" {
		t.Errorf("expected TypeName 'truenas_virt_config', got %q", resp.TypeName)
	}
}

func TestVirtConfigDataSource_Schema(t *testing.T) {
	ds := NewVirtConfigDataSource()

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), req, resp)

	// Verify schema has description
	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify bridge attribute exists and is computed
	bridgeAttr, ok := resp.Schema.Attributes["bridge"]
	if !ok {
		t.Fatal("expected 'bridge' attribute in schema")
	}
	if !bridgeAttr.IsComputed() {
		t.Error("expected 'bridge' attribute to be computed")
	}

	// Verify v4_network attribute exists and is computed
	v4NetworkAttr, ok := resp.Schema.Attributes["v4_network"]
	if !ok {
		t.Fatal("expected 'v4_network' attribute in schema")
	}
	if !v4NetworkAttr.IsComputed() {
		t.Error("expected 'v4_network' attribute to be computed")
	}

	// Verify v6_network attribute exists and is computed
	v6NetworkAttr, ok := resp.Schema.Attributes["v6_network"]
	if !ok {
		t.Fatal("expected 'v6_network' attribute in schema")
	}
	if !v6NetworkAttr.IsComputed() {
		t.Error("expected 'v6_network' attribute to be computed")
	}

	// Verify pool attribute exists and is computed
	preferredPoolAttr, ok := resp.Schema.Attributes["pool"]
	if !ok {
		t.Fatal("expected 'pool' attribute in schema")
	}
	if !preferredPoolAttr.IsComputed() {
		t.Error("expected 'pool' attribute to be computed")
	}
}

func TestVirtConfigDataSource_Configure_Success(t *testing.T) {
	ds := NewVirtConfigDataSource().(*VirtConfigDataSource)

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

func TestVirtConfigDataSource_Configure_NilProviderData(t *testing.T) {
	ds := NewVirtConfigDataSource().(*VirtConfigDataSource)

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

func TestVirtConfigDataSource_Configure_WrongType(t *testing.T) {
	ds := NewVirtConfigDataSource().(*VirtConfigDataSource)

	req := datasource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

// createVirtConfigTestReadRequest creates a datasource.ReadRequest for the virtualization config data source
func createVirtConfigTestReadRequest(t *testing.T) datasource.ReadRequest {
	t.Helper()

	// Get the schema
	ds := NewVirtConfigDataSource()
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	// Build config value - all fields are computed, so they start as null
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"bridge":         tftypes.String,
			"v4_network":     tftypes.String,
			"v6_network":     tftypes.String,
			"pool": tftypes.String,
		},
	}, map[string]tftypes.Value{
		"bridge":         tftypes.NewValue(tftypes.String, nil),
		"v4_network":     tftypes.NewValue(tftypes.String, nil),
		"v6_network":     tftypes.NewValue(tftypes.String, nil),
		"pool": tftypes.NewValue(tftypes.String, nil),
	})

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    configValue,
	}

	return datasource.ReadRequest{
		Config: config,
	}
}

func TestVirtConfigDataSource_Read_Success(t *testing.T) {
	ds := &VirtConfigDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method != "virt.global.config" {
					t.Errorf("expected method 'virt.global.config', got %q", method)
				}
				// Verify no params are passed
				if params != nil {
					t.Errorf("expected nil params, got %v", params)
				}
				// Return a full config response
				return json.RawMessage(`{
					"bridge": "br0",
					"v4_network": "10.0.0.0/24",
					"v6_network": "fd00::/64",
					"pool": "tank"
				}`), nil
			},
		},
	}

	req := createVirtConfigTestReadRequest(t)

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
	var model VirtConfigDataSourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.Bridge.ValueString() != "br0" {
		t.Errorf("expected Bridge 'br0', got %q", model.Bridge.ValueString())
	}
	if model.V4Network.ValueString() != "10.0.0.0/24" {
		t.Errorf("expected V4Network '10.0.0.0/24', got %q", model.V4Network.ValueString())
	}
	if model.V6Network.ValueString() != "fd00::/64" {
		t.Errorf("expected V6Network 'fd00::/64', got %q", model.V6Network.ValueString())
	}
	if model.Pool.ValueString() != "tank" {
		t.Errorf("expected Pool 'tank', got %q", model.Pool.ValueString())
	}
}

func TestVirtConfigDataSource_Read_NullFields(t *testing.T) {
	ds := &VirtConfigDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// Return response with null fields
				return json.RawMessage(`{
					"bridge": "br0",
					"v4_network": null,
					"v6_network": null,
					"pool": null
				}`), nil
			},
		},
	}

	req := createVirtConfigTestReadRequest(t)

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

	var model VirtConfigDataSourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// Bridge should have a value
	if model.Bridge.ValueString() != "br0" {
		t.Errorf("expected Bridge 'br0', got %q", model.Bridge.ValueString())
	}

	// Null fields should be null in the model
	if !model.V4Network.IsNull() {
		t.Errorf("expected V4Network to be null, got %q", model.V4Network.ValueString())
	}
	if !model.V6Network.IsNull() {
		t.Errorf("expected V6Network to be null, got %q", model.V6Network.ValueString())
	}
	if !model.Pool.IsNull() {
		t.Errorf("expected Pool to be null, got %q", model.Pool.ValueString())
	}
}

func TestVirtConfigDataSource_Read_AllNullFields(t *testing.T) {
	ds := &VirtConfigDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// Return response with all null fields
				return json.RawMessage(`{
					"bridge": null,
					"v4_network": null,
					"v6_network": null,
					"pool": null
				}`), nil
			},
		},
	}

	req := createVirtConfigTestReadRequest(t)

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

	var model VirtConfigDataSourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// All fields should be null
	if !model.Bridge.IsNull() {
		t.Errorf("expected Bridge to be null, got %q", model.Bridge.ValueString())
	}
	if !model.V4Network.IsNull() {
		t.Errorf("expected V4Network to be null, got %q", model.V4Network.ValueString())
	}
	if !model.V6Network.IsNull() {
		t.Errorf("expected V6Network to be null, got %q", model.V6Network.ValueString())
	}
	if !model.Pool.IsNull() {
		t.Errorf("expected Pool to be null, got %q", model.Pool.ValueString())
	}
}

func TestVirtConfigDataSource_Read_APIError(t *testing.T) {
	ds := &VirtConfigDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection failed")
			},
		},
	}

	req := createVirtConfigTestReadRequest(t)

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

func TestVirtConfigDataSource_Read_InvalidJSON(t *testing.T) {
	ds := &VirtConfigDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		},
	}

	req := createVirtConfigTestReadRequest(t)

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

// Test that VirtConfigDataSource implements the DataSource interface
func TestVirtConfigDataSource_ImplementsInterfaces(t *testing.T) {
	ds := NewVirtConfigDataSource()

	_ = datasource.DataSource(ds)
	_ = datasource.DataSourceWithConfigure(ds.(*VirtConfigDataSource))
}
