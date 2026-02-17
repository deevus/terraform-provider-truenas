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

func TestNewGroupDataSource(t *testing.T) {
	ds := NewGroupDataSource()
	if ds == nil {
		t.Fatal("expected non-nil data source")
	}

	_ = datasource.DataSource(ds)
	_ = datasource.DataSourceWithConfigure(ds.(*GroupDataSource))
}

func TestGroupDataSource_Metadata(t *testing.T) {
	ds := NewGroupDataSource()

	req := datasource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_group" {
		t.Errorf("expected TypeName 'truenas_group', got %q", resp.TypeName)
	}
}

func TestGroupDataSource_Schema(t *testing.T) {
	ds := NewGroupDataSource()

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), req, resp)

	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	for _, name := range []string{"id", "gid", "name", "smb", "builtin", "local", "sudo_commands", "sudo_commands_nopasswd", "users"} {
		if attrs[name] == nil {
			t.Errorf("expected '%s' attribute", name)
		}
	}

	if !attrs["name"].IsRequired() {
		t.Error("expected 'name' attribute to be required")
	}
	if !attrs["id"].IsComputed() {
		t.Error("expected 'id' attribute to be computed")
	}
	if !attrs["gid"].IsComputed() {
		t.Error("expected 'gid' attribute to be computed")
	}
}

func TestGroupDataSource_Configure_Success(t *testing.T) {
	ds := NewGroupDataSource().(*GroupDataSource)

	req := datasource.ConfigureRequest{ProviderData: &client.MockClient{}}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestGroupDataSource_Configure_NilProviderData(t *testing.T) {
	ds := NewGroupDataSource().(*GroupDataSource)

	req := datasource.ConfigureRequest{ProviderData: nil}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestGroupDataSource_Configure_WrongType(t *testing.T) {
	ds := NewGroupDataSource().(*GroupDataSource)

	req := datasource.ConfigureRequest{ProviderData: "not a client"}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

func createGroupReadRequest(t *testing.T, name string) (datasource.ReadRequest, datasource.SchemaResponse) {
	t.Helper()

	ds := NewGroupDataSource()
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                     tftypes.String,
			"gid":                    tftypes.Number,
			"name":                   tftypes.String,
			"smb":                    tftypes.Bool,
			"builtin":                tftypes.Bool,
			"local":                  tftypes.Bool,
			"sudo_commands":          tftypes.List{ElementType: tftypes.String},
			"sudo_commands_nopasswd": tftypes.List{ElementType: tftypes.String},
			"users":                  tftypes.List{ElementType: tftypes.Number},
		},
	}, map[string]tftypes.Value{
		"id":                     tftypes.NewValue(tftypes.String, nil),
		"gid":                    tftypes.NewValue(tftypes.Number, nil),
		"name":                   tftypes.NewValue(tftypes.String, name),
		"smb":                    tftypes.NewValue(tftypes.Bool, nil),
		"builtin":                tftypes.NewValue(tftypes.Bool, nil),
		"local":                  tftypes.NewValue(tftypes.Bool, nil),
		"sudo_commands":          tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"sudo_commands_nopasswd": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"users":                  tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, nil),
	})

	return datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configValue},
	}, *schemaResp
}

func TestGroupDataSource_Read_Success(t *testing.T) {
	ds := &GroupDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": 10,
					"gid": 1000,
					"name": "developers",
					"builtin": false,
					"smb": true,
					"sudo_commands": ["/usr/bin/apt"],
					"sudo_commands_nopasswd": [],
					"users": [5, 10],
					"local": true,
					"immutable": false
				}]`), nil
			},
		},
	}

	req, schemaResp := createGroupReadRequest(t, "developers")
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model GroupDataSourceModel
	resp.State.Get(context.Background(), &model)

	if model.ID.ValueString() != "10" {
		t.Errorf("expected ID '10', got %q", model.ID.ValueString())
	}
	if model.GID.ValueInt64() != 1000 {
		t.Errorf("expected GID 1000, got %d", model.GID.ValueInt64())
	}
	if model.Name.ValueString() != "developers" {
		t.Errorf("expected name 'developers', got %q", model.Name.ValueString())
	}
	if model.SMB.ValueBool() != true {
		t.Errorf("expected smb true, got %v", model.SMB.ValueBool())
	}
	if model.Builtin.ValueBool() != false {
		t.Errorf("expected builtin false, got %v", model.Builtin.ValueBool())
	}
	if model.Local.ValueBool() != true {
		t.Errorf("expected local true, got %v", model.Local.ValueBool())
	}
}

func TestGroupDataSource_Read_NotFound(t *testing.T) {
	ds := &GroupDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		},
	}

	req, schemaResp := createGroupReadRequest(t, "nonexistent")
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for group not found")
	}
}

func TestGroupDataSource_Read_APIError(t *testing.T) {
	ds := &GroupDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection failed")
			},
		},
	}

	req, schemaResp := createGroupReadRequest(t, "developers")
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestGroupDataSource_Read_InvalidJSON(t *testing.T) {
	ds := &GroupDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		},
	}

	req, schemaResp := createGroupReadRequest(t, "developers")
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestGroupDataSource_Read_VerifyFilterParams(t *testing.T) {
	var capturedParams any

	ds := &GroupDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`[{
					"id": 10, "gid": 1000, "name": "wheel",
					"builtin": true, "smb": false,
					"sudo_commands": [], "sudo_commands_nopasswd": [],
					"users": [], "local": true, "immutable": false
				}]`), nil
			},
		},
	}

	req, schemaResp := createGroupReadRequest(t, "wheel")
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	filters, ok := capturedParams.([][]string)
	if !ok {
		t.Fatalf("expected params to be [][]string, got %T", capturedParams)
	}

	if len(filters) != 1 || len(filters[0]) != 3 {
		t.Fatalf("expected 1 filter with 3 parts, got %v", filters)
	}

	if filters[0][0] != "group" || filters[0][1] != "=" || filters[0][2] != "wheel" {
		t.Errorf("expected filter ['group', '=', 'wheel'], got %v", filters[0])
	}
}
