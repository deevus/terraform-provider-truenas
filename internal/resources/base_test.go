package resources

import (
	"context"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestBaseResource_Configure_NilProviderData(t *testing.T) {
	b := &BaseResource{}

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	b.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if b.services != nil {
		t.Error("expected services to remain nil")
	}
}

func TestBaseResource_Configure_WrongType(t *testing.T) {
	b := &BaseResource{}

	req := resource.ConfigureRequest{
		ProviderData: "not a services",
	}
	resp := &resource.ConfigureResponse{}

	b.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}

	found := false
	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == "Unexpected Resource Configure Type" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Unexpected Resource Configure Type' diagnostic")
	}
}

func TestBaseResource_Configure_Success(t *testing.T) {
	b := &BaseResource{}

	services := &services.TrueNASServices{}

	req := resource.ConfigureRequest{
		ProviderData: services,
	}
	resp := &resource.ConfigureResponse{}

	b.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if b.services != services {
		t.Error("expected services to be set")
	}
}

func TestBaseResource_ImportState(t *testing.T) {
	b := &BaseResource{}

	idType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id": tftypes.String,
		},
	}

	rawState := tftypes.NewValue(idType, map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, "test-id"),
	})

	req := resource.ImportStateRequest{
		ID: "imported-id",
	}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Raw:    rawState,
			Schema: importStateTestSchema(),
		},
	}

	b.ImportState(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func importStateTestSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}
