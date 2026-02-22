package resources

import (
	"context"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// BaseResource provides shared Configure and ImportState behavior for all resources.
// Embed this in resource structs to inherit the services field and standard methods.
type BaseResource struct {
	services *services.TrueNASServices
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
}

func (b *BaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
