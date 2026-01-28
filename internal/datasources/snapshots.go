package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/deevus/terraform-provider-truenas/internal/api"
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
	var data SnapshotsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get TrueNAS version for API method resolution
	version, ok := api.GetVersionOrDiag(ctx, d.client, &resp.Diagnostics)
	if !ok {
		return
	}

	// Build filter for dataset
	datasetID := data.DatasetID.ValueString()
	filter := [][]any{{"dataset", "=", datasetID}}

	// If recursive, match exact dataset OR child datasets (dataset/)
	if !data.Recursive.IsNull() && data.Recursive.ValueBool() {
		filter = [][]any{
			{"OR", [][]any{
				{"dataset", "=", datasetID},
				{"dataset", "^", datasetID + "/"},
			}},
		}
	}

	method := api.ResolveSnapshotMethod(version, api.MethodSnapshotQuery)
	result, err := d.client.Call(ctx, method, filter)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Snapshots",
			fmt.Sprintf("Unable to query snapshots: %s", err.Error()),
		)
		return
	}

	var snapshots []api.SnapshotResponse
	if err := json.Unmarshal(result, &snapshots); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Snapshots Response",
			fmt.Sprintf("Unable to parse snapshots response: %s", err.Error()),
		)
		return
	}

	// Filter by name pattern if specified
	namePattern := data.NamePattern.ValueString()

	data.Snapshots = make([]SnapshotModel, 0, len(snapshots))
	for _, snap := range snapshots {
		// Apply name pattern filter
		if namePattern != "" {
			matched, err := filepath.Match(namePattern, snap.Name)
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Name Pattern",
					fmt.Sprintf("Invalid glob pattern %q: %s", namePattern, err.Error()),
				)
				return
			}
			if !matched {
				continue
			}
		}

		data.Snapshots = append(data.Snapshots, SnapshotModel{
			ID:              types.StringValue(snap.ID),
			Name:            types.StringValue(snap.SnapshotName), // Use SnapshotName, not Name (which is full ID)
			DatasetID:       types.StringValue(snap.Dataset),
			UsedBytes:       types.Int64Value(snap.Properties.Used.Parsed),
			ReferencedBytes: types.Int64Value(snap.Properties.Referenced.Parsed),
			Hold:            types.BoolValue(snap.HasHold()),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
