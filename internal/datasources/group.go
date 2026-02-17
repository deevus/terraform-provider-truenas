package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/deevus/terraform-provider-truenas/internal/api"
	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &GroupDataSource{}
var _ datasource.DataSourceWithConfigure = &GroupDataSource{}

// GroupDataSource defines the data source implementation.
type GroupDataSource struct {
	client client.Client
}

// GroupDataSourceModel describes the data source data model.
type GroupDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	GID                  types.Int64  `tfsdk:"gid"`
	Name                 types.String `tfsdk:"name"`
	SMB                  types.Bool   `tfsdk:"smb"`
	Builtin              types.Bool   `tfsdk:"builtin"`
	Local                types.Bool   `tfsdk:"local"`
	SudoCommands         types.List   `tfsdk:"sudo_commands"`
	SudoCommandsNopasswd types.List   `tfsdk:"sudo_commands_nopasswd"`
	Users                types.List   `tfsdk:"users"`
}

// NewGroupDataSource creates a new GroupDataSource.
func NewGroupDataSource() datasource.DataSource {
	return &GroupDataSource{}
}

func (d *GroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *GroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a TrueNAS group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Internal group ID.",
				Computed:    true,
			},
			"gid": schema.Int64Attribute{
				Description: "UNIX group ID.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the group to look up.",
				Required:    true,
			},
			"smb": schema.BoolAttribute{
				Description: "Whether the group is eligible for SMB share ACLs.",
				Computed:    true,
			},
			"builtin": schema.BoolAttribute{
				Description: "Whether this is a built-in system group.",
				Computed:    true,
			},
			"local": schema.BoolAttribute{
				Description: "Whether this is a local group (vs. directory service).",
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
			"users": schema.ListAttribute{
				Description: "List of user IDs that are members of this group.",
				Computed:    true,
				ElementType: types.Int64Type,
			},
		},
	}
}

func (d *GroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *GroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GroupDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filter := [][]string{{"group", "=", data.Name.ValueString()}}

	result, err := d.client.Call(ctx, "group.query", filter)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Group",
			fmt.Sprintf("Unable to read group %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	var groups []api.GroupResponse
	if err := json.Unmarshal(result, &groups); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Group Response",
			fmt.Sprintf("Unable to parse group response: %s", err.Error()),
		)
		return
	}

	if len(groups) == 0 {
		resp.Diagnostics.AddError(
			"Group Not Found",
			fmt.Sprintf("Group %q was not found.", data.Name.ValueString()),
		)
		return
	}

	group := groups[0]

	data.ID = types.StringValue(strconv.FormatInt(group.ID, 10))
	data.GID = types.Int64Value(group.GID)
	data.Name = types.StringValue(group.Name)
	data.SMB = types.BoolValue(group.SMB)
	data.Builtin = types.BoolValue(group.Builtin)
	data.Local = types.BoolValue(group.Local)

	data.SudoCommands, _ = types.ListValueFrom(ctx, types.StringType, group.SudoCommands)
	data.SudoCommandsNopasswd, _ = types.ListValueFrom(ctx, types.StringType, group.SudoCommandsNopasswd)
	data.Users, _ = types.ListValueFrom(ctx, types.Int64Type, group.Users)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
