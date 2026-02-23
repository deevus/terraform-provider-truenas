package datasources

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SnapshotsDataSource{}
var _ datasource.DataSourceWithConfigure = &SnapshotsDataSource{}

// SnapshotsDataSource defines the data source implementation.
type SnapshotsDataSource struct {
	services *services.TrueNASServices
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

func (d *SnapshotsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SnapshotsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// List all snapshots via the service
	snapshots, err := d.services.Snapshot.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Snapshots",
			fmt.Sprintf("Unable to query snapshots: %s", err.Error()),
		)
		return
	}

	// Filter by dataset
	datasetID := data.DatasetID.ValueString()
	recursive := !data.Recursive.IsNull() && data.Recursive.ValueBool()
	namePattern := data.NamePattern.ValueString()

	data.Snapshots = make([]SnapshotModel, 0, len(snapshots))
	for _, snap := range snapshots {
		// Filter by dataset ID
		if recursive {
			if snap.Dataset != datasetID && !strings.HasPrefix(snap.Dataset, datasetID+"/") {
				continue
			}
		} else {
			if snap.Dataset != datasetID {
				continue
			}
		}

		// Apply name pattern filter
		if namePattern != "" {
			matched, err := filepath.Match(namePattern, snap.SnapshotName)
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
			Name:            types.StringValue(snap.SnapshotName),
			DatasetID:       types.StringValue(snap.Dataset),
			UsedBytes:       types.Int64Value(snap.Used),
			ReferencedBytes: types.Int64Value(snap.Referenced),
			Hold:            types.BoolValue(snap.HasHold),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
