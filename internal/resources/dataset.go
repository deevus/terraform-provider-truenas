package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &DatasetResource{}
var _ resource.ResourceWithConfigure = &DatasetResource{}
var _ resource.ResourceWithImportState = &DatasetResource{}

// DatasetResource defines the resource implementation.
type DatasetResource struct {
	client client.Client
}

// DatasetResourceModel describes the resource data model.
type DatasetResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Pool         types.String `tfsdk:"pool"`
	Path         types.String `tfsdk:"path"`
	Parent       types.String `tfsdk:"parent"`
	Name         types.String `tfsdk:"name"`
	MountPath    types.String `tfsdk:"mount_path"`
	Compression  types.String `tfsdk:"compression"`
	Quota        types.String `tfsdk:"quota"`
	RefQuota     types.String `tfsdk:"refquota"`
	Atime        types.String `tfsdk:"atime"`
	ForceDestroy types.Bool   `tfsdk:"force_destroy"`
}

// datasetCreateResponse represents the JSON response from pool.dataset.create.
type datasetCreateResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Mountpoint string `json:"mountpoint"`
}

// datasetQueryResponse represents the JSON response from pool.dataset.query.
type datasetQueryResponse struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Mountpoint  string             `json:"mountpoint"`
	Compression propertyValueField `json:"compression"`
	Quota       propertyValueField `json:"quota"`
	RefQuota    propertyValueField `json:"refquota"`
	Atime       propertyValueField `json:"atime"`
}

// propertyValueField represents the nested property value in API responses.
type propertyValueField struct {
	Value string `json:"value"`
}

// queryDataset queries a dataset by ID and returns the response.
// Returns nil if the dataset is not found.
func (r *DatasetResource) queryDataset(ctx context.Context, datasetID string) (*datasetQueryResponse, error) {
	filter := [][]any{{"id", "=", datasetID}}
	result, err := r.client.Call(ctx, "pool.dataset.query", filter)
	if err != nil {
		return nil, err
	}

	var datasets []datasetQueryResponse
	if err := json.Unmarshal(result, &datasets); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(datasets) == 0 {
		return nil, nil
	}

	return &datasets[0], nil
}

// mapDatasetToModel maps API response fields to the Terraform model.
func mapDatasetToModel(ds *datasetQueryResponse, data *DatasetResourceModel) {
	data.ID = types.StringValue(ds.ID)
	data.MountPath = types.StringValue(ds.Mountpoint)
	data.Compression = types.StringValue(ds.Compression.Value)
	data.Quota = types.StringValue(ds.Quota.Value)
	data.RefQuota = types.StringValue(ds.RefQuota.Value)
	data.Atime = types.StringValue(ds.Atime.Value)
}

// NewDatasetResource creates a new DatasetResource.
func NewDatasetResource() resource.Resource {
	return &DatasetResource{}
}

