package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
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

// virtConfigAPIResponse represents the JSON response from virt.global.config.
type virtConfigAPIResponse struct {
	Bridge    *string `json:"bridge"`
	V4Network *string `json:"v4_network"`
	V6Network *string `json:"v6_network"`
	Pool      *string `json:"pool"`
}

// VirtConfigResource defines the resource implementation.
type VirtConfigResource struct {
	client client.Client
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

func (r *VirtConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VirtConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VirtConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build params
	params := r.buildParams(&data)

	// Call API - Create uses virt.global.update
	_, err := r.client.Call(ctx, "virt.global.update", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update LXC Config",
			fmt.Sprintf("Unable to update virtualization configuration: %s", err.Error()),
		)
		return
	}

	// Read back the config to get the actual state
	config, err := r.readConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read LXC Config",
			fmt.Sprintf("virtualization config updated but unable to read: %s", err.Error()),
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

	config, err := r.readConfig(ctx)
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

	// Build params from plan
	params := r.buildParams(&plan)

	// Call API
	_, err := r.client.Call(ctx, "virt.global.update", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update LXC Config",
			fmt.Sprintf("Unable to update virtualization configuration: %s", err.Error()),
		)
		return
	}

	// Read back the config to get the actual state
	config, err := r.readConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read LXC Config",
			fmt.Sprintf("virtualization config updated but unable to read: %s", err.Error()),
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
	// Reset to defaults by setting all fields to nil
	params := map[string]any{
		"bridge":     nil,
		"v4_network": nil,
		"v6_network": nil,
		"pool":       nil,
	}

	_, err := r.client.Call(ctx, "virt.global.update", params)
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

// readConfig queries the virtualization config from the API.
func (r *VirtConfigResource) readConfig(ctx context.Context) (*virtConfigAPIResponse, error) {
	result, err := r.client.Call(ctx, "virt.global.config", nil)
	if err != nil {
		return nil, err
	}

	var config virtConfigAPIResponse
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &config, nil
}

// buildParams builds the API params from the resource model.
func (r *VirtConfigResource) buildParams(data *VirtConfigResourceModel) map[string]any {
	params := map[string]any{}

	if !data.Bridge.IsNull() {
		params["bridge"] = data.Bridge.ValueString()
	}
	if !data.V4Network.IsNull() {
		params["v4_network"] = data.V4Network.ValueString()
	}
	if !data.V6Network.IsNull() {
		params["v6_network"] = data.V6Network.ValueString()
	}
	if !data.Pool.IsNull() {
		params["pool"] = data.Pool.ValueString()
	}

	return params
}

// mapConfigToModel maps an API response to the resource model.
func (r *VirtConfigResource) mapConfigToModel(config *virtConfigAPIResponse, data *VirtConfigResourceModel) {
	if config.Bridge != nil {
		data.Bridge = types.StringValue(*config.Bridge)
	} else {
		data.Bridge = types.StringNull()
	}

	if config.V4Network != nil {
		data.V4Network = types.StringValue(*config.V4Network)
	} else {
		data.V4Network = types.StringNull()
	}

	if config.V6Network != nil {
		data.V6Network = types.StringValue(*config.V6Network)
	} else {
		data.V6Network = types.StringNull()
	}

	if config.Pool != nil {
		data.Pool = types.StringValue(*config.Pool)
	} else {
		data.Pool = types.StringNull()
	}
}
