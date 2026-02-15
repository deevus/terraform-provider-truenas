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

func TestNewVirtConfigResource(t *testing.T) {
	r := NewVirtConfigResource()
	if r == nil {
		t.Fatal("NewVirtConfigResource returned nil")
	}

	virtConfigResource, ok := r.(*VirtConfigResource)
	if !ok {
		t.Fatalf("expected *VirtConfigResource, got %T", r)
	}

	// Verify interface implementations
	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(virtConfigResource)
	_ = resource.ResourceWithImportState(virtConfigResource)
}

func TestVirtConfigResource_Metadata(t *testing.T) {
	r := NewVirtConfigResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_virt_config" {
		t.Errorf("expected TypeName 'truenas_virt_config', got %q", resp.TypeName)
	}
}

func TestVirtConfigResource_Schema(t *testing.T) {
	r := NewVirtConfigResource()

	ctx := context.Background()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}

	r.Schema(ctx, schemaReq, schemaResp)

	if schemaResp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify required attributes exist
	attrs := schemaResp.Schema.Attributes

	// Check id attribute - should be computed
	idAttr, ok := attrs["id"]
	if !ok {
		t.Fatal("expected 'id' attribute")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' attribute to be computed")
	}

	// Check bridge attribute - should be optional
	bridgeAttr, ok := attrs["bridge"]
	if !ok {
		t.Fatal("expected 'bridge' attribute")
	}
	if !bridgeAttr.IsOptional() {
		t.Error("expected 'bridge' attribute to be optional")
	}

	// Check v4_network attribute - should be optional
	v4NetworkAttr, ok := attrs["v4_network"]
	if !ok {
		t.Fatal("expected 'v4_network' attribute")
	}
	if !v4NetworkAttr.IsOptional() {
		t.Error("expected 'v4_network' attribute to be optional")
	}

	// Check v6_network attribute - should be optional
	v6NetworkAttr, ok := attrs["v6_network"]
	if !ok {
		t.Fatal("expected 'v6_network' attribute")
	}
	if !v6NetworkAttr.IsOptional() {
		t.Error("expected 'v6_network' attribute to be optional")
	}

	// Check pool attribute - should be optional
	preferredPoolAttr, ok := attrs["pool"]
	if !ok {
		t.Fatal("expected 'pool' attribute")
	}
	if !preferredPoolAttr.IsOptional() {
		t.Error("expected 'pool' attribute to be optional")
	}
}


// Test helpers

func getVirtConfigResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewVirtConfigResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("failed to get schema: %v", schemaResp.Diagnostics)
	}
	return *schemaResp
}

// virtConfigModelParams holds parameters for creating test model values.
type virtConfigModelParams struct {
	ID            interface{}
	Bridge        interface{}
	V4Network     interface{}
	V6Network     interface{}
	Pool interface{}
}

func createVirtConfigModelValue(p virtConfigModelParams) tftypes.Value {
	// Build the values map
	values := map[string]tftypes.Value{
		"id":             tftypes.NewValue(tftypes.String, p.ID),
		"bridge":         tftypes.NewValue(tftypes.String, p.Bridge),
		"v4_network":     tftypes.NewValue(tftypes.String, p.V4Network),
		"v6_network":     tftypes.NewValue(tftypes.String, p.V6Network),
		"pool": tftypes.NewValue(tftypes.String, p.Pool),
	}

	// Create object type matching the schema
	objectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":             tftypes.String,
			"bridge":         tftypes.String,
			"v4_network":     tftypes.String,
			"v6_network":     tftypes.String,
			"pool": tftypes.String,
		},
	}

	return tftypes.NewValue(objectType, values)
}

