package resources

import (
	"context"
	"errors"
	"math/big"
	"testing"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

var testUser = &truenas.User{
	ID:       50,
	UID:      1001,
	Username: "jdoe",
	FullName: "John Doe",
	Email:    "john@example.com",
	Home:     "/home/jdoe",
	Shell:    "/usr/bin/zsh",
	HomeMode: "700",
	GroupID:  100,
	SMB:      true,
}

func TestUserResource_Create_Success(t *testing.T) {
	var capturedOpts truenas.CreateUserOpts

	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateUserOpts) (*truenas.User, error) {
					capturedOpts = opts
					return testUser, nil
				},
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

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if capturedOpts.Username != "jdoe" {
		t.Errorf("expected username 'jdoe', got %q", capturedOpts.Username)
	}
	if capturedOpts.FullName != "John Doe" {
		t.Errorf("expected full_name 'John Doe', got %q", capturedOpts.FullName)
	}
	if !capturedOpts.GroupCreate {
		t.Error("expected group_create true")
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
	var capturedOpts truenas.CreateUserOpts

	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateUserOpts) (*truenas.User, error) {
					capturedOpts = opts
					return testUser, nil
				},
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	planValue := createUserModelValue(userModelParams{
		Username:    "jdoe",
		FullName:    "John Doe",
		Email:       "john@example.com",
		Password:    "s3cret!",
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

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if capturedOpts.Password != "s3cret!" {
		t.Errorf("expected password 's3cret!', got %q", capturedOpts.Password)
	}
}

func TestUserResource_Create_WithGroupID(t *testing.T) {
	var capturedOpts truenas.CreateUserOpts

	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateUserOpts) (*truenas.User, error) {
					capturedOpts = opts
					return testUser, nil
				},
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

	if capturedOpts.Group != 200 {
		t.Errorf("expected group 200, got %d", capturedOpts.Group)
	}
	if capturedOpts.GroupCreate {
		t.Error("expected group_create to be false when not set")
	}
}

func TestUserResource_Create_WithAllOptions(t *testing.T) {
	var capturedOpts truenas.CreateUserOpts

	fullUser := &truenas.User{
		ID: 53, UID: 2000, Username: "admin", FullName: "Admin User",
		Email: "admin@example.com", Home: "/home/admin", Shell: "/bin/bash",
		HomeMode: "755", GroupID: 300, Groups: []int64{100, 200},
		SMB: false, SSHPasswordEnabled: true, SSHPubKey: "ssh-rsa AAAA...",
		SudoCommands: []string{"/usr/bin/apt"}, SudoCommandsNopasswd: []string{"/usr/bin/reboot"},
	}

	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateUserOpts) (*truenas.User, error) {
					capturedOpts = opts
					return fullUser, nil
				},
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

	if capturedOpts.UID != 2000 {
		t.Errorf("expected uid 2000, got %d", capturedOpts.UID)
	}
	if !capturedOpts.HomeCreate {
		t.Error("expected home_create true")
	}
	if capturedOpts.SSHPubKey != "ssh-rsa AAAA..." {
		t.Errorf("expected sshpubkey, got %q", capturedOpts.SSHPubKey)
	}
	if !capturedOpts.SSHPasswordEnabled {
		t.Error("expected ssh_password_enabled true")
	}
	if len(capturedOpts.Groups) != 2 || capturedOpts.Groups[0] != 100 || capturedOpts.Groups[1] != 200 {
		t.Errorf("unexpected groups: %v", capturedOpts.Groups)
	}
	if len(capturedOpts.SudoCommands) != 1 || capturedOpts.SudoCommands[0] != "/usr/bin/apt" {
		t.Errorf("unexpected sudo_commands: %v", capturedOpts.SudoCommands)
	}

	// Verify state
	var resultData UserResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.UID.ValueInt64() != 2000 {
		t.Errorf("expected UID 2000, got %d", resultData.UID.ValueInt64())
	}
	if !resultData.SSHPasswordEnabled.ValueBool() {
		t.Error("expected ssh_password_enabled true in state")
	}
	if resultData.SSHPubKey.ValueString() != "ssh-rsa AAAA..." {
		t.Errorf("expected sshpubkey 'ssh-rsa AAAA...', got %q", resultData.SSHPubKey.ValueString())
	}
}

func TestUserResource_Create_ServiceError(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateUserOpts) (*truenas.User, error) {
					return nil, errors.New("connection refused")
				},
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
		t.Fatal("expected error for service error")
	}

	if !resp.State.Raw.IsNull() {
		t.Error("expected state to not be set when service returns error")
	}
}

func TestUserResource_Create_NotFound(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateUserOpts) (*truenas.User, error) {
					return nil, nil
				},
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
		t.Fatal("expected error when service returns nil user after create")
	}
}

func TestUserResource_Read_Success(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				GetFunc: func(ctx context.Context, id int64) (*truenas.User, error) {
					return testUser, nil
				},
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
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				GetFunc: func(ctx context.Context, id int64) (*truenas.User, error) {
					return nil, nil
				},
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

func TestUserResource_Read_ServiceError(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				GetFunc: func(ctx context.Context, id int64) (*truenas.User, error) {
					return nil, errors.New("connection refused")
				},
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
		t.Fatal("expected error for service error")
	}
}

