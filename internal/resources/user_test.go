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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewUserResource(t *testing.T) {
	r := NewUserResource()
	if r == nil {
		t.Fatal("NewUserResource returned nil")
	}

	_, ok := r.(*UserResource)
	if !ok {
		t.Fatalf("expected *UserResource, got %T", r)
	}

	// Verify interface implementations
	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(r.(*UserResource))
	_ = resource.ResourceWithImportState(r.(*UserResource))
}

func TestUserResource_Metadata(t *testing.T) {
	r := NewUserResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_user" {
		t.Errorf("expected TypeName 'truenas_user', got %q", resp.TypeName)
	}
}

func TestUserResource_Configure_Success(t *testing.T) {
	r := NewUserResource().(*UserResource)

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

func TestUserResource_Configure_NilProviderData(t *testing.T) {
	r := NewUserResource().(*UserResource)

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestUserResource_Configure_WrongType(t *testing.T) {
	r := NewUserResource().(*UserResource)

	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

func TestUserResource_Schema(t *testing.T) {
	r := NewUserResource()

	ctx := context.Background()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}

	r.Schema(ctx, schemaReq, schemaResp)

	if schemaResp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	attrs := schemaResp.Schema.Attributes
	for _, name := range []string{
		"id", "uid", "username", "full_name", "email", "password",
		"password_disabled", "group_id", "group_create", "groups",
		"home", "home_create", "home_mode", "shell", "smb",
		"ssh_password_enabled", "sshpubkey", "locked",
		"sudo_commands", "sudo_commands_nopasswd", "builtin",
	} {
		if attrs[name] == nil {
			t.Errorf("expected '%s' attribute", name)
		}
	}

	// Verify password is sensitive
	pwdAttr := attrs["password"].(schema.StringAttribute)
	if !pwdAttr.Sensitive {
		t.Error("expected password to be sensitive")
	}
}

// Test helpers

func getUserResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewUserResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("failed to get schema: %v", schemaResp.Diagnostics)
	}
	return *schemaResp
}

type userModelParams struct {
	ID                   interface{}
	UID                  interface{} // *big.Float or nil
	Username             interface{}
	FullName             interface{}
	Email                interface{}
	Password             interface{}
	PasswordDisabled     bool
	GroupID              interface{} // *big.Float or nil
	GroupCreate          interface{} // bool or nil
	Groups               []int64
	Home                 interface{}
	HomeCreate           interface{} // bool or nil
	HomeMode             interface{}
	Shell                interface{}
	SMB                  bool
	SSHPasswordEnabled   bool
	SSHPubKey            interface{}
	Locked               bool
	SudoCommands         []string
	SudoCommandsNopasswd []string
	Builtin              bool
}

