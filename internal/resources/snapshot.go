package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	client client.Client
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

// snapshotQueryResponse represents the JSON response from pool.snapshot.query.
type snapshotQueryResponse struct {
	ID         string                     `json:"id"`
	Name       string                     `json:"name"`
	Dataset    string                     `json:"dataset"`
	Holds      map[string]any             `json:"holds"`
	Properties snapshotPropertiesResponse `json:"properties"`
}

type snapshotPropertiesResponse struct {
	CreateTXG  propertyValue `json:"createtxg"`
	Used       parsedValue   `json:"used"`
	Referenced parsedValue   `json:"referenced"`
}

type propertyValue struct {
	Value string `json:"value"`
}

type parsedValue struct {
	Parsed int64 `json:"parsed"`
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

func (r *SnapshotResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// querySnapshot queries a snapshot by ID and returns the response.
// Returns nil if the snapshot is not found.
func (r *SnapshotResource) querySnapshot(ctx context.Context, snapshotID string) (*snapshotQueryResponse, error) {
	filter := [][]any{{"id", "=", snapshotID}}
	result, err := r.client.Call(ctx, "pool.snapshot.query", filter)
	if err != nil {
		return nil, err
	}

	var snapshots []snapshotQueryResponse
	if err := json.Unmarshal(result, &snapshots); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(snapshots) == 0 {
		return nil, nil
	}

	return &snapshots[0], nil
}

// mapSnapshotToModel maps API response fields to the Terraform model.
func mapSnapshotToModel(snap *snapshotQueryResponse, data *SnapshotResourceModel) {
	data.ID = types.StringValue(snap.ID)
	data.DatasetID = types.StringValue(snap.Dataset)
	data.Name = types.StringValue(snap.Name)
	data.Hold = types.BoolValue(len(snap.Holds) > 0)
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

	// Build create params
	params := map[string]any{
		"dataset": data.DatasetID.ValueString(),
		"name":    data.Name.ValueString(),
	}

	if !data.Recursive.IsNull() && data.Recursive.ValueBool() {
		params["recursive"] = true
	}

	// Create the snapshot
	_, err := r.client.Call(ctx, "pool.snapshot.create", params)
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
		_, err := r.client.Call(ctx, "pool.snapshot.hold", snapshotID)
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
	// TODO: implement
	resp.Diagnostics.AddError("Not Implemented", "Read not yet implemented")
}

func (r *SnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO: implement
	resp.Diagnostics.AddError("Not Implemented", "Update not yet implemented")
}

func (r *SnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: implement
	resp.Diagnostics.AddError("Not Implemented", "Delete not yet implemented")
}

func (r *SnapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
