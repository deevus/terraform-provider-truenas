package datasources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &UserDataSource{}
var _ datasource.DataSourceWithConfigure = &UserDataSource{}

// UserDataSource defines the data source implementation.
type UserDataSource struct {
	services *services.TrueNASServices
}

// UserDataSourceModel describes the data source data model.
type UserDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	UID                  types.Int64  `tfsdk:"uid"`
	Username             types.String `tfsdk:"username"`
	FullName             types.String `tfsdk:"full_name"`
	Email                types.String `tfsdk:"email"`
	Home                 types.String `tfsdk:"home"`
	Shell                types.String `tfsdk:"shell"`
	GroupID              types.Int64  `tfsdk:"group_id"`
	Groups               types.List   `tfsdk:"groups"`
	SMB                  types.Bool   `tfsdk:"smb"`
	PasswordDisabled     types.Bool   `tfsdk:"password_disabled"`
	SSHPasswordEnabled   types.Bool   `tfsdk:"ssh_password_enabled"`
	SSHPubKey            types.String `tfsdk:"sshpubkey"`
	Locked               types.Bool   `tfsdk:"locked"`
	SudoCommands         types.List   `tfsdk:"sudo_commands"`
	SudoCommandsNopasswd types.List   `tfsdk:"sudo_commands_nopasswd"`
	Builtin              types.Bool   `tfsdk:"builtin"`
	Local                types.Bool   `tfsdk:"local"`
}

// NewUserDataSource creates a new UserDataSource.
func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

func (d *UserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a TrueNAS user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Internal user ID.",
				Computed:    true,
			},
			"uid": schema.Int64Attribute{
				Description: "UNIX user ID.",
				Computed:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username to look up.",
				Required:    true,
			},
			"full_name": schema.StringAttribute{
				Description: "Full name of the user.",
				Computed:    true,
			},
			"email": schema.StringAttribute{
				Description: "Email address.",
				Computed:    true,
			},
			"home": schema.StringAttribute{
				Description: "Home directory path.",
				Computed:    true,
			},
			"shell": schema.StringAttribute{
				Description: "Login shell path.",
				Computed:    true,
			},
			"group_id": schema.Int64Attribute{
				Description: "Primary group internal ID.",
				Computed:    true,
			},
			"groups": schema.ListAttribute{
				Description: "List of secondary group IDs.",
				Computed:    true,
				ElementType: types.Int64Type,
			},
			"smb": schema.BoolAttribute{
				Description: "Whether SMB authentication is enabled.",
				Computed:    true,
			},
			"password_disabled": schema.BoolAttribute{
				Description: "Whether password login is disabled.",
				Computed:    true,
			},
			"ssh_password_enabled": schema.BoolAttribute{
				Description: "Whether SSH password authentication is enabled.",
				Computed:    true,
			},
			"sshpubkey": schema.StringAttribute{
				Description: "SSH public key.",
				Computed:    true,
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the account is locked.",
				Computed:    true,
			},
			"sudo_commands": schema.ListAttribute{
				Description: "List of allowed sudo commands.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"sudo_commands_nopasswd": schema.ListAttribute{
				Description: "List of allowed sudo commands without password.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"builtin": schema.BoolAttribute{
				Description: "Whether this is a built-in system user.",
				Computed:    true,
			},
			"local": schema.BoolAttribute{
				Description: "Whether this is a local user (vs. directory service).",
				Computed:    true,
			},
		},
	}
}

func (d *UserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	s, ok := req.ProviderData.(*services.TrueNASServices)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *services.TrueNASServices, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.services = s
}

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := d.services.User.GetByUsername(ctx, data.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read User",
			fmt.Sprintf("Unable to read user %q: %s", data.Username.ValueString(), err.Error()),
		)
		return
	}

	if user == nil {
		resp.Diagnostics.AddError(
			"User Not Found",
			fmt.Sprintf("User %q was not found.", data.Username.ValueString()),
		)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(user.ID, 10))
	data.UID = types.Int64Value(user.UID)
	data.Username = types.StringValue(user.Username)
	data.FullName = types.StringValue(user.FullName)
	data.Email = types.StringValue(user.Email)
	data.Home = types.StringValue(user.Home)
	data.Shell = types.StringValue(user.Shell)
	data.GroupID = types.Int64Value(user.GroupID)
	data.SMB = types.BoolValue(user.SMB)
	data.PasswordDisabled = types.BoolValue(user.PasswordDisabled)
	data.SSHPasswordEnabled = types.BoolValue(user.SSHPasswordEnabled)
	data.SSHPubKey = types.StringValue(user.SSHPubKey)
	data.Locked = types.BoolValue(user.Locked)
	data.Builtin = types.BoolValue(user.Builtin)
	data.Local = types.BoolValue(user.Local)

	data.Groups, _ = types.ListValueFrom(ctx, types.Int64Type, user.Groups)
	data.SudoCommands, _ = types.ListValueFrom(ctx, types.StringType, user.SudoCommands)
	data.SudoCommandsNopasswd, _ = types.ListValueFrom(ctx, types.StringType, user.SudoCommandsNopasswd)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
