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

var _ datasource.DataSource = &PoolDataSource{}
var _ datasource.DataSourceWithConfigure = &PoolDataSource{}

// PoolDataSource defines the data source implementation.
type PoolDataSource struct {
	client client.Client
}

// PoolDataSourceModel describes the data source data model.
type PoolDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Path           types.String `tfsdk:"path"`
	Status         types.String `tfsdk:"status"`
	AvailableBytes types.Int64  `tfsdk:"available_bytes"`
	UsedBytes      types.Int64  `tfsdk:"used_bytes"`
}

// poolResponse represents the JSON response from pool.query.
type poolResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Status    string `json:"status"`
	Size      int64  `json:"size"`
	Allocated int64  `json:"allocated"`
	Free      int64  `json:"free"`
}

// NewPoolDataSource creates a new PoolDataSource.
func NewPoolDataSource() datasource.DataSource {
	return &PoolDataSource{}
}

func (d *PoolDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pool"
}

func (d *PoolDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a TrueNAS storage pool.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the pool.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the pool to look up.",
				Required:    true,
			},
			"path": schema.StringAttribute{
				Description: "The mount path of the pool.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The status of the pool (e.g., ONLINE, DEGRADED, OFFLINE).",
				Computed:    true,
			},
			"available_bytes": schema.Int64Attribute{
				Description: "The available space in the pool in bytes.",
				Computed:    true,
			},
			"used_bytes": schema.Int64Attribute{
				Description: "The used space in the pool in bytes.",
				Computed:    true,
			},
		},
	}
}

func (d *PoolDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PoolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PoolDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build filter params: [["name", "=", "poolname"]]
	filters := [][]string{{"name", "=", data.Name.ValueString()}}

	// Call the TrueNAS API
	result, err := d.client.Call(ctx, "pool.query", filters)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Pool",
			fmt.Sprintf("Unable to read pool %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Parse the response
	var pools []poolResponse
	if err := json.Unmarshal(result, &pools); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Pool Response",
			fmt.Sprintf("Unable to parse pool response: %s", err.Error()),
		)
		return
	}

	// Check if pool was found
	if len(pools) == 0 {
		resp.Diagnostics.AddError(
			"Pool Not Found",
			fmt.Sprintf("Pool %q was not found.", data.Name.ValueString()),
		)
		return
	}

	pool := pools[0]

	// Map response to model
	data.ID = types.StringValue(fmt.Sprintf("%d", pool.ID))
	data.Name = types.StringValue(pool.Name)
	data.Path = types.StringValue(pool.Path)
	data.Status = types.StringValue(pool.Status)
	data.AvailableBytes = types.Int64Value(pool.Free)
	data.UsedBytes = types.Int64Value(pool.Allocated)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
