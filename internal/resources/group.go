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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &GroupResource{}
	_ resource.ResourceWithConfigure   = &GroupResource{}
	_ resource.ResourceWithImportState = &GroupResource{}
)

// GroupResourceModel describes the resource data model.
type GroupResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	GID                  types.Int64  `tfsdk:"gid"`
	Name                 types.String `tfsdk:"name"`
	SMB                  types.Bool   `tfsdk:"smb"`
	SudoCommands         types.List   `tfsdk:"sudo_commands"`
	SudoCommandsNopasswd types.List   `tfsdk:"sudo_commands_nopasswd"`
	Builtin              types.Bool   `tfsdk:"builtin"`
}

// GroupResource defines the resource implementation.
type GroupResource struct {
	BaseResource
}

// NewGroupResource creates a new GroupResource.
func NewGroupResource() resource.Resource {
	return &GroupResource{}
}

func (r *GroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *GroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages local groups on TrueNAS.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Group ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"gid": schema.Int64Attribute{
				Description: "UNIX group ID. If not specified, TrueNAS assigns the next available GID.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Group name.",
				Required:    true,
			},
			"smb": schema.BoolAttribute{
				Description: "Allow group to be used for SMB permissions.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
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
				Description: "Whether this is a built-in system group.",
				Computed:    true,
			},
		},
	}
}

func (r *GroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := buildGroupCreateParams(ctx, &data)

	result, err := r.client.Call(ctx, "group.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Group",
			fmt.Sprintf("Unable to create group: %s", err.Error()),
		)
		return
	}

	var createdID int64
	if err := json.Unmarshal(result, &createdID); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Response",
			fmt.Sprintf("Unable to parse create response: %s", err.Error()),
		)
		return
	}

	group, err := r.queryGroup(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Group",
			fmt.Sprintf("Group created but unable to read: %s", err.Error()),
		)
		return
	}

	if group == nil {
		resp.Diagnostics.AddError(
			"Group Not Found",
			"Group was created but could not be found.",
		)
		return
	}

	mapGroupToModel(ctx, group, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GroupResourceModel

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

	group, err := r.queryGroup(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Group",
			fmt.Sprintf("Unable to query group: %s", err.Error()),
		)
		return
	}

	if group == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	mapGroupToModel(ctx, group, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state GroupResourceModel
	var plan GroupResourceModel

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

	params := buildGroupUpdateParams(ctx, &plan)

	_, err = r.client.Call(ctx, "group.update", []any{id, params})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Group",
			fmt.Sprintf("Unable to update group: %s", err.Error()),
		)
		return
	}

	group, err := r.queryGroup(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Group",
			fmt.Sprintf("Group updated but unable to read: %s", err.Error()),
		)
		return
	}

	if group == nil {
		resp.Diagnostics.AddError(
			"Group Not Found",
			"Group was updated but could not be found.",
		)
		return
	}

	mapGroupToModel(ctx, group, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GroupResourceModel

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

	_, err = r.client.Call(ctx, "group.delete", []any{id, map[string]any{"delete_users": false}})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Group",
			fmt.Sprintf("Unable to delete group: %s", err.Error()),
		)
		return
	}
}

// ImportState imports a group by GID.
func (r *GroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	gid, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric GID, got %q", req.ID),
		)
		return
	}

	group, err := r.queryGroupByField(ctx, "gid", gid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Import Group", err.Error())
		return
	}

	if group == nil {
		resp.Diagnostics.AddError(
			"Group Not Found",
			fmt.Sprintf("No group found with GID %d", gid),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.FormatInt(group.ID, 10))...)
}

// queryGroup queries a group by internal ID and returns the response.
func (r *GroupResource) queryGroup(ctx context.Context, id int64) (*api.GroupResponse, error) {
	return r.queryGroupByField(ctx, "id", id)
}

// queryGroupByField queries a group by an arbitrary field and returns the response.
func (r *GroupResource) queryGroupByField(ctx context.Context, field string, value int64) (*api.GroupResponse, error) {
	filter := [][]any{{field, "=", value}}
	result, err := r.client.Call(ctx, "group.query", filter)
	if err != nil {
		return nil, err
	}

	var groups []api.GroupResponse
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(groups) == 0 {
		return nil, nil
	}

	return &groups[0], nil
}

// buildGroupCreateParams builds the API create params from the resource model.
func buildGroupCreateParams(ctx context.Context, data *GroupResourceModel) map[string]any {
	params := map[string]any{
		"name": data.Name.ValueString(),
		"smb":  data.SMB.ValueBool(),
	}

	if !data.GID.IsNull() && !data.GID.IsUnknown() {
		params["gid"] = data.GID.ValueInt64()
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

// buildGroupUpdateParams builds the API update params (excludes gid).
func buildGroupUpdateParams(ctx context.Context, data *GroupResourceModel) map[string]any {
	params := map[string]any{
		"name": data.Name.ValueString(),
		"smb":  data.SMB.ValueBool(),
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

// mapGroupToModel maps an API response to the resource model.
func mapGroupToModel(ctx context.Context, group *api.GroupResponse, data *GroupResourceModel) {
	data.ID = types.StringValue(strconv.FormatInt(group.ID, 10))
	data.GID = types.Int64Value(group.GID)
	data.Name = types.StringValue(group.Name)
	data.SMB = types.BoolValue(group.SMB)
	data.Builtin = types.BoolValue(group.Builtin)

	if !data.SudoCommands.IsNull() {
		data.SudoCommands, _ = types.ListValueFrom(ctx, types.StringType, group.SudoCommands)
	} else if len(group.SudoCommands) > 0 {
		data.SudoCommands, _ = types.ListValueFrom(ctx, types.StringType, group.SudoCommands)
	}

	if !data.SudoCommandsNopasswd.IsNull() {
		data.SudoCommandsNopasswd, _ = types.ListValueFrom(ctx, types.StringType, group.SudoCommandsNopasswd)
	} else if len(group.SudoCommandsNopasswd) > 0 {
		data.SudoCommandsNopasswd, _ = types.ListValueFrom(ctx, types.StringType, group.SudoCommandsNopasswd)
	}
}