func TestVirtConfigResource_Create_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &VirtConfigResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.global.update" {
					capturedMethod = method
					capturedParams = params
					return json.RawMessage(`{
						"bridge": "br0",
						"v4_network": "10.0.0.0/24",
						"v6_network": "fd00::/64",
						"pool": "tank"
					}`), nil
				}
				if method == "virt.global.config" {
					return json.RawMessage(`{
						"bridge": "br0",
						"v4_network": "10.0.0.0/24",
						"v6_network": "fd00::/64",
						"pool": "tank"
					}`), nil
				}
				return nil, nil
			},
		}},

	}

	schemaResp := getVirtConfigResourceSchema(t)
	planValue := createVirtConfigModelValue(virtConfigModelParams{
		Bridge:        "br0",
		V4Network:     "10.0.0.0/24",
		V6Network:     "fd00::/64",
		Pool: "tank",
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

	if capturedMethod != "virt.global.update" {
		t.Errorf("expected method 'virt.global.update', got %q", capturedMethod)
	}

	// Verify params
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["bridge"] != "br0" {
		t.Errorf("expected bridge 'br0', got %v", params["bridge"])
	}
	if params["v4_network"] != "10.0.0.0/24" {
		t.Errorf("expected v4_network '10.0.0.0/24', got %v", params["v4_network"])
	}
	if params["v6_network"] != "fd00::/64" {
		t.Errorf("expected v6_network 'fd00::/64', got %v", params["v6_network"])
	}
	if params["pool"] != "tank" {
		t.Errorf("expected pool 'tank', got %v", params["pool"])
	}

	// Verify state was set
	var resultData VirtConfigResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "virt_config" {
		t.Errorf("expected ID 'virt_config', got %q", resultData.ID.ValueString())
	}
	if resultData.Bridge.ValueString() != "br0" {
		t.Errorf("expected bridge 'br0', got %q", resultData.Bridge.ValueString())
	}
	if resultData.V4Network.ValueString() != "10.0.0.0/24" {
		t.Errorf("expected v4_network '10.0.0.0/24', got %q", resultData.V4Network.ValueString())
	}
	if resultData.V6Network.ValueString() != "fd00::/64" {
		t.Errorf("expected v6_network 'fd00::/64', got %q", resultData.V6Network.ValueString())
	}
	if resultData.Pool.ValueString() != "tank" {
		t.Errorf("expected pool 'tank', got %q", resultData.Pool.ValueString())
	}
}

func TestVirtConfigResource_Create_PartialConfig(t *testing.T) {
	var capturedParams any

	r := &VirtConfigResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.global.update" {
					capturedParams = params
					return json.RawMessage(`{
						"bridge": null,
						"v4_network": "10.0.0.0/24",
						"v6_network": null,
						"pool": null
					}`), nil
				}
				if method == "virt.global.config" {
					return json.RawMessage(`{
						"bridge": null,
						"v4_network": "10.0.0.0/24",
						"v6_network": null,
						"pool": null
					}`), nil
				}
				return nil, nil
			},
		}},

	}

	schemaResp := getVirtConfigResourceSchema(t)
	// Only set v4_network, leave others null
	planValue := createVirtConfigModelValue(virtConfigModelParams{
		Bridge:        nil,
		V4Network:     "10.0.0.0/24",
		V6Network:     nil,
		Pool: nil,
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

	// Verify params only contains the set value
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	// Only v4_network should be in params
	if params["v4_network"] != "10.0.0.0/24" {
		t.Errorf("expected v4_network '10.0.0.0/24', got %v", params["v4_network"])
	}
	// Other fields should not be in params (null values are not sent)
	if _, exists := params["bridge"]; exists {
		t.Error("expected bridge to not be in params")
	}
	if _, exists := params["v6_network"]; exists {
		t.Error("expected v6_network to not be in params")
	}
	if _, exists := params["pool"]; exists {
		t.Error("expected pool to not be in params")
	}

	// Verify state was set
	var resultData VirtConfigResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "virt_config" {
		t.Errorf("expected ID 'virt_config', got %q", resultData.ID.ValueString())
	}
	if resultData.V4Network.ValueString() != "10.0.0.0/24" {
		t.Errorf("expected v4_network '10.0.0.0/24', got %q", resultData.V4Network.ValueString())
	}
	if !resultData.Bridge.IsNull() {
		t.Error("expected bridge to be null")
	}
	if !resultData.V6Network.IsNull() {
		t.Error("expected v6_network to be null")
	}
	if !resultData.Pool.IsNull() {
		t.Error("expected pool to be null")
	}
}

func TestVirtConfigResource_Create_APIError(t *testing.T) {
	r := &VirtConfigResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		}},

	}

	schemaResp := getVirtConfigResourceSchema(t)
	planValue := createVirtConfigModelValue(virtConfigModelParams{
		Bridge:        "br0",
		V4Network:     "10.0.0.0/24",
		V6Network:     "fd00::/64",
		Pool: "tank",
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

	// Verify state was not set (should remain empty/null)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to not be set when API returns error")
	}
}

