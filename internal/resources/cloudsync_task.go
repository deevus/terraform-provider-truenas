package resources

import (
	"context"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                = &CloudSyncTaskResource{}
	_ resource.ResourceWithConfigure   = &CloudSyncTaskResource{}
	_ resource.ResourceWithImportState = &CloudSyncTaskResource{}
)

// CloudSyncTaskResource defines the resource implementation.
type CloudSyncTaskResource struct {
	client client.Client
}

// NewCloudSyncTaskResource creates a new CloudSyncTaskResource.
func NewCloudSyncTaskResource() resource.Resource {
	return &CloudSyncTaskResource{}
}

func (r *CloudSyncTaskResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloudsync_task"
}

func (r *CloudSyncTaskResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// TODO: Implement schema
}

func (r *CloudSyncTaskResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CloudSyncTaskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// TODO: Implement
}

func (r *CloudSyncTaskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// TODO: Implement
}

func (r *CloudSyncTaskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO: Implement
}

func (r *CloudSyncTaskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: Implement
}

func (r *CloudSyncTaskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
