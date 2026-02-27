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

func TestNewUserDataSource(t *testing.T) {
	ds := NewUserDataSource()
	if ds == nil {
		t.Fatal("expected non-nil data source")
	}

	_ = datasource.DataSource(ds)
	_ = datasource.DataSourceWithConfigure(ds.(*UserDataSource))
}

func TestUserDataSource_Metadata(t *testing.T) {
	ds := NewUserDataSource()

	req := datasource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_user" {
		t.Errorf("expected TypeName 'truenas_user', got %q", resp.TypeName)
	}
}

func TestUserDataSource_Schema(t *testing.T) {
	ds := NewUserDataSource()

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), req, resp)

	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	for _, name := range []string{
		"id", "uid", "username", "full_name", "email", "home", "shell",
		"group_id", "groups", "smb", "password_disabled", "ssh_password_enabled",
		"sshpubkey", "locked", "sudo_commands", "sudo_commands_nopasswd",
		"builtin", "local",
	} {
		if attrs[name] == nil {
			t.Errorf("expected '%s' attribute", name)
		}
	}

	if !attrs["username"].IsRequired() {
		t.Error("expected 'username' attribute to be required")
	}
	if !attrs["id"].IsComputed() {
		t.Error("expected 'id' attribute to be computed")
	}
	if !attrs["uid"].IsComputed() {
		t.Error("expected 'uid' attribute to be computed")
	}
}

func TestUserDataSource_Configure_Success(t *testing.T) {
	ds := NewUserDataSource().(*UserDataSource)

	req := datasource.ConfigureRequest{ProviderData: &client.MockClient{}}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestUserDataSource_Configure_NilProviderData(t *testing.T) {
	ds := NewUserDataSource().(*UserDataSource)

	req := datasource.ConfigureRequest{ProviderData: nil}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestUserDataSource_Configure_WrongType(t *testing.T) {
	ds := NewUserDataSource().(*UserDataSource)

	req := datasource.ConfigureRequest{ProviderData: "not a client"}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

func createUserReadRequest(t *testing.T, username string) (datasource.ReadRequest, datasource.SchemaResponse) {
	t.Helper()

	ds := NewUserDataSource()
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                     tftypes.String,
			"uid":                    tftypes.Number,
			"username":               tftypes.String,
			"full_name":              tftypes.String,
			"email":                  tftypes.String,
			"home":                   tftypes.String,
			"shell":                  tftypes.String,
			"group_id":               tftypes.Number,
			"groups":                 tftypes.List{ElementType: tftypes.Number},
			"smb":                    tftypes.Bool,
			"password_disabled":      tftypes.Bool,
			"ssh_password_enabled":   tftypes.Bool,
			"sshpubkey":              tftypes.String,
			"locked":                 tftypes.Bool,
			"sudo_commands":          tftypes.List{ElementType: tftypes.String},
			"sudo_commands_nopasswd": tftypes.List{ElementType: tftypes.String},
			"builtin":                tftypes.Bool,
			"local":                  tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":                     tftypes.NewValue(tftypes.String, nil),
		"uid":                    tftypes.NewValue(tftypes.Number, nil),
		"username":               tftypes.NewValue(tftypes.String, username),
		"full_name":              tftypes.NewValue(tftypes.String, nil),
		"email":                  tftypes.NewValue(tftypes.String, nil),
		"home":                   tftypes.NewValue(tftypes.String, nil),
		"shell":                  tftypes.NewValue(tftypes.String, nil),
		"group_id":               tftypes.NewValue(tftypes.Number, nil),
		"groups":                 tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, nil),
		"smb":                    tftypes.NewValue(tftypes.Bool, nil),
		"password_disabled":      tftypes.NewValue(tftypes.Bool, nil),
		"ssh_password_enabled":   tftypes.NewValue(tftypes.Bool, nil),
		"sshpubkey":              tftypes.NewValue(tftypes.String, nil),
		"locked":                 tftypes.NewValue(tftypes.Bool, nil),
		"sudo_commands":          tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"sudo_commands_nopasswd": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"builtin":                tftypes.NewValue(tftypes.Bool, nil),
		"local":                  tftypes.NewValue(tftypes.Bool, nil),
	})

	return datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configValue},
	}, *schemaResp
}

