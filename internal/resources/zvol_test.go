package resources

import (
	"context"
	"errors"
	"testing"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func TestZvolResource_Create_Basic(t *testing.T) {
	var capturedOpts truenas.CreateZvolOpts

	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				CreateZvolFunc: func(ctx context.Context, opts truenas.CreateZvolOpts) (*truenas.Zvol, error) {
					capturedOpts = opts
					return &truenas.Zvol{
						ID:           "tank/myvol",
						Name:         "tank/myvol",
						Pool:         "tank",
						Compression:  "lz4",
						Volsize:      10737418240,
						Volblocksize: "16K",
					}, nil
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)
	planValue := createZvolModelValue(defaultZvolPlanParams())

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if capturedOpts.Name != "tank/myvol" {
		t.Errorf("expected name 'tank/myvol', got %v", capturedOpts.Name)
	}
	if capturedOpts.Volsize != int64(10737418240) {
		t.Errorf("expected volsize 10737418240, got %v", capturedOpts.Volsize)
	}

	var model ZvolResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.ID.ValueString() != "tank/myvol" {
		t.Errorf("expected ID 'tank/myvol', got %q", model.ID.ValueString())
	}
}

func TestZvolResource_Create_WithOptionalFields(t *testing.T) {
	var capturedOpts truenas.CreateZvolOpts

	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				CreateZvolFunc: func(ctx context.Context, opts truenas.CreateZvolOpts) (*truenas.Zvol, error) {
					capturedOpts = opts
					return &truenas.Zvol{
						ID:           "tank/myvol",
						Name:         "tank/myvol",
						Pool:         "tank",
						Compression:  "zstd",
						Comments:     "test vol",
						Volsize:      10737418240,
						Volblocksize: "64K",
						Sparse:       true,
					}, nil
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.Volblocksize = strPtr("64K")
	p.Sparse = boolPtr(true)
	p.Compression = strPtr("zstd")
	p.Comments = strPtr("test vol")
	planValue := createZvolModelValue(p)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if capturedOpts.Volblocksize != "64K" {
		t.Errorf("expected volblocksize '64K', got %v", capturedOpts.Volblocksize)
	}
	if capturedOpts.Sparse != true {
		t.Errorf("expected sparse true, got %v", capturedOpts.Sparse)
	}
	if capturedOpts.Compression != "zstd" {
		t.Errorf("expected compression 'zstd', got %v", capturedOpts.Compression)
	}
	if capturedOpts.Comments != "test vol" {
		t.Errorf("expected comments 'test vol', got %v", capturedOpts.Comments)
	}
}

func TestZvolResource_Create_InvalidName(t *testing.T) {
	r := &ZvolResource{BaseResource: BaseResource{services: &services.TrueNASServices{}}}

	schemaResp := getZvolResourceSchema(t)
	p := zvolModelParams{Volsize: strPtr("10G")} // no pool/path
	planValue := createZvolModelValue(p)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid name")
	}
}

func TestZvolResource_Create_APIError(t *testing.T) {
	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				CreateZvolFunc: func(ctx context.Context, opts truenas.CreateZvolOpts) (*truenas.Zvol, error) {
					return nil, errors.New("pool not found")
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)
	planValue := createZvolModelValue(defaultZvolPlanParams())

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API failure")
	}
}

func TestZvolResource_Create_NotFoundAfterCreate(t *testing.T) {
	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				CreateZvolFunc: func(ctx context.Context, opts truenas.CreateZvolOpts) (*truenas.Zvol, error) {
					return nil, nil
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)
	planValue := createZvolModelValue(defaultZvolPlanParams())

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when zvol not found after create")
	}
}

