package resources

import (
	"context"
	"errors"
	"testing"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewSnapshotResource(t *testing.T) {
	r := NewSnapshotResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(r.(*SnapshotResource))
	_ = resource.ResourceWithImportState(r.(*SnapshotResource))
}

func TestSnapshotResource_Metadata(t *testing.T) {
	r := NewSnapshotResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_snapshot" {
		t.Errorf("expected TypeName 'truenas_snapshot', got %q", resp.TypeName)
	}
}

func TestSnapshotResource_Schema(t *testing.T) {
	r := NewSnapshotResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify id attribute exists and is computed
	idAttr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal("expected 'id' attribute in schema")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' attribute to be computed")
	}

	// Verify dataset_id attribute exists and is required
	datasetIDAttr, ok := resp.Schema.Attributes["dataset_id"]
	if !ok {
		t.Fatal("expected 'dataset_id' attribute in schema")
	}
	if !datasetIDAttr.IsRequired() {
		t.Error("expected 'dataset_id' attribute to be required")
	}

	// Verify name attribute exists and is required
	nameAttr, ok := resp.Schema.Attributes["name"]
	if !ok {
		t.Fatal("expected 'name' attribute in schema")
	}
	if !nameAttr.IsRequired() {
		t.Error("expected 'name' attribute to be required")
	}

	// Verify hold attribute exists and is optional
	holdAttr, ok := resp.Schema.Attributes["hold"]
	if !ok {
		t.Fatal("expected 'hold' attribute in schema")
	}
	if !holdAttr.IsOptional() {
		t.Error("expected 'hold' attribute to be optional")
	}

	// Verify recursive attribute exists and is optional
	recursiveAttr, ok := resp.Schema.Attributes["recursive"]
	if !ok {
		t.Fatal("expected 'recursive' attribute in schema")
	}
	if !recursiveAttr.IsOptional() {
		t.Error("expected 'recursive' attribute to be optional")
	}

	// Verify computed attributes
	for _, attr := range []string{"createtxg", "used_bytes", "referenced_bytes"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Fatalf("expected '%s' attribute in schema", attr)
		}
		if !a.IsComputed() {
			t.Errorf("expected '%s' attribute to be computed", attr)
		}
	}
}

func TestSnapshotResource_Configure_Success(t *testing.T) {
	r := NewSnapshotResource().(*SnapshotResource)

	svc := &services.TrueNASServices{}

	req := resource.ConfigureRequest{
		ProviderData: svc,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestSnapshotResource_Configure_NilProviderData(t *testing.T) {
	r := NewSnapshotResource().(*SnapshotResource)

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestSnapshotResource_Configure_WrongType(t *testing.T) {
	r := NewSnapshotResource().(*SnapshotResource)

	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

// Test helpers

func getSnapshotResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewSnapshotResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	return *schemaResp
}

// snapshotModelParams holds parameters for creating test model values.
// Using a struct instead of 8 individual parameters per the 3-param rule.
type snapshotModelParams struct {
	ID              interface{}
	DatasetID       interface{}
	Name            interface{}
	Hold            interface{}
	Recursive       interface{}
	CreateTXG       interface{}
	UsedBytes       interface{}
	ReferencedBytes interface{}
}

func createSnapshotResourceModelValue(p snapshotModelParams) tftypes.Value {
	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":               tftypes.String,
			"dataset_id":       tftypes.String,
			"name":             tftypes.String,
			"hold":             tftypes.Bool,
			"recursive":        tftypes.Bool,
			"createtxg":        tftypes.String,
			"used_bytes":       tftypes.Number,
			"referenced_bytes": tftypes.Number,
		},
	}, map[string]tftypes.Value{
		"id":               tftypes.NewValue(tftypes.String, p.ID),
		"dataset_id":       tftypes.NewValue(tftypes.String, p.DatasetID),
		"name":             tftypes.NewValue(tftypes.String, p.Name),
		"hold":             tftypes.NewValue(tftypes.Bool, p.Hold),
		"recursive":        tftypes.NewValue(tftypes.Bool, p.Recursive),
		"createtxg":        tftypes.NewValue(tftypes.String, p.CreateTXG),
		"used_bytes":       tftypes.NewValue(tftypes.Number, p.UsedBytes),
		"referenced_bytes": tftypes.NewValue(tftypes.Number, p.ReferencedBytes),
	})
}