func TestUserDataSource_Read_Success(t *testing.T) {
	ds := &UserDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": 1,
					"uid": 0,
					"username": "root",
					"full_name": "root",
					"email": "admin@example.com",
					"home": "/root",
					"shell": "/usr/bin/zsh",
					"home_mode": "755",
					"group": {"id": 1, "bsdgrp_gid": 0, "bsdgrp_group": "wheel"},
					"groups": [40, 50],
					"smb": false,
					"password_disabled": true,
					"ssh_password_enabled": false,
					"sshpubkey": "ssh-ed25519 AAAA...",
					"locked": false,
					"sudo_commands": [],
					"sudo_commands_nopasswd": ["ALL"],
					"builtin": true,
					"local": true,
					"immutable": true
				}]`), nil
			},
		},
	}

	req, schemaResp := createUserReadRequest(t, "root")
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model UserDataSourceModel
	resp.State.Get(context.Background(), &model)

	if model.ID.ValueString() != "1" {
		t.Errorf("expected ID '1', got %q", model.ID.ValueString())
	}
	if model.UID.ValueInt64() != 0 {
		t.Errorf("expected UID 0, got %d", model.UID.ValueInt64())
	}
	if model.Username.ValueString() != "root" {
		t.Errorf("expected username 'root', got %q", model.Username.ValueString())
	}
	if model.FullName.ValueString() != "root" {
		t.Errorf("expected full_name 'root', got %q", model.FullName.ValueString())
	}
	if model.Email.ValueString() != "admin@example.com" {
		t.Errorf("expected email 'admin@example.com', got %q", model.Email.ValueString())
	}
	if model.Home.ValueString() != "/root" {
		t.Errorf("expected home '/root', got %q", model.Home.ValueString())
	}
	if model.Shell.ValueString() != "/usr/bin/zsh" {
		t.Errorf("expected shell '/usr/bin/zsh', got %q", model.Shell.ValueString())
	}
	if model.GroupID.ValueInt64() != 1 {
		t.Errorf("expected group_id 1, got %d", model.GroupID.ValueInt64())
	}
	if model.SMB.ValueBool() != false {
		t.Errorf("expected smb false, got %v", model.SMB.ValueBool())
	}
	if model.PasswordDisabled.ValueBool() != true {
		t.Errorf("expected password_disabled true, got %v", model.PasswordDisabled.ValueBool())
	}
	if model.SSHPubKey.ValueString() != "ssh-ed25519 AAAA..." {
		t.Errorf("expected sshpubkey 'ssh-ed25519 AAAA...', got %q", model.SSHPubKey.ValueString())
	}
	if model.Locked.ValueBool() != false {
		t.Errorf("expected locked false, got %v", model.Locked.ValueBool())
	}
	if model.Builtin.ValueBool() != true {
		t.Errorf("expected builtin true, got %v", model.Builtin.ValueBool())
	}
	if model.Local.ValueBool() != true {
		t.Errorf("expected local true, got %v", model.Local.ValueBool())
	}
}

func TestUserDataSource_Read_NullEmail(t *testing.T) {
	ds := &UserDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": 5, "uid": 1000, "username": "testuser",
					"full_name": "Test User", "email": null,
					"home": "/var/empty", "shell": "/usr/bin/zsh", "home_mode": "",
					"group": {"id": 10, "bsdgrp_gid": 1000, "bsdgrp_group": "testuser"},
					"groups": [], "smb": false, "password_disabled": false,
					"ssh_password_enabled": false, "sshpubkey": null,
					"locked": false, "sudo_commands": [], "sudo_commands_nopasswd": [],
					"builtin": false, "local": true, "immutable": false
				}]`), nil
			},
		},
	}

	req, schemaResp := createUserReadRequest(t, "testuser")
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model UserDataSourceModel
	resp.State.Get(context.Background(), &model)

	if model.Email.ValueString() != "" {
		t.Errorf("expected email '', got %q", model.Email.ValueString())
	}
	if model.SSHPubKey.ValueString() != "" {
		t.Errorf("expected sshpubkey '', got %q", model.SSHPubKey.ValueString())
	}
}

func TestUserDataSource_Read_NotFound(t *testing.T) {
	ds := &UserDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		},
	}

	req, schemaResp := createUserReadRequest(t, "nonexistent")
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for user not found")
	}
}

func TestUserDataSource_Read_APIError(t *testing.T) {
	ds := &UserDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection failed")
			},
		},
	}

	req, schemaResp := createUserReadRequest(t, "root")
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestUserDataSource_Read_InvalidJSON(t *testing.T) {
	ds := &UserDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		},
	}

	req, schemaResp := createUserReadRequest(t, "root")
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestUserDataSource_Read_VerifyFilterParams(t *testing.T) {
	var capturedParams any

	ds := &UserDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`[{
					"id": 1, "uid": 0, "username": "root", "full_name": "root",
					"email": null, "home": "/root", "shell": "/usr/bin/zsh", "home_mode": "755",
					"group": {"id": 1, "bsdgrp_gid": 0, "bsdgrp_group": "wheel"},
					"groups": [], "smb": false, "password_disabled": true,
					"ssh_password_enabled": false, "sshpubkey": null, "locked": false,
					"sudo_commands": [], "sudo_commands_nopasswd": [],
					"builtin": true, "local": true, "immutable": true
				}]`), nil
			},
		},
	}

	req, schemaResp := createUserReadRequest(t, "root")
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

	if filters[0][0] != "username" || filters[0][1] != "=" || filters[0][2] != "root" {
		t.Errorf("expected filter ['username', '=', 'root'], got %v", filters[0])
	}
}
