package resources

import (
	"context"
	"fmt"
	"strconv"

	truenas "github.com/deevus/truenas-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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
				Description: "Create a new primary group with the same name as the user. Changing this value forces the user to be destroyed and recreated.",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
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

	opts := buildCreateUserOpts(ctx, &data)

	user, err := r.services.User.Create(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create User",
			fmt.Sprintf("Unable to create user: %s", err.Error()),
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

	user, err := r.services.User.Get(ctx, id)
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

	opts := buildUpdateUserOpts(ctx, &plan)

	user, err := r.services.User.Update(ctx, id, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update User",
			fmt.Sprintf("Unable to update user: %s", err.Error()),
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

	err = r.services.User.Delete(ctx, id, data.GroupCreate.ValueBool())
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

	user, err := r.services.User.GetByUID(ctx, uid)
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

// buildCreateUserOpts builds typed create options from the resource model.
func buildCreateUserOpts(ctx context.Context, data *UserResourceModel) truenas.CreateUserOpts {
	opts := truenas.CreateUserOpts{
		Username:           data.Username.ValueString(),
		FullName:           data.FullName.ValueString(),
		Email:              data.Email.ValueString(),
		PasswordDisabled:   data.PasswordDisabled.ValueBool(),
		Home:               data.Home.ValueString(),
		HomeMode:           data.HomeMode.ValueString(),
		Shell:              data.Shell.ValueString(),
		SMB:                data.SMB.ValueBool(),
		SSHPasswordEnabled: data.SSHPasswordEnabled.ValueBool(),
		Locked:             data.Locked.ValueBool(),
	}

	if !data.UID.IsNull() && !data.UID.IsUnknown() {
		opts.UID = data.UID.ValueInt64()
	}

	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		opts.Password = data.Password.ValueString()
	}

	if !data.GroupID.IsNull() && !data.GroupID.IsUnknown() {
		opts.Group = data.GroupID.ValueInt64()
	}

	if !data.GroupCreate.IsNull() && !data.GroupCreate.IsUnknown() {
		opts.GroupCreate = data.GroupCreate.ValueBool()
	}

	if !data.Groups.IsNull() && !data.Groups.IsUnknown() {
		var items []int64
		data.Groups.ElementsAs(ctx, &items, false)
		opts.Groups = items
	}

	if !data.HomeCreate.IsNull() && !data.HomeCreate.IsUnknown() {
		opts.HomeCreate = data.HomeCreate.ValueBool()
	}

	if !data.SSHPubKey.IsNull() && !data.SSHPubKey.IsUnknown() {
		opts.SSHPubKey = data.SSHPubKey.ValueString()
	}

	if !data.SudoCommands.IsNull() && !data.SudoCommands.IsUnknown() {
		var items []string
		data.SudoCommands.ElementsAs(ctx, &items, false)
		opts.SudoCommands = items
	}

	if !data.SudoCommandsNopasswd.IsNull() && !data.SudoCommandsNopasswd.IsUnknown() {
		var items []string
		data.SudoCommandsNopasswd.ElementsAs(ctx, &items, false)
		opts.SudoCommandsNopasswd = items
	}

	return opts
}

// buildUpdateUserOpts builds typed update options from the resource model.
// UID, GroupCreate, and HomeCreate are excluded (immutable after creation).
func buildUpdateUserOpts(ctx context.Context, data *UserResourceModel) truenas.UpdateUserOpts {
	opts := truenas.UpdateUserOpts{
		Username:           data.Username.ValueString(),
		FullName:           data.FullName.ValueString(),
		Email:              data.Email.ValueString(),
		PasswordDisabled:   data.PasswordDisabled.ValueBool(),
		Home:               data.Home.ValueString(),
		HomeMode:           data.HomeMode.ValueString(),
		Shell:              data.Shell.ValueString(),
		SMB:                data.SMB.ValueBool(),
		SSHPasswordEnabled: data.SSHPasswordEnabled.ValueBool(),
		Locked:             data.Locked.ValueBool(),
	}

	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		opts.Password = data.Password.ValueString()
	}

	if !data.GroupID.IsNull() && !data.GroupID.IsUnknown() {
		opts.Group = data.GroupID.ValueInt64()
	}

	if !data.Groups.IsNull() && !data.Groups.IsUnknown() {
		var items []int64
		data.Groups.ElementsAs(ctx, &items, false)
		opts.Groups = items
	}

	if !data.SSHPubKey.IsNull() && !data.SSHPubKey.IsUnknown() {
		opts.SSHPubKey = data.SSHPubKey.ValueString()
	}

	if !data.SudoCommands.IsNull() && !data.SudoCommands.IsUnknown() {
		var items []string
		data.SudoCommands.ElementsAs(ctx, &items, false)
		opts.SudoCommands = items
	}

	if !data.SudoCommandsNopasswd.IsNull() && !data.SudoCommandsNopasswd.IsUnknown() {
		var items []string
		data.SudoCommandsNopasswd.ElementsAs(ctx, &items, false)
		opts.SudoCommandsNopasswd = items
	}

	return opts
}

// mapUserToModel maps a typed User to the resource model.
func mapUserToModel(ctx context.Context, user *truenas.User, data *UserResourceModel) {
	data.ID = types.StringValue(strconv.FormatInt(user.ID, 10))
	data.UID = types.Int64Value(user.UID)
	data.Username = types.StringValue(user.Username)
	data.FullName = types.StringValue(user.FullName)
	data.Email = types.StringValue(user.Email)
	data.PasswordDisabled = types.BoolValue(user.PasswordDisabled)
	data.GroupID = types.Int64Value(user.GroupID)
	data.Home = types.StringValue(user.Home)
	if user.HomeMode != "" {
		data.HomeMode = types.StringValue(user.HomeMode)
	}
	data.Shell = types.StringValue(user.Shell)
	data.SMB = types.BoolValue(user.SMB)
	data.SSHPasswordEnabled = types.BoolValue(user.SSHPasswordEnabled)
	if user.SSHPubKey != "" {
		data.SSHPubKey = types.StringValue(user.SSHPubKey)
	}
	data.Locked = types.BoolValue(user.Locked)
	data.Builtin = types.BoolValue(user.Builtin)

	// Do not overwrite Password - it is write-only
	// Do not overwrite GroupCreate - it is create-only
	// Do not overwrite HomeCreate - it is create-only

	if !data.Groups.IsNull() {
		data.Groups, _ = types.ListValueFrom(ctx, types.Int64Type, user.Groups)
	}

	if !data.SudoCommands.IsNull() {
		data.SudoCommands, _ = types.ListValueFrom(ctx, types.StringType, user.SudoCommands)
	}

	if !data.SudoCommandsNopasswd.IsNull() {
		data.SudoCommandsNopasswd, _ = types.ListValueFrom(ctx, types.StringType, user.SudoCommandsNopasswd)
	}
}
