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

var _ datasource.DataSource = &DatasetDataSource{}
var _ datasource.DataSourceWithConfigure = &DatasetDataSource{}

// DatasetDataSource defines the data source implementation.
type DatasetDataSource struct {
	client client.Client
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

// datasetResponse represents the JSON response from pool.dataset.query.
type datasetResponse struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Pool        string           `json:"pool"`
	Mountpoint  string           `json:"mountpoint"`
	Compression compressionValue `json:"compression"`
	Used        parsedValue      `json:"used"`
	Available   parsedValue      `json:"available"`
}

type compressionValue struct {
	Value string `json:"value"`
}

type parsedValue struct {
	Parsed int64 `json:"parsed"`
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

func (d *DatasetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DatasetDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build full dataset name (pool/path)
	fullName := fmt.Sprintf("%s/%s", data.Pool.ValueString(), data.Path.ValueString())

	// Build filter params: [["id", "=", "pool/path"]]
	filter := [][]any{{"id", "=", fullName}}

	// Call the TrueNAS API
	result, err := d.client.Call(ctx, "pool.dataset.query", filter)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Dataset",
			fmt.Sprintf("Unable to read dataset %q: %s", fullName, err.Error()),
		)
		return
	}

	// Parse the response
	var datasets []datasetResponse
	if err := json.Unmarshal(result, &datasets); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Dataset Response",
			fmt.Sprintf("Unable to parse dataset response: %s", err.Error()),
		)
		return
	}

	// Check if dataset was found
	if len(datasets) == 0 {
		resp.Diagnostics.AddError(
			"Dataset Not Found",
			fmt.Sprintf("Dataset %q was not found.", fullName),
		)
		return
	}

	ds := datasets[0]

	// Map response to model
	data.ID = types.StringValue(ds.ID)
	data.MountPath = types.StringValue(ds.Mountpoint)
	data.Compression = types.StringValue(ds.Compression.Value)
	data.UsedBytes = types.Int64Value(ds.Used.Parsed)
	data.AvailableBytes = types.Int64Value(ds.Available.Parsed)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
