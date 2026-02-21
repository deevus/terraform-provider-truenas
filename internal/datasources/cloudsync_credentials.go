package datasources

import (
	"context"
	"fmt"
	"strings"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/truenas-go/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CloudSyncCredentialsDataSource{}
var _ datasource.DataSourceWithConfigure = &CloudSyncCredentialsDataSource{}

// CloudSyncCredentialsDataSource defines the data source implementation.
type CloudSyncCredentialsDataSource struct {
	client client.Client
}

// CloudSyncCredentialsDataSourceModel describes the data source data model.
type CloudSyncCredentialsDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	ProviderType types.String `tfsdk:"provider_type"`
}

// NewCloudSyncCredentialsDataSource creates a new CloudSyncCredentialsDataSource.
func NewCloudSyncCredentialsDataSource() datasource.DataSource {
	return &CloudSyncCredentialsDataSource{}
}

func (d *CloudSyncCredentialsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloudsync_credentials"
}

func (d *CloudSyncCredentialsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about TrueNAS cloud sync credentials by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the credentials.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the cloud sync credentials to look up.",
				Required:    true,
			},
			"provider_type": schema.StringAttribute{
				Description: "The type of cloud provider (s3, b2, gcs, azure).",
				Computed:    true,
			},
		},
	}
}

func (d *CloudSyncCredentialsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CloudSyncCredentialsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CloudSyncCredentialsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get version for API compatibility
	version := d.client.Version()

	// Query all credentials (no filter - API returns all)
	result, err := d.client.Call(ctx, "cloudsync.credentials.query", [][]any{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Cloud Sync Credentials",
			fmt.Sprintf("Unable to read cloud sync credentials: %s", err.Error()),
		)
		return
	}

	// Parse the response using version-aware parser
	credentials, err := truenas.ParseCredentials(result, version)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Credentials Response",
			fmt.Sprintf("Unable to parse credentials response: %s", err.Error()),
		)
		return
	}

	// Find the credential with matching name
	var found *truenas.CloudSyncCredentialResponse
	searchName := data.Name.ValueString()
	for i := range credentials {
		if credentials[i].Name == searchName {
			found = &credentials[i]
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError(
			"Credentials Not Found",
			fmt.Sprintf("Cloud sync credentials %q was not found.", searchName),
		)
		return
	}

	// Map response to model
	data.ID = types.StringValue(fmt.Sprintf("%d", found.ID))
	data.Name = types.StringValue(found.Name)
	data.ProviderType = types.StringValue(mapAPIProviderToTerraform(found.Provider.Type))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// mapAPIProviderToTerraform maps API provider values to lowercase Terraform values.
func mapAPIProviderToTerraform(provider string) string {
	switch provider {
	case "S3":
		return "s3"
	case "B2":
		return "b2"
	case "GOOGLE_CLOUD_STORAGE":
		return "gcs"
	case "AZUREBLOB":
		return "azure"
	default:
		// Return unknown providers as-is in lowercase
		return strings.ToLower(provider)
	}
}