// testSnapshot returns a standard test snapshot.
func testSnapshot() *truenas.Snapshot {
	return &truenas.Snapshot{
		ID:           "tank/data@snap1",
		Dataset:      "tank/data",
		SnapshotName: "snap1",
		CreateTXG:    "12345",
		Used:         1024,
		Referenced:   2048,
		HasHold:      false,
	}
}

func TestSnapshotResource_Create_Success(t *testing.T) {
	var capturedOpts truenas.CreateSnapshotOpts

	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateSnapshotOpts) (*truenas.Snapshot, error) {
					capturedOpts = opts
					return testSnapshot(), nil
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		DatasetID: "tank/data",
		Name:      "snap1",
		Hold:      false,
		Recursive: false,
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if capturedOpts.Dataset != "tank/data" {
		t.Errorf("expected dataset 'tank/data', got %q", capturedOpts.Dataset)
	}
	if capturedOpts.Name != "snap1" {
		t.Errorf("expected name 'snap1', got %q", capturedOpts.Name)
	}

	// Verify model mapping
	var data SnapshotResourceModel
	resp.State.Get(context.Background(), &data)

	if data.ID.ValueString() != "tank/data@snap1" {
		t.Errorf("expected ID 'tank/data@snap1', got %q", data.ID.ValueString())
	}
	if data.CreateTXG.ValueString() != "12345" {
		t.Errorf("expected createtxg '12345', got %q", data.CreateTXG.ValueString())
	}
	if data.UsedBytes.ValueInt64() != 1024 {
		t.Errorf("expected used_bytes 1024, got %d", data.UsedBytes.ValueInt64())
	}
	if data.ReferencedBytes.ValueInt64() != 2048 {
		t.Errorf("expected referenced_bytes 2048, got %d", data.ReferencedBytes.ValueInt64())
	}
}

func TestSnapshotResource_Create_WithHold(t *testing.T) {
	var holdCalled bool

	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateSnapshotOpts) (*truenas.Snapshot, error) {
					return testSnapshot(), nil
				},
				HoldFunc: func(ctx context.Context, id string) error {
					holdCalled = true
					return nil
				},
				GetFunc: func(ctx context.Context, id string) (*truenas.Snapshot, error) {
					snap := testSnapshot()
					snap.HasHold = true
					return snap, nil
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		DatasetID: "tank/data",
		Name:      "snap1",
		Hold:      true,
		Recursive: false,
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !holdCalled {
		t.Error("expected Hold to be called when hold=true")
	}
}

func TestSnapshotResource_Create_APIError(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateSnapshotOpts) (*truenas.Snapshot, error) {
					return nil, errors.New("snapshot already exists")
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		DatasetID: "tank/data",
		Name:      "snap1",
		Hold:      false,
		Recursive: false,
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestSnapshotResource_Read_Success(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				GetFunc: func(ctx context.Context, id string) (*truenas.Snapshot, error) {
					return testSnapshot(), nil
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "",
		UsedBytes:       float64(0),
		ReferencedBytes: float64(0),
	})

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotResourceModel
	resp.State.Get(context.Background(), &data)

	if data.CreateTXG.ValueString() != "12345" {
		t.Errorf("expected createtxg '12345', got %q", data.CreateTXG.ValueString())
	}
	if data.UsedBytes.ValueInt64() != 1024 {
		t.Errorf("expected used_bytes 1024, got %d", data.UsedBytes.ValueInt64())
	}
}

func TestSnapshotResource_Read_NotFound(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				GetFunc: func(ctx context.Context, id string) (*truenas.Snapshot, error) {
					return nil, nil
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "",
		UsedBytes:       float64(0),
		ReferencedBytes: float64(0),
	})

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	// Should not error - just remove from state
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// State should be empty (null)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be null when snapshot not found")
	}
}