func TestZvolResource_Read_Basic(t *testing.T) {
	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetZvolFunc: func(ctx context.Context, id string) (*truenas.Zvol, error) {
					return &truenas.Zvol{
						ID:           "tank/myvol",
						Name:         "tank/myvol",
						Pool:         "tank",
						Compression:  "lz4",
						Volsize:      10737418240,
						Volblocksize: "16K",
					}, nil
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	stateValue := createZvolModelValue(p)

	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model ZvolResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.ID.ValueString() != "tank/myvol" {
		t.Errorf("expected ID 'tank/myvol', got %q", model.ID.ValueString())
	}
	if model.Compression.ValueString() != "lz4" {
		t.Errorf("expected compression 'lz4', got %q", model.Compression.ValueString())
	}
}

func TestZvolResource_Read_NotFound(t *testing.T) {
	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetZvolFunc: func(ctx context.Context, id string) (*truenas.Zvol, error) {
					return nil, nil
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/deleted")
	stateValue := createZvolModelValue(p)

	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// State should be removed (resource deleted outside Terraform)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed for deleted zvol")
	}
}

func TestZvolResource_Update_Volsize(t *testing.T) {
	var capturedID string
	var capturedOpts truenas.UpdateZvolOpts

	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				UpdateZvolFunc: func(ctx context.Context, id string, opts truenas.UpdateZvolOpts) (*truenas.Zvol, error) {
					capturedID = id
					capturedOpts = opts
					return &truenas.Zvol{
						ID:           "tank/myvol",
						Name:         "tank/myvol",
						Pool:         "tank",
						Compression:  "lz4",
						Volsize:      21474836480,
						Volblocksize: "16K",
					}, nil
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)

	stateP := defaultZvolPlanParams()
	stateP.ID = strPtr("tank/myvol")
	stateP.Volblocksize = strPtr("16K")
	stateP.Compression = strPtr("lz4")
	stateValue := createZvolModelValue(stateP)

	planP := defaultZvolPlanParams()
	planP.ID = strPtr("tank/myvol")
	planP.Volsize = strPtr("21474836480") // doubled
	planP.Volblocksize = strPtr("16K")
	planP.Compression = strPtr("lz4")
	planValue := createZvolModelValue(planP)

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if capturedID != "tank/myvol" {
		t.Errorf("expected update ID 'tank/myvol', got %q", capturedID)
	}
	if capturedOpts.Volsize == nil || *capturedOpts.Volsize != int64(21474836480) {
		t.Errorf("expected volsize 21474836480, got %v", capturedOpts.Volsize)
	}
}

func TestZvolResource_Update_NoChanges(t *testing.T) {
	var updateCalled bool

	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				UpdateZvolFunc: func(ctx context.Context, id string, opts truenas.UpdateZvolOpts) (*truenas.Zvol, error) {
					updateCalled = true
					return nil, nil
				},
				GetZvolFunc: func(ctx context.Context, id string) (*truenas.Zvol, error) {
					return &truenas.Zvol{
						ID:           "tank/myvol",
						Name:         "tank/myvol",
						Pool:         "tank",
						Compression:  "lz4",
						Volsize:      10737418240,
						Volblocksize: "16K",
					}, nil
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)

	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	p.Volblocksize = strPtr("16K")
	p.Compression = strPtr("lz4")
	value := createZvolModelValue(p)

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: value},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: value},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if updateCalled {
		t.Error("expected UpdateZvol to NOT be called when nothing changed")
	}
}

func TestZvolResource_Update_APIError(t *testing.T) {
	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				UpdateZvolFunc: func(ctx context.Context, id string, opts truenas.UpdateZvolOpts) (*truenas.Zvol, error) {
					return nil, errors.New("update failed")
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)

	stateP := defaultZvolPlanParams()
	stateP.ID = strPtr("tank/myvol")
	stateP.Volblocksize = strPtr("16K")
	stateP.Compression = strPtr("lz4")
	stateValue := createZvolModelValue(stateP)

	planP := defaultZvolPlanParams()
	planP.ID = strPtr("tank/myvol")
	planP.Volsize = strPtr("21474836480")
	planP.Volblocksize = strPtr("16K")
	planP.Compression = strPtr("lz4")
	planValue := createZvolModelValue(planP)

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for update API failure")
	}
}