func TestUserResource_Read_PreservesPassword(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				GetFunc: func(ctx context.Context, id int64) (*truenas.User, error) {
					return testUser, nil
				},
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
	var capturedID int64
	var capturedOpts truenas.UpdateUserOpts

	updatedUser := &truenas.User{
		ID: 50, UID: 1001, Username: "jdoe", FullName: "Jane Doe",
		Email: "jane@example.com", Home: "/home/jdoe", Shell: "/bin/bash",
		HomeMode: "700", GroupID: 100, SMB: true,
	}

	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				UpdateFunc: func(ctx context.Context, id int64, opts truenas.UpdateUserOpts) (*truenas.User, error) {
					capturedID = id
					capturedOpts = opts
					return updatedUser, nil
				},
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

	if capturedID != 50 {
		t.Errorf("expected ID 50, got %d", capturedID)
	}

	if capturedOpts.FullName != "Jane Doe" {
		t.Errorf("expected full_name 'Jane Doe', got %q", capturedOpts.FullName)
	}
	if capturedOpts.Shell != "/bin/bash" {
		t.Errorf("expected shell '/bin/bash', got %q", capturedOpts.Shell)
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
	var capturedOpts truenas.UpdateUserOpts

	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				UpdateFunc: func(ctx context.Context, id int64, opts truenas.UpdateUserOpts) (*truenas.User, error) {
					capturedOpts = opts
					return testUser, nil
				},
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

	if capturedOpts.Password != "new-password" {
		t.Errorf("expected password 'new-password', got %q", capturedOpts.Password)
	}
}

func TestUserResource_Update_ServiceError(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				UpdateFunc: func(ctx context.Context, id int64, opts truenas.UpdateUserOpts) (*truenas.User, error) {
					return nil, errors.New("connection refused")
				},
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
		t.Fatal("expected error for service error")
	}
}

func TestUserResource_Update_NotFound(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				UpdateFunc: func(ctx context.Context, id int64, opts truenas.UpdateUserOpts) (*truenas.User, error) {
					return nil, nil
				},
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
		t.Fatal("expected error when service returns nil user after update")
	}
}

func TestUserResource_Delete_Success(t *testing.T) {
	var capturedID int64
	var capturedDeleteGroup bool

	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				DeleteFunc: func(ctx context.Context, id int64, deleteGroup bool) error {
					capturedID = id
					capturedDeleteGroup = deleteGroup
					return nil
				},
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

	if capturedID != 50 {
		t.Errorf("expected ID 50, got %d", capturedID)
	}
	if capturedDeleteGroup {
		t.Error("expected deleteGroup false when group_create not set in state")
	}
}

