package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/deevus/terraform-provider-truenas/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &UserResource{}
	_ resource.ResourceWithConfigure   = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
)

// UserResourceModel describes the resource data model.
type UserResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	UID                  types.Int64  `tfsdk:"uid"`
	Username             types.String `tfsdk:"username"`
	FullName             types.String `tfsdk:"full_name"`
	Email                types.String `tfsdk:"email"`
	Password             types.String `tfsdk:"password"`
	PasswordDisabled     types.Bool   `tfsdk:"password_disabled"`
	GroupID              types.Int64  `tfsdk:"group_id"`
	GroupCreate          types.Bool   `tfsdk:"group_create"`
	Groups               types.List   `tfsdk:"groups"`
	Home                 types.String `tfsdk:"home"`
	HomeCreate           types.Bool   `tfsdk:"home_create"`
	HomeMode             types.String `tfsdk:"home_mode"`
	Shell                types.String `tfsdk:"shell"`
	SMB                  types.Bool   `tfsdk:"smb"`
	SSHPasswordEnabled   types.Bool   `tfsdk:"ssh_password_enabled"`
	SSHPubKey            types.String `tfsdk:"sshpubkey"`
	Locked               types.Bool   `tfsdk:"locked"`
	SudoCommands         types.List   `tfsdk:"sudo_commands"`
	SudoCommandsNopasswd types.List   `tfsdk:"sudo_commands_nopasswd"`
	Builtin              types.Bool   `tfsdk:"builtin"`
}

// UserResource defines the resource implementation.
type UserResource struct {
	BaseResource
}

// NewUserResource creates a new UserResource.
func NewUserResource() resource.Resource {
	return &UserResource{}
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages local users on TrueNAS.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "User ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uid": schema.Int64Attribute{
				Description: "UNIX user ID. If not specified, TrueNAS assigns the next available UID.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Description: "Login username.",
				Required:    true,
			},
			"full_name": schema.StringAttribute{
				Description: "Full name (GECOS field).",
				Required:    true,
			},
			"email": schema.StringAttribute{
				Description: "Email address.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"password": schema.StringAttribute{
				Description: "User password.",
				Optional:    true,
				Sensitive:   true,
			},
			"password_disabled": schema.BoolAttribute{
				Description: "Disable password login.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"group_id": schema.Int64Attribute{
				Description: "Primary group ID.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"group_create": schema.BoolAttribute{
				Description: "Create a new primary group with the same name as the user. Only used during creation.",
				Optional:    true,
			},
			"groups": schema.ListAttribute{
				Description: "List of secondary group IDs.",
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"home": schema.StringAttribute{
				Description: "Home directory path.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("/var/empty"),
			},
			"home_create": schema.BoolAttribute{
				Description: "Create the home directory if it does not exist. Only used during creation.",
				Optional:    true,
			},
			"home_mode": schema.StringAttribute{
				Description: "Home directory permissions (octal).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("700"),
			},
			"shell": schema.StringAttribute{
				Description: "Login shell path.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("/usr/bin/zsh"),
			},
			"smb": schema.BoolAttribute{
				Description: "Allow user for SMB authentication.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"ssh_password_enabled": schema.BoolAttribute{
				Description: "Allow SSH password authentication.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"sshpubkey": schema.StringAttribute{
				Description: "SSH public key.",
				Optional:    true,
			},
			"locked": schema.BoolAttribute{
				Description: "Lock user account.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"sudo_commands": schema.ListAttribute{
				Description: "List of allowed sudo commands.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"sudo_commands_nopasswd": schema.ListAttribute{
				Description: "List of allowed sudo commands without password.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"builtin": schema.BoolAttribute{
				Description: "Whether this is a built-in system user.",
				Computed:    true,
			},
		},
	}
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := buildUserCreateParams(ctx, &data)

	result, err := r.client.Call(ctx, "user.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create User",
			fmt.Sprintf("Unable to create user: %s", err.Error()),
		)
		return
	}

	var createResp struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(result, &createResp); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Response",
			fmt.Sprintf("Unable to parse create response: %s", err.Error()),
		)
		return
	}

	user, err := r.queryUser(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read User",
			fmt.Sprintf("User created but unable to read: %s", err.Error()),
		)
		return
	}

	if user == nil {
		resp.Diagnostics.AddError(
			"User Not Found",
			"User was created but could not be found.",
		)
		return
	}

	mapUserToModel(ctx, user, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	user, err := r.queryUser(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read User",
			fmt.Sprintf("Unable to query user: %s", err.Error()),
		)
		return
	}

	if user == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	mapUserToModel(ctx, user, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state UserResourceModel
	var plan UserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	params := buildUserUpdateParams(ctx, &plan)

	_, err = r.client.Call(ctx, "user.update", []any{id, params})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update User",
			fmt.Sprintf("Unable to update user: %s", err.Error()),
		)
		return
	}

	user, err := r.queryUser(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read User",
			fmt.Sprintf("User updated but unable to read: %s", err.Error()),
		)
		return
	}

	if user == nil {
		resp.Diagnostics.AddError(
			"User Not Found",
			"User was updated but could not be found.",
		)
		return
	}

	mapUserToModel(ctx, user, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	_, err = r.client.Call(ctx, "user.delete", []any{id, map[string]any{"delete_group": true}})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete User",
			fmt.Sprintf("Unable to delete user: %s", err.Error()),
		)
		return
	}
}