func TestZvolResource_Delete_Basic(t *testing.T) {
	var deleteCalled bool
	var deleteID string

	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				DeleteZvolFunc: func(ctx context.Context, id string) error {
					deleteCalled = true
					deleteID = id
					return nil
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	stateValue := createZvolModelValue(p)

	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if !deleteCalled {
		t.Fatal("expected DeleteZvol to be called")
	}
	if deleteID != "tank/myvol" {
		t.Errorf("expected delete ID 'tank/myvol', got %q", deleteID)
	}
}

func TestZvolResource_Delete_ForceDestroy(t *testing.T) {
	var deleteDatasetCalled bool
	var deleteDatasetID string
	var deleteDatasetRecursive bool

	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				DeleteDatasetFunc: func(ctx context.Context, id string, recursive bool) error {
					deleteDatasetCalled = true
					deleteDatasetID = id
					deleteDatasetRecursive = recursive
					return nil
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	p.ForceDestroy = boolPtr(true)
	stateValue := createZvolModelValue(p)

	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if !deleteDatasetCalled {
		t.Fatal("expected DeleteDataset to be called for force_destroy")
	}
	if deleteDatasetID != "tank/myvol" {
		t.Errorf("expected delete ID 'tank/myvol', got %q", deleteDatasetID)
	}
	if !deleteDatasetRecursive {
		t.Error("expected recursive=true for force_destroy")
	}
}

func TestZvolResource_Delete_APIError(t *testing.T) {
	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				DeleteZvolFunc: func(ctx context.Context, id string) error {
					return errors.New("zvol is busy")
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	stateValue := createZvolModelValue(p)

	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for delete API failure")
	}
}

func TestZvolResource_Update_CompressionAndComments(t *testing.T) {
	var capturedOpts truenas.UpdateZvolOpts

	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				UpdateZvolFunc: func(ctx context.Context, id string, opts truenas.UpdateZvolOpts) (*truenas.Zvol, error) {
					capturedOpts = opts
					return &truenas.Zvol{
						ID:           "tank/myvol",
						Name:         "tank/myvol",
						Pool:         "tank",
						Compression:  "zstd",
						Comments:     "new comment",
						Volsize:      10737418240,
						Volblocksize: "16K",
					}, nil
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)

	stateP := defaultZvolPlanParams()
	stateP.ID = strPtr("tank/myvol")
	stateP.Volblocksize = strPtr("16K")
	stateP.Compression = strPtr("lz4")
	stateP.Comments = strPtr("old comment")
	stateValue := createZvolModelValue(stateP)

	planP := defaultZvolPlanParams()
	planP.ID = strPtr("tank/myvol")
	planP.Volblocksize = strPtr("16K")
	planP.Compression = strPtr("zstd")
	planP.Comments = strPtr("new comment")
	planValue := createZvolModelValue(planP)

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if capturedOpts.Compression != "zstd" {
		t.Errorf("expected compression 'zstd', got %v", capturedOpts.Compression)
	}
	if capturedOpts.Comments == nil || *capturedOpts.Comments != "new comment" {
		t.Errorf("expected comments 'new comment', got %v", capturedOpts.Comments)
	}
}

func TestZvolResource_Update_ClearComments(t *testing.T) {
	var capturedOpts truenas.UpdateZvolOpts

	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				UpdateZvolFunc: func(ctx context.Context, id string, opts truenas.UpdateZvolOpts) (*truenas.Zvol, error) {
					capturedOpts = opts
					return &truenas.Zvol{
						ID:           "tank/myvol",
						Name:         "tank/myvol",
						Pool:         "tank",
						Compression:  "lz4",
						Volsize:      10737418240,
						Volblocksize: "16K",
					}, nil
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)

	stateP := defaultZvolPlanParams()
	stateP.ID = strPtr("tank/myvol")
	stateP.Volblocksize = strPtr("16K")
	stateP.Compression = strPtr("lz4")
	stateP.Comments = strPtr("old comment")
	stateValue := createZvolModelValue(stateP)

	planP := defaultZvolPlanParams()
	planP.ID = strPtr("tank/myvol")
	planP.Volblocksize = strPtr("16K")
	planP.Compression = strPtr("lz4")
	// Comments nil = clear
	planValue := createZvolModelValue(planP)

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if capturedOpts.Comments == nil || *capturedOpts.Comments != "" {
		t.Errorf("expected comments to be empty string, got %v", capturedOpts.Comments)
	}
}

func TestZvolResource_Create_InvalidVolsizeFormat(t *testing.T) {
	r := &ZvolResource{BaseResource: BaseResource{services: &services.TrueNASServices{}}}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.Volsize = strPtr("not-a-size")
	planValue := createZvolModelValue(p)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid volsize format")
	}
}

func TestZvolResource_Update_ReadAfterUpdateFails(t *testing.T) {
	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				UpdateZvolFunc: func(ctx context.Context, id string, opts truenas.UpdateZvolOpts) (*truenas.Zvol, error) {
					return nil, errors.New("read failed")
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)

	stateP := defaultZvolPlanParams()
	stateP.ID = strPtr("tank/myvol")
	stateP.Volblocksize = strPtr("16K")
	stateP.Compression = strPtr("lz4")
	stateValue := createZvolModelValue(stateP)

	planP := defaultZvolPlanParams()
	planP.ID = strPtr("tank/myvol")
	planP.Volsize = strPtr("21474836480")
	planP.Volblocksize = strPtr("16K")
	planP.Compression = strPtr("lz4")
	planValue := createZvolModelValue(planP)

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when read after update fails")
	}
}