func TestUserResource_Delete_WithGroupCreate(t *testing.T) {
	var capturedDeleteGroup bool

	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				DeleteFunc: func(ctx context.Context, id int64, deleteGroup bool) error {
					capturedDeleteGroup = deleteGroup
					return nil
				},
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	stateValue := createUserModelValue(userModelParams{
		ID:          "50",
		UID:         big.NewFloat(1001),
		Username:    "jdoe",
		FullName:    "John Doe",
		Email:       "john@example.com",
		GroupCreate: true,
		Home:        "/home/jdoe",
		HomeMode:    "700",
		Shell:       "/usr/bin/zsh",
		SMB:         true,
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

	if !capturedDeleteGroup {
		t.Error("expected deleteGroup true when group_create=true in state")
	}
}

func TestUserResource_Delete_ServiceError(t *testing.T) {
	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				DeleteFunc: func(ctx context.Context, id int64, deleteGroup bool) error {
					return errors.New("user in use")
				},
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
		t.Fatal("expected error for service error")
	}
}

func TestUserResource_Update_WithAllOptionalFields(t *testing.T) {
	var capturedOpts truenas.UpdateUserOpts

	r := &UserResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			User: &truenas.MockUserService{
				UpdateFunc: func(ctx context.Context, id int64, opts truenas.UpdateUserOpts) (*truenas.User, error) {
					capturedOpts = opts
					return &truenas.User{
						ID: 50, UID: 1001, Username: "jdoe", FullName: "John Doe",
						Email: "john@example.com", Home: "/home/jdoe", Shell: "/usr/bin/zsh",
						HomeMode: "700", GroupID: 100, Groups: []int64{200, 300},
						SMB: true, SSHPasswordEnabled: true, SSHPubKey: "ssh-rsa key",
						SudoCommands: []string{"/usr/bin/apt"}, SudoCommandsNopasswd: []string{"/usr/bin/reboot"},
					}, nil
				},
			},
		}},
	}

	schemaResp := getUserResourceSchema(t)
	stateValue := createUserModelValue(userModelParams{
		ID: "50", UID: big.NewFloat(1001), Username: "jdoe", FullName: "John Doe",
		Email: "john@example.com", GroupID: big.NewFloat(100),
		Groups: []int64{200, 300}, Home: "/home/jdoe", HomeMode: "700",
		Shell: "/usr/bin/zsh", SMB: true, SSHPasswordEnabled: true,
		SSHPubKey:            "ssh-rsa key",
		SudoCommands:         []string{"/usr/bin/apt"},
		SudoCommandsNopasswd: []string{"/usr/bin/reboot"},
	})
	planValue := createUserModelValue(userModelParams{
		ID: "50", UID: big.NewFloat(1001), Username: "jdoe", FullName: "John Doe",
		Email: "john@example.com", GroupID: big.NewFloat(100),
		Groups: []int64{200, 300}, Home: "/home/jdoe", HomeMode: "700",
		Shell: "/usr/bin/zsh", SMB: true, SSHPasswordEnabled: true,
		SSHPubKey:            "ssh-rsa key",
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

	if capturedOpts.SSHPubKey != "ssh-rsa key" {
		t.Errorf("expected sshpubkey in update opts, got %q", capturedOpts.SSHPubKey)
	}
	if capturedOpts.Group != 100 {
		t.Errorf("expected group 100, got %d", capturedOpts.Group)
	}
	if len(capturedOpts.SudoCommands) != 1 || capturedOpts.SudoCommands[0] != "/usr/bin/apt" {
		t.Errorf("expected sudo_commands in update opts, got %v", capturedOpts.SudoCommands)
	}
	if len(capturedOpts.SudoCommandsNopasswd) != 1 || capturedOpts.SudoCommandsNopasswd[0] != "/usr/bin/reboot" {
		t.Errorf("expected sudo_commands_nopasswd in update opts, got %v", capturedOpts.SudoCommandsNopasswd)
	}
	if len(capturedOpts.Groups) != 2 {
		t.Errorf("expected groups in update opts, got %v", capturedOpts.Groups)
	}
}

func TestUserResource_MapUserToModel_EmptyEmail(t *testing.T) {
	user := &truenas.User{
		ID: 50, UID: 1001, Username: "jdoe", FullName: "John Doe",
		Email: "", Home: "/home/jdoe", Shell: "/usr/bin/zsh", HomeMode: "700",
		GroupID: 100, SMB: true,
	}
	data := &UserResourceModel{}
	mapUserToModel(context.Background(), user, data)

	if data.Email.ValueString() != "" {
		t.Errorf("expected empty email, got %q", data.Email.ValueString())
	}
}

func TestUserResource_MapUserToModel_WithSSHPubKey(t *testing.T) {
	user := &truenas.User{
		ID: 50, UID: 1001, Username: "jdoe", FullName: "John Doe",
		Email: "", Home: "/home/jdoe", Shell: "/usr/bin/zsh", HomeMode: "700",
		GroupID: 100, SMB: true, SSHPubKey: "ssh-rsa AAAA...",
	}
	data := &UserResourceModel{}
	mapUserToModel(context.Background(), user, data)

	if data.SSHPubKey.ValueString() != "ssh-rsa AAAA..." {
		t.Errorf("expected sshpubkey to be set, got %q", data.SSHPubKey.ValueString())
	}
}

func TestUserResource_MapUserToModel_GroupsFromAPI(t *testing.T) {
	user := &truenas.User{
		ID: 50, UID: 1001, Username: "jdoe", FullName: "John Doe",
		Email: "", Home: "/home/jdoe", Shell: "/usr/bin/zsh", HomeMode: "700",
		GroupID:              100,
		SMB:                  true,
		Groups:               []int64{200, 300},
		SudoCommands:         []string{"/usr/bin/apt"},
		SudoCommandsNopasswd: []string{"/usr/bin/reboot"},
	}

	// When data fields are null (not set in config), they must stay null even
	// if the API returns values — overwriting would cause a plan inconsistency.
	data := &UserResourceModel{}
	mapUserToModel(context.Background(), user, data)

	if !data.Groups.IsNull() {
		t.Error("expected groups to stay null when not set in config")
	}
	if !data.SudoCommands.IsNull() {
		t.Error("expected sudo_commands to stay null when not set in config")
	}
	if !data.SudoCommandsNopasswd.IsNull() {
		t.Error("expected sudo_commands_nopasswd to stay null when not set in config")
	}

	// When data fields are already non-null (user set them in config), they
	// must be refreshed from the API response.
	data2 := &UserResourceModel{}
	data2.Groups, _ = types.ListValueFrom(context.Background(), types.Int64Type, []int64{})
	data2.SudoCommands, _ = types.ListValueFrom(context.Background(), types.StringType, []string{})
	data2.SudoCommandsNopasswd, _ = types.ListValueFrom(context.Background(), types.StringType, []string{})
	mapUserToModel(context.Background(), user, data2)

	if data2.Groups.IsNull() {
		t.Error("expected groups to be refreshed from API when already set in config")
	}
	if data2.SudoCommands.IsNull() {
		t.Error("expected sudo_commands to be refreshed from API when already set in config")
	}
	if data2.SudoCommandsNopasswd.IsNull() {
		t.Error("expected sudo_commands_nopasswd to be refreshed from API when already set in config")
	}
}
