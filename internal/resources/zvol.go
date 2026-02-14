package resources

import (
	"context"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	customtypes "github.com/deevus/terraform-provider-truenas/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	client client.Client
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

func (r *ZvolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ZvolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("Not Implemented", "Create is not yet implemented")
}

func (r *ZvolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("Not Implemented", "Read is not yet implemented")
}

func (r *ZvolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Not Implemented", "Update is not yet implemented")
}

func (r *ZvolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("Not Implemented", "Delete is not yet implemented")
}

func (r *ZvolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
