package resources

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/api"
	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewGroupResource(t *testing.T) {
	r := NewGroupResource()
	if r == nil {
		t.Fatal("NewGroupResource returned nil")
	}

	_, ok := r.(*GroupResource)
	if !ok {
		t.Fatalf("expected *GroupResource, got %T", r)
	}

	// Verify interface implementations
	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(r.(*GroupResource))
	_ = resource.ResourceWithImportState(r.(*GroupResource))
}

func TestGroupResource_Metadata(t *testing.T) {
	r := NewGroupResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_group" {
		t.Errorf("expected TypeName 'truenas_group', got %q", resp.TypeName)
	}
}

func TestGroupResource_Configure_Success(t *testing.T) {
	r := NewGroupResource().(*GroupResource)

	mockClient := &client.MockClient{}

	req := resource.ConfigureRequest{
		ProviderData: mockClient,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if r.client == nil {
		t.Error("expected client to be set")
	}
}

func TestGroupResource_Configure_NilProviderData(t *testing.T) {
	r := NewGroupResource().(*GroupResource)

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestGroupResource_Configure_WrongType(t *testing.T) {
	r := NewGroupResource().(*GroupResource)

	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

func TestGroupResource_Schema(t *testing.T) {
	r := NewGroupResource()

	ctx := context.Background()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}

	r.Schema(ctx, schemaReq, schemaResp)

	if schemaResp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	attrs := schemaResp.Schema.Attributes
	for _, name := range []string{"id", "gid", "name", "smb", "sudo_commands", "sudo_commands_nopasswd", "builtin"} {
		if attrs[name] == nil {
			t.Errorf("expected '%s' attribute", name)
		}
	}
}

// Test helpers

func getGroupResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewGroupResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("failed to get schema: %v", schemaResp.Diagnostics)
	}
	return *schemaResp
}

type groupModelParams struct {
	ID                   interface{}
	GID                  interface{} // *big.Float or nil
	Name                 interface{}
	SMB                  bool
	SudoCommands         []string
	SudoCommandsNopasswd []string
	Builtin              bool
}

func createGroupModelValue(p groupModelParams) tftypes.Value {
	values := map[string]tftypes.Value{
		"id":      tftypes.NewValue(tftypes.String, p.ID),
		"name":    tftypes.NewValue(tftypes.String, p.Name),
		"smb":     tftypes.NewValue(tftypes.Bool, p.SMB),
		"builtin": tftypes.NewValue(tftypes.Bool, p.Builtin),
	}

	if p.GID != nil {
		values["gid"] = tftypes.NewValue(tftypes.Number, p.GID)
	} else {
		values["gid"] = tftypes.NewValue(tftypes.Number, nil)
	}

	// Handle sudo_commands list
	if p.SudoCommands != nil {
		sudoValues := make([]tftypes.Value, len(p.SudoCommands))
		for i, v := range p.SudoCommands {
			sudoValues[i] = tftypes.NewValue(tftypes.String, v)
		}
		values["sudo_commands"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, sudoValues)
	} else {
		values["sudo_commands"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil)
	}

	// Handle sudo_commands_nopasswd list
	if p.SudoCommandsNopasswd != nil {
		sudoValues := make([]tftypes.Value, len(p.SudoCommandsNopasswd))
		for i, v := range p.SudoCommandsNopasswd {
			sudoValues[i] = tftypes.NewValue(tftypes.String, v)
		}
		values["sudo_commands_nopasswd"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, sudoValues)
	} else {
		values["sudo_commands_nopasswd"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil)
	}

	objectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                     tftypes.String,
			"gid":                    tftypes.Number,
			"name":                   tftypes.String,
			"smb":                    tftypes.Bool,
			"sudo_commands":          tftypes.List{ElementType: tftypes.String},
			"sudo_commands_nopasswd": tftypes.List{ElementType: tftypes.String},
			"builtin":                tftypes.Bool,
		},
	}

	return tftypes.NewValue(objectType, values)
}

