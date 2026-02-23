package datasources

import (
	"context"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VirtConfigDataSource{}
var _ datasource.DataSourceWithConfigure = &VirtConfigDataSource{}

// VirtConfigDataSource defines the data source implementation.
type VirtConfigDataSource struct {
	services *services.TrueNASServices
}

// VirtConfigDataSourceModel describes the data source data model.
type VirtConfigDataSourceModel struct {
	Bridge    types.String `tfsdk:"bridge"`
	V4Network types.String `tfsdk:"v4_network"`
	V6Network types.String `tfsdk:"v6_network"`
	Pool      types.String `tfsdk:"pool"`
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

	s, ok := req.ProviderData.(*services.TrueNASServices)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *services.TrueNASServices, got: %T.", req.ProviderData),
		)
		return
	}

	d.services = s
}

func (d *VirtConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VirtConfigDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get global config via the service
	config, err := d.services.Virt.GetGlobalConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read LXC Config",
			fmt.Sprintf("Unable to read virtualization configuration: %s", err.Error()),
		)
		return
	}

	// Map response to model, handling empty strings as null
	// VirtGlobalConfig converts nil pointer fields to empty strings,
	// so we treat empty strings as null for Terraform state.
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

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
