package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/api"
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

// queryCredential queries a credential by ID and returns the response.
func (r *CloudSyncCredentialsResource) queryCredential(ctx context.Context, id int64) (*api.CloudSyncCredentialResponse, error) {
	filter := [][]any{{"id", "=", id}}
	result, err := r.client.Call(ctx, "cloudsync.credentials.query", filter)
	if err != nil {
		return nil, err
	}

	var credentials []api.CloudSyncCredentialResponse
	if err := json.Unmarshal(result, &credentials); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(credentials) == 0 {
		return nil, nil
	}

	return &credentials[0], nil
}

// getProviderAndAttributes extracts provider type and attributes from the model.
func getProviderAndAttributes(data *CloudSyncCredentialsResourceModel) (string, map[string]any) {
	if data.S3 != nil {
		attrs := map[string]any{
			"access_key_id":     data.S3.AccessKeyID.ValueString(),
			"secret_access_key": data.S3.SecretAccessKey.ValueString(),
		}
		if !data.S3.Endpoint.IsNull() {
			attrs["endpoint"] = data.S3.Endpoint.ValueString()
		}
		if !data.S3.Region.IsNull() {
			attrs["region"] = data.S3.Region.ValueString()
		}
		return "S3", attrs
	}
	if data.B2 != nil {
		return "B2", map[string]any{
			"account": data.B2.Account.ValueString(),
			"key":     data.B2.Key.ValueString(),
		}
	}
	if data.GCS != nil {
		return "GOOGLE_CLOUD_STORAGE", map[string]any{
			"service_account_credentials": data.GCS.ServiceAccountCredentials.ValueString(),
		}
	}
	if data.Azure != nil {
		return "AZUREBLOB", map[string]any{
			"account": data.Azure.Account.ValueString(),
			"key":     data.Azure.Key.ValueString(),
		}
	}
	return "", nil
}

func (r *CloudSyncCredentialsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CloudSyncCredentialsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	provider, attributes := getProviderAndAttributes(&data)
	if provider == "" {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Exactly one provider block (s3, b2, gcs, or azure) must be specified.",
		)
		return
	}

	params := map[string]any{
		"name":       data.Name.ValueString(),
		"provider":   provider,
		"attributes": attributes,
	}

	result, err := r.client.Call(ctx, "cloudsync.credentials.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Cloud Sync Credentials",
			fmt.Sprintf("Unable to create credentials: %s", err.Error()),
		)
		return
	}

	// Parse response to get ID
	var createResp struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(result, &createResp); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Response",
			fmt.Sprintf("Unable to parse create response: %s", err.Error()),
		)
		return
	}

	// Query to get full state
	cred, err := r.queryCredential(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Credentials",
			fmt.Sprintf("Credentials created but unable to read: %s", err.Error()),
		)
		return
	}

	if cred == nil {
		resp.Diagnostics.AddError(
			"Credentials Not Found",
			"Credentials were created but could not be found.",
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d", cred.ID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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
