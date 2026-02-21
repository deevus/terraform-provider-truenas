package resources

import (
	"context"
	"encoding/json"
	"fmt"

	truenas "github.com/deevus/truenas-go"
	customtypes "github.com/deevus/terraform-provider-truenas/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ZvolResource{}
var _ resource.ResourceWithConfigure = &ZvolResource{}
var _ resource.ResourceWithImportState = &ZvolResource{}

type ZvolResource struct {
	BaseResource
}

type ZvolResourceModel struct {
	ID           types.String                `tfsdk:"id"`
	Pool         types.String                `tfsdk:"pool"`
	Path         types.String                `tfsdk:"path"`
	Parent       types.String                `tfsdk:"parent"`
	Volsize      customtypes.SizeStringValue `tfsdk:"volsize"`
	Volblocksize types.String                `tfsdk:"volblocksize"`
	Sparse       types.Bool                  `tfsdk:"sparse"`
	ForceSize    types.Bool                  `tfsdk:"force_size"`
	Compression  types.String                `tfsdk:"compression"`
	Comments     types.String                `tfsdk:"comments"`
	ForceDestroy types.Bool                  `tfsdk:"force_destroy"`
}

// zvolQueryResponse represents the JSON response from pool.dataset.query for a zvol.
type zvolQueryResponse struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Pool         string             `json:"pool"`
	Volsize      sizePropertyField  `json:"volsize"`
	Volblocksize propertyValueField `json:"volblocksize"`
	Sparse       propertyValueField `json:"sparse"`
	Compression  propertyValueField `json:"compression"`
	Comments     propertyValueField `json:"comments"`
}

func NewZvolResource() resource.Resource {
	return &ZvolResource{}
}

func (r *ZvolResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zvol"
}

func (r *ZvolResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := poolDatasetIdentitySchema()

	// Zvol-specific attributes
	attrs["volsize"] = schema.StringAttribute{
		CustomType:  customtypes.SizeStringType{},
		Description: "Volume size. Accepts human-readable sizes (e.g., '10G', '500M', '1T') or bytes. Must be a multiple of volblocksize.",
		Required:    true,
	}
	attrs["volblocksize"] = schema.StringAttribute{
		Description: "Volume block size. Cannot be changed after creation. Options: 512, 512B, 1K, 2K, 4K, 8K, 16K, 32K, 64K, 128K.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
			stringplanmodifier.RequiresReplace(),
		},
	}
	attrs["sparse"] = schema.BoolAttribute{
		Description: "Create a sparse (thin-provisioned) volume. Defaults to false.",
		Optional:    true,
	}
	attrs["force_size"] = schema.BoolAttribute{
		Description: "Allow setting volsize that is not a multiple of volblocksize, or allow shrinking. Not stored in state.",
		Optional:    true,
	}
	attrs["compression"] = schema.StringAttribute{
		Description: "Compression algorithm (e.g., 'LZ4', 'ZSTD', 'OFF').",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	attrs["comments"] = schema.StringAttribute{
		Description: "Comments / description for this volume.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	attrs["force_destroy"] = schema.BoolAttribute{
		Description: "Force destroy including child datasets. Defaults to false.",
		Optional:    true,
	}

	resp.Schema = schema.Schema{
		Description: "Manages a ZFS volume (zvol) on TrueNAS. Zvols are block devices backed by ZFS, commonly used as VM disks or iSCSI targets.",
		Attributes:  attrs,
	}
}


