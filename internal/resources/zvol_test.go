package resources

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewZvolResource(t *testing.T) {
	r := NewZvolResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(r.(*ZvolResource))
	_ = resource.ResourceWithImportState(r.(*ZvolResource))
}

func TestZvolResource_Metadata(t *testing.T) {
	r := NewZvolResource().(*ZvolResource)
	req := resource.MetadataRequest{ProviderTypeName: "truenas"}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_zvol" {
		t.Errorf("expected type name 'truenas_zvol', got %q", resp.TypeName)
	}
}

func TestZvolResource_Schema(t *testing.T) {
	schemaResp := getZvolResourceSchema(t)

	expectedAttrs := []string{
		"id", "pool", "path", "parent",
		"volsize", "volblocksize", "sparse", "force_size",
		"compression", "comments",
		"force_destroy",
	}

	for _, attr := range expectedAttrs {
		if _, ok := schemaResp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing expected attribute %q", attr)
		}
	}
}

// -- Test helpers --

func getZvolResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewZvolResource().(*ZvolResource)
	resp := resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema error: %v", resp.Diagnostics)
	}
	return resp
}

// zvolObjectType returns the tftypes.Object for constructing test values.
func zvolObjectType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"pool":          tftypes.String,
			"path":          tftypes.String,
			"parent":        tftypes.String,
			"volsize":       tftypes.String,
			"volblocksize":  tftypes.String,
			"sparse":        tftypes.Bool,
			"force_size":    tftypes.Bool,
			"compression":   tftypes.String,
			"comments":      tftypes.String,
			"force_destroy": tftypes.Bool,
		},
	}
}

type zvolModelParams struct {
	ID           *string
	Pool         *string
	Path         *string
	Parent       *string
	Volsize      *string
	Volblocksize *string
	Sparse       *bool
	ForceSize    *bool
	Compression  *string
	Comments     *string
	ForceDestroy *bool
}

func createZvolModelValue(p zvolModelParams) tftypes.Value {
	strVal := func(s *string) tftypes.Value {
		if s == nil {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, *s)
	}
	boolVal := func(b *bool) tftypes.Value {
		if b == nil {
			return tftypes.NewValue(tftypes.Bool, nil)
		}
		return tftypes.NewValue(tftypes.Bool, *b)
	}

	return tftypes.NewValue(zvolObjectType(), map[string]tftypes.Value{
		"id":            strVal(p.ID),
		"pool":          strVal(p.Pool),
		"path":          strVal(p.Path),
		"parent":        strVal(p.Parent),
		"volsize":       strVal(p.Volsize),
		"volblocksize":  strVal(p.Volblocksize),
		"sparse":        boolVal(p.Sparse),
		"force_size":    boolVal(p.ForceSize),
		"compression":   strVal(p.Compression),
		"comments":      strVal(p.Comments),
		"force_destroy": boolVal(p.ForceDestroy),
	})
}

func boolPtr(b bool) *bool { return &b }

func defaultZvolPlanParams() zvolModelParams {
	return zvolModelParams{
		Pool:    strPtr("tank"),
		Path:    strPtr("myvol"),
		Volsize: strPtr("10737418240"),
	}
}

// mockZvolQueryResponse returns a mock pool.dataset.query response for a zvol.
func mockZvolQueryResponse(id, compression, comments string, volsize int64, volblocksize string, sparse bool) string {
	return fmt.Sprintf(`[{
		"id": %q,
		"type": "VOLUME",
		"name": %q,
		"pool": "tank",
		"volsize": {"value": "%d", "parsed": %d},
		"volblocksize": {"value": %q, "parsed": 0},
		"sparse": {"value": "%t", "parsed": %t},
		"compression": {"value": %q},
		"comments": {"value": %q}
	}]`, id, id, volsize, volsize, volblocksize, sparse, sparse, compression, comments)
}
