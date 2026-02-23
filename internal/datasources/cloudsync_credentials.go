package datasources

import (
	"context"
	"fmt"
	"strings"

	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CloudSyncCredentialsDataSource{}
var _ datasource.DataSourceWithConfigure = &CloudSyncCredentialsDataSource{}

// CloudSyncCredentialsDataSource defines the data source implementation.
type CloudSyncCredentialsDataSource struct {
	services *services.TrueNASServices
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

func (d *CloudSyncCredentialsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CloudSyncCredentialsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// List all credentials via the service
	credentials, err := d.services.CloudSync.ListCredentials(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Cloud Sync Credentials",
			fmt.Sprintf("Unable to read cloud sync credentials: %s", err.Error()),
		)
		return
	}

	// Find the credential with matching name
	searchName := data.Name.ValueString()
	found := false
	for _, cred := range credentials {
		if cred.Name == searchName {
			data.ID = types.StringValue(fmt.Sprintf("%d", cred.ID))
			data.Name = types.StringValue(cred.Name)
			data.ProviderType = types.StringValue(mapAPIProviderToTerraform(cred.ProviderType))
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"Credentials Not Found",
			fmt.Sprintf("Cloud sync credentials %q was not found.", searchName),
		)
		return
	}

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
