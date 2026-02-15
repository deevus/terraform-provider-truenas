package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SnapshotResource{}
var _ resource.ResourceWithConfigure = &SnapshotResource{}
var _ resource.ResourceWithImportState = &SnapshotResource{}

// SnapshotResource defines the resource implementation.
type SnapshotResource struct {
	BaseResource
}

// SnapshotResourceModel describes the resource data model.
type SnapshotResourceModel struct {
	ID              types.String `tfsdk:"id"`
	DatasetID       types.String `tfsdk:"dataset_id"`
	Name            types.String `tfsdk:"name"`
	Hold            types.Bool   `tfsdk:"hold"`
	Recursive       types.Bool   `tfsdk:"recursive"`
	CreateTXG       types.String `tfsdk:"createtxg"`
	UsedBytes       types.Int64  `tfsdk:"used_bytes"`
	ReferencedBytes types.Int64  `tfsdk:"referenced_bytes"`
}


// NewSnapshotResource creates a new SnapshotResource.
func NewSnapshotResource() resource.Resource {
	return &SnapshotResource{}
}

func (r *SnapshotResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot"
}

func (r *SnapshotResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a ZFS snapshot. Use for pre-upgrade backups and point-in-time recovery.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Snapshot identifier (dataset@name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dataset_id": schema.StringAttribute{
				Description: "Dataset ID to snapshot. Reference a truenas_dataset resource or data source.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Snapshot name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hold": schema.BoolAttribute{
				Description: "Prevent automatic deletion. Default: false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"recursive": schema.BoolAttribute{
				Description: "Include child datasets. Default: false. Only used at create time.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"createtxg": schema.StringAttribute{
				Description: "Transaction group when snapshot was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"used_bytes": schema.Int64Attribute{
				Description: "Space consumed by snapshot.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"referenced_bytes": schema.Int64Attribute{
				Description: "Space referenced by snapshot.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}


// querySnapshot queries a snapshot by ID and returns the response.
// Returns nil if the snapshot is not found.
func (r *SnapshotResource) querySnapshot(ctx context.Context, snapshotID string) (*api.SnapshotResponse, error) {
	version := r.client.Version()

	method := api.ResolveSnapshotMethod(version, api.MethodSnapshotQuery)
	filter := [][]any{{"id", "=", snapshotID}}
	result, err := r.client.Call(ctx, method, filter)
	if err != nil {
		return nil, err
	}

	var snapshots []api.SnapshotResponse
	if err := json.Unmarshal(result, &snapshots); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(snapshots) == 0 {
		return nil, nil
	}

	return &snapshots[0], nil
}

// mapSnapshotToModel maps API response fields to the Terraform model.
func mapSnapshotToModel(snap *api.SnapshotResponse, data *SnapshotResourceModel) {
	data.ID = types.StringValue(snap.ID)
	data.DatasetID = types.StringValue(snap.Dataset)
	data.Name = types.StringValue(snap.SnapshotName) // Use SnapshotName, not Name (which is full ID)
	data.Hold = types.BoolValue(snap.HasHold())
	data.CreateTXG = types.StringValue(snap.Properties.CreateTXG.Value)
	data.UsedBytes = types.Int64Value(snap.Properties.Used.Parsed)
	data.ReferencedBytes = types.Int64Value(snap.Properties.Referenced.Parsed)
}

func (r *SnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SnapshotResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get TrueNAS version for API method resolution
	version := r.client.Version()

	// Build create params
	params := map[string]any{
		"dataset": data.DatasetID.ValueString(),
		"name":    data.Name.ValueString(),
	}

	if !data.Recursive.IsNull() && data.Recursive.ValueBool() {
		params["recursive"] = true
	}

	// Create the snapshot
	method := api.ResolveSnapshotMethod(version, api.MethodSnapshotCreate)
	_, err := r.client.Call(ctx, method, params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Snapshot",
			fmt.Sprintf("Unable to create snapshot: %s", err.Error()),
		)
		return
	}

	// Build snapshot ID
	snapshotID := fmt.Sprintf("%s@%s", data.DatasetID.ValueString(), data.Name.ValueString())

	// If hold is requested, apply it
	if !data.Hold.IsNull() && data.Hold.ValueBool() {
		holdMethod := api.ResolveSnapshotMethod(version, api.MethodSnapshotHold)
		_, err := r.client.Call(ctx, holdMethod, snapshotID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Hold Snapshot",
				fmt.Sprintf("Snapshot created but failed to apply hold: %s", err.Error()),
			)
			return
		}
	}

	// Query the snapshot to get computed fields
	snap, err := r.querySnapshot(ctx, snapshotID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Snapshot",
			fmt.Sprintf("Snapshot created but unable to read: %s", err.Error()),
		)
		return
	}

	if snap == nil {
		resp.Diagnostics.AddError(
			"Snapshot Not Found",
			"Snapshot was created but could not be found.",
		)
		return
	}

	mapSnapshotToModel(snap, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SnapshotResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snap, err := r.querySnapshot(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Snapshot",
			fmt.Sprintf("Unable to read snapshot %q: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	if snap == nil {
		// Snapshot no longer exists - remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	mapSnapshotToModel(snap, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state SnapshotResourceModel
	var plan SnapshotResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get TrueNAS version for API method resolution
	version := r.client.Version()

	snapshotID := state.ID.ValueString()

	// Handle hold changes
	stateHold := state.Hold.ValueBool()
	planHold := plan.Hold.ValueBool()

	if stateHold && !planHold {
		// Release hold
		method := api.ResolveSnapshotMethod(version, api.MethodSnapshotRelease)
		_, err := r.client.Call(ctx, method, snapshotID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Release Snapshot Hold",
				fmt.Sprintf("Unable to release hold on snapshot %q: %s", snapshotID, err.Error()),
			)
			return
		}
	} else if !stateHold && planHold {
		// Apply hold
		method := api.ResolveSnapshotMethod(version, api.MethodSnapshotHold)
		_, err := r.client.Call(ctx, method, snapshotID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Hold Snapshot",
				fmt.Sprintf("Unable to hold snapshot %q: %s", snapshotID, err.Error()),
			)
			return
		}
	}

	// Refresh state from API
	snap, err := r.querySnapshot(ctx, snapshotID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Snapshot",
			fmt.Sprintf("Unable to read snapshot %q: %s", snapshotID, err.Error()),
		)
		return
	}

	if snap == nil {
		resp.Diagnostics.AddError(
			"Snapshot Not Found",
			fmt.Sprintf("Snapshot %q no longer exists.", snapshotID),
		)
		return
	}

	mapSnapshotToModel(snap, &plan)
	plan.Hold = types.BoolValue(planHold) // Preserve the planned hold value

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SnapshotResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get TrueNAS version for API method resolution
	version := r.client.Version()

	snapshotID := data.ID.ValueString()

	// If held, release first
	if data.Hold.ValueBool() {
		method := api.ResolveSnapshotMethod(version, api.MethodSnapshotRelease)
		_, err := r.client.Call(ctx, method, snapshotID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Release Snapshot Hold",
				fmt.Sprintf("Unable to release hold before delete: %s", err.Error()),
			)
			return
		}
	}

	// Delete the snapshot
	method := api.ResolveSnapshotMethod(version, api.MethodSnapshotDelete)
	_, err := r.client.Call(ctx, method, snapshotID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Snapshot",
			fmt.Sprintf("Unable to delete snapshot %q: %s", snapshotID, err.Error()),
		)
		return
	}
}

