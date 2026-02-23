package resources

import (
	"context"
	"fmt"

	truenas "github.com/deevus/truenas-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &VirtConfigResource{}
	_ resource.ResourceWithConfigure   = &VirtConfigResource{}
	_ resource.ResourceWithImportState = &VirtConfigResource{}
)

// VirtConfigResourceModel describes the resource data model.
type VirtConfigResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Bridge        types.String `tfsdk:"bridge"`
	V4Network     types.String `tfsdk:"v4_network"`
	V6Network     types.String `tfsdk:"v6_network"`
	Pool types.String `tfsdk:"pool"`
}

// VirtConfigResource defines the resource implementation.
type VirtConfigResource struct {
	BaseResource
}

// NewVirtConfigResource creates a new VirtConfigResource.
func NewVirtConfigResource() resource.Resource {
	return &VirtConfigResource{}
}

func (r *VirtConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virt_config"
}

func (r *VirtConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the global virtualization configuration on TrueNAS.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource ID (always 'virt_config').",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bridge": schema.StringAttribute{
				Description: "The network bridge used for virtualizations. Set to null to auto-detect.",
				Optional:    true,
			},
			"v4_network": schema.StringAttribute{
				Description: "The IPv4 network CIDR for virtualizations.",
				Optional:    true,
			},
			"v6_network": schema.StringAttribute{
				Description: "The IPv6 network CIDR for virtualizations.",
				Optional:    true,
			},
			"pool": schema.StringAttribute{
				Description: "The default storage pool for virtualizations.",
				Optional:    true,
			},
		},
	}
}


func (r *VirtConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VirtConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build opts and call API
	opts := r.buildConfigOpts(&data)
	config, err := r.services.Virt.UpdateGlobalConfig(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update LXC Config",
			fmt.Sprintf("Unable to update virtualization configuration: %s", err.Error()),
		)
		return
	}

	// Map response to model
	r.mapConfigToModel(config, &data)

	// Set the singleton ID
	data.ID = types.StringValue("virt_config")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VirtConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VirtConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.services.Virt.GetGlobalConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read LXC Config",
			fmt.Sprintf("Unable to read virtualization configuration: %s", err.Error()),
		)
		return
	}

	r.mapConfigToModel(config, &data)

	// Ensure ID is set
	data.ID = types.StringValue("virt_config")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VirtConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VirtConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build opts and call API
	opts := r.buildConfigOpts(&plan)
	config, err := r.services.Virt.UpdateGlobalConfig(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update LXC Config",
			fmt.Sprintf("Unable to update virtualization configuration: %s", err.Error()),
		)
		return
	}

	// Map response to model
	r.mapConfigToModel(config, &plan)

	// Ensure ID is set
	plan.ID = types.StringValue("virt_config")

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VirtConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Reset to defaults by setting all fields to empty strings
	empty := ""
	opts := truenas.UpdateVirtGlobalConfigOpts{
		Bridge:    &empty,
		V4Network: &empty,
		V6Network: &empty,
		Pool:      &empty,
	}

	_, err := r.services.Virt.UpdateGlobalConfig(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Reset LXC Config",
			fmt.Sprintf("Unable to reset virtualization configuration: %s", err.Error()),
		)
		return
	}
}

func (r *VirtConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Validate the import ID - must be "virt_config"
	if req.ID != "virt_config" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID 'virt_config', got %q. This resource is a singleton.", req.ID),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// buildConfigOpts builds the API opts from the resource model.
// Only non-null terraform fields are set as pointers.
func (r *VirtConfigResource) buildConfigOpts(data *VirtConfigResourceModel) truenas.UpdateVirtGlobalConfigOpts {
	var opts truenas.UpdateVirtGlobalConfigOpts

	if !data.Bridge.IsNull() {
		opts.Bridge = truenas.StringPtr(data.Bridge.ValueString())
	}
	if !data.V4Network.IsNull() {
		opts.V4Network = truenas.StringPtr(data.V4Network.ValueString())
	}
	if !data.V6Network.IsNull() {
		opts.V6Network = truenas.StringPtr(data.V6Network.ValueString())
	}
	if !data.Pool.IsNull() {
		opts.Pool = truenas.StringPtr(data.Pool.ValueString())
	}

	return opts
}

// mapConfigToModel maps a VirtGlobalConfig to the resource model.
// Empty strings from the API are mapped to null in the terraform model.
func (r *VirtConfigResource) mapConfigToModel(config *truenas.VirtGlobalConfig, data *VirtConfigResourceModel) {
	if config.Bridge != "" {
		data.Bridge = types.StringValue(config.Bridge)
	} else {
		data.Bridge = types.StringNull()
	}

	if config.V4Network != "" {
		data.V4Network = types.StringValue(config.V4Network)
	} else {
		data.V4Network = types.StringNull()
	}

	if config.V6Network != "" {
		data.V6Network = types.StringValue(config.V6Network)
	} else {
		data.V6Network = types.StringNull()
	}

	if config.Pool != "" {
		data.Pool = types.StringValue(config.Pool)
	} else {
		data.Pool = types.StringNull()
	}
}