func TestGroupResource_Create_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "group.create" {
					capturedMethod = method
					capturedParams = params
					return json.RawMessage(`100`), nil
				}
				if method == "group.query" {
					return json.RawMessage(`[{
						"id": 100,
						"gid": 3000,
						"name": "developers",
						"builtin": false,
						"smb": true,
						"sudo_commands": [],
						"sudo_commands_nopasswd": [],
						"users": [],
						"local": true,
						"immutable": false
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)
	planValue := createGroupModelValue(groupModelParams{
		Name: "developers",
		SMB:  true,
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

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if capturedMethod != "group.create" {
		t.Errorf("expected method 'group.create', got %q", capturedMethod)
	}

	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["name"] != "developers" {
		t.Errorf("expected name 'developers', got %v", params["name"])
	}
	if params["smb"] != true {
		t.Errorf("expected smb true, got %v", params["smb"])
	}
	if _, hasGID := params["gid"]; hasGID {
		t.Error("expected gid to not be in params when not set")
	}

	var resultData GroupResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "100" {
		t.Errorf("expected ID '100', got %q", resultData.ID.ValueString())
	}
	if resultData.GID.ValueInt64() != 3000 {
		t.Errorf("expected GID 3000, got %d", resultData.GID.ValueInt64())
	}
	if resultData.Name.ValueString() != "developers" {
		t.Errorf("expected name 'developers', got %q", resultData.Name.ValueString())
	}
	if resultData.SMB.ValueBool() != true {
		t.Errorf("expected smb true, got %v", resultData.SMB.ValueBool())
	}
	if resultData.Builtin.ValueBool() != false {
		t.Errorf("expected builtin false, got %v", resultData.Builtin.ValueBool())
	}
}

func TestGroupResource_Create_WithGID(t *testing.T) {
	var capturedParams any

	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "group.create" {
					capturedParams = params
					return json.RawMessage(`101`), nil
				}
				if method == "group.query" {
					return json.RawMessage(`[{
						"id": 101,
						"gid": 5000,
						"name": "custom",
						"builtin": false,
						"smb": false,
						"sudo_commands": [],
						"sudo_commands_nopasswd": [],
						"users": [],
						"local": true,
						"immutable": false
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)
	planValue := createGroupModelValue(groupModelParams{
		GID:  big.NewFloat(5000),
		Name: "custom",
		SMB:  false,
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

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["gid"] != int64(5000) {
		t.Errorf("expected gid 5000, got %v", params["gid"])
	}
}

func TestGroupResource_Create_WithSudoCommands(t *testing.T) {
	var capturedParams any

	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "group.create" {
					capturedParams = params
					return json.RawMessage(`102`), nil
				}
				if method == "group.query" {
					return json.RawMessage(`[{
						"id": 102,
						"gid": 3001,
						"name": "admins",
						"builtin": false,
						"smb": true,
						"sudo_commands": ["/usr/bin/apt", "/usr/bin/systemctl"],
						"sudo_commands_nopasswd": ["/usr/bin/reboot"],
						"users": [],
						"local": true,
						"immutable": false
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)
	planValue := createGroupModelValue(groupModelParams{
		Name:                 "admins",
		SMB:                  true,
		SudoCommands:         []string{"/usr/bin/apt", "/usr/bin/systemctl"},
		SudoCommandsNopasswd: []string{"/usr/bin/reboot"},
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

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	sudoCmds, ok := params["sudo_commands"].([]string)
	if !ok {
		t.Fatalf("expected sudo_commands to be []string, got %T", params["sudo_commands"])
	}
	if len(sudoCmds) != 2 || sudoCmds[0] != "/usr/bin/apt" || sudoCmds[1] != "/usr/bin/systemctl" {
		t.Errorf("unexpected sudo_commands: %v", sudoCmds)
	}

	sudoNP, ok := params["sudo_commands_nopasswd"].([]string)
	if !ok {
		t.Fatalf("expected sudo_commands_nopasswd to be []string, got %T", params["sudo_commands_nopasswd"])
	}
	if len(sudoNP) != 1 || sudoNP[0] != "/usr/bin/reboot" {
		t.Errorf("unexpected sudo_commands_nopasswd: %v", sudoNP)
	}

	// Verify state has the sudo commands
	var resultData GroupResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.SudoCommands.IsNull() {
		t.Error("expected sudo_commands to be set")
	}
	if resultData.SudoCommandsNopasswd.IsNull() {
		t.Error("expected sudo_commands_nopasswd to be set")
	}
}

func TestGroupResource_Create_APIError(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)
	planValue := createGroupModelValue(groupModelParams{
		Name: "developers",
		SMB:  true,
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
		t.Fatal("expected error for API error")
	}

	if !resp.State.Raw.IsNull() {
		t.Error("expected state to not be set when API returns error")
	}
}

func TestGroupResource_Read_Success(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": 100,
					"gid": 3000,
					"name": "developers",
					"builtin": false,
					"smb": true,
					"sudo_commands": [],
					"sudo_commands_nopasswd": [],
					"users": [1, 2],
					"local": true,
					"immutable": false
				}]`), nil
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)
	stateValue := createGroupModelValue(groupModelParams{
		ID:   "100",
		GID:  big.NewFloat(3000),
		Name: "developers",
		SMB:  true,
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

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var resultData GroupResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "100" {
		t.Errorf("expected ID '100', got %q", resultData.ID.ValueString())
	}
	if resultData.GID.ValueInt64() != 3000 {
		t.Errorf("expected GID 3000, got %d", resultData.GID.ValueInt64())
	}
	if resultData.Name.ValueString() != "developers" {
		t.Errorf("expected name 'developers', got %q", resultData.Name.ValueString())
	}
	if resultData.SMB.ValueBool() != true {
		t.Errorf("expected smb true, got %v", resultData.SMB.ValueBool())
	}
	if resultData.Builtin.ValueBool() != false {
		t.Errorf("expected builtin false, got %v", resultData.Builtin.ValueBool())
	}
}

func TestGroupResource_Read_NotFound(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)
	stateValue := createGroupModelValue(groupModelParams{
		ID:   "100",
		GID:  big.NewFloat(3000),
		Name: "deleted-group",
		SMB:  true,
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

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed when resource not found")
	}
}

func TestGroupResource_Read_APIError(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)
	stateValue := createGroupModelValue(groupModelParams{
		ID:   "100",
		GID:  big.NewFloat(3000),
		Name: "developers",
		SMB:  true,
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
		t.Fatal("expected error for API error")
	}
}

func TestGroupResource_Update_Success(t *testing.T) {
	var capturedMethod string
	var capturedID int64
	var capturedUpdateData map[string]any

	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "group.update" {
					capturedMethod = method
					args := params.([]any)
					capturedID = args[0].(int64)
					capturedUpdateData = args[1].(map[string]any)
					return json.RawMessage(`{"id": 100}`), nil
				}
				if method == "group.query" {
					return json.RawMessage(`[{
						"id": 100,
						"gid": 3000,
						"name": "devs",
						"builtin": false,
						"smb": false,
						"sudo_commands": [],
						"sudo_commands_nopasswd": [],
						"users": [],
						"local": true,
						"immutable": false
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)

	stateValue := createGroupModelValue(groupModelParams{
		ID:   "100",
		GID:  big.NewFloat(3000),
		Name: "developers",
		SMB:  true,
	})

	planValue := createGroupModelValue(groupModelParams{
		ID:   "100",
		GID:  big.NewFloat(3000),
		Name: "devs",
		SMB:  false,
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

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if capturedMethod != "group.update" {
		t.Errorf("expected method 'group.update', got %q", capturedMethod)
	}

	if capturedID != 100 {
		t.Errorf("expected ID 100, got %d", capturedID)
	}

	if capturedUpdateData["name"] != "devs" {
		t.Errorf("expected name 'devs', got %v", capturedUpdateData["name"])
	}
	if capturedUpdateData["smb"] != false {
		t.Errorf("expected smb false, got %v", capturedUpdateData["smb"])
	}
	// gid should NOT be in update params
	if _, hasGID := capturedUpdateData["gid"]; hasGID {
		t.Error("expected gid to not be in update params")
	}

	var resultData GroupResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "100" {
		t.Errorf("expected ID '100', got %q", resultData.ID.ValueString())
	}
	if resultData.Name.ValueString() != "devs" {
		t.Errorf("expected name 'devs', got %q", resultData.Name.ValueString())
	}
	if resultData.SMB.ValueBool() != false {
		t.Errorf("expected smb false, got %v", resultData.SMB.ValueBool())
	}
}

func TestGroupResource_Update_APIError(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)

	stateValue := createGroupModelValue(groupModelParams{
		ID:   "100",
		GID:  big.NewFloat(3000),
		Name: "developers",
		SMB:  true,
	})

	planValue := createGroupModelValue(groupModelParams{
		ID:   "100",
		GID:  big.NewFloat(3000),
		Name: "devs",
		SMB:  false,
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
		t.Fatal("expected error for API error")
	}
}

func TestGroupResource_Delete_Success(t *testing.T) {
	var capturedMethod string
	var capturedArgs []any

	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedArgs = params.([]any)
				return json.RawMessage(`true`), nil
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)
	stateValue := createGroupModelValue(groupModelParams{
		ID:   "100",
		GID:  big.NewFloat(3000),
		Name: "developers",
		SMB:  true,
	})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if capturedMethod != "group.delete" {
		t.Errorf("expected method 'group.delete', got %q", capturedMethod)
	}

	if capturedArgs[0] != int64(100) {
		t.Errorf("expected ID 100, got %v", capturedArgs[0])
	}

	opts, ok := capturedArgs[1].(map[string]any)
	if !ok {
		t.Fatalf("expected opts to be map[string]any, got %T", capturedArgs[1])
	}
	if opts["delete_users"] != false {
		t.Errorf("expected delete_users false, got %v", opts["delete_users"])
	}
}

func TestGroupResource_Delete_APIError(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("group in use")
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)
	stateValue := createGroupModelValue(groupModelParams{
		ID:   "100",
		GID:  big.NewFloat(3000),
		Name: "developers",
		SMB:  true,
	})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestGroupResource_Create_QueryError(t *testing.T) {
	callCount := 0
	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "group.create" {
					return json.RawMessage(`100`), nil
				}
				callCount++
				return nil, errors.New("query failed")
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)
	planValue := createGroupModelValue(groupModelParams{
		Name: "developers",
		SMB:  true,
	})

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

func TestGroupResource_Create_QueryNotFound(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "group.create" {
					return json.RawMessage(`100`), nil
				}
				return json.RawMessage(`[]`), nil
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)
	planValue := createGroupModelValue(groupModelParams{
		Name: "developers",
		SMB:  true,
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when query returns empty after create")
	}
}

func TestGroupResource_Create_ParseError(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not json`), nil
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)
	planValue := createGroupModelValue(groupModelParams{
		Name: "developers",
		SMB:  true,
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when response is not valid JSON")
	}
}

func TestGroupResource_Update_WithSudoCommands(t *testing.T) {
	var capturedUpdateData map[string]any

	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "group.update" {
					args := params.([]any)
					capturedUpdateData = args[1].(map[string]any)
					return json.RawMessage(`{"id": 100}`), nil
				}
				if method == "group.query" {
					return json.RawMessage(`[{
						"id": 100,
						"gid": 3000,
						"name": "admins",
						"builtin": false,
						"smb": true,
						"sudo_commands": ["/usr/bin/apt"],
						"sudo_commands_nopasswd": ["/usr/bin/reboot"],
						"users": [],
						"local": true,
						"immutable": false
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)

	stateValue := createGroupModelValue(groupModelParams{
		ID:                   "100",
		GID:                  big.NewFloat(3000),
		Name:                 "admins",
		SMB:                  true,
		SudoCommands:         []string{"/usr/bin/apt"},
		SudoCommandsNopasswd: []string{"/usr/bin/reboot"},
	})

	planValue := createGroupModelValue(groupModelParams{
		ID:                   "100",
		GID:                  big.NewFloat(3000),
		Name:                 "admins",
		SMB:                  true,
		SudoCommands:         []string{"/usr/bin/apt"},
		SudoCommandsNopasswd: []string{"/usr/bin/reboot"},
	})

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	sudoCmds, ok := capturedUpdateData["sudo_commands"].([]string)
	if !ok {
		t.Fatalf("expected sudo_commands to be []string, got %T", capturedUpdateData["sudo_commands"])
	}
	if len(sudoCmds) != 1 || sudoCmds[0] != "/usr/bin/apt" {
		t.Errorf("unexpected sudo_commands: %v", sudoCmds)
	}

	sudoNP, ok := capturedUpdateData["sudo_commands_nopasswd"].([]string)
	if !ok {
		t.Fatalf("expected sudo_commands_nopasswd to be []string, got %T", capturedUpdateData["sudo_commands_nopasswd"])
	}
	if len(sudoNP) != 1 || sudoNP[0] != "/usr/bin/reboot" {
		t.Errorf("unexpected sudo_commands_nopasswd: %v", sudoNP)
	}
}

func TestGroupResource_Update_QueryError(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "group.update" {
					return json.RawMessage(`{"id": 100}`), nil
				}
				return nil, errors.New("query failed")
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)
	stateValue := createGroupModelValue(groupModelParams{
		ID: "100", GID: big.NewFloat(3000), Name: "devs", SMB: true,
	})
	planValue := createGroupModelValue(groupModelParams{
		ID: "100", GID: big.NewFloat(3000), Name: "devs2", SMB: true,
	})

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when query after update fails")
	}
}

func TestGroupResource_Update_QueryNotFound(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "group.update" {
					return json.RawMessage(`{"id": 100}`), nil
				}
				return json.RawMessage(`[]`), nil
			},
		}},
	}

	schemaResp := getGroupResourceSchema(t)
	stateValue := createGroupModelValue(groupModelParams{
		ID: "100", GID: big.NewFloat(3000), Name: "devs", SMB: true,
	})
	planValue := createGroupModelValue(groupModelParams{
		ID: "100", GID: big.NewFloat(3000), Name: "devs2", SMB: true,
	})

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when query returns empty after update")
	}
}

func TestGroupResource_MapGroupToModel_SudoCommandsFromAPI(t *testing.T) {
	// Test when data has null sudo_commands but API returns non-empty
	group := &api.GroupResponse{
		ID: 100, GID: 3000, Name: "admins", SMB: true,
		SudoCommands:         []string{"/usr/bin/apt"},
		SudoCommandsNopasswd: []string{"/usr/bin/reboot"},
	}
	data := &GroupResourceModel{}
	mapGroupToModel(context.Background(), group, data)

	if data.SudoCommands.IsNull() {
		t.Error("expected sudo_commands to be set from API when not null and API has values")
	}
	if data.SudoCommandsNopasswd.IsNull() {
		t.Error("expected sudo_commands_nopasswd to be set from API when not null and API has values")
	}
}