func TestZvolResource_ImportState(t *testing.T) {
	r := NewZvolResource().(*ZvolResource)
	schemaResp := getZvolResourceSchema(t)

	emptyState := createZvolModelValue(defaultZvolPlanParams())

	req := resource.ImportStateRequest{ID: "tank/myvol"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: emptyState},
	}

	r.ImportState(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model ZvolResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.ID.ValueString() != "tank/myvol" {
		t.Errorf("expected ID 'tank/myvol', got %q", model.ID.ValueString())
	}
}

func TestZvolResource_Read_PopulatesPoolPath_AfterImport(t *testing.T) {
	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetZvolFunc: func(ctx context.Context, id string) (*truenas.Zvol, error) {
					return &truenas.Zvol{
						ID:           "tank/vms/disk0",
						Name:         "tank/vms/disk0",
						Pool:         "tank",
						Compression:  "lz4",
						Volsize:      10737418240,
						Volblocksize: "16K",
					}, nil
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)
	// After import, only ID is set -- pool/path/parent are null
	p := zvolModelParams{
		ID:      strPtr("tank/vms/disk0"),
		Volsize: strPtr("10737418240"),
	}
	stateValue := createZvolModelValue(p)

	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model ZvolResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.Pool.ValueString() != "tank" {
		t.Errorf("expected pool 'tank', got %q", model.Pool.ValueString())
	}
	if model.Path.ValueString() != "vms/disk0" {
		t.Errorf("expected path 'vms/disk0', got %q", model.Path.ValueString())
	}
}

func TestZvolResource_Read_APIError(t *testing.T) {
	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetZvolFunc: func(ctx context.Context, id string) (*truenas.Zvol, error) {
					return nil, errors.New("connection failed")
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	stateValue := createZvolModelValue(p)

	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for read API failure")
	}
}

