package resources

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewSnapshotResource(t *testing.T) {
	r := NewSnapshotResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*SnapshotResource)
	var _ resource.ResourceWithImportState = r.(*SnapshotResource)
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

	mockClient := &client.MockClient{}

	req := resource.ConfigureRequest{
		ProviderData: mockClient,
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

func TestSnapshotResource_Create_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.snapshot.create" {
					capturedMethod = method
					capturedParams = params
					return json.RawMessage(`{"id": "tank/data@snap1"}`), nil
				}
				if method == "pool.snapshot.query" {
					return json.RawMessage(`[{
						"id": "tank/data@snap1",
						"name": "snap1",
						"dataset": "tank/data",
						"properties": {
							"createtxg": {"value": "12345"},
							"used": {"parsed": 1024},
							"referenced": {"parsed": 2048}
						}
					}]`), nil
				}
				return nil, nil
			},
		},
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

	if capturedMethod != "pool.snapshot.create" {
		t.Errorf("expected method 'pool.snapshot.create', got %q", capturedMethod)
	}

	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["dataset"] != "tank/data" {
		t.Errorf("expected dataset 'tank/data', got %v", params["dataset"])
	}
	if params["name"] != "snap1" {
		t.Errorf("expected name 'snap1', got %v", params["name"])
	}
}

func TestSnapshotResource_Create_WithHold(t *testing.T) {
	var methods []string

	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				if method == "pool.snapshot.create" {
					return json.RawMessage(`{"id": "tank/data@snap1"}`), nil
				}
				if method == "pool.snapshot.hold" {
					return json.RawMessage(`true`), nil
				}
				if method == "pool.snapshot.query" {
					return json.RawMessage(`[{
						"id": "tank/data@snap1",
						"name": "snap1",
						"dataset": "tank/data",
						"properties": {
							"createtxg": {"value": "12345"},
							"used": {"parsed": 1024},
							"referenced": {"parsed": 2048}
						}
					}]`), nil
				}
				return nil, nil
			},
		},
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

	// Verify pool.snapshot.hold was called
	holdCalled := false
	for _, m := range methods {
		if m == "pool.snapshot.hold" {
			holdCalled = true
			break
		}
	}
	if !holdCalled {
		t.Error("expected pool.snapshot.hold to be called when hold=true")
	}
}

func TestSnapshotResource_Create_APIError(t *testing.T) {
	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("snapshot already exists")
			},
		},
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