func TestVirtConfigResource_Read_Success(t *testing.T) {
	r := &VirtConfigResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method != "virt.global.config" {
					t.Errorf("expected method 'virt.global.config', got %q", method)
				}
				// Verify no params are passed
				if params != nil {
					t.Errorf("expected nil params, got %v", params)
				}
				return json.RawMessage(`{
					"bridge": "br0",
					"v4_network": "10.0.0.0/24",
					"v6_network": "fd00::/64",
					"pool": "tank"
				}`), nil
			},
		}},

	}

	schemaResp := getVirtConfigResourceSchema(t)
	stateValue := createVirtConfigModelValue(virtConfigModelParams{
		ID:            "virt_config",
		Bridge:        "br0",
		V4Network:     "10.0.0.0/24",
		V6Network:     "fd00::/64",
		Pool: "tank",
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

	// Verify state was updated
	var resultData VirtConfigResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "virt_config" {
		t.Errorf("expected ID 'virt_config', got %q", resultData.ID.ValueString())
	}
	if resultData.Bridge.ValueString() != "br0" {
		t.Errorf("expected bridge 'br0', got %q", resultData.Bridge.ValueString())
	}
	if resultData.V4Network.ValueString() != "10.0.0.0/24" {
		t.Errorf("expected v4_network '10.0.0.0/24', got %q", resultData.V4Network.ValueString())
	}
	if resultData.V6Network.ValueString() != "fd00::/64" {
		t.Errorf("expected v6_network 'fd00::/64', got %q", resultData.V6Network.ValueString())
	}
	if resultData.Pool.ValueString() != "tank" {
		t.Errorf("expected pool 'tank', got %q", resultData.Pool.ValueString())
	}
}

func TestVirtConfigResource_Read_NullFields(t *testing.T) {
	r := &VirtConfigResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`{
					"bridge": null,
					"v4_network": "10.0.0.0/24",
					"v6_network": null,
					"pool": null
				}`), nil
			},
		}},

	}

	schemaResp := getVirtConfigResourceSchema(t)
	stateValue := createVirtConfigModelValue(virtConfigModelParams{
		ID:            "virt_config",
		Bridge:        nil,
		V4Network:     "10.0.0.0/24",
		V6Network:     nil,
		Pool: nil,
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

	var resultData VirtConfigResourceModel
	resp.State.Get(context.Background(), &resultData)

	// v4_network should have a value
	if resultData.V4Network.ValueString() != "10.0.0.0/24" {
		t.Errorf("expected v4_network '10.0.0.0/24', got %q", resultData.V4Network.ValueString())
	}

	// Null fields should be null in the model
	if !resultData.Bridge.IsNull() {
		t.Errorf("expected bridge to be null, got %q", resultData.Bridge.ValueString())
	}
	if !resultData.V6Network.IsNull() {
		t.Errorf("expected v6_network to be null, got %q", resultData.V6Network.ValueString())
	}
	if !resultData.Pool.IsNull() {
		t.Errorf("expected pool to be null, got %q", resultData.Pool.ValueString())
	}
}

func TestVirtConfigResource_Read_APIError(t *testing.T) {
	r := &VirtConfigResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		}},

	}

	schemaResp := getVirtConfigResourceSchema(t)
	stateValue := createVirtConfigModelValue(virtConfigModelParams{
		ID:            "virt_config",
		Bridge:        "br0",
		V4Network:     "10.0.0.0/24",
		V6Network:     "fd00::/64",
		Pool: "tank",
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

func TestVirtConfigResource_Read_InvalidJSON(t *testing.T) {
	r := &VirtConfigResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		}},

	}

	schemaResp := getVirtConfigResourceSchema(t)
	stateValue := createVirtConfigModelValue(virtConfigModelParams{
		ID:            "virt_config",
		Bridge:        "br0",
		V4Network:     "10.0.0.0/24",
		V6Network:     "fd00::/64",
		Pool: "tank",
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
		t.Fatal("expected error for invalid JSON")
	}
}

func TestVirtConfigResource_Update_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &VirtConfigResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.global.update" {
					capturedMethod = method
					capturedParams = params
					return json.RawMessage(`{
						"bridge": "br1",
						"v4_network": "192.168.1.0/24",
						"v6_network": "fd01::/64",
						"pool": "storage"
					}`), nil
				}
				if method == "virt.global.config" {
					return json.RawMessage(`{
						"bridge": "br1",
						"v4_network": "192.168.1.0/24",
						"v6_network": "fd01::/64",
						"pool": "storage"
					}`), nil
				}
				return nil, nil
			},
		}},

	}

	schemaResp := getVirtConfigResourceSchema(t)

	// Current state
	stateValue := createVirtConfigModelValue(virtConfigModelParams{
		ID:            "virt_config",
		Bridge:        "br0",
		V4Network:     "10.0.0.0/24",
		V6Network:     "fd00::/64",
		Pool: "tank",
	})

	// Updated plan
	planValue := createVirtConfigModelValue(virtConfigModelParams{
		ID:            "virt_config",
		Bridge:        "br1",
		V4Network:     "192.168.1.0/24",
		V6Network:     "fd01::/64",
		Pool: "storage",
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

	if capturedMethod != "virt.global.update" {
		t.Errorf("expected method 'virt.global.update', got %q", capturedMethod)
	}

	// Verify params
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["bridge"] != "br1" {
		t.Errorf("expected bridge 'br1', got %v", params["bridge"])
	}
	if params["v4_network"] != "192.168.1.0/24" {
		t.Errorf("expected v4_network '192.168.1.0/24', got %v", params["v4_network"])
	}
	if params["v6_network"] != "fd01::/64" {
		t.Errorf("expected v6_network 'fd01::/64', got %v", params["v6_network"])
	}
	if params["pool"] != "storage" {
		t.Errorf("expected pool 'storage', got %v", params["pool"])
	}

	// Verify state was set
	var resultData VirtConfigResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "virt_config" {
		t.Errorf("expected ID 'virt_config', got %q", resultData.ID.ValueString())
	}
	if resultData.Bridge.ValueString() != "br1" {
		t.Errorf("expected bridge 'br1', got %q", resultData.Bridge.ValueString())
	}
	if resultData.V4Network.ValueString() != "192.168.1.0/24" {
		t.Errorf("expected v4_network '192.168.1.0/24', got %q", resultData.V4Network.ValueString())
	}
	if resultData.V6Network.ValueString() != "fd01::/64" {
		t.Errorf("expected v6_network 'fd01::/64', got %q", resultData.V6Network.ValueString())
	}
	if resultData.Pool.ValueString() != "storage" {
		t.Errorf("expected pool 'storage', got %q", resultData.Pool.ValueString())
	}
}

func TestVirtConfigResource_Update_APIError(t *testing.T) {
	r := &VirtConfigResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		}},

	}

	schemaResp := getVirtConfigResourceSchema(t)

	// Current state
	stateValue := createVirtConfigModelValue(virtConfigModelParams{
		ID:            "virt_config",
		Bridge:        "br0",
		V4Network:     "10.0.0.0/24",
		V6Network:     "fd00::/64",
		Pool: "tank",
	})

	// Updated plan
	planValue := createVirtConfigModelValue(virtConfigModelParams{
		ID:            "virt_config",
		Bridge:        "br1",
		V4Network:     "192.168.1.0/24",
		V6Network:     "fd01::/64",
		Pool: "storage",
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
		t.Fatal("expected error for API error")
	}
}