func TestSnapshotResource_Update_HoldToRelease(t *testing.T) {
	var releaseCalled bool

	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				ReleaseFunc: func(ctx context.Context, id string) error {
					releaseCalled = true
					return nil
				},
				GetFunc: func(ctx context.Context, id string) (*truenas.Snapshot, error) {
					return testSnapshot(), nil
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)

	// State has hold=true
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            true,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	// Plan has hold=false
	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !releaseCalled {
		t.Error("expected Release to be called")
	}
}

func TestSnapshotResource_Update_ReleaseToHold(t *testing.T) {
	var holdCalled bool

	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				HoldFunc: func(ctx context.Context, id string) error {
					holdCalled = true
					return nil
				},
				GetFunc: func(ctx context.Context, id string) (*truenas.Snapshot, error) {
					return testSnapshot(), nil
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)

	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            true,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !holdCalled {
		t.Error("expected Hold to be called")
	}
}

func TestSnapshotResource_Delete_Success(t *testing.T) {
	var deleteCalled bool
	var deleteID string

	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				DeleteFunc: func(ctx context.Context, id string) error {
					deleteCalled = true
					deleteID = id
					return nil
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !deleteCalled {
		t.Error("expected Delete to be called")
	}

	if deleteID != "tank/data@snap1" {
		t.Errorf("expected delete ID 'tank/data@snap1', got %q", deleteID)
	}
}

func TestSnapshotResource_Delete_WithHold(t *testing.T) {
	var releaseCalled, deleteCalled bool
	var callOrder []string

	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				ReleaseFunc: func(ctx context.Context, id string) error {
					releaseCalled = true
					callOrder = append(callOrder, "release")
					return nil
				},
				DeleteFunc: func(ctx context.Context, id string) error {
					deleteCalled = true
					callOrder = append(callOrder, "delete")
					return nil
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            true,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !releaseCalled {
		t.Error("expected Release to be called")
	}
	if !deleteCalled {
		t.Error("expected Delete to be called")
	}

	// Verify release was called before delete
	if len(callOrder) != 2 || callOrder[0] != "release" || callOrder[1] != "delete" {
		t.Errorf("expected [release, delete], got %v", callOrder)
	}
}

func TestSnapshotResource_Read_APIError(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				GetFunc: func(ctx context.Context, id string) (*truenas.Snapshot, error) {
					return nil, errors.New("connection refused")
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestSnapshotResource_Delete_APIError(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				DeleteFunc: func(ctx context.Context, id string) error {
					return errors.New("snapshot is busy")
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestSnapshotResource_ImportState(t *testing.T) {
	r := NewSnapshotResource().(*SnapshotResource)

	schemaResp := getSnapshotResourceSchema(t)

	// Create an initial empty state with the correct schema
	emptyState := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              nil,
		DatasetID:       nil,
		Name:            nil,
		Hold:            nil,
		Recursive:       nil,
		CreateTXG:       nil,
		UsedBytes:       nil,
		ReferencedBytes: nil,
	})

	req := resource.ImportStateRequest{
		ID: "tank/data@snap1",
	}

	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    emptyState,
		},
	}

	r.ImportState(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotResourceModel
	diags := resp.State.Get(context.Background(), &data)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if data.ID.ValueString() != "tank/data@snap1" {
		t.Errorf("expected ID 'tank/data@snap1', got %q", data.ID.ValueString())
	}
}

func TestSnapshotResource_Create_WithRecursive(t *testing.T) {
	var capturedOpts truenas.CreateSnapshotOpts

	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateSnapshotOpts) (*truenas.Snapshot, error) {
					capturedOpts = opts
					return testSnapshot(), nil
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		DatasetID: "tank/data",
		Name:      "snap1",
		Hold:      false,
		Recursive: true,
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !capturedOpts.Recursive {
		t.Error("expected recursive=true")
	}
}

func TestSnapshotResource_Create_HoldError(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateSnapshotOpts) (*truenas.Snapshot, error) {
					return testSnapshot(), nil
				},
				HoldFunc: func(ctx context.Context, id string) error {
					return errors.New("hold failed")
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		DatasetID: "tank/data",
		Name:      "snap1",
		Hold:      true,
		Recursive: false,
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for hold failure")
	}
}

func TestSnapshotResource_Create_GetAfterHoldError(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateSnapshotOpts) (*truenas.Snapshot, error) {
					return testSnapshot(), nil
				},
				HoldFunc: func(ctx context.Context, id string) error {
					return nil
				},
				GetFunc: func(ctx context.Context, id string) (*truenas.Snapshot, error) {
					return nil, errors.New("query failed")
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		DatasetID: "tank/data",
		Name:      "snap1",
		Hold:      true,
		Recursive: false,
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for query failure after hold")
	}
}

func TestSnapshotResource_Update_ReleaseError(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				ReleaseFunc: func(ctx context.Context, id string) error {
					return errors.New("release failed")
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)

	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            true,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for release failure")
	}
}

