package resources

import (
	"context"
	"errors"
	"math/big"
	"testing"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/terraform-provider-truenas/internal/services"
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

	req := resource.ConfigureRequest{
		ProviderData: &services.TrueNASServices{},
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if r.services == nil {
		t.Error("expected services to be set")
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
	var capturedOpts truenas.CreateGroupOpts

	r := &GroupResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateGroupOpts) (*truenas.Group, error) {
					capturedOpts = opts
					return &truenas.Group{ID: 100, GID: 3000, Name: "developers", SMB: true}, nil
				},
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

	if capturedOpts.Name != "developers" {
		t.Errorf("expected name 'developers', got %q", capturedOpts.Name)
	}
	if capturedOpts.SMB != true {
		t.Errorf("expected smb true, got %v", capturedOpts.SMB)
	}
	if capturedOpts.GID != 0 {
		t.Errorf("expected gid to not be set (0), got %d", capturedOpts.GID)
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
	var capturedOpts truenas.CreateGroupOpts

	r := &GroupResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateGroupOpts) (*truenas.Group, error) {
					capturedOpts = opts
					return &truenas.Group{ID: 101, GID: 5000, Name: "custom"}, nil
				},
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

	if capturedOpts.GID != 5000 {
		t.Errorf("expected GID 5000, got %d", capturedOpts.GID)
	}
}

func TestGroupResource_Create_WithSudoCommands(t *testing.T) {
	var capturedOpts truenas.CreateGroupOpts

	r := &GroupResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateGroupOpts) (*truenas.Group, error) {
					capturedOpts = opts
					return &truenas.Group{
						ID: 102, GID: 3001, Name: "admins", SMB: true,
						SudoCommands:         []string{"/usr/bin/apt", "/usr/bin/systemctl"},
						SudoCommandsNopasswd: []string{"/usr/bin/reboot"},
					}, nil
				},
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

	if len(capturedOpts.SudoCommands) != 2 || capturedOpts.SudoCommands[0] != "/usr/bin/apt" || capturedOpts.SudoCommands[1] != "/usr/bin/systemctl" {
		t.Errorf("unexpected sudo_commands: %v", capturedOpts.SudoCommands)
	}

	if len(capturedOpts.SudoCommandsNopasswd) != 1 || capturedOpts.SudoCommandsNopasswd[0] != "/usr/bin/reboot" {
		t.Errorf("unexpected sudo_commands_nopasswd: %v", capturedOpts.SudoCommandsNopasswd)
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
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateGroupOpts) (*truenas.Group, error) {
					return nil, errors.New("connection refused")
				},
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
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				GetFunc: func(ctx context.Context, id int64) (*truenas.Group, error) {
					return &truenas.Group{ID: 100, GID: 3000, Name: "developers", SMB: true, Users: []int64{1, 2}}, nil
				},
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
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				GetFunc: func(ctx context.Context, id int64) (*truenas.Group, error) {
					return nil, nil
				},
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
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				GetFunc: func(ctx context.Context, id int64) (*truenas.Group, error) {
					return nil, errors.New("connection refused")
				},
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
	var capturedID int64
	var capturedOpts truenas.UpdateGroupOpts

	r := &GroupResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				UpdateFunc: func(ctx context.Context, id int64, opts truenas.UpdateGroupOpts) (*truenas.Group, error) {
					capturedID = id
					capturedOpts = opts
					return &truenas.Group{ID: 100, GID: 3000, Name: "devs", SMB: false}, nil
				},
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

	if capturedID != 100 {
		t.Errorf("expected ID 100, got %d", capturedID)
	}

	if capturedOpts.Name != "devs" {
		t.Errorf("expected name 'devs', got %q", capturedOpts.Name)
	}
	if capturedOpts.SMB != false {
		t.Errorf("expected smb false, got %v", capturedOpts.SMB)
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
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				UpdateFunc: func(ctx context.Context, id int64, opts truenas.UpdateGroupOpts) (*truenas.Group, error) {
					return nil, errors.New("connection refused")
				},
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
	var capturedID int64

	r := &GroupResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				DeleteFunc: func(ctx context.Context, id int64) error {
					capturedID = id
					return nil
				},
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

	if capturedID != 100 {
		t.Errorf("expected ID 100, got %d", capturedID)
	}
}

func TestGroupResource_Delete_APIError(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				DeleteFunc: func(ctx context.Context, id int64) error {
					return errors.New("group in use")
				},
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

func TestGroupResource_Create_ServiceError(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateGroupOpts) (*truenas.Group, error) {
					return nil, errors.New("query failed")
				},
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
		t.Fatal("expected error when service returns error")
	}
}

func TestGroupResource_Create_NotFound(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateGroupOpts) (*truenas.Group, error) {
					return nil, nil
				},
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
		t.Fatal("expected error when service returns nil group")
	}
}

func TestGroupResource_Update_WithSudoCommands(t *testing.T) {
	var capturedOpts truenas.UpdateGroupOpts

	r := &GroupResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				UpdateFunc: func(ctx context.Context, id int64, opts truenas.UpdateGroupOpts) (*truenas.Group, error) {
					capturedOpts = opts
					return &truenas.Group{
						ID: 100, GID: 3000, Name: "admins", SMB: true,
						SudoCommands:         []string{"/usr/bin/apt"},
						SudoCommandsNopasswd: []string{"/usr/bin/reboot"},
					}, nil
				},
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

	if len(capturedOpts.SudoCommands) != 1 || capturedOpts.SudoCommands[0] != "/usr/bin/apt" {
		t.Errorf("unexpected sudo_commands: %v", capturedOpts.SudoCommands)
	}

	if len(capturedOpts.SudoCommandsNopasswd) != 1 || capturedOpts.SudoCommandsNopasswd[0] != "/usr/bin/reboot" {
		t.Errorf("unexpected sudo_commands_nopasswd: %v", capturedOpts.SudoCommandsNopasswd)
	}
}

func TestGroupResource_Update_ServiceError(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				UpdateFunc: func(ctx context.Context, id int64, opts truenas.UpdateGroupOpts) (*truenas.Group, error) {
					return nil, errors.New("query failed")
				},
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
		t.Fatal("expected error when service returns error")
	}
}

func TestGroupResource_Update_NotFound(t *testing.T) {
	r := &GroupResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Group: &truenas.MockGroupService{
				UpdateFunc: func(ctx context.Context, id int64, opts truenas.UpdateGroupOpts) (*truenas.Group, error) {
					return nil, nil
				},
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
		t.Fatal("expected error when service returns nil group")
	}
}

func TestGroupResource_MapGroupToModel_SudoCommandsFromAPI(t *testing.T) {
	// Test when data has null sudo_commands but API returns non-empty
	group := &truenas.Group{
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