// ImportState imports a user by UID.
func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	uid, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric UID, got %q", req.ID),
		)
		return
	}

	user, err := r.queryUserByField(ctx, "uid", uid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Import User", err.Error())
		return
	}

	if user == nil {
		resp.Diagnostics.AddError(
			"User Not Found",
			fmt.Sprintf("No user found with UID %d", uid),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.FormatInt(user.ID, 10))...)
}

// queryUser queries a user by internal ID and returns the response.
func (r *UserResource) queryUser(ctx context.Context, id int64) (*api.UserResponse, error) {
	return r.queryUserByField(ctx, "id", id)
}

// queryUserByField queries a user by an arbitrary field and returns the response.
func (r *UserResource) queryUserByField(ctx context.Context, field string, value int64) (*api.UserResponse, error) {
	filter := [][]any{{field, "=", value}}
	result, err := r.client.Call(ctx, "user.query", filter)
	if err != nil {
		return nil, err
	}

	var users []api.UserResponse
	if err := json.Unmarshal(result, &users); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(users) == 0 {
		return nil, nil
	}

	return &users[0], nil
}

// buildUserCreateParams builds the API create params from the resource model.
func buildUserCreateParams(ctx context.Context, data *UserResourceModel) map[string]any {
	params := map[string]any{
		"username":            data.Username.ValueString(),
		"full_name":           data.FullName.ValueString(),
		"email":               data.Email.ValueString(),
		"password_disabled":   data.PasswordDisabled.ValueBool(),
		"home":                data.Home.ValueString(),
		"home_mode":           data.HomeMode.ValueString(),
		"shell":               data.Shell.ValueString(),
		"smb":                 data.SMB.ValueBool(),
		"ssh_password_enabled": data.SSHPasswordEnabled.ValueBool(),
		"locked":              data.Locked.ValueBool(),
	}

	if !data.UID.IsNull() && !data.UID.IsUnknown() {
		params["uid"] = data.UID.ValueInt64()
	}

	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		params["password"] = data.Password.ValueString()
	}

	if !data.GroupID.IsNull() && !data.GroupID.IsUnknown() {
		params["group"] = data.GroupID.ValueInt64()
	}

	if !data.GroupCreate.IsNull() && !data.GroupCreate.IsUnknown() {
		params["group_create"] = data.GroupCreate.ValueBool()
	}

	if !data.Groups.IsNull() && !data.Groups.IsUnknown() {
		var items []int64
		data.Groups.ElementsAs(ctx, &items, false)
		params["groups"] = items
	}

	if !data.HomeCreate.IsNull() && !data.HomeCreate.IsUnknown() {
		params["home_create"] = data.HomeCreate.ValueBool()
	}

	if !data.SSHPubKey.IsNull() && !data.SSHPubKey.IsUnknown() {
		params["sshpubkey"] = data.SSHPubKey.ValueString()
	}

	if !data.SudoCommands.IsNull() && !data.SudoCommands.IsUnknown() {
		var items []string
		data.SudoCommands.ElementsAs(ctx, &items, false)
		params["sudo_commands"] = items
	}

	if !data.SudoCommandsNopasswd.IsNull() && !data.SudoCommandsNopasswd.IsUnknown() {
		var items []string
		data.SudoCommandsNopasswd.ElementsAs(ctx, &items, false)
		params["sudo_commands_nopasswd"] = items
	}

	return params
}

