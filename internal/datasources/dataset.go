package datasources

import (
	"context"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DatasetDataSource{}
var _ datasource.DataSourceWithConfigure = &DatasetDataSource{}

// DatasetDataSource defines the data source implementation.
type DatasetDataSource struct {
	services *services.TrueNASServices
}

// DatasetDataSourceModel describes the data source data model.
type DatasetDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Pool           types.String `tfsdk:"pool"`
	Path           types.String `tfsdk:"path"`
	MountPath      types.String `tfsdk:"mount_path"`
	Compression    types.String `tfsdk:"compression"`
	UsedBytes      types.Int64  `tfsdk:"used_bytes"`
	AvailableBytes types.Int64  `tfsdk:"available_bytes"`
}

// NewDatasetDataSource creates a new DatasetDataSource.
func NewDatasetDataSource() datasource.DataSource {
	return &DatasetDataSource{}
}

func (d *DatasetDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

func (d *DatasetDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about an existing TrueNAS dataset.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Dataset identifier (pool/path).",
				Computed:    true,
			},
			"pool": schema.StringAttribute{
				Description: "Pool name.",
				Required:    true,
			},
			"path": schema.StringAttribute{
				Description: "Dataset path within the pool.",
				Required:    true,
			},
			"mount_path": schema.StringAttribute{
				Description: "Filesystem mount path.",
				Computed:    true,
			},
			"compression": schema.StringAttribute{
				Description: "Compression algorithm.",
				Computed:    true,
			},
			"used_bytes": schema.Int64Attribute{
				Description: "Space used in bytes.",
				Computed:    true,
			},
			"available_bytes": schema.Int64Attribute{
				Description: "Space available in bytes.",
				Computed:    true,
			},
		},
	}
}

func (d *DatasetDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DatasetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DatasetDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build full dataset name (pool/path)
	fullName := fmt.Sprintf("%s/%s", data.Pool.ValueString(), data.Path.ValueString())

	// Get the dataset via the service
	ds, err := d.services.Dataset.GetDataset(ctx, fullName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Dataset",
			fmt.Sprintf("Unable to read dataset %q: %s", fullName, err.Error()),
		)
		return
	}

	// Check if dataset was found
	if ds == nil {
		resp.Diagnostics.AddError(
			"Dataset Not Found",
			fmt.Sprintf("Dataset %q was not found.", fullName),
		)
		return
	}

	// Map response to model
	data.ID = types.StringValue(ds.ID)
	data.MountPath = types.StringValue(ds.Mountpoint)
	data.Compression = types.StringValue(ds.Compression)
	data.UsedBytes = types.Int64Value(ds.Used)
	data.AvailableBytes = types.Int64Value(ds.Available)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
