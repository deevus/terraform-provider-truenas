package resources

import (
	"context"
	"fmt"

	"github.com/deevus/truenas-go/client"
	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// BaseResource provides shared Configure and ImportState behavior for all resources.
// Embed this in resource structs to inherit the services field and standard methods.
type BaseResource struct {
	services *services.TrueNASServices

	// client provides backward-compatible access to the raw client.Client
	// for resources that haven't been migrated to typed services yet.
	// Migrated resources should use r.services.<Service> instead.
	// Remove this field once all resources use typed service methods.
	client client.Client
}

func (b *BaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	s, ok := req.ProviderData.(*services.TrueNASServices)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *services.TrueNASServices, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	b.services = s
	b.client = s.Client
}

func (b *BaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