func (r *DatasetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

func (r *DatasetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS dataset.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Dataset identifier (pool/path).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pool": schema.StringAttribute{
				Description: "Pool name. Use with 'path' attribute.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"path": schema.StringAttribute{
				Description: "Dataset path within the pool. Use with 'pool' attribute.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parent": schema.StringAttribute{
				Description: "Parent dataset path (e.g., 'tank/data'). Use with 'name' attribute.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Dataset name. Use with 'parent' attribute.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mount_path": schema.StringAttribute{
				Description: "Filesystem mount path.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// Optional+Computed attributes use UseStateForUnknown() to prevent Terraform
			// from showing "known after apply" on every plan when the user hasn't specified
			// a value. After Create, these are always populated from the API response, so
			// subsequent plans use the known state value instead of showing as unknown.
			"compression": schema.StringAttribute{
				Description: "Compression algorithm (e.g., 'lz4', 'zstd', 'off').",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"quota": schema.StringAttribute{
				Description: "Dataset quota (e.g., '10G', '1T').",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"refquota": schema.StringAttribute{
				Description: "Dataset reference quota (e.g., '10G', '1T').",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"atime": schema.StringAttribute{
				Description: "Access time tracking ('on' or 'off').",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"force_destroy": schema.BoolAttribute{
				Description: "When destroying this resource, also delete all child datasets. Defaults to false.",
				Optional:    true,
			},
		},
	}
}

func (r *DatasetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *DatasetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DatasetResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the full dataset name
	fullName := getFullName(&data)
	if fullName == "" {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Either 'pool' and 'path' or 'parent' and 'name' must be provided.",
		)
		return
	}

	// Build create params
	params := map[string]any{
		"name": fullName,
	}

	if !data.Compression.IsNull() && !data.Compression.IsUnknown() {
		params["compression"] = data.Compression.ValueString()
	}

	if !data.Quota.IsNull() && !data.Quota.IsUnknown() {
		params["quota"] = data.Quota.ValueString()
	}

	if !data.RefQuota.IsNull() && !data.RefQuota.IsUnknown() {
		params["refquota"] = data.RefQuota.ValueString()
	}

	if !data.Atime.IsNull() && !data.Atime.IsUnknown() {
		params["atime"] = data.Atime.ValueString()
	}

	// Call the TrueNAS API
	result, err := r.client.Call(ctx, "pool.dataset.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Dataset",
			fmt.Sprintf("Unable to create dataset %q: %s", fullName, err.Error()),
		)
		return
	}

	// Parse the response
	var createResp datasetCreateResponse
	if err := json.Unmarshal(result, &createResp); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Dataset Response",
			fmt.Sprintf("Unable to parse dataset create response: %s", err.Error()),
		)
		return
	}

	// Query the created dataset to get all computed attributes
	ds, err := r.queryDataset(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Dataset After Create",
			fmt.Sprintf("Dataset was created but unable to read it: %s", err.Error()),
		)
		return
	}

	if ds == nil {
		resp.Diagnostics.AddError(
			"Dataset Not Found After Create",
			fmt.Sprintf("Dataset %q was created but could not be found", createResp.ID),
		)
		return
	}

	// Map all attributes from query response
	mapDatasetToModel(ds, &data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatasetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DatasetResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	datasetID := data.ID.ValueString()

	ds, err := r.queryDataset(ctx, datasetID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Dataset",
			fmt.Sprintf("Unable to read dataset %q: %s", datasetID, err.Error()),
		)
		return
	}

	// Dataset was deleted outside of Terraform - remove from state
	if ds == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response to model - always set all computed attributes
	mapDatasetToModel(ds, &data)

	// Populate pool/path from ID if not set (e.g., after import)
	// ID format is "pool/path/to/dataset"
	if data.Pool.IsNull() && data.Path.IsNull() && data.Parent.IsNull() && data.Name.IsNull() {
		parts := strings.SplitN(ds.ID, "/", 2)
		if len(parts) == 2 {
			data.Pool = types.StringValue(parts[0])
			data.Path = types.StringValue(parts[1])
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatasetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DatasetResourceModel
	var state DatasetResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read current state
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build update params - only include changed values
	updateParams := map[string]any{}

	if !data.Compression.Equal(state.Compression) && !data.Compression.IsNull() {
		updateParams["compression"] = data.Compression.ValueString()
	}

	if !data.Quota.Equal(state.Quota) && !data.Quota.IsNull() {
		updateParams["quota"] = data.Quota.ValueString()
	}

	if !data.RefQuota.Equal(state.RefQuota) && !data.RefQuota.IsNull() {
		updateParams["refquota"] = data.RefQuota.ValueString()
	}

	if !data.Atime.Equal(state.Atime) && !data.Atime.IsNull() {
		updateParams["atime"] = data.Atime.ValueString()
	}

	// If no changes, copy computed values from state and save
	if len(updateParams) == 0 {
		data.MountPath = state.MountPath
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// Call the TrueNAS API with positional args [id, params_object]
	datasetID := data.ID.ValueString()
	params := []any{datasetID, updateParams}

	result, err := r.client.Call(ctx, "pool.dataset.update", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Dataset",
			fmt.Sprintf("Unable to update dataset %q: %s", datasetID, err.Error()),
		)
		return
	}

	// Parse the response
	var updateResp datasetQueryResponse
	if err := json.Unmarshal(result, &updateResp); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Dataset Response",
			fmt.Sprintf("Unable to parse dataset update response: %s", err.Error()),
		)
		return
	}

	// Map response to model
	mapDatasetToModel(&updateResp, &data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatasetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DatasetResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call the TrueNAS API
	datasetID := data.ID.ValueString()

	// Build delete params - API expects [id, options] when options provided
	var params any = datasetID
	if !data.ForceDestroy.IsNull() && data.ForceDestroy.ValueBool() {
		params = []any{datasetID, map[string]bool{"recursive": true}}
	}

	_, err := r.client.Call(ctx, "pool.dataset.delete", params)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Dataset",
			fmt.Sprintf("Unable to delete dataset %q: %s", datasetID, err.Error()),
		)
		return
	}
}

func (r *DatasetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// getFullName returns the full dataset name from the model.
// It supports two modes:
// 1. pool + path (e.g., pool="tank", path="data/apps" -> "tank/data/apps")
// 2. parent + name (e.g., parent="tank/data", name="apps" -> "tank/data/apps")
// If neither mode is valid, it returns an empty string.
// If both modes are provided, pool/path takes precedence.
func getFullName(data *DatasetResourceModel) string {
	// Mode 1: pool + path
	hasPool := !data.Pool.IsNull() && !data.Pool.IsUnknown() && data.Pool.ValueString() != ""
	hasPath := !data.Path.IsNull() && !data.Path.IsUnknown() && data.Path.ValueString() != ""

	if hasPool && hasPath {
		return fmt.Sprintf("%s/%s", data.Pool.ValueString(), data.Path.ValueString())
	}

	// Mode 2: parent + name
	hasParent := !data.Parent.IsNull() && !data.Parent.IsUnknown() && data.Parent.ValueString() != ""
	hasName := !data.Name.IsNull() && !data.Name.IsUnknown() && data.Name.ValueString() != ""

	if hasParent && hasName {
		return fmt.Sprintf("%s/%s", data.Parent.ValueString(), data.Name.ValueString())
	}

	// Invalid configuration
	return ""
}