func TestVirtConfigResource_Delete_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &VirtConfigResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedParams = params
				return json.RawMessage(`{
					"bridge": null,
					"v4_network": null,
					"v6_network": null,
					"pool": null
				}`), nil
			},
		}},

	}

	schemaResp := getVirtConfigResourceSchema(t)
	stateValue := createVirtConfigModelValue(virtConfigModelParams{
		ID:            "virt_config",
		Bridge:        "br0",
		V4Network:     "10.0.0.0/24",
		V6Network:     "fd00::/64",
		Pool: "tank",
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

	if capturedMethod != "virt.global.update" {
		t.Errorf("expected method 'virt.global.update', got %q", capturedMethod)
	}

	// Verify params - should reset all fields to nil
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["bridge"] != nil {
		t.Errorf("expected bridge nil, got %v", params["bridge"])
	}
	if params["v4_network"] != nil {
		t.Errorf("expected v4_network nil, got %v", params["v4_network"])
	}
	if params["v6_network"] != nil {
		t.Errorf("expected v6_network nil, got %v", params["v6_network"])
	}
	if params["pool"] != nil {
		t.Errorf("expected pool nil, got %v", params["pool"])
	}
}

func TestVirtConfigResource_Delete_APIError(t *testing.T) {
	r := &VirtConfigResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("unable to reset config")
			},
		}},

	}

	schemaResp := getVirtConfigResourceSchema(t)
	stateValue := createVirtConfigModelValue(virtConfigModelParams{
		ID:            "virt_config",
		Bridge:        "br0",
		V4Network:     "10.0.0.0/24",
		V6Network:     "fd00::/64",
		Pool: "tank",
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

func TestVirtConfigResource_ImportState(t *testing.T) {
	r := NewVirtConfigResource().(*VirtConfigResource)

	schemaResp := getVirtConfigResourceSchema(t)

	// Create an initial empty state with the correct schema
	emptyState := createVirtConfigModelValue(virtConfigModelParams{
		ID:            nil,
		Bridge:        nil,
		V4Network:     nil,
		V6Network:     nil,
		Pool: nil,
	})

	req := resource.ImportStateRequest{
		ID: "virt_config",
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

	// Verify state was set with the ID
	var resultData VirtConfigResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "virt_config" {
		t.Errorf("expected ID 'virt_config', got %q", resultData.ID.ValueString())
	}
}

func TestVirtConfigResource_ImportState_InvalidID(t *testing.T) {
	r := &VirtConfigResource{
		BaseResource: BaseResource{client: &client.MockClient{}},
	}

	schemaResp := getVirtConfigResourceSchema(t)

	// Import with wrong ID
	req := resource.ImportStateRequest{
		ID: "wrong_id",
	}

	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.ImportState(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid import ID")
	}
}

// Test that VirtConfigResource implements the required interfaces
func TestVirtConfigResource_ImplementsInterfaces(t *testing.T) {
	r := NewVirtConfigResource()

	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(r.(*VirtConfigResource))
	_ = resource.ResourceWithImportState(r.(*VirtConfigResource))
}
