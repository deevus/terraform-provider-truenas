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

func TestNewSnapshotsDataSource(t *testing.T) {
	ds := NewSnapshotsDataSource()
	if ds == nil {
		t.Fatal("expected non-nil data source")
	}

	_ = datasource.DataSource(ds)
	_ = datasource.DataSourceWithConfigure(ds.(*SnapshotsDataSource))
}

func TestSnapshotsDataSource_Metadata(t *testing.T) {
	ds := NewSnapshotsDataSource()

	req := datasource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_snapshots" {
		t.Errorf("expected TypeName 'truenas_snapshots', got %q", resp.TypeName)
	}
}

func TestSnapshotsDataSource_Schema(t *testing.T) {
	ds := NewSnapshotsDataSource()

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), req, resp)

	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify dataset_id is required
	datasetIDAttr, ok := resp.Schema.Attributes["dataset_id"]
	if !ok {
		t.Fatal("expected 'dataset_id' attribute in schema")
	}
	if !datasetIDAttr.IsRequired() {
		t.Error("expected 'dataset_id' attribute to be required")
	}

	// Verify snapshots is computed
	snapshotsAttr, ok := resp.Schema.Attributes["snapshots"]
	if !ok {
		t.Fatal("expected 'snapshots' attribute in schema")
	}
	if !snapshotsAttr.IsComputed() {
		t.Error("expected 'snapshots' attribute to be computed")
	}
}

func TestSnapshotsDataSource_Configure_Success(t *testing.T) {
	ds := NewSnapshotsDataSource().(*SnapshotsDataSource)

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

func getSnapshotsDataSourceSchema(t *testing.T) datasource.SchemaResponse {
	t.Helper()
	ds := NewSnapshotsDataSource()
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)
	return *schemaResp
}

func TestSnapshotsDataSource_Read_Success(t *testing.T) {
	ds := &SnapshotsDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[
					{
						"id": "tank/data@snap1",
						"name": "snap1",
						"dataset": "tank/data",
						"holds": {},
						"properties": {
							"used": {"parsed": 1024},
							"referenced": {"parsed": 2048}
						}
					},
					{
						"id": "tank/data@snap2",
						"name": "snap2",
						"dataset": "tank/data",
						"holds": {"terraform": true},
						"properties": {
							"used": {"parsed": 512},
							"referenced": {"parsed": 1024}
						}
					}
				]`), nil
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	if len(data.Snapshots) != 2 {
		t.Errorf("expected 2 snapshots, got %d", len(data.Snapshots))
	}
}

func TestSnapshotsDataSource_Read_Empty(t *testing.T) {
	ds := &SnapshotsDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	if len(data.Snapshots) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(data.Snapshots))
	}
}

func TestSnapshotsDataSource_Configure_NilProviderData(t *testing.T) {
	ds := NewSnapshotsDataSource().(*SnapshotsDataSource)

	req := datasource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors for nil provider data: %v", resp.Diagnostics)
	}
}

func TestSnapshotsDataSource_Configure_WrongType(t *testing.T) {
	ds := NewSnapshotsDataSource().(*SnapshotsDataSource)

	req := datasource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong provider data type")
	}
}

func TestSnapshotsDataSource_Read_APIError(t *testing.T) {
	ds := &SnapshotsDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API failure")
	}
}

func TestSnapshotsDataSource_Read_InvalidJSON(t *testing.T) {
	ds := &SnapshotsDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`{invalid json`), nil
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestSnapshotsDataSource_Read_Recursive(t *testing.T) {
	var capturedFilter any

	ds := &SnapshotsDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedFilter = params
				return json.RawMessage(`[
					{
						"id": "tank/data@snap1",
						"name": "snap1",
						"dataset": "tank/data",
						"holds": {},
						"properties": {
							"used": {"parsed": 1024},
							"referenced": {"parsed": 2048}
						}
					},
					{
						"id": "tank/data/child@snap2",
						"name": "snap2",
						"dataset": "tank/data/child",
						"holds": {},
						"properties": {
							"used": {"parsed": 512},
							"referenced": {"parsed": 1024}
						}
					}
				]`), nil
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, true),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify recursive filter was used (OR clause)
	filter, ok := capturedFilter.([][]any)
	if !ok {
		t.Fatalf("expected filter to be [][]any, got %T", capturedFilter)
	}
	if len(filter) != 1 || filter[0][0] != "OR" {
		t.Errorf("expected OR filter for recursive, got %v", filter)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	if len(data.Snapshots) != 2 {
		t.Errorf("expected 2 snapshots, got %d", len(data.Snapshots))
	}
}

func TestSnapshotsDataSource_Read_NamePattern_Match(t *testing.T) {
	ds := &SnapshotsDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[
					{
						"id": "tank/data@pre-upgrade-1",
						"name": "pre-upgrade-1",
						"dataset": "tank/data",
						"holds": {},
						"properties": {
							"used": {"parsed": 1024},
							"referenced": {"parsed": 2048}
						}
					},
					{
						"id": "tank/data@post-upgrade",
						"name": "post-upgrade",
						"dataset": "tank/data",
						"holds": {},
						"properties": {
							"used": {"parsed": 512},
							"referenced": {"parsed": 1024}
						}
					},
					{
						"id": "tank/data@pre-upgrade-2",
						"name": "pre-upgrade-2",
						"dataset": "tank/data",
						"holds": {},
						"properties": {
							"used": {"parsed": 256},
							"referenced": {"parsed": 512}
						}
					}
				]`), nil
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, "pre-*"),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	// Should only match pre-upgrade-1 and pre-upgrade-2
	if len(data.Snapshots) != 2 {
		t.Errorf("expected 2 snapshots matching 'pre-*', got %d", len(data.Snapshots))
	}
}

func TestSnapshotsDataSource_Read_NamePattern_NoMatch(t *testing.T) {
	ds := &SnapshotsDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[
					{
						"id": "tank/data@snap1",
						"name": "snap1",
						"dataset": "tank/data",
						"holds": {},
						"properties": {
							"used": {"parsed": 1024},
							"referenced": {"parsed": 2048}
						}
					}
				]`), nil
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, "backup-*"),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	if len(data.Snapshots) != 0 {
		t.Errorf("expected 0 snapshots matching 'backup-*', got %d", len(data.Snapshots))
	}
}

func TestSnapshotsDataSource_Read_NamePattern_Invalid(t *testing.T) {
	ds := &SnapshotsDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[
					{
						"id": "tank/data@snap1",
						"name": "snap1",
						"dataset": "tank/data",
						"holds": {},
						"properties": {
							"used": {"parsed": 1024},
							"referenced": {"parsed": 2048}
						}
					}
				]`), nil
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, "[invalid"),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid glob pattern")
	}
}

func TestSnapshotsDataSource_Read_GetConfigError(t *testing.T) {
	ds := &SnapshotsDataSource{
		client: &client.MockClient{},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)
	// Create an invalid config with wrong type for dataset_id (number instead of string)
	invalidValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.Number, // Wrong type - should be String
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.Number, 12345), // Wrong value type
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    invalidValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid config value")
	}
}