func (r *ZvolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ZvolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fullName := poolDatasetFullName(data.Pool, data.Path, data.Parent, types.StringNull())
	if fullName == "" {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Either 'pool' with 'path', or 'parent' with 'path' must be provided.",
		)
		return
	}

	// Parse volsize
	volsizeBytes, err := truenas.ParseSize(data.Volsize.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Volsize", fmt.Sprintf("Unable to parse volsize %q: %s", data.Volsize.ValueString(), err.Error()))
		return
	}

	params := map[string]any{
		"name":    fullName,
		"type":    "VOLUME",
		"volsize": volsizeBytes,
	}

	if !data.Volblocksize.IsNull() && !data.Volblocksize.IsUnknown() {
		params["volblocksize"] = data.Volblocksize.ValueString()
	}
	if !data.Sparse.IsNull() && !data.Sparse.IsUnknown() {
		params["sparse"] = data.Sparse.ValueBool()
	}
	if !data.ForceSize.IsNull() && !data.ForceSize.IsUnknown() {
		params["force_size"] = data.ForceSize.ValueBool()
	}
	if !data.Compression.IsNull() && !data.Compression.IsUnknown() {
		params["compression"] = data.Compression.ValueString()
	}
	if !data.Comments.IsNull() && !data.Comments.IsUnknown() {
		params["comments"] = data.Comments.ValueString()
	}

	result, err := r.client.Call(ctx, "pool.dataset.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Zvol",
			fmt.Sprintf("Unable to create zvol %q: %s", fullName, err.Error()),
		)
		return
	}

	// Parse create response to get ID
	var createResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(result, &createResp); err != nil {
		resp.Diagnostics.AddError("Unable to Parse Response", fmt.Sprintf("Unable to parse create response: %s", err.Error()))
		return
	}

	// Query to get all computed attributes
	r.readZvolAfterCreate(ctx, createResp.ID, &data, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZvolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ZvolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zvolID := data.ID.ValueString()

	raw, err := queryPoolDataset(ctx, r.client, zvolID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Zvol", fmt.Sprintf("Unable to read zvol %q: %s", zvolID, err.Error()))
		return
	}

	if raw == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var zvol zvolQueryResponse
	if err := json.Unmarshal(raw, &zvol); err != nil {
		resp.Diagnostics.AddError("Unable to Parse Response", fmt.Sprintf("Unable to parse zvol response: %s", err.Error()))
		return
	}

	mapZvolToModel(&zvol, &data)

	// Populate pool/path from ID if not set (e.g., after import)
	if data.Pool.IsNull() && data.Path.IsNull() && data.Parent.IsNull() {
		pool, path := poolDatasetIDToParts(zvol.ID)
		data.Pool = types.StringValue(pool)
		data.Path = types.StringValue(path)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZvolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ZvolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateParams := map[string]any{}

	// Check volsize change
	if !plan.Volsize.Equal(state.Volsize) {
		volsizeBytes, err := truenas.ParseSize(plan.Volsize.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid Volsize", fmt.Sprintf("Unable to parse volsize %q: %s", plan.Volsize.ValueString(), err.Error()))
			return
		}
		updateParams["volsize"] = volsizeBytes
	}

	if !plan.Compression.Equal(state.Compression) && !plan.Compression.IsNull() {
		updateParams["compression"] = plan.Compression.ValueString()
	}

	if !plan.Comments.Equal(state.Comments) {
		if plan.Comments.IsNull() {
			updateParams["comments"] = ""
		} else {
			updateParams["comments"] = plan.Comments.ValueString()
		}
	}

	if !plan.ForceSize.IsNull() && !plan.ForceSize.IsUnknown() && plan.ForceSize.ValueBool() {
		updateParams["force_size"] = true
	}

	zvolID := state.ID.ValueString()

	if len(updateParams) > 0 {
		_, err := r.client.Call(ctx, "pool.dataset.update", []any{zvolID, updateParams})
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Update Zvol",
				fmt.Sprintf("Unable to update zvol %q: %s", zvolID, err.Error()),
			)
			return
		}
	}

	// Re-read to get current state
	raw, err := queryPoolDataset(ctx, r.client, zvolID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Zvol After Update", fmt.Sprintf("Unable to read zvol %q: %s", zvolID, err.Error()))
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Zvol Not Found After Update", fmt.Sprintf("Zvol %q not found after update", zvolID))
		return
	}

	var zvol zvolQueryResponse
	if err := json.Unmarshal(raw, &zvol); err != nil {
		resp.Diagnostics.AddError("Unable to Parse Response", fmt.Sprintf("Unable to parse zvol response: %s", err.Error()))
		return
	}

	mapZvolToModel(&zvol, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZvolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ZvolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zvolID := data.ID.ValueString()
	recursive := !data.ForceDestroy.IsNull() && data.ForceDestroy.ValueBool()

	if err := deletePoolDataset(ctx, r.client, zvolID, recursive); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Zvol",
			fmt.Sprintf("Unable to delete zvol %q: %s", zvolID, err.Error()),
		)
	}
}


// readZvolAfterCreate queries a zvol and maps it into the model after creation.
func (r *ZvolResource) readZvolAfterCreate(ctx context.Context, zvolID string, data *ZvolResourceModel, resp *resource.CreateResponse) {
	raw, err := queryPoolDataset(ctx, r.client, zvolID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Zvol After Create", fmt.Sprintf("Zvol was created but unable to read it: %s", err.Error()))
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Zvol Not Found After Create", fmt.Sprintf("Zvol %q was created but could not be found", zvolID))
		return
	}

	var zvol zvolQueryResponse
	if err := json.Unmarshal(raw, &zvol); err != nil {
		resp.Diagnostics.AddError("Unable to Parse Response", fmt.Sprintf("Unable to parse zvol response: %s", err.Error()))
		return
	}

	mapZvolToModel(&zvol, data)
}

// mapZvolToModel maps a query response to the resource model.
func mapZvolToModel(zvol *zvolQueryResponse, data *ZvolResourceModel) {
	data.ID = types.StringValue(zvol.ID)
	data.Volsize = customtypes.NewSizeStringValue(fmt.Sprintf("%d", zvol.Volsize.Parsed))
	data.Volblocksize = types.StringValue(zvol.Volblocksize.Value)
	data.Compression = types.StringValue(zvol.Compression.Value)

	if zvol.Comments.Value != "" {
		data.Comments = types.StringValue(zvol.Comments.Value)
	} else {
		data.Comments = types.StringNull()
	}
}
