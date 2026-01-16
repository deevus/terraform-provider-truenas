package datasources

import (
	"context"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SnapshotsDataSource{}
var _ datasource.DataSourceWithConfigure = &SnapshotsDataSource{}

// SnapshotsDataSource defines the data source implementation.
type SnapshotsDataSource struct {
	client client.Client
}

// SnapshotsDataSourceModel describes the data source data model.
type SnapshotsDataSourceModel struct {
	DatasetID   types.String    `tfsdk:"dataset_id"`
	Recursive   types.Bool      `tfsdk:"recursive"`
	NamePattern types.String    `tfsdk:"name_pattern"`
	Snapshots   []SnapshotModel `tfsdk:"snapshots"`
}

// SnapshotModel represents a snapshot in the list.
type SnapshotModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	DatasetID       types.String `tfsdk:"dataset_id"`
	UsedBytes       types.Int64  `tfsdk:"used_bytes"`
	ReferencedBytes types.Int64  `tfsdk:"referenced_bytes"`
	Hold            types.Bool   `tfsdk:"hold"`
}

// NewSnapshotsDataSource creates a new SnapshotsDataSource.
func NewSnapshotsDataSource() datasource.DataSource {
	return &SnapshotsDataSource{}
}

func (d *SnapshotsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshots"
}

func (d *SnapshotsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves snapshots for a dataset.",
		Attributes: map[string]schema.Attribute{
			"dataset_id": schema.StringAttribute{
				Description: "Dataset ID to query snapshots for.",
				Required:    true,
			},
			"recursive": schema.BoolAttribute{
				Description: "Include child dataset snapshots. Default: false.",
				Optional:    true,
			},
			"name_pattern": schema.StringAttribute{
				Description: "Glob pattern to filter snapshot names.",
				Optional:    true,
			},
			"snapshots": schema.ListNestedAttribute{
				Description: "List of snapshots.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Snapshot ID (dataset@name).",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Snapshot name.",
							Computed:    true,
						},
						"dataset_id": schema.StringAttribute{
							Description: "Parent dataset ID.",
							Computed:    true,
						},
						"used_bytes": schema.Int64Attribute{
							Description: "Space consumed by snapshot.",
							Computed:    true,
						},
						"referenced_bytes": schema.Int64Attribute{
							Description: "Space referenced by snapshot.",
							Computed:    true,
						},
						"hold": schema.BoolAttribute{
							Description: "Whether snapshot is held.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *SnapshotsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SnapshotsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// TODO: implement
	resp.Diagnostics.AddError("Not Implemented", "Read not yet implemented")
}