func createUserModelValue(p userModelParams) tftypes.Value {
	values := map[string]tftypes.Value{
		"id":                     tftypes.NewValue(tftypes.String, p.ID),
		"username":               tftypes.NewValue(tftypes.String, p.Username),
		"full_name":              tftypes.NewValue(tftypes.String, p.FullName),
		"email":                  tftypes.NewValue(tftypes.String, p.Email),
		"password":               tftypes.NewValue(tftypes.String, p.Password),
		"password_disabled":      tftypes.NewValue(tftypes.Bool, p.PasswordDisabled),
		"home":                   tftypes.NewValue(tftypes.String, p.Home),
		"home_mode":              tftypes.NewValue(tftypes.String, p.HomeMode),
		"shell":                  tftypes.NewValue(tftypes.String, p.Shell),
		"smb":                    tftypes.NewValue(tftypes.Bool, p.SMB),
		"ssh_password_enabled":   tftypes.NewValue(tftypes.Bool, p.SSHPasswordEnabled),
		"sshpubkey":              tftypes.NewValue(tftypes.String, p.SSHPubKey),
		"locked":                 tftypes.NewValue(tftypes.Bool, p.Locked),
		"builtin":                tftypes.NewValue(tftypes.Bool, p.Builtin),
	}

	if p.UID != nil {
		values["uid"] = tftypes.NewValue(tftypes.Number, p.UID)
	} else {
		values["uid"] = tftypes.NewValue(tftypes.Number, nil)
	}

	if p.GroupID != nil {
		values["group_id"] = tftypes.NewValue(tftypes.Number, p.GroupID)
	} else {
		values["group_id"] = tftypes.NewValue(tftypes.Number, nil)
	}

	if p.GroupCreate != nil {
		values["group_create"] = tftypes.NewValue(tftypes.Bool, p.GroupCreate)
	} else {
		values["group_create"] = tftypes.NewValue(tftypes.Bool, nil)
	}

	if p.HomeCreate != nil {
		values["home_create"] = tftypes.NewValue(tftypes.Bool, p.HomeCreate)
	} else {
		values["home_create"] = tftypes.NewValue(tftypes.Bool, nil)
	}

	// Handle groups list (Int64)
	if p.Groups != nil {
		groupValues := make([]tftypes.Value, len(p.Groups))
		for i, v := range p.Groups {
			groupValues[i] = tftypes.NewValue(tftypes.Number, big.NewFloat(float64(v)))
		}
		values["groups"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, groupValues)
	} else {
		values["groups"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, nil)
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
			"uid":                    tftypes.Number,
			"username":               tftypes.String,
			"full_name":              tftypes.String,
			"email":                  tftypes.String,
			"password":               tftypes.String,
			"password_disabled":      tftypes.Bool,
			"group_id":               tftypes.Number,
			"group_create":           tftypes.Bool,
			"groups":                 tftypes.List{ElementType: tftypes.Number},
			"home":                   tftypes.String,
			"home_create":            tftypes.Bool,
			"home_mode":              tftypes.String,
			"shell":                  tftypes.String,
			"smb":                    tftypes.Bool,
			"ssh_password_enabled":   tftypes.Bool,
			"sshpubkey":              tftypes.String,
			"locked":                 tftypes.Bool,
			"sudo_commands":          tftypes.List{ElementType: tftypes.String},
			"sudo_commands_nopasswd": tftypes.List{ElementType: tftypes.String},
			"builtin":                tftypes.Bool,
		},
	}

	return tftypes.NewValue(objectType, values)
}

var userQueryResponse = `[{
	"id": 50,
	"uid": 1001,
	"username": "jdoe",
	"full_name": "John Doe",
	"email": "john@example.com",
	"home": "/home/jdoe",
	"shell": "/usr/bin/zsh",
	"home_mode": "700",
	"group": {"id": 100, "bsdgrp_gid": 1001, "bsdgrp_group": "jdoe"},
	"groups": [],
	"smb": true,
	"password_disabled": false,
	"ssh_password_enabled": false,
	"sshpubkey": null,
	"locked": false,
	"sudo_commands": [],
	"sudo_commands_nopasswd": [],
	"builtin": false,
	"local": true,
	"immutable": false
}]`