func TestZvolResource_Configure_NilProviderData(t *testing.T) {
	r := NewZvolResource().(*ZvolResource)

	req := resource.ConfigureRequest{ProviderData: nil}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestZvolResource_Configure_WrongType(t *testing.T) {
	r := NewZvolResource().(*ZvolResource)

	req := resource.ConfigureRequest{ProviderData: "not a client"}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong provider data type")
	}
}

func TestZvolResource_Update_NotFoundAfterUpdate(t *testing.T) {
	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				UpdateZvolFunc: func(ctx context.Context, id string, opts truenas.UpdateZvolOpts) (*truenas.Zvol, error) {
					return nil, nil
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)

	stateP := defaultZvolPlanParams()
	stateP.ID = strPtr("tank/myvol")
	stateP.Volblocksize = strPtr("16K")
	stateP.Compression = strPtr("lz4")
	stateValue := createZvolModelValue(stateP)

	planP := defaultZvolPlanParams()
	planP.ID = strPtr("tank/myvol")
	planP.Volsize = strPtr("21474836480")
	planP.Volblocksize = strPtr("16K")
	planP.Compression = strPtr("lz4")
	planValue := createZvolModelValue(planP)

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when zvol not found after update")
	}
}

func TestZvolResource_Update_NoChanges_ReadFails(t *testing.T) {
	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetZvolFunc: func(ctx context.Context, id string) (*truenas.Zvol, error) {
					return nil, errors.New("read failed")
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)

	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	p.Volblocksize = strPtr("16K")
	p.Compression = strPtr("lz4")
	value := createZvolModelValue(p)

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: value},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: value},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when read fails in no-changes path")
	}
}

func TestZvolResource_Update_NoChanges_NotFound(t *testing.T) {
	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetZvolFunc: func(ctx context.Context, id string) (*truenas.Zvol, error) {
					return nil, nil
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)

	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	p.Volblocksize = strPtr("16K")
	p.Compression = strPtr("lz4")
	value := createZvolModelValue(p)

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: value},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: value},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when zvol not found in no-changes path")
	}
}

func TestZvolResource_Delete_ForceDestroy_APIError(t *testing.T) {
	r := &ZvolResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				DeleteDatasetFunc: func(ctx context.Context, id string, recursive bool) error {
					return errors.New("delete failed")
				},
			},
		}},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	p.ForceDestroy = boolPtr(true)
	stateValue := createZvolModelValue(p)

	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for force_destroy delete API failure")
	}
}

// -- Shared helper tests --

func TestPoolDatasetFullName(t *testing.T) {
	tests := []struct {
		name     string
		pool     string
		path     string
		parent   string
		nameAttr string
		want     string
	}{
		{"pool+path", "tank", "vms/disk0", "", "", "tank/vms/disk0"},
		{"parent+path", "", "disk0", "tank/vms", "", "tank/vms/disk0"},
		{"parent+name", "", "", "tank/vms", "disk0", "tank/vms/disk0"},
		{"nothing", "", "", "", "", ""},
		{"pool only", "tank", "", "", "", ""},
		{"path only", "", "disk0", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toStr := func(s string) types.String {
				if s == "" {
					return types.StringNull()
				}
				return types.StringValue(s)
			}
			got := poolDatasetFullName(toStr(tt.pool), toStr(tt.path), toStr(tt.parent), toStr(tt.nameAttr))
			if got != tt.want {
				t.Errorf("poolDatasetFullName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPoolDatasetIDToParts(t *testing.T) {
	tests := []struct {
		id       string
		wantPool string
		wantPath string
	}{
		{"tank/vms/disk0", "tank", "vms/disk0"},
		{"tank/disk0", "tank", "disk0"},
		{"tank", "tank", ""},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			pool, path := poolDatasetIDToParts(tt.id)
			if pool != tt.wantPool {
				t.Errorf("pool = %q, want %q", pool, tt.wantPool)
			}
			if path != tt.wantPath {
				t.Errorf("path = %q, want %q", path, tt.wantPath)
			}
		})
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

