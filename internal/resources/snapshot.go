package resources

import (
	"context"
	"fmt"

	truenas "github.com/deevus/truenas-go"
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

// mapSnapshotToModel maps a typed Snapshot to the Terraform model.
func mapSnapshotToModel(snap *truenas.Snapshot, data *SnapshotResourceModel) {
	data.ID = types.StringValue(snap.ID)
	data.DatasetID = types.StringValue(snap.Dataset)
	data.Name = types.StringValue(snap.SnapshotName)
	data.Hold = types.BoolValue(snap.HasHold)
	data.CreateTXG = types.StringValue(snap.CreateTXG)
	data.UsedBytes = types.Int64Value(snap.Used)
	data.ReferencedBytes = types.Int64Value(snap.Referenced)
}

func (r *SnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SnapshotResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the snapshot
	snap, err := r.services.Snapshot.Create(ctx, truenas.CreateSnapshotOpts{
		Dataset:   data.DatasetID.ValueString(),
		Name:      data.Name.ValueString(),
		Recursive: !data.Recursive.IsNull() && data.Recursive.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Snapshot",
			fmt.Sprintf("Unable to create snapshot: %s", err.Error()),
		)
		return
	}

	// If hold is requested, apply it
	if !data.Hold.IsNull() && data.Hold.ValueBool() {
		err := r.services.Snapshot.Hold(ctx, snap.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Hold Snapshot",
				fmt.Sprintf("Snapshot created but failed to apply hold: %s", err.Error()),
			)
			return
		}

		// Re-read snapshot to get updated hold state
		snap, err = r.services.Snapshot.Get(ctx, snap.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read Snapshot",
				fmt.Sprintf("Snapshot created but unable to read: %s", err.Error()),
			)
			return
		}
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

	snap, err := r.services.Snapshot.Get(ctx, data.ID.ValueString())
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

	snapshotID := state.ID.ValueString()

	// Handle hold changes
	stateHold := state.Hold.ValueBool()
	planHold := plan.Hold.ValueBool()

	if stateHold && !planHold {
		// Release hold
		err := r.services.Snapshot.Release(ctx, snapshotID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Release Snapshot Hold",
				fmt.Sprintf("Unable to release hold on snapshot %q: %s", snapshotID, err.Error()),
			)
			return
		}
	} else if !stateHold && planHold {
		// Apply hold
		err := r.services.Snapshot.Hold(ctx, snapshotID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Hold Snapshot",
				fmt.Sprintf("Unable to hold snapshot %q: %s", snapshotID, err.Error()),
			)
			return
		}
	}

	// Refresh state from API
	snap, err := r.services.Snapshot.Get(ctx, snapshotID)
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

	snapshotID := data.ID.ValueString()

	// If held, release first
	if data.Hold.ValueBool() {
		err := r.services.Snapshot.Release(ctx, snapshotID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Release Snapshot Hold",
				fmt.Sprintf("Unable to release hold before delete: %s", err.Error()),
			)
			return
		}
	}

	// Delete the snapshot
	err := r.services.Snapshot.Delete(ctx, snapshotID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Snapshot",
			fmt.Sprintf("Unable to delete snapshot %q: %s", snapshotID, err.Error()),
		)
		return
	}
}