func TestUserResource_Create_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "user.create" {
					capturedMethod = method
					capturedParams = params
					return json.RawMessage(`{"id": 50}`), nil
				}
				if method == "user.query" {
					return json.RawMessage(userQueryResponse), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	planValue := createUserModelValue(userModelParams{
		Username:     "jdoe",
		FullName:     "John Doe",
		Email:        "john@example.com",
		GroupCreate:  true,
		Home:         "/home/jdoe",
		HomeMode:     "700",
		Shell:        "/usr/bin/zsh",
		SMB:          true,
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

	if capturedMethod != "user.create" {
		t.Errorf("expected method 'user.create', got %q", capturedMethod)
	}

	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["username"] != "jdoe" {
		t.Errorf("expected username 'jdoe', got %v", params["username"])
	}
	if params["full_name"] != "John Doe" {
		t.Errorf("expected full_name 'John Doe', got %v", params["full_name"])
	}
	if params["group_create"] != true {
		t.Errorf("expected group_create true, got %v", params["group_create"])
	}

	var resultData UserResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "50" {
		t.Errorf("expected ID '50', got %q", resultData.ID.ValueString())
	}
	if resultData.UID.ValueInt64() != 1001 {
		t.Errorf("expected UID 1001, got %d", resultData.UID.ValueInt64())
	}
	if resultData.Username.ValueString() != "jdoe" {
		t.Errorf("expected username 'jdoe', got %q", resultData.Username.ValueString())
	}
	if resultData.GroupID.ValueInt64() != 100 {
		t.Errorf("expected group_id 100, got %d", resultData.GroupID.ValueInt64())
	}
}

func TestUserResource_Create_WithPassword(t *testing.T) {
	var capturedParams any

	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "user.create" {
					capturedParams = params
					return json.RawMessage(`{"id": 51}`), nil
				}
				if method == "user.query" {
					return json.RawMessage(userQueryResponse), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	planValue := createUserModelValue(userModelParams{
		Username:     "jdoe",
		FullName:     "John Doe",
		Email:        "john@example.com",
		Password:     "s3cret!",
		GroupCreate:  true,
		Home:         "/home/jdoe",
		HomeMode:     "700",
		Shell:        "/usr/bin/zsh",
		SMB:          true,
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

	if params["password"] != "s3cret!" {
		t.Errorf("expected password 's3cret!', got %v", params["password"])
	}
}

func TestUserResource_Create_WithGroupID(t *testing.T) {
	var capturedParams any

	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "user.create" {
					capturedParams = params
					return json.RawMessage(`{"id": 52}`), nil
				}
				if method == "user.query" {
					return json.RawMessage(userQueryResponse), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	planValue := createUserModelValue(userModelParams{
		Username: "jdoe",
		FullName: "John Doe",
		Email:    "john@example.com",
		GroupID:  big.NewFloat(200),
		Home:     "/home/jdoe",
		HomeMode: "700",
		Shell:    "/usr/bin/zsh",
		SMB:      true,
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

	// API param is "group" (not "group_id")
	if params["group"] != int64(200) {
		t.Errorf("expected group 200, got %v", params["group"])
	}
	if _, hasGroupCreate := params["group_create"]; hasGroupCreate {
		t.Error("expected group_create to not be in params when not set")
	}
}

func TestUserResource_Create_WithAllOptions(t *testing.T) {
	var capturedParams any

	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "user.create" {
					capturedParams = params
					return json.RawMessage(`{"id": 53}`), nil
				}
				if method == "user.query" {
					return json.RawMessage(`[{
						"id": 53,
						"uid": 2000,
						"username": "admin",
						"full_name": "Admin User",
						"email": "admin@example.com",
						"home": "/home/admin",
						"shell": "/bin/bash",
						"home_mode": "755",
						"group": {"id": 300, "bsdgrp_gid": 2000, "bsdgrp_group": "admin"},
						"groups": [100, 200],
						"smb": false,
						"password_disabled": false,
						"ssh_password_enabled": true,
						"sshpubkey": "ssh-rsa AAAA...",
						"locked": false,
						"sudo_commands": ["/usr/bin/apt"],
						"sudo_commands_nopasswd": ["/usr/bin/reboot"],
						"builtin": false,
						"local": true,
						"immutable": false
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	planValue := createUserModelValue(userModelParams{
		UID:                  big.NewFloat(2000),
		Username:             "admin",
		FullName:             "Admin User",
		Email:                "admin@example.com",
		Password:             "p@ssw0rd",
		GroupID:              big.NewFloat(300),
		Groups:               []int64{100, 200},
		Home:                 "/home/admin",
		HomeCreate:           true,
		HomeMode:             "755",
		Shell:                "/bin/bash",
		SMB:                  false,
		SSHPasswordEnabled:   true,
		SSHPubKey:            "ssh-rsa AAAA...",
		SudoCommands:         []string{"/usr/bin/apt"},
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

	if params["uid"] != int64(2000) {
		t.Errorf("expected uid 2000, got %v", params["uid"])
	}
	if params["home_create"] != true {
		t.Errorf("expected home_create true, got %v", params["home_create"])
	}
	if params["sshpubkey"] != "ssh-rsa AAAA..." {
		t.Errorf("expected sshpubkey, got %v", params["sshpubkey"])
	}
	if params["ssh_password_enabled"] != true {
		t.Errorf("expected ssh_password_enabled true, got %v", params["ssh_password_enabled"])
	}

	groups, ok := params["groups"].([]int64)
	if !ok {
		t.Fatalf("expected groups to be []int64, got %T", params["groups"])
	}
	if len(groups) != 2 || groups[0] != 100 || groups[1] != 200 {
		t.Errorf("unexpected groups: %v", groups)
	}

	sudoCmds, ok := params["sudo_commands"].([]string)
	if !ok {
		t.Fatalf("expected sudo_commands to be []string, got %T", params["sudo_commands"])
	}
	if len(sudoCmds) != 1 || sudoCmds[0] != "/usr/bin/apt" {
		t.Errorf("unexpected sudo_commands: %v", sudoCmds)
	}

	// Verify state
	var resultData UserResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.UID.ValueInt64() != 2000 {
		t.Errorf("expected UID 2000, got %d", resultData.UID.ValueInt64())
	}
	if resultData.SSHPasswordEnabled.ValueBool() != true {
		t.Errorf("expected ssh_password_enabled true, got %v", resultData.SSHPasswordEnabled.ValueBool())
	}
	if resultData.SSHPubKey.ValueString() != "ssh-rsa AAAA..." {
		t.Errorf("expected sshpubkey 'ssh-rsa AAAA...', got %q", resultData.SSHPubKey.ValueString())
	}
}

func TestUserResource_Create_APIError(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	planValue := createUserModelValue(userModelParams{
		Username:    "jdoe",
		FullName:    "John Doe",
		Email:       "john@example.com",
		GroupCreate: true,
		Home:        "/home/jdoe",
		HomeMode:    "700",
		Shell:       "/usr/bin/zsh",
		SMB:         true,
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

func TestUserResource_Read_Success(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(userQueryResponse), nil
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	stateValue := createUserModelValue(userModelParams{
		ID:       "50",
		UID:      big.NewFloat(1001),
		Username: "jdoe",
		FullName: "John Doe",
		Email:    "john@example.com",
		GroupID:  big.NewFloat(100),
		Home:     "/home/jdoe",
		HomeMode: "700",
		Shell:    "/usr/bin/zsh",
		SMB:      true,
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

	var resultData UserResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "50" {
		t.Errorf("expected ID '50', got %q", resultData.ID.ValueString())
	}
	if resultData.UID.ValueInt64() != 1001 {
		t.Errorf("expected UID 1001, got %d", resultData.UID.ValueInt64())
	}
	if resultData.Username.ValueString() != "jdoe" {
		t.Errorf("expected username 'jdoe', got %q", resultData.Username.ValueString())
	}
	if resultData.FullName.ValueString() != "John Doe" {
		t.Errorf("expected full_name 'John Doe', got %q", resultData.FullName.ValueString())
	}
	if resultData.Email.ValueString() != "john@example.com" {
		t.Errorf("expected email 'john@example.com', got %q", resultData.Email.ValueString())
	}
	if resultData.GroupID.ValueInt64() != 100 {
		t.Errorf("expected group_id 100, got %d", resultData.GroupID.ValueInt64())
	}
	if resultData.Home.ValueString() != "/home/jdoe" {
		t.Errorf("expected home '/home/jdoe', got %q", resultData.Home.ValueString())
	}
	if resultData.Shell.ValueString() != "/usr/bin/zsh" {
		t.Errorf("expected shell '/usr/bin/zsh', got %q", resultData.Shell.ValueString())
	}
	if resultData.Builtin.ValueBool() != false {
		t.Errorf("expected builtin false, got %v", resultData.Builtin.ValueBool())
	}
}

func TestUserResource_Read_NotFound(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	stateValue := createUserModelValue(userModelParams{
		ID:       "50",
		UID:      big.NewFloat(1001),
		Username: "deleted-user",
		FullName: "Deleted User",
		Email:    "",
		Home:     "/var/empty",
		HomeMode: "700",
		Shell:    "/usr/bin/zsh",
		SMB:      true,
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

func TestUserResource_Read_APIError(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	stateValue := createUserModelValue(userModelParams{
		ID:       "50",
		UID:      big.NewFloat(1001),
		Username: "jdoe",
		FullName: "John Doe",
		Email:    "john@example.com",
		Home:     "/home/jdoe",
		HomeMode: "700",
		Shell:    "/usr/bin/zsh",
		SMB:      true,
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

func TestUserResource_Read_PreservesPassword(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(userQueryResponse), nil
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	stateValue := createUserModelValue(userModelParams{
		ID:       "50",
		UID:      big.NewFloat(1001),
		Username: "jdoe",
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "my-secret-password",
		GroupID:  big.NewFloat(100),
		Home:     "/home/jdoe",
		HomeMode: "700",
		Shell:    "/usr/bin/zsh",
		SMB:      true,
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

	var resultData UserResourceModel
	resp.State.Get(context.Background(), &resultData)

	// Password should be preserved from state, not overwritten by API read
	if resultData.Password.ValueString() != "my-secret-password" {
		t.Errorf("expected password to be preserved as 'my-secret-password', got %q", resultData.Password.ValueString())
	}
}

func TestUserResource_Update_Success(t *testing.T) {
	var capturedMethod string
	var capturedID int64
	var capturedUpdateData map[string]any

	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "user.update" {
					capturedMethod = method
					args := params.([]any)
					capturedID = args[0].(int64)
					capturedUpdateData = args[1].(map[string]any)
					return json.RawMessage(`{"id": 50}`), nil
				}
				if method == "user.query" {
					return json.RawMessage(`[{
						"id": 50,
						"uid": 1001,
						"username": "jdoe",
						"full_name": "Jane Doe",
						"email": "jane@example.com",
						"home": "/home/jdoe",
						"shell": "/bin/bash",
						"home_mode": "700",
						"group": {"id": 100, "bsdgrp_gid": 1001, "bsdgrp_group": "jdoe"},
						"groups": [],
						"smb": true,
						"password_disabled": false,
						"ssh_password_enabled": false,
						"sshpubkey": null,
						"locked": false,
						"sudo_commands": [],
						"sudo_commands_nopasswd": [],
						"builtin": false,
						"local": true,
						"immutable": false
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)

	stateValue := createUserModelValue(userModelParams{
		ID:       "50",
		UID:      big.NewFloat(1001),
		Username: "jdoe",
		FullName: "John Doe",
		Email:    "john@example.com",
		GroupID:  big.NewFloat(100),
		Home:     "/home/jdoe",
		HomeMode: "700",
		Shell:    "/usr/bin/zsh",
		SMB:      true,
	})

	planValue := createUserModelValue(userModelParams{
		ID:       "50",
		UID:      big.NewFloat(1001),
		Username: "jdoe",
		FullName: "Jane Doe",
		Email:    "jane@example.com",
		GroupID:  big.NewFloat(100),
		Home:     "/home/jdoe",
		HomeMode: "700",
		Shell:    "/bin/bash",
		SMB:      true,
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

	if capturedMethod != "user.update" {
		t.Errorf("expected method 'user.update', got %q", capturedMethod)
	}

	if capturedID != 50 {
		t.Errorf("expected ID 50, got %d", capturedID)
	}

	if capturedUpdateData["full_name"] != "Jane Doe" {
		t.Errorf("expected full_name 'Jane Doe', got %v", capturedUpdateData["full_name"])
	}
	if capturedUpdateData["shell"] != "/bin/bash" {
		t.Errorf("expected shell '/bin/bash', got %v", capturedUpdateData["shell"])
	}

	// group_create and home_create should NOT be in update params
	if _, has := capturedUpdateData["group_create"]; has {
		t.Error("expected group_create to not be in update params")
	}
	if _, has := capturedUpdateData["home_create"]; has {
		t.Error("expected home_create to not be in update params")
	}
	// uid should NOT be in update params
	if _, has := capturedUpdateData["uid"]; has {
		t.Error("expected uid to not be in update params")
	}

	var resultData UserResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.FullName.ValueString() != "Jane Doe" {
		t.Errorf("expected full_name 'Jane Doe', got %q", resultData.FullName.ValueString())
	}
	if resultData.Shell.ValueString() != "/bin/bash" {
		t.Errorf("expected shell '/bin/bash', got %q", resultData.Shell.ValueString())
	}
}

func TestUserResource_Update_WithPassword(t *testing.T) {
	var capturedUpdateData map[string]any

	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "user.update" {
					args := params.([]any)
					capturedUpdateData = args[1].(map[string]any)
					return json.RawMessage(`{"id": 50}`), nil
				}
				if method == "user.query" {
					return json.RawMessage(userQueryResponse), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)

	stateValue := createUserModelValue(userModelParams{
		ID:       "50",
		UID:      big.NewFloat(1001),
		Username: "jdoe",
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "old-password",
		GroupID:  big.NewFloat(100),
		Home:     "/home/jdoe",
		HomeMode: "700",
		Shell:    "/usr/bin/zsh",
		SMB:      true,
	})

	planValue := createUserModelValue(userModelParams{
		ID:       "50",
		UID:      big.NewFloat(1001),
		Username: "jdoe",
		FullName: "John Doe",
		Email:    "john@example.com",
		Password: "new-password",
		GroupID:  big.NewFloat(100),
		Home:     "/home/jdoe",
		HomeMode: "700",
		Shell:    "/usr/bin/zsh",
		SMB:      true,
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

	if capturedUpdateData["password"] != "new-password" {
		t.Errorf("expected password 'new-password', got %v", capturedUpdateData["password"])
	}
}

func TestUserResource_Update_APIError(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)

	stateValue := createUserModelValue(userModelParams{
		ID:       "50",
		UID:      big.NewFloat(1001),
		Username: "jdoe",
		FullName: "John Doe",
		Email:    "john@example.com",
		Home:     "/home/jdoe",
		HomeMode: "700",
		Shell:    "/usr/bin/zsh",
		SMB:      true,
	})

	planValue := createUserModelValue(userModelParams{
		ID:       "50",
		UID:      big.NewFloat(1001),
		Username: "jdoe",
		FullName: "Jane Doe",
		Email:    "jane@example.com",
		Home:     "/home/jdoe",
		HomeMode: "700",
		Shell:    "/usr/bin/zsh",
		SMB:      true,
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

func TestUserResource_Delete_Success(t *testing.T) {
	var capturedMethod string
	var capturedArgs []any

	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedArgs = params.([]any)
				return json.RawMessage(`true`), nil
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	stateValue := createUserModelValue(userModelParams{
		ID:       "50",
		UID:      big.NewFloat(1001),
		Username: "jdoe",
		FullName: "John Doe",
		Email:    "john@example.com",
		Home:     "/home/jdoe",
		HomeMode: "700",
		Shell:    "/usr/bin/zsh",
		SMB:      true,
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

	if capturedMethod != "user.delete" {
		t.Errorf("expected method 'user.delete', got %q", capturedMethod)
	}

	if capturedArgs[0] != int64(50) {
		t.Errorf("expected ID 50, got %v", capturedArgs[0])
	}

	opts, ok := capturedArgs[1].(map[string]any)
	if !ok {
		t.Fatalf("expected opts to be map[string]any, got %T", capturedArgs[1])
	}
	if opts["delete_group"] != true {
		t.Errorf("expected delete_group true, got %v", opts["delete_group"])
	}
}

func TestUserResource_Delete_APIError(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("user in use")
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	stateValue := createUserModelValue(userModelParams{
		ID:       "50",
		UID:      big.NewFloat(1001),
		Username: "jdoe",
		FullName: "John Doe",
		Email:    "john@example.com",
		Home:     "/home/jdoe",
		HomeMode: "700",
		Shell:    "/usr/bin/zsh",
		SMB:      true,
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

func TestUserResource_Create_QueryError(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "user.create" {
					return json.RawMessage(`{"id": 50}`), nil
				}
				return nil, errors.New("query failed")
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	planValue := createUserModelValue(userModelParams{
		Username: "jdoe", FullName: "John Doe", Email: "", GroupCreate: true,
		Home: "/home/jdoe", HomeMode: "700", Shell: "/usr/bin/zsh", SMB: true,
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

func TestUserResource_Create_QueryNotFound(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "user.create" {
					return json.RawMessage(`{"id": 50}`), nil
				}
				return json.RawMessage(`[]`), nil
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	planValue := createUserModelValue(userModelParams{
		Username: "jdoe", FullName: "John Doe", Email: "", GroupCreate: true,
		Home: "/home/jdoe", HomeMode: "700", Shell: "/usr/bin/zsh", SMB: true,
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

func TestUserResource_Create_ParseError(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not json`), nil
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	planValue := createUserModelValue(userModelParams{
		Username: "jdoe", FullName: "John Doe", Email: "", GroupCreate: true,
		Home: "/home/jdoe", HomeMode: "700", Shell: "/usr/bin/zsh", SMB: true,
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

func TestUserResource_Update_WithAllOptionalFields(t *testing.T) {
	var capturedUpdateData map[string]any

	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "user.update" {
					args := params.([]any)
					capturedUpdateData = args[1].(map[string]any)
					return json.RawMessage(`{"id": 50}`), nil
				}
				if method == "user.query" {
					return json.RawMessage(`[{
						"id": 50, "uid": 1001, "username": "jdoe", "full_name": "John Doe",
						"email": "john@example.com", "home": "/home/jdoe", "shell": "/usr/bin/zsh",
						"home_mode": "700",
						"group": {"id": 100, "bsdgrp_gid": 1001, "bsdgrp_group": "jdoe"},
						"groups": [200, 300],
						"smb": true, "password_disabled": false,
						"ssh_password_enabled": true, "sshpubkey": "ssh-rsa key",
						"locked": false,
						"sudo_commands": ["/usr/bin/apt"],
						"sudo_commands_nopasswd": ["/usr/bin/reboot"],
						"builtin": false, "local": true, "immutable": false
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	stateValue := createUserModelValue(userModelParams{
		ID: "50", UID: big.NewFloat(1001), Username: "jdoe", FullName: "John Doe",
		Email: "john@example.com", GroupID: big.NewFloat(100),
		Groups: []int64{200, 300}, Home: "/home/jdoe", HomeMode: "700",
		Shell: "/usr/bin/zsh", SMB: true, SSHPasswordEnabled: true,
		SSHPubKey: "ssh-rsa key",
		SudoCommands: []string{"/usr/bin/apt"}, SudoCommandsNopasswd: []string{"/usr/bin/reboot"},
	})
	planValue := createUserModelValue(userModelParams{
		ID: "50", UID: big.NewFloat(1001), Username: "jdoe", FullName: "John Doe",
		Email: "john@example.com", GroupID: big.NewFloat(100),
		Groups: []int64{200, 300}, Home: "/home/jdoe", HomeMode: "700",
		Shell: "/usr/bin/zsh", SMB: true, SSHPasswordEnabled: true,
		SSHPubKey: "ssh-rsa key",
		SudoCommands: []string{"/usr/bin/apt"}, SudoCommandsNopasswd: []string{"/usr/bin/reboot"},
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

	// Verify optional fields are in update params
	if capturedUpdateData["sshpubkey"] != "ssh-rsa key" {
		t.Errorf("expected sshpubkey in update params, got %v", capturedUpdateData["sshpubkey"])
	}
	if capturedUpdateData["group"] != int64(100) {
		t.Errorf("expected group 100, got %v", capturedUpdateData["group"])
	}
	sudoCmds, ok := capturedUpdateData["sudo_commands"].([]string)
	if !ok || len(sudoCmds) != 1 {
		t.Errorf("expected sudo_commands in update params, got %v", capturedUpdateData["sudo_commands"])
	}
	sudoNP, ok := capturedUpdateData["sudo_commands_nopasswd"].([]string)
	if !ok || len(sudoNP) != 1 {
		t.Errorf("expected sudo_commands_nopasswd in update params, got %v", capturedUpdateData["sudo_commands_nopasswd"])
	}
	groups, ok := capturedUpdateData["groups"].([]int64)
	if !ok || len(groups) != 2 {
		t.Errorf("expected groups in update params, got %v", capturedUpdateData["groups"])
	}
}

func TestUserResource_Update_QueryError(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "user.update" {
					return json.RawMessage(`{"id": 50}`), nil
				}
				return nil, errors.New("query failed")
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	stateValue := createUserModelValue(userModelParams{
		ID: "50", UID: big.NewFloat(1001), Username: "jdoe", FullName: "John Doe",
		Email: "", Home: "/home/jdoe", HomeMode: "700", Shell: "/usr/bin/zsh", SMB: true,
	})
	planValue := createUserModelValue(userModelParams{
		ID: "50", UID: big.NewFloat(1001), Username: "jdoe", FullName: "Jane Doe",
		Email: "", Home: "/home/jdoe", HomeMode: "700", Shell: "/usr/bin/zsh", SMB: true,
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

func TestUserResource_Update_QueryNotFound(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "user.update" {
					return json.RawMessage(`{"id": 50}`), nil
				}
				return json.RawMessage(`[]`), nil
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	stateValue := createUserModelValue(userModelParams{
		ID: "50", UID: big.NewFloat(1001), Username: "jdoe", FullName: "John Doe",
		Email: "", Home: "/home/jdoe", HomeMode: "700", Shell: "/usr/bin/zsh", SMB: true,
	})
	planValue := createUserModelValue(userModelParams{
		ID: "50", UID: big.NewFloat(1001), Username: "jdoe", FullName: "Jane Doe",
		Email: "", Home: "/home/jdoe", HomeMode: "700", Shell: "/usr/bin/zsh", SMB: true,
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

func TestUserResource_MapUserToModel_NullEmail(t *testing.T) {
	user := &api.UserResponse{
		ID: 50, UID: 1001, Username: "jdoe", FullName: "John Doe",
		Email: nil, Home: "/home/jdoe", Shell: "/usr/bin/zsh", HomeMode: "700",
		Group: api.UserGroupResponse{ID: 100}, SMB: true,
	}
	data := &UserResourceModel{}
	mapUserToModel(context.Background(), user, data)

	if data.Email.ValueString() != "" {
		t.Errorf("expected empty email for nil, got %q", data.Email.ValueString())
	}
}

func TestUserResource_MapUserToModel_WithSSHPubKey(t *testing.T) {
	key := "ssh-rsa AAAA..."
	user := &api.UserResponse{
		ID: 50, UID: 1001, Username: "jdoe", FullName: "John Doe",
		Email: nil, Home: "/home/jdoe", Shell: "/usr/bin/zsh", HomeMode: "700",
		Group: api.UserGroupResponse{ID: 100}, SMB: true,
		SSHPubKey: &key,
	}
	data := &UserResourceModel{}
	mapUserToModel(context.Background(), user, data)

	if data.SSHPubKey.ValueString() != "ssh-rsa AAAA..." {
		t.Errorf("expected sshpubkey to be set, got %q", data.SSHPubKey.ValueString())
	}
}

func TestUserResource_MapUserToModel_GroupsFromAPI(t *testing.T) {
	user := &api.UserResponse{
		ID: 50, UID: 1001, Username: "jdoe", FullName: "John Doe",
		Email: nil, Home: "/home/jdoe", Shell: "/usr/bin/zsh", HomeMode: "700",
		Group: api.UserGroupResponse{ID: 100}, SMB: true,
		Groups:               []int64{200, 300},
		SudoCommands:         []string{"/usr/bin/apt"},
		SudoCommandsNopasswd: []string{"/usr/bin/reboot"},
	}
	data := &UserResourceModel{}
	mapUserToModel(context.Background(), user, data)

	// When data.Groups is null but API returns non-empty, it should be set
	if data.Groups.IsNull() {
		t.Error("expected groups to be set from API when data was null and API has values")
	}
	if data.SudoCommands.IsNull() {
		t.Error("expected sudo_commands to be set from API")
	}
	if data.SudoCommandsNopasswd.IsNull() {
		t.Error("expected sudo_commands_nopasswd to be set from API")
	}
}