func TestSnapshotResource_Update_HoldError(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				HoldFunc: func(ctx context.Context, id string) error {
					return errors.New("hold failed")
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)

	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            true,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for hold failure")
	}
}

func TestSnapshotResource_Update_QueryError(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				GetFunc: func(ctx context.Context, id string) (*truenas.Snapshot, error) {
					return nil, errors.New("query failed")
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)

	// No hold change, so it goes straight to Get
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for query failure")
	}
}

func TestSnapshotResource_Update_SnapshotNotFound(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				GetFunc: func(ctx context.Context, id string) (*truenas.Snapshot, error) {
					return nil, nil
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)

	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when snapshot not found during update")
	}
}

func TestSnapshotResource_Delete_ReleaseError(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				ReleaseFunc: func(ctx context.Context, id string) error {
					return errors.New("release failed")
				},
			},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            true,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for release failure before delete")
	}
}

// Helper to create an invalid model value with type mismatch
func createInvalidSnapshotModelValue() tftypes.Value {
	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":               tftypes.String,
			"dataset_id":       tftypes.String,
			"name":             tftypes.String,
			"hold":             tftypes.String, // Wrong type - should be Bool
			"recursive":        tftypes.Bool,
			"createtxg":        tftypes.String,
			"used_bytes":       tftypes.Number,
			"referenced_bytes": tftypes.Number,
		},
	}, map[string]tftypes.Value{
		"id":               tftypes.NewValue(tftypes.String, nil),
		"dataset_id":       tftypes.NewValue(tftypes.String, "tank/data"),
		"name":             tftypes.NewValue(tftypes.String, "snap1"),
		"hold":             tftypes.NewValue(tftypes.String, "not-a-bool"), // Wrong value type
		"recursive":        tftypes.NewValue(tftypes.Bool, false),
		"createtxg":        tftypes.NewValue(tftypes.String, nil),
		"used_bytes":       tftypes.NewValue(tftypes.Number, nil),
		"referenced_bytes": tftypes.NewValue(tftypes.Number, nil),
	})
}

func TestSnapshotResource_Create_GetPlanError(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	invalidValue := createInvalidSnapshotModelValue()

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    invalidValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid plan value")
	}
}

func TestSnapshotResource_Read_GetStateError(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	invalidValue := createInvalidSnapshotModelValue()

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    invalidValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid state value")
	}
}

func TestSnapshotResource_Update_GetStateOrPlanError(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	invalidValue := createInvalidSnapshotModelValue()

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    invalidValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    invalidValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid state/plan value")
	}
}

func TestSnapshotResource_Delete_GetStateError(t *testing.T) {
	r := &SnapshotResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{},
		}},
	}

	schemaResp := getSnapshotResourceSchema(t)
	invalidValue := createInvalidSnapshotModelValue()

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    invalidValue,
		},
	}

	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid state value")
	}
}