// buildUserUpdateParams builds the API update params (excludes uid, group_create, home_create).
func buildUserUpdateParams(ctx context.Context, data *UserResourceModel) map[string]any {
	params := map[string]any{
		"username":            data.Username.ValueString(),
		"full_name":           data.FullName.ValueString(),
		"email":               data.Email.ValueString(),
		"password_disabled":   data.PasswordDisabled.ValueBool(),
		"home":                data.Home.ValueString(),
		"home_mode":           data.HomeMode.ValueString(),
		"shell":               data.Shell.ValueString(),
		"smb":                 data.SMB.ValueBool(),
		"ssh_password_enabled": data.SSHPasswordEnabled.ValueBool(),
		"locked":              data.Locked.ValueBool(),
	}

	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		params["password"] = data.Password.ValueString()
	}

	if !data.GroupID.IsNull() && !data.GroupID.IsUnknown() {
		params["group"] = data.GroupID.ValueInt64()
	}

	if !data.Groups.IsNull() && !data.Groups.IsUnknown() {
		var items []int64
		data.Groups.ElementsAs(ctx, &items, false)
		params["groups"] = items
	}

	if !data.SSHPubKey.IsNull() && !data.SSHPubKey.IsUnknown() {
		params["sshpubkey"] = data.SSHPubKey.ValueString()
	}

	if !data.SudoCommands.IsNull() && !data.SudoCommands.IsUnknown() {
		var items []string
		data.SudoCommands.ElementsAs(ctx, &items, false)
		params["sudo_commands"] = items
	}

	if !data.SudoCommandsNopasswd.IsNull() && !data.SudoCommandsNopasswd.IsUnknown() {
		var items []string
		data.SudoCommandsNopasswd.ElementsAs(ctx, &items, false)
		params["sudo_commands_nopasswd"] = items
	}

	return params
}

// mapUserToModel maps an API response to the resource model.
func mapUserToModel(ctx context.Context, user *api.UserResponse, data *UserResourceModel) {
	data.ID = types.StringValue(strconv.FormatInt(user.ID, 10))
	data.UID = types.Int64Value(user.UID)
	data.Username = types.StringValue(user.Username)
	data.FullName = types.StringValue(user.FullName)

	if user.Email != nil {
		data.Email = types.StringValue(*user.Email)
	} else {
		data.Email = types.StringValue("")
	}

	data.PasswordDisabled = types.BoolValue(user.PasswordDisabled)
	data.GroupID = types.Int64Value(user.Group.ID)
	data.Home = types.StringValue(user.Home)
	if user.HomeMode != "" {
		data.HomeMode = types.StringValue(user.HomeMode)
	}
	data.Shell = types.StringValue(user.Shell)
	data.SMB = types.BoolValue(user.SMB)
	data.SSHPasswordEnabled = types.BoolValue(user.SSHPasswordEnabled)

	if user.SSHPubKey != nil {
		data.SSHPubKey = types.StringValue(*user.SSHPubKey)
	}
	// Leave SSHPubKey unchanged if API returns nil and user didn't configure it

	data.Locked = types.BoolValue(user.Locked)
	data.Builtin = types.BoolValue(user.Builtin)

	// Do not overwrite Password - it is write-only
	// Do not overwrite GroupCreate - it is create-only
	// Do not overwrite HomeCreate - it is create-only

	// Map groups list
	if !data.Groups.IsNull() {
		data.Groups, _ = types.ListValueFrom(ctx, types.Int64Type, user.Groups)
	} else if len(user.Groups) > 0 {
		data.Groups, _ = types.ListValueFrom(ctx, types.Int64Type, user.Groups)
	}

	if !data.SudoCommands.IsNull() {
		data.SudoCommands, _ = types.ListValueFrom(ctx, types.StringType, user.SudoCommands)
	} else if len(user.SudoCommands) > 0 {
		data.SudoCommands, _ = types.ListValueFrom(ctx, types.StringType, user.SudoCommands)
	}

	if !data.SudoCommandsNopasswd.IsNull() {
		data.SudoCommandsNopasswd, _ = types.ListValueFrom(ctx, types.StringType, user.SudoCommandsNopasswd)
	} else if len(user.SudoCommandsNopasswd) > 0 {
		data.SudoCommandsNopasswd, _ = types.ListValueFrom(ctx, types.StringType, user.SudoCommandsNopasswd)
	}
}
