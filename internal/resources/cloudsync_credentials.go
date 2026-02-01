package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

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
						Optional:    true,
						Sensitive:   true,
					},
					"secret_access_key": schema.StringAttribute{
						Description: "Secret access key.",
						Optional:    true,
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
						Optional:    true,
						Sensitive:   true,
					},
					"key": schema.StringAttribute{
						Description: "Application key.",
						Optional:    true,
						Sensitive:   true,
					},
				},
			},
			"gcs": schema.SingleNestedBlock{
				Description: "Google Cloud Storage credentials.",
				Attributes: map[string]schema.Attribute{
					"service_account_credentials": schema.StringAttribute{
						Description: "Service account JSON credentials.",
						Optional:    true,
						Sensitive:   true,
					},
				},
			},
			"azure": schema.SingleNestedBlock{
				Description: "Azure Blob Storage credentials.",
				Attributes: map[string]schema.Attribute{
					"account": schema.StringAttribute{
						Description: "Storage account name.",
						Optional:    true,
						Sensitive:   true,
					},
					"key": schema.StringAttribute{
						Description: "Account key.",
						Optional:    true,
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
func (r *CloudSyncCredentialsResource) queryCredential(ctx context.Context, id int64, version api.Version) (*api.CloudSyncCredentialResponse, error) {
	filter := [][]any{{"id", "=", id}}
	result, err := r.client.Call(ctx, "cloudsync.credentials.query", filter)
	if err != nil {
		return nil, err
	}

	credentials, err := api.ParseCredentials(result, version)
	if err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(credentials) == 0 {
		return nil, nil
	}

	return &credentials[0], nil
}

// validateProviderBlock validates that required fields are present in the specified provider block.
func validateProviderBlock(data *CloudSyncCredentialsResourceModel) []string {
	var errors []string

	if data.S3 != nil {
		if data.S3.AccessKeyID.IsNull() || data.S3.AccessKeyID.ValueString() == "" {
			errors = append(errors, "s3.access_key_id is required when s3 block is specified")
		}
		if data.S3.SecretAccessKey.IsNull() || data.S3.SecretAccessKey.ValueString() == "" {
			errors = append(errors, "s3.secret_access_key is required when s3 block is specified")
		}
	}
	if data.B2 != nil {
		if data.B2.Account.IsNull() || data.B2.Account.ValueString() == "" {
			errors = append(errors, "b2.account is required when b2 block is specified")
		}
		if data.B2.Key.IsNull() || data.B2.Key.ValueString() == "" {
			errors = append(errors, "b2.key is required when b2 block is specified")
		}
	}
	if data.GCS != nil {
		if data.GCS.ServiceAccountCredentials.IsNull() || data.GCS.ServiceAccountCredentials.ValueString() == "" {
			errors = append(errors, "gcs.service_account_credentials is required when gcs block is specified")
		}
	}
	if data.Azure != nil {
		if data.Azure.Account.IsNull() || data.Azure.Account.ValueString() == "" {
			errors = append(errors, "azure.account is required when azure block is specified")
		}
		if data.Azure.Key.IsNull() || data.Azure.Key.ValueString() == "" {
			errors = append(errors, "azure.key is required when azure block is specified")
		}
	}

	return errors
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

	// Validate required fields within provider blocks
	if validationErrors := validateProviderBlock(&data); len(validationErrors) > 0 {
		for _, err := range validationErrors {
			resp.Diagnostics.AddError("Invalid Configuration", err)
		}
		return
	}

	providerType, attributes := getProviderAndAttributes(&data)
	if providerType == "" {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Exactly one provider block (s3, b2, gcs, or azure) must be specified.",
		)
		return
	}

	// Get version for API compatibility
	version := r.client.Version()

	params := api.BuildCredentialsParams(version, data.Name.ValueString(), providerType, attributes)

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
	cred, err := r.queryCredential(ctx, createResp.ID, version)
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
	var data CloudSyncCredentialsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	// Get version for API compatibility
	version := r.client.Version()

	cred, err := r.queryCredential(ctx, id, version)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Credentials",
			fmt.Sprintf("Unable to read credentials %q: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	if cred == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(cred.Name)
	// Note: We preserve the existing block values since the API returns
	// sensitive data that we don't want to overwrite from state

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CloudSyncCredentialsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state CloudSyncCredentialsResourceModel
	var plan CloudSyncCredentialsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate required fields within provider blocks
	if validationErrors := validateProviderBlock(&plan); len(validationErrors) > 0 {
		for _, err := range validationErrors {
			resp.Diagnostics.AddError("Invalid Configuration", err)
		}
		return
	}

	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	providerType, attributes := getProviderAndAttributes(&plan)
	if providerType == "" {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Exactly one provider block (s3, b2, gcs, or azure) must be specified.",
		)
		return
	}

	// Get version for API compatibility
	version := r.client.Version()

	updateData := api.BuildCredentialsParams(version, plan.Name.ValueString(), providerType, attributes)

	_, err = r.client.Call(ctx, "cloudsync.credentials.update", []any{id, updateData})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Credentials",
			fmt.Sprintf("Unable to update credentials: %s", err.Error()),
		)
		return
	}

	// Query to refresh state
	cred, err := r.queryCredential(ctx, id, version)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Credentials",
			fmt.Sprintf("Credentials updated but unable to read: %s", err.Error()),
		)
		return
	}

	if cred == nil {
		resp.Diagnostics.AddError(
			"Credentials Not Found",
			"Credentials were updated but could not be found.",
		)
		return
	}

	plan.ID = state.ID
	plan.Name = types.StringValue(cred.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CloudSyncCredentialsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CloudSyncCredentialsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	_, err = r.client.Call(ctx, "cloudsync.credentials.delete", id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Credentials",
			fmt.Sprintf("Unable to delete credentials: %s", err.Error()),
		)
		return
	}
}

func (r *CloudSyncCredentialsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
