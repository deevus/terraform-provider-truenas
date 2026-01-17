package resources

import (
	"context"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &CloudSyncCredentialsResource{}
var _ resource.ResourceWithConfigure = &CloudSyncCredentialsResource{}
var _ resource.ResourceWithImportState = &CloudSyncCredentialsResource{}

// CloudSyncCredentialsResourceModel describes the resource data model.
type CloudSyncCredentialsResourceModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	S3    *S3Block     `tfsdk:"s3"`
	B2    *B2Block     `tfsdk:"b2"`
	GCS   *GCSBlock    `tfsdk:"gcs"`
	Azure *AzureBlock  `tfsdk:"azure"`
}

// S3Block represents S3 credentials.
type S3Block struct {
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
	Endpoint        types.String `tfsdk:"endpoint"`
	Region          types.String `tfsdk:"region"`
}

// B2Block represents Backblaze B2 credentials.
type B2Block struct {
	Account types.String `tfsdk:"account"`
	Key     types.String `tfsdk:"key"`
}

// GCSBlock represents Google Cloud Storage credentials.
type GCSBlock struct {
	ServiceAccountCredentials types.String `tfsdk:"service_account_credentials"`
}

// AzureBlock represents Azure Blob Storage credentials.
type AzureBlock struct {
	Account types.String `tfsdk:"account"`
	Key     types.String `tfsdk:"key"`
}

// CloudSyncCredentialsResource defines the resource implementation.
type CloudSyncCredentialsResource struct {
	client client.Client
}

// NewCloudSyncCredentialsResource creates a new CloudSyncCredentialsResource.
func NewCloudSyncCredentialsResource() resource.Resource {
	return &CloudSyncCredentialsResource{}
}

func (r *CloudSyncCredentialsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloudsync_credentials"
}

func (r *CloudSyncCredentialsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages cloud sync credentials for backup tasks.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Credential ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Credential name.",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"s3": schema.SingleNestedBlock{
				Description: "S3-compatible storage credentials.",
				Attributes: map[string]schema.Attribute{
					"access_key_id": schema.StringAttribute{
						Description: "Access key ID.",
						Required:    true,
						Sensitive:   true,
					},
					"secret_access_key": schema.StringAttribute{
						Description: "Secret access key.",
						Required:    true,
						Sensitive:   true,
					},
					"endpoint": schema.StringAttribute{
						Description: "Custom endpoint URL for S3-compatible storage.",
						Optional:    true,
					},
					"region": schema.StringAttribute{
						Description: "Region.",
						Optional:    true,
					},
				},
			},
			"b2": schema.SingleNestedBlock{
				Description: "Backblaze B2 credentials.",
				Attributes: map[string]schema.Attribute{
					"account": schema.StringAttribute{
						Description: "Account ID.",
						Required:    true,
						Sensitive:   true,
					},
					"key": schema.StringAttribute{
						Description: "Application key.",
						Required:    true,
						Sensitive:   true,
					},
				},
			},
			"gcs": schema.SingleNestedBlock{
				Description: "Google Cloud Storage credentials.",
				Attributes: map[string]schema.Attribute{
					"service_account_credentials": schema.StringAttribute{
						Description: "Service account JSON credentials.",
						Required:    true,
						Sensitive:   true,
					},
				},
			},
			"azure": schema.SingleNestedBlock{
				Description: "Azure Blob Storage credentials.",
				Attributes: map[string]schema.Attribute{
					"account": schema.StringAttribute{
						Description: "Storage account name.",
						Required:    true,
						Sensitive:   true,
					},
					"key": schema.StringAttribute{
						Description: "Account key.",
						Required:    true,
						Sensitive:   true,
					},
				},
			},
		},
	}
}

func (r *CloudSyncCredentialsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *CloudSyncCredentialsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
