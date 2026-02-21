package datasources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deevus/truenas-go/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VirtConfigDataSource{}
var _ datasource.DataSourceWithConfigure = &VirtConfigDataSource{}

// VirtConfigDataSource defines the data source implementation.
type VirtConfigDataSource struct {
	client client.Client
}

// VirtConfigDataSourceModel describes the data source data model.
type VirtConfigDataSourceModel struct {
	Bridge        types.String `tfsdk:"bridge"`
	V4Network     types.String `tfsdk:"v4_network"`
	V6Network     types.String `tfsdk:"v6_network"`
	Pool types.String `tfsdk:"pool"`
}

// virtConfigResponse represents the JSON response from virt.global.config.
type virtConfigResponse struct {
	Bridge    *string `json:"bridge"`
	V4Network *string `json:"v4_network"`
	V6Network *string `json:"v6_network"`
	Pool      *string `json:"pool"`
}

// NewVirtConfigDataSource creates a new VirtConfigDataSource.
func NewVirtConfigDataSource() datasource.DataSource {
	return &VirtConfigDataSource{}
}

func (d *VirtConfigDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virt_config"
}

func (d *VirtConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the global virtualization configuration from TrueNAS.",
		Attributes: map[string]schema.Attribute{
			"bridge": schema.StringAttribute{
				Description: "The network bridge used for virtualizations.",
				Computed:    true,
			},
			"v4_network": schema.StringAttribute{
				Description: "The IPv4 network CIDR for virtualizations.",
				Computed:    true,
			},
			"v6_network": schema.StringAttribute{
				Description: "The IPv6 network CIDR for virtualizations.",
				Computed:    true,
			},
			"pool": schema.StringAttribute{
				Description: "The default storage pool for virtualizations.",
				Computed:    true,
			},
		},
	}
}

func (d *VirtConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
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

func (d *VirtConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VirtConfigDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call the TrueNAS API - virt.global.config takes no parameters
	result, err := d.client.Call(ctx, "virt.global.config", nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read LXC Config",
			fmt.Sprintf("Unable to read virtualization configuration: %s", err.Error()),
		)
		return
	}

	// Parse the response
	var config virtConfigResponse
	if err := json.Unmarshal(result, &config); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse LXC Config Response",
			fmt.Sprintf("Unable to parse virtualization config response: %s", err.Error()),
		)
		return
	}

	// Map response to model, handling nullable fields
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

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
