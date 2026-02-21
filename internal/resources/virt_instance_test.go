package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/truenas-go/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewVirtInstanceResource(t *testing.T) {
	r := NewVirtInstanceResource()
	if r == nil {
		t.Fatal("NewVirtInstanceResource returned nil")
	}

	containerResource, ok := r.(*VirtInstanceResource)
	if !ok {
		t.Fatalf("expected *VirtInstanceResource, got %T", r)
	}

	// Verify interface implementations
	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(containerResource)
	_ = resource.ResourceWithImportState(containerResource)
}

func TestVirtInstanceResource_Metadata(t *testing.T) {
	r := NewVirtInstanceResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_virt_instance" {
		t.Errorf("expected TypeName 'truenas_virt_instance', got %q", resp.TypeName)
	}
}

func TestVirtInstanceResource_Schema(t *testing.T) {
	r := NewVirtInstanceResource()

	ctx := context.Background()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}

	r.Schema(ctx, schemaReq, schemaResp)

	if schemaResp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	attrs := schemaResp.Schema.Attributes

	// Test required attributes
	requiredAttrs := []string{"name", "storage_pool", "image_name", "image_version"}
	for _, name := range requiredAttrs {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("expected %q attribute", name)
		}
		if !attr.IsRequired() {
			t.Errorf("expected %q attribute to be required", name)
		}
	}

	// Test computed attributes
	computedAttrs := []string{"id", "uuid", "state", "addresses"}
	for _, name := range computedAttrs {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("expected %q attribute", name)
		}
		if !attr.IsComputed() {
			t.Errorf("expected %q attribute to be computed", name)
		}
	}

	// Test optional attributes
	optionalAttrs := []string{
		"autostart", "desired_state", "state_timeout", "shutdown_timeout",
	}
	for _, name := range optionalAttrs {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("expected %q attribute", name)
		}
		if !attr.IsOptional() {
			t.Errorf("expected %q attribute to be optional", name)
		}
	}
}


// Test helpers

func getVirtInstanceResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewVirtInstanceResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("failed to get schema: %v", schemaResp.Diagnostics)
	}
	return *schemaResp
}

// mockVirtInstanceResponse generates a valid virt.instance.get_instance JSON response.
func mockVirtInstanceResponse(name, status string, autostart bool) json.RawMessage {
	return mockVirtInstanceResponseWithAliases(name, status, autostart, nil)
}

// mockVirtInstanceResponseWithAliases generates a virt.instance.get_instance response with custom aliases.
func mockVirtInstanceResponseWithAliases(name, status string, autostart bool, aliases []map[string]interface{}) json.RawMessage {
	aliasesJSON := "[]"
	if len(aliases) > 0 {
		aliasBytes, _ := json.Marshal(aliases)
		aliasesJSON = string(aliasBytes)
	}
	return json.RawMessage(fmt.Sprintf(`{
		"id": %q,
		"name": %q,
		"storage_pool": "tank",
		"image": {"os": "ubuntu", "release": "24.04", "architecture": "amd64", "description": "", "variant": ""},
		"status": %q,
		"autostart": %t,
		"aliases": %s
	}`, name, name, status, autostart, aliasesJSON))
}

// virtInstanceModelParams holds parameters for creating test model values.
type virtInstanceModelParams struct {
	ID              interface{}
	Name            interface{}
	StoragePool     interface{}
	ImageName       interface{}
	ImageVersion    interface{}
	Autostart       interface{}
	DesiredState    interface{}
	StateTimeout    interface{}
	State           interface{}
	UUID            interface{}
	ShutdownTimeout interface{}
	Addresses       []addressParams
	Disks           []diskParams
	NICs            []nicParams
	Proxies         []proxyParams
}

type addressParams struct {
	Type    interface{}
	Address interface{}
	Netmask interface{}
}

type diskParams struct {
	Name        interface{}
	Source      interface{}
	Destination interface{}
	Readonly    interface{}
}

type nicParams struct {
	Name    interface{}
	Network interface{}
	NICType interface{}
	Parent  interface{}
}

type proxyParams struct {
	Name        interface{}
	SourceProto interface{}
	SourcePort  interface{}
	DestProto   interface{}
	DestPort    interface{}
}

// diskBlockType returns the tftypes.Object type for disk blocks.
func diskBlockType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"name":        tftypes.String,
			"source":      tftypes.String,
			"destination": tftypes.String,
			"readonly":    tftypes.Bool,
		},
	}
}

// nicBlockType returns the tftypes.Object type for nic blocks.
func nicBlockType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"name":     tftypes.String,
			"network":  tftypes.String,
			"nic_type": tftypes.String,
			"parent":   tftypes.String,
		},
	}
}

// proxyBlockType returns the tftypes.Object type for proxy blocks.
func proxyBlockType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"name":         tftypes.String,
			"source_proto": tftypes.String,
			"source_port":  tftypes.Number,
			"dest_proto":   tftypes.String,
			"dest_port":    tftypes.Number,
		},
	}
}

// addressAttrType returns the tftypes.Object type for address attributes.
func addressAttrType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"type":    tftypes.String,
			"address": tftypes.String,
			"netmask": tftypes.Number,
		},
	}
}

func createVirtInstanceModelValue(p virtInstanceModelParams) tftypes.Value {
	// Build disk block values
	var diskValues []tftypes.Value
	for _, d := range p.Disks {
		diskValues = append(diskValues, tftypes.NewValue(diskBlockType(), map[string]tftypes.Value{
			"name":        tftypes.NewValue(tftypes.String, d.Name),
			"source":      tftypes.NewValue(tftypes.String, d.Source),
			"destination": tftypes.NewValue(tftypes.String, d.Destination),
			"readonly":    tftypes.NewValue(tftypes.Bool, d.Readonly),
		}))
	}
	var diskListValue tftypes.Value
	if len(diskValues) == 0 {
		diskListValue = tftypes.NewValue(tftypes.List{ElementType: diskBlockType()}, []tftypes.Value{})
	} else {
		diskListValue = tftypes.NewValue(tftypes.List{ElementType: diskBlockType()}, diskValues)
	}

	// Build nic block values
	var nicValues []tftypes.Value
	for _, n := range p.NICs {
		nicValues = append(nicValues, tftypes.NewValue(nicBlockType(), map[string]tftypes.Value{
			"name":     tftypes.NewValue(tftypes.String, n.Name),
			"network":  tftypes.NewValue(tftypes.String, n.Network),
			"nic_type": tftypes.NewValue(tftypes.String, n.NICType),
			"parent":   tftypes.NewValue(tftypes.String, n.Parent),
		}))
	}
	var nicListValue tftypes.Value
	if len(nicValues) == 0 {
		nicListValue = tftypes.NewValue(tftypes.List{ElementType: nicBlockType()}, []tftypes.Value{})
	} else {
		nicListValue = tftypes.NewValue(tftypes.List{ElementType: nicBlockType()}, nicValues)
	}

	// Build proxy block values
	var proxyValues []tftypes.Value
	for _, pr := range p.Proxies {
		proxyValues = append(proxyValues, tftypes.NewValue(proxyBlockType(), map[string]tftypes.Value{
			"name":         tftypes.NewValue(tftypes.String, pr.Name),
			"source_proto": tftypes.NewValue(tftypes.String, pr.SourceProto),
			"source_port":  tftypes.NewValue(tftypes.Number, pr.SourcePort),
			"dest_proto":   tftypes.NewValue(tftypes.String, pr.DestProto),
			"dest_port":    tftypes.NewValue(tftypes.Number, pr.DestPort),
		}))
	}
	var proxyListValue tftypes.Value
	if len(proxyValues) == 0 {
		proxyListValue = tftypes.NewValue(tftypes.List{ElementType: proxyBlockType()}, []tftypes.Value{})
	} else {
		proxyListValue = tftypes.NewValue(tftypes.List{ElementType: proxyBlockType()}, proxyValues)
	}

	// Build address attribute values
	var addressValues []tftypes.Value
	for _, a := range p.Addresses {
		addressValues = append(addressValues, tftypes.NewValue(addressAttrType(), map[string]tftypes.Value{
			"type":    tftypes.NewValue(tftypes.String, a.Type),
			"address": tftypes.NewValue(tftypes.String, a.Address),
			"netmask": tftypes.NewValue(tftypes.Number, a.Netmask),
		}))
	}
	var addressListValue tftypes.Value
	if len(addressValues) == 0 {
		addressListValue = tftypes.NewValue(tftypes.List{ElementType: addressAttrType()}, []tftypes.Value{})
	} else {
		addressListValue = tftypes.NewValue(tftypes.List{ElementType: addressAttrType()}, addressValues)
	}

	values := map[string]tftypes.Value{
		"id":               tftypes.NewValue(tftypes.String, p.ID),
		"name":             tftypes.NewValue(tftypes.String, p.Name),
		"storage_pool":     tftypes.NewValue(tftypes.String, p.StoragePool),
		"image_name":       tftypes.NewValue(tftypes.String, p.ImageName),
		"image_version":    tftypes.NewValue(tftypes.String, p.ImageVersion),
		"autostart":        tftypes.NewValue(tftypes.Bool, p.Autostart),
		"desired_state":    tftypes.NewValue(tftypes.String, p.DesiredState),
		"state_timeout":    tftypes.NewValue(tftypes.Number, p.StateTimeout),
		"state":            tftypes.NewValue(tftypes.String, p.State),
		"uuid":             tftypes.NewValue(tftypes.String, p.UUID),
		"shutdown_timeout": tftypes.NewValue(tftypes.Number, p.ShutdownTimeout),
		"addresses":        addressListValue,
		"disk":             diskListValue,
		"nic":              nicListValue,
		"proxy":            proxyListValue,
	}

	objectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":               tftypes.String,
			"name":             tftypes.String,
			"storage_pool":     tftypes.String,
			"image_name":       tftypes.String,
			"image_version":    tftypes.String,
			"autostart":        tftypes.Bool,
			"desired_state":    tftypes.String,
			"state_timeout":    tftypes.Number,
			"state":            tftypes.String,
			"uuid":             tftypes.String,
			"shutdown_timeout": tftypes.Number,
			"addresses":        tftypes.List{ElementType: addressAttrType()},
			"disk":             tftypes.List{ElementType: diskBlockType()},
			"nic":              tftypes.List{ElementType: nicBlockType()},
			"proxy":            tftypes.List{ElementType: proxyBlockType()},
		},
	}

	return tftypes.NewValue(objectType, values)
}

// Version check tests

func TestVirtInstanceResource_Create_VersionCheck(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 24, Minor: 10, Patch: 2, Build: 4},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	planValue := createVirtInstanceModelValue(virtInstanceModelParams{
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
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
		t.Fatal("expected error for TrueNAS 24.x")
	}

	// Verify error message mentions version requirement
	foundVersionError := false
	for _, d := range resp.Diagnostics.Errors() {
		if strings.Contains(d.Summary(), "Unsupported TrueNAS Version") {
			foundVersionError = true
			break
		}
	}
	if !foundVersionError {
		t.Error("expected version check error")
	}
}

func TestVirtInstanceResource_Read_VersionCheck(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 24, Minor: 10, Patch: 2, Build: 4},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "1",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
		State:        "RUNNING",
		UUID:         "abc-123",
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
		t.Fatal("expected error for TrueNAS 24.x")
	}
}

func TestVirtInstanceResource_Update_VersionCheck(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 24, Minor: 10, Patch: 2, Build: 4},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "1",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
		State:        "RUNNING",
	})
	planValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "1",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "STOPPED",
		StateTimeout: float64(90),
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
		t.Fatal("expected error for TrueNAS 24.x")
	}
}

// Create tests

func TestVirtInstanceResource_Create_Success(t *testing.T) {
	var capturedCreateMethod string
	var capturedCreateParams any

	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedCreateMethod = method
				capturedCreateParams = params
				// container.create returns the ID
				return json.RawMessage(`1`), nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return mockVirtInstanceResponse("test-container", "RUNNING", false), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	planValue := createVirtInstanceModelValue(virtInstanceModelParams{
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
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

	// Verify container.create was called
	if capturedCreateMethod != "virt.instance.create" {
		t.Errorf("expected method 'container.create', got %q", capturedCreateMethod)
	}

	// Verify params
	params, ok := capturedCreateParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedCreateParams)
	}
	if params["name"] != "test-container" {
		t.Errorf("expected name 'test-container', got %v", params["name"])
	}
	if params["storage_pool"] != "tank" {
		t.Errorf("expected storage_pool 'tank', got %v", params["storage_pool"])
	}
	image, ok := params["image"].(string)
	if !ok {
		t.Fatalf("expected image to be string, got %T", params["image"])
	}
	if image != "ubuntu/24.04" {
		t.Errorf("expected image 'ubuntu/24.04', got %v", image)
	}

	// Verify state was set
	var resultData VirtInstanceResourceModel
	resp.State.Get(context.Background(), &resultData)
	// virt.instance uses container name as ID
	if resultData.ID.ValueString() != "test-container" {
		t.Errorf("expected ID 'test-container', got %q", resultData.ID.ValueString())
	}
	if resultData.State.ValueString() != "RUNNING" {
		t.Errorf("expected State 'RUNNING', got %q", resultData.State.ValueString())
	}
	// UUID is set from container ID in virt.instance API
	if resultData.UUID.ValueString() != "test-container" {
		t.Errorf("expected UUID 'test-container', got %q", resultData.UUID.ValueString())
	}
}

func TestVirtInstanceResource_Create_WithDesiredStateStopped(t *testing.T) {
	var methods []string
	queryCount := 0
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				return json.RawMessage(`1`), nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				queryCount++
				// First query after create: RUNNING, subsequent: STOPPED
				if queryCount == 1 {
					return mockVirtInstanceResponse("test-container", "RUNNING", false), nil
				}
				return mockVirtInstanceResponse("test-container", "STOPPED", false), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	planValue := createVirtInstanceModelValue(virtInstanceModelParams{
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "STOPPED",
		StateTimeout: float64(90),
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

	// Verify container.create was called, then container.stop
	if len(methods) < 2 {
		t.Fatalf("expected at least 2 API calls, got %d: %v", len(methods), methods)
	}
	if methods[0] != "virt.instance.create" {
		t.Errorf("expected first call to be container.create, got %q", methods[0])
	}
	if methods[1] != "virt.instance.stop" {
		t.Errorf("expected second call to be container.stop, got %q", methods[1])
	}

	// Verify final state
	var model VirtInstanceResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.State.ValueString() != "STOPPED" {
		t.Errorf("expected final state STOPPED, got %q", model.State.ValueString())
	}
}

func TestVirtInstanceResource_Create_APIError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("container already exists")
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	planValue := createVirtInstanceModelValue(virtInstanceModelParams{
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
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

func TestVirtInstanceResource_Create_QueryErrorAfterCreate(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`1`), nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("query failed")
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	planValue := createVirtInstanceModelValue(virtInstanceModelParams{
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
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
		t.Fatal("expected error when query fails after create")
	}
}

func TestVirtInstanceResource_Create_NotFoundAfterCreate(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`1`), nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// virt.instance.get_instance returns error when not found
				return nil, errors.New("No such instance: test-container")
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	planValue := createVirtInstanceModelValue(virtInstanceModelParams{
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
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
		t.Fatal("expected error when container not found after create")
	}
}

// Read tests

func TestVirtInstanceResource_Read_Success(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method != "virt.instance.get_instance" {
					t.Errorf("expected method 'virt.instance.get_instance', got %q", method)
				}
				return mockVirtInstanceResponse("test-container", "RUNNING", true), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
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

	var resultData VirtInstanceResourceModel
	resp.State.Get(context.Background(), &resultData)

	// virt.instance uses container name as ID
	if resultData.ID.ValueString() != "test-container" {
		t.Errorf("expected ID 'test-container', got %q", resultData.ID.ValueString())
	}
	if resultData.State.ValueString() != "RUNNING" {
		t.Errorf("expected State 'RUNNING', got %q", resultData.State.ValueString())
	}
	// UUID is set from container ID
	if resultData.UUID.ValueString() != "test-container" {
		t.Errorf("expected UUID 'test-container', got %q", resultData.UUID.ValueString())
	}
	if !resultData.Autostart.ValueBool() {
		t.Error("expected Autostart to be true")
	}
}

func TestVirtInstanceResource_Read_NotFound(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// virt.instance.get_instance returns an error when not found
				return nil, errors.New("No such instance: test-container")
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "1",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
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

	// State should be empty (removed)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed (null) when container not found")
	}
}

func TestVirtInstanceResource_Read_APIError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection failed")
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "1",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
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

func TestVirtInstanceResource_Read_InvalidJSON(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "1",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
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

func TestVirtInstanceResource_Read_PreservesDesiredState(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// API reports RUNNING state, but user wants it STOPPED
				return mockVirtInstanceResponse("test-container", "RUNNING", false), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	// Prior state has desired_state = "STOPPED"
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "STOPPED",
		StateTimeout: float64(180),
		State:        "STOPPED",
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

	var model VirtInstanceResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// desired_state should be preserved from prior state
	if model.DesiredState.ValueString() != "STOPPED" {
		t.Errorf("expected desired_state 'STOPPED' to be preserved, got %q", model.DesiredState.ValueString())
	}
	// state_timeout should be preserved from prior state
	if model.StateTimeout.ValueInt64() != 180 {
		t.Errorf("expected state_timeout 180 to be preserved, got %d", model.StateTimeout.ValueInt64())
	}
	// state should reflect actual API state
	if model.State.ValueString() != "RUNNING" {
		t.Errorf("expected state 'RUNNING' from API, got %q", model.State.ValueString())
	}
}

// Update tests

func TestVirtInstanceResource_Update_ChangeConfig(t *testing.T) {
	var capturedUpdateParams any

	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.update" {
					capturedUpdateParams = params
				}
				return json.RawMessage(`null`), nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return mockVirtInstanceResponse("test-container", "RUNNING", true), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
		State:        "RUNNING",
		Autostart:    false,
	})
	planValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
		Autostart:    true,
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

	// Verify container.update was called with correct params
	if capturedUpdateParams == nil {
		t.Fatal("expected virt.instance.update to be called")
	}
	params, ok := capturedUpdateParams.([]any)
	if !ok {
		t.Fatalf("expected params to be []any, got %T", capturedUpdateParams)
	}
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	updateData, ok := params[1].(map[string]any)
	if !ok {
		t.Fatalf("expected second param to be map[string]any, got %T", params[1])
	}
	if updateData["autostart"] != true {
		t.Errorf("expected autostart true, got %v", updateData["autostart"])
	}
}

func TestVirtInstanceResource_Update_ChangeDesiredState(t *testing.T) {
	var methods []string
	queryCount := 0
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				return json.RawMessage(`null`), nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				queryCount++
				// First query: RUNNING, after stop: STOPPED
				if queryCount == 1 {
					return mockVirtInstanceResponse("test-container", "RUNNING", false), nil
				}
				return mockVirtInstanceResponse("test-container", "STOPPED", false), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
		State:        "RUNNING",
	})
	planValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "STOPPED",
		StateTimeout: float64(90),
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

	// Verify container.stop was called
	foundStop := false
	for _, m := range methods {
		if m == "virt.instance.stop" {
			foundStop = true
			break
		}
	}
	if !foundStop {
		t.Errorf("expected container.stop to be called, got methods: %v", methods)
	}

	// Verify final state
	var model VirtInstanceResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.State.ValueString() != "STOPPED" {
		t.Errorf("expected state 'STOPPED', got %q", model.State.ValueString())
	}
}

func TestVirtInstanceResource_Update_APIError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("update failed")
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": 1,
					"name": "test-container",
					"pool": "tank",
					"image": {"name": "ubuntu", "version": "24.04"},
					"uuid": "abc-123",
					"state": "RUNNING",
					"dataset": "tank/containers/test-container"
				}]`), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "1",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
		State:        "RUNNING",
		Autostart:    false,
	})
	planValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "1",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
		Autostart:    true,
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

// Delete tests

func TestVirtInstanceResource_Delete_RunningContainer(t *testing.T) {
	var methods []string
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				return json.RawMessage(`null`), nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return mockVirtInstanceResponse("test-container", "RUNNING", false), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
		State:        "RUNNING",
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

	// Verify virt.instance.stop was called first, then virt.instance.delete
	if len(methods) < 2 {
		t.Fatalf("expected at least 2 API calls, got %d: %v", len(methods), methods)
	}
	if methods[0] != "virt.instance.stop" {
		t.Errorf("expected first call to be virt.instance.stop, got %q", methods[0])
	}
	if methods[1] != "virt.instance.delete" {
		t.Errorf("expected second call to be virt.instance.delete, got %q", methods[1])
	}
}

func TestVirtInstanceResource_Delete_StoppedContainer(t *testing.T) {
	var methods []string
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				return json.RawMessage(`null`), nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return mockVirtInstanceResponse("test-container", "STOPPED", false), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "STOPPED",
		StateTimeout: float64(90),
		State:        "STOPPED",
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

	// Verify only container.delete was called (no stop needed)
	if len(methods) != 1 {
		t.Fatalf("expected 1 API call, got %d: %v", len(methods), methods)
	}
	if methods[0] != "virt.instance.delete" {
		t.Errorf("expected container.delete, got %q", methods[0])
	}
}

func TestVirtInstanceResource_Delete_APIError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("delete failed")
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": 1,
					"name": "test-container",
					"state": "STOPPED"
				}]`), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "1",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "STOPPED",
		StateTimeout: float64(90),
		State:        "STOPPED",
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

// Import tests

func TestVirtInstanceResource_ImportState(t *testing.T) {
	r := NewVirtInstanceResource().(*VirtInstanceResource)

	schemaResp := getVirtInstanceResourceSchema(t)
	emptyState := createVirtInstanceModelValue(virtInstanceModelParams{})

	req := resource.ImportStateRequest{
		ID: "test-container",
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

	// Verify name was set from import ID
	var model VirtInstanceResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.Name.ValueString() != "test-container" {
		t.Errorf("expected Name 'test-container', got %q", model.Name.ValueString())
	}
}

// Additional edge case tests

func TestVirtInstanceResource_Create_WithDevices(t *testing.T) {
	var capturedParams any
	var deviceListCalled bool

	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.create" {
					capturedParams = params
				}
				return json.RawMessage(`"1"`), nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.device_list" {
					deviceListCalled = true
					return json.RawMessage(`[
						{"dev_type": "DISK", "name": "data", "source": "/mnt/tank/data", "destination": "/data", "readonly": false}
					]`), nil
				}
				return json.RawMessage(`{
					"id": "1",
					"name": "test-container",
					"storage_pool": "tank",
					"image": {"os": "ubuntu", "release": "24.04", "architecture": "amd64", "description": "", "variant": ""},
					"status": "RUNNING",
					"autostart": true,
					"aliases": []
				}`), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	planValue := createVirtInstanceModelValue(virtInstanceModelParams{
		Name:            "test-container",
		StoragePool:     "tank",
		ImageName:       "ubuntu",
		ImageVersion:    "24.04",
		DesiredState:    "RUNNING",
		StateTimeout:    float64(90),
		Autostart:       true,
		ShutdownTimeout: float64(60),
		Disks: []diskParams{
			{Name: "data", Source: "/mnt/tank/data", Destination: "/data", Readonly: false},
		},
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

	// Verify params
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}
	if params["autostart"] != true {
		t.Errorf("expected autostart true, got %v", params["autostart"])
	}

	// Verify devices were included in create params
	devices, ok := params["devices"].([]map[string]any)
	if !ok || len(devices) == 0 {
		t.Errorf("expected devices in params, got %v", params["devices"])
	}

	if !deviceListCalled {
		t.Error("expected device_list to be called after create")
	}
}

func TestVirtInstanceResource_getVirtInstanceState(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method != "virt.instance.get_instance" {
					t.Errorf("expected method 'virt.instance.get_instance', got %q", method)
				}
				return json.RawMessage(`{"id": "1", "name": "test-container", "status": "RUNNING", "storage_pool": "tank", "autostart": false, "aliases": [], "image": {"os": "ubuntu", "release": "24.04", "architecture": "amd64", "description": "", "variant": ""}}`), nil
			},
		}},

	}

	state, err := r.getVirtInstanceState(context.Background(), "test-container")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != "RUNNING" {
		t.Errorf("expected state RUNNING, got %q", state)
	}
}

func TestVirtInstanceResource_getVirtInstanceState_NotFound(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// virt.instance.get_instance returns error when not found
				return nil, errors.New("No such instance: test-container")
			},
		}},

	}

	_, err := r.getVirtInstanceState(context.Background(), "test-container")
	if err == nil {
		t.Fatal("expected error for container not found")
	}
}

func TestVirtInstanceResource_reconcileDesiredState_StartContainer(t *testing.T) {
	var calledMethod string
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				calledMethod = method
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`{"id": "1", "name": "test-container", "status": "RUNNING", "storage_pool": "tank", "autostart": false, "aliases": [], "image": {"os": "ubuntu", "release": "24.04", "architecture": "amd64", "description": "", "variant": ""}}`), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}
	err := r.reconcileDesiredState(context.Background(), "test-container", "1", VirtInstanceStateStopped, VirtInstanceStateRunning, 30*time.Second, 30, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledMethod != "virt.instance.start" {
		t.Errorf("expected container.start to be called, got %q", calledMethod)
	}
}

func TestVirtInstanceResource_reconcileDesiredState_StopContainer(t *testing.T) {
	var calledMethod string
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				calledMethod = method
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`{"id": "1", "name": "test-container", "status": "STOPPED", "storage_pool": "tank", "autostart": false, "aliases": [], "image": {"os": "ubuntu", "release": "24.04", "architecture": "amd64", "description": "", "variant": ""}}`), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}
	err := r.reconcileDesiredState(context.Background(), "test-container", "1", VirtInstanceStateRunning, VirtInstanceStateStopped, 30*time.Second, 30, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledMethod != "virt.instance.stop" {
		t.Errorf("expected container.stop to be called, got %q", calledMethod)
	}
}

func TestVirtInstanceResource_reconcileDesiredState_NoChangeNeeded(t *testing.T) {
	callCount := 0
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				callCount++
				return nil, nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}
	err := r.reconcileDesiredState(context.Background(), "test-container", "1", VirtInstanceStateRunning, VirtInstanceStateRunning, 30*time.Second, 30, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 0 {
		t.Errorf("expected no API calls when state matches, got %d calls", callCount)
	}
}

// Test interface compliance
func TestVirtInstanceResource_ImplementsInterfaces(t *testing.T) {
	r := NewVirtInstanceResource()

	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(r.(*VirtInstanceResource))
	_ = resource.ResourceWithImportState(r.(*VirtInstanceResource))
}

// Helper to create a string pointer
func strPtr(s string) *string {
	return &s
}

// Helper to create an int64 pointer
func int64Ptr(i int64) *int64 {
	return &i
}

// Tests for getManagedDeviceNames
func TestGetManagedDeviceNames_Empty(t *testing.T) {
	data := &VirtInstanceResourceModel{}
	names := getManagedDeviceNames(data)
	if len(names) != 0 {
		t.Errorf("expected empty map, got %d entries", len(names))
	}
}

func TestGetManagedDeviceNames_WithDisks(t *testing.T) {
	data := &VirtInstanceResourceModel{
		Disks: []DiskModel{
			{Name: types.StringValue("disk1")},
			{Name: types.StringValue("disk2")},
			{Name: types.StringNull()}, // Should be skipped
			{Name: types.StringValue("")}, // Should be skipped
		},
	}
	names := getManagedDeviceNames(data)
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}
	if !names["disk1"] {
		t.Error("expected disk1 to be in names")
	}
	if !names["disk2"] {
		t.Error("expected disk2 to be in names")
	}
}

func TestGetManagedDeviceNames_WithNICs(t *testing.T) {
	data := &VirtInstanceResourceModel{
		NICs: []NICModel{
			{Name: types.StringValue("eth0")},
			{Name: types.StringNull()},
		},
	}
	names := getManagedDeviceNames(data)
	if len(names) != 1 {
		t.Errorf("expected 1 name, got %d", len(names))
	}
	if !names["eth0"] {
		t.Error("expected eth0 to be in names")
	}
}

func TestGetManagedDeviceNames_WithProxies(t *testing.T) {
	data := &VirtInstanceResourceModel{
		Proxies: []ProxyModel{
			{Name: types.StringValue("proxy1")},
			{Name: types.StringValue("proxy2")},
		},
	}
	names := getManagedDeviceNames(data)
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}
}

func TestGetManagedDeviceNames_AllDeviceTypes(t *testing.T) {
	data := &VirtInstanceResourceModel{
		Disks: []DiskModel{
			{Name: types.StringValue("disk1")},
		},
		NICs: []NICModel{
			{Name: types.StringValue("eth0")},
		},
		Proxies: []ProxyModel{
			{Name: types.StringValue("proxy1")},
		},
	}
	names := getManagedDeviceNames(data)
	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}
}

// Tests for buildDevices
func TestVirtInstanceResource_buildDevices_Empty(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{}
	devices := r.buildDevices(data)
	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}
}

func TestVirtInstanceResource_buildDevices_Disks(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{
		Disks: []DiskModel{
			{
				Name:        types.StringValue("data"),
				Source:      types.StringValue("/mnt/tank/data"),
				Destination: types.StringValue("/data"),
				Readonly:    types.BoolValue(true),
			},
			{
				Name:        types.StringNull(), // No name
				Source:      types.StringValue("/mnt/tank/backup"),
				Destination: types.StringValue("/backup"),
				Readonly:    types.BoolNull(), // No readonly specified
			},
		},
	}
	devices := r.buildDevices(data)
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}

	// First disk with all fields
	if devices[0]["dev_type"] != "DISK" {
		t.Errorf("expected dev_type 'DISK', got %v", devices[0]["dev_type"])
	}
	if devices[0]["name"] != "data" {
		t.Errorf("expected name 'data', got %v", devices[0]["name"])
	}
	if devices[0]["source"] != "/mnt/tank/data" {
		t.Errorf("expected source '/mnt/tank/data', got %v", devices[0]["source"])
	}
	if devices[0]["readonly"] != true {
		t.Errorf("expected readonly true, got %v", devices[0]["readonly"])
	}

	// Second disk without name or readonly
	if _, hasName := devices[1]["name"]; hasName {
		t.Error("expected no name field for second disk")
	}
	if _, hasReadonly := devices[1]["readonly"]; hasReadonly {
		t.Error("expected no readonly field for second disk")
	}
}

func TestVirtInstanceResource_buildDevices_NICs(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{
		NICs: []NICModel{
			{
				Name:    types.StringValue("eth0"),
				Network: types.StringValue("bridge0"),
				NICType: types.StringValue("BRIDGED"),
				Parent:  types.StringNull(),
			},
			{
				Name:    types.StringValue("eth1"),
				Network: types.StringNull(),
				NICType: types.StringValue("MACVLAN"),
				Parent:  types.StringValue("enp0s3"),
			},
			{
				Name:    types.StringNull(),
				Network: types.StringNull(),
				NICType: types.StringNull(),
				Parent:  types.StringNull(),
			},
		},
	}
	devices := r.buildDevices(data)
	if len(devices) != 3 {
		t.Fatalf("expected 3 devices, got %d", len(devices))
	}

	// First NIC with network
	if devices[0]["dev_type"] != "NIC" {
		t.Errorf("expected dev_type 'NIC', got %v", devices[0]["dev_type"])
	}
	if devices[0]["network"] != "bridge0" {
		t.Errorf("expected network 'bridge0', got %v", devices[0]["network"])
	}
	if devices[0]["nic_type"] != "BRIDGED" {
		t.Errorf("expected nic_type 'BRIDGED', got %v", devices[0]["nic_type"])
	}

	// Second NIC with parent (MACVLAN)
	if devices[1]["parent"] != "enp0s3" {
		t.Errorf("expected parent 'enp0s3', got %v", devices[1]["parent"])
	}

	// Third NIC with minimal fields
	if _, hasNetwork := devices[2]["network"]; hasNetwork {
		t.Error("expected no network field for third NIC")
	}
}

func TestVirtInstanceResource_buildDevices_Proxies(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{
		Proxies: []ProxyModel{
			{
				Name:        types.StringValue("http"),
				SourceProto: types.StringValue("TCP"),
				SourcePort:  types.Int64Value(8080),
				DestProto:   types.StringValue("TCP"),
				DestPort:    types.Int64Value(80),
			},
			{
				Name:        types.StringNull(), // No name
				SourceProto: types.StringValue("UDP"),
				SourcePort:  types.Int64Value(5353),
				DestProto:   types.StringValue("UDP"),
				DestPort:    types.Int64Value(53),
			},
		},
	}
	devices := r.buildDevices(data)
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}

	// First proxy with name
	if devices[0]["dev_type"] != "PROXY" {
		t.Errorf("expected dev_type 'PROXY', got %v", devices[0]["dev_type"])
	}
	if devices[0]["name"] != "http" {
		t.Errorf("expected name 'http', got %v", devices[0]["name"])
	}
	if devices[0]["source_proto"] != "TCP" {
		t.Errorf("expected source_proto 'TCP', got %v", devices[0]["source_proto"])
	}
	if devices[0]["source_port"] != int64(8080) {
		t.Errorf("expected source_port 8080, got %v", devices[0]["source_port"])
	}

	// Second proxy without name
	if _, hasName := devices[1]["name"]; hasName {
		t.Error("expected no name field for second proxy")
	}
}

func TestVirtInstanceResource_buildDevices_AllTypes(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{
		Disks: []DiskModel{
			{Name: types.StringValue("disk1"), Source: types.StringValue("/src"), Destination: types.StringValue("/dst")},
		},
		NICs: []NICModel{
			{Name: types.StringValue("eth0"), Network: types.StringValue("bridge0")},
		},
		Proxies: []ProxyModel{
			{Name: types.StringValue("proxy1"), SourceProto: types.StringValue("TCP"), SourcePort: types.Int64Value(80), DestProto: types.StringValue("TCP"), DestPort: types.Int64Value(80)},
		},
	}
	devices := r.buildDevices(data)
	if len(devices) != 3 {
		t.Fatalf("expected 3 devices, got %d", len(devices))
	}
}

// Tests for mapDevicesToModel
func TestVirtInstanceResource_mapDevicesToModel_Empty(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{}
	r.mapDevicesToModel([]deviceAPIResponse{}, data, nil)
	if len(data.Disks) != 0 || len(data.NICs) != 0 || len(data.Proxies) != 0 {
		t.Error("expected empty device lists")
	}
}

func TestVirtInstanceResource_mapDevicesToModel_Disks(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{}
	devices := []deviceAPIResponse{
		{
			DevType:     "DISK",
			Name:        strPtr("data"),
			Source:      strPtr("/mnt/tank/data"),
			Destination: strPtr("/data"),
			Readonly:    true,
		},
		{
			DevType:     "DISK",
			Name:        nil, // No name
			Source:      nil,
			Destination: nil,
			Readonly:    false,
		},
	}
	r.mapDevicesToModel(devices, data, nil)
	if len(data.Disks) != 2 {
		t.Fatalf("expected 2 disks, got %d", len(data.Disks))
	}
	if data.Disks[0].Name.ValueString() != "data" {
		t.Errorf("expected disk name 'data', got %q", data.Disks[0].Name.ValueString())
	}
	if data.Disks[0].Source.ValueString() != "/mnt/tank/data" {
		t.Errorf("expected source '/mnt/tank/data', got %q", data.Disks[0].Source.ValueString())
	}
	if !data.Disks[0].Readonly.ValueBool() {
		t.Error("expected readonly true")
	}
	if !data.Disks[1].Name.IsNull() {
		t.Error("expected null name for second disk")
	}
}

func TestVirtInstanceResource_mapDevicesToModel_NICs(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{}
	devices := []deviceAPIResponse{
		{
			DevType: "NIC",
			Name:    strPtr("eth0"),
			Network: strPtr("bridge0"),
			NICType: strPtr("BRIDGED"),
			Parent:  nil,
		},
		{
			DevType: "NIC",
			Name:    strPtr("eth1"),
			Network: nil,
			NICType: nil,
			Parent:  strPtr("enp0s3"),
		},
	}
	r.mapDevicesToModel(devices, data, nil)
	if len(data.NICs) != 2 {
		t.Fatalf("expected 2 NICs, got %d", len(data.NICs))
	}
	if data.NICs[0].Network.ValueString() != "bridge0" {
		t.Errorf("expected network 'bridge0', got %q", data.NICs[0].Network.ValueString())
	}
	if data.NICs[0].NICType.ValueString() != "BRIDGED" {
		t.Errorf("expected nic_type 'BRIDGED', got %q", data.NICs[0].NICType.ValueString())
	}
	if !data.NICs[0].Parent.IsNull() {
		t.Error("expected null parent for first NIC")
	}
	if data.NICs[1].Parent.ValueString() != "enp0s3" {
		t.Errorf("expected parent 'enp0s3', got %q", data.NICs[1].Parent.ValueString())
	}
}

func TestVirtInstanceResource_mapDevicesToModel_Proxies(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{}
	devices := []deviceAPIResponse{
		{
			DevType:     "PROXY",
			Name:        strPtr("http"),
			SourceProto: strPtr("TCP"),
			SourcePort:  int64Ptr(8080),
			DestProto:   strPtr("TCP"),
			DestPort:    int64Ptr(80),
		},
		{
			DevType:     "PROXY",
			Name:        nil,
			SourceProto: nil,
			SourcePort:  nil,
			DestProto:   nil,
			DestPort:    nil,
		},
	}
	r.mapDevicesToModel(devices, data, nil)
	if len(data.Proxies) != 2 {
		t.Fatalf("expected 2 proxies, got %d", len(data.Proxies))
	}
	if data.Proxies[0].Name.ValueString() != "http" {
		t.Errorf("expected name 'http', got %q", data.Proxies[0].Name.ValueString())
	}
	if data.Proxies[0].SourcePort.ValueInt64() != 8080 {
		t.Errorf("expected source_port 8080, got %d", data.Proxies[0].SourcePort.ValueInt64())
	}
}

func TestVirtInstanceResource_mapDevicesToModel_WithManagedFilter(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{}
	managedNames := map[string]bool{"data": true}
	devices := []deviceAPIResponse{
		{DevType: "DISK", Name: strPtr("data"), Source: strPtr("/src"), Destination: strPtr("/dst")},
		{DevType: "DISK", Name: strPtr("system"), Source: strPtr("/sys"), Destination: strPtr("/system")}, // Should be filtered out
		{DevType: "DISK", Name: nil}, // Should be filtered out (no name when filtering)
	}
	r.mapDevicesToModel(devices, data, managedNames)
	if len(data.Disks) != 1 {
		t.Fatalf("expected 1 disk (filtered), got %d", len(data.Disks))
	}
	if data.Disks[0].Name.ValueString() != "data" {
		t.Errorf("expected disk name 'data', got %q", data.Disks[0].Name.ValueString())
	}
}

func TestVirtInstanceResource_mapDevicesToModel_AllTypes(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{}
	devices := []deviceAPIResponse{
		{DevType: "DISK", Name: strPtr("disk1"), Source: strPtr("/src"), Destination: strPtr("/dst")},
		{DevType: "NIC", Name: strPtr("eth0"), Network: strPtr("bridge0")},
		{DevType: "PROXY", Name: strPtr("http"), SourceProto: strPtr("TCP"), SourcePort: int64Ptr(80), DestProto: strPtr("TCP"), DestPort: int64Ptr(80)},
		{DevType: "UNKNOWN", Name: strPtr("unknown")}, // Unknown type should be ignored
	}
	r.mapDevicesToModel(devices, data, nil)
	if len(data.Disks) != 1 {
		t.Errorf("expected 1 disk, got %d", len(data.Disks))
	}
	if len(data.NICs) != 1 {
		t.Errorf("expected 1 NIC, got %d", len(data.NICs))
	}
	if len(data.Proxies) != 1 {
		t.Errorf("expected 1 proxy, got %d", len(data.Proxies))
	}
}

// Tests for matchCreatedDevices
func TestVirtInstanceResource_matchCreatedDevices_Disks(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{
		Disks: []DiskModel{
			{Name: types.StringValue("existing"), Source: types.StringValue("/src1"), Destination: types.StringValue("/dst1")},
			{Name: types.StringNull(), Source: types.StringValue("/src2"), Destination: types.StringValue("/dst2")}, // Should get name from API
		},
	}
	apiDevices := []deviceAPIResponse{
		{DevType: "DISK", Name: strPtr("existing"), Source: strPtr("/src1"), Destination: strPtr("/dst1")},
		{DevType: "DISK", Name: strPtr("auto-disk-1"), Source: strPtr("/src2"), Destination: strPtr("/dst2")},
		{DevType: "DISK", Name: nil, Source: strPtr("/src3"), Destination: strPtr("/dst3")}, // No name, should be skipped
	}
	r.matchCreatedDevices(apiDevices, data)
	if data.Disks[0].Name.ValueString() != "existing" {
		t.Errorf("expected 'existing', got %q", data.Disks[0].Name.ValueString())
	}
	if data.Disks[1].Name.ValueString() != "auto-disk-1" {
		t.Errorf("expected 'auto-disk-1', got %q", data.Disks[1].Name.ValueString())
	}
}

func TestVirtInstanceResource_matchCreatedDevices_NICs_ByNetwork(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{
		NICs: []NICModel{
			{Name: types.StringNull(), Network: types.StringValue("bridge0"), NICType: types.StringNull(), Parent: types.StringNull()},
		},
	}
	apiDevices := []deviceAPIResponse{
		{DevType: "NIC", Name: strPtr("eth0"), Network: strPtr("bridge0")},
	}
	r.matchCreatedDevices(apiDevices, data)
	if data.NICs[0].Name.ValueString() != "eth0" {
		t.Errorf("expected 'eth0', got %q", data.NICs[0].Name.ValueString())
	}
}

func TestVirtInstanceResource_matchCreatedDevices_NICs_ByParent(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{
		NICs: []NICModel{
			{Name: types.StringNull(), Network: types.StringNull(), NICType: types.StringValue("MACVLAN"), Parent: types.StringValue("enp0s3")},
		},
	}
	apiDevices := []deviceAPIResponse{
		{DevType: "NIC", Name: strPtr("macvlan0"), Parent: strPtr("enp0s3")},
	}
	r.matchCreatedDevices(apiDevices, data)
	if data.NICs[0].Name.ValueString() != "macvlan0" {
		t.Errorf("expected 'macvlan0', got %q", data.NICs[0].Name.ValueString())
	}
}

func TestVirtInstanceResource_matchCreatedDevices_NICs_AlreadyNamed(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{
		NICs: []NICModel{
			{Name: types.StringValue("mynic"), Network: types.StringValue("bridge0")},
		},
	}
	apiDevices := []deviceAPIResponse{
		{DevType: "NIC", Name: strPtr("eth0"), Network: strPtr("bridge0")},
	}
	r.matchCreatedDevices(apiDevices, data)
	// Should keep the original name
	if data.NICs[0].Name.ValueString() != "mynic" {
		t.Errorf("expected 'mynic', got %q", data.NICs[0].Name.ValueString())
	}
}

func TestVirtInstanceResource_matchCreatedDevices_Proxies(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{
		Proxies: []ProxyModel{
			{
				Name:        types.StringNull(),
				SourceProto: types.StringValue("TCP"),
				SourcePort:  types.Int64Value(8080),
				DestProto:   types.StringValue("TCP"),
				DestPort:    types.Int64Value(80),
			},
		},
	}
	apiDevices := []deviceAPIResponse{
		{
			DevType:     "PROXY",
			Name:        strPtr("proxy-tcp-8080"),
			SourceProto: strPtr("TCP"),
			SourcePort:  int64Ptr(8080),
			DestProto:   strPtr("TCP"),
			DestPort:    int64Ptr(80),
		},
	}
	r.matchCreatedDevices(apiDevices, data)
	if data.Proxies[0].Name.ValueString() != "proxy-tcp-8080" {
		t.Errorf("expected 'proxy-tcp-8080', got %q", data.Proxies[0].Name.ValueString())
	}
}

func TestVirtInstanceResource_matchCreatedDevices_NoMatch(t *testing.T) {
	r := &VirtInstanceResource{}
	data := &VirtInstanceResourceModel{
		Disks: []DiskModel{
			{Name: types.StringNull(), Source: types.StringValue("/src"), Destination: types.StringValue("/dst")},
		},
		NICs: []NICModel{
			{Name: types.StringNull(), Network: types.StringValue("nonexistent")},
		},
		Proxies: []ProxyModel{
			{Name: types.StringNull(), SourceProto: types.StringValue("UDP"), SourcePort: types.Int64Value(1234), DestProto: types.StringValue("UDP"), DestPort: types.Int64Value(5678)},
		},
	}
	apiDevices := []deviceAPIResponse{
		{DevType: "DISK", Name: strPtr("other"), Source: strPtr("/other"), Destination: strPtr("/other")},
		{DevType: "NIC", Name: strPtr("eth0"), Network: strPtr("different")},
		{DevType: "PROXY", Name: strPtr("proxy"), SourceProto: strPtr("TCP"), SourcePort: int64Ptr(80), DestProto: strPtr("TCP"), DestPort: int64Ptr(80)},
	}
	r.matchCreatedDevices(apiDevices, data)
	// Names should remain null since no match
	if !data.Disks[0].Name.IsNull() {
		t.Error("expected disk name to remain null")
	}
	if !data.NICs[0].Name.IsNull() {
		t.Error("expected NIC name to remain null")
	}
	if !data.Proxies[0].Name.IsNull() {
		t.Error("expected proxy name to remain null")
	}
}

// Tests for reconcileDevices
func TestVirtInstanceResource_reconcileDevices_NoChanges(t *testing.T) {
	callCount := 0
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				callCount++
				return nil, nil
			},
		}},

	}
	plan := &VirtInstanceResourceModel{
		Disks: []DiskModel{{Name: types.StringValue("disk1")}},
	}
	state := &VirtInstanceResourceModel{
		Disks: []DiskModel{{Name: types.StringValue("disk1")}},
	}
	err := r.reconcileDevices(context.Background(), "test-id", plan, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 0 {
		t.Errorf("expected 0 API calls, got %d", callCount)
	}
}

func TestVirtInstanceResource_reconcileDevices_DeleteDevice(t *testing.T) {
	var deletedDevices []string
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.device_delete" {
					p := params.([]any)
					deletedDevices = append(deletedDevices, p[1].(string))
				}
				return nil, nil
			},
		}},

	}
	plan := &VirtInstanceResourceModel{
		Disks: []DiskModel{}, // No disks in plan
	}
	state := &VirtInstanceResourceModel{
		Disks: []DiskModel{
			{Name: types.StringValue("disk1")},
			{Name: types.StringValue("disk2")},
		},
	}
	err := r.reconcileDevices(context.Background(), "test-id", plan, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deletedDevices) != 2 {
		t.Errorf("expected 2 deletes, got %d", len(deletedDevices))
	}
}

func TestVirtInstanceResource_reconcileDevices_AddDisk(t *testing.T) {
	var addedDevices []map[string]any
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.device_add" {
					p := params.([]any)
					addedDevices = append(addedDevices, p[1].(map[string]any))
				}
				return nil, nil
			},
		}},

	}
	plan := &VirtInstanceResourceModel{
		Disks: []DiskModel{
			{Name: types.StringValue("newdisk"), Source: types.StringValue("/src"), Destination: types.StringValue("/dst"), Readonly: types.BoolValue(true)},
		},
	}
	state := &VirtInstanceResourceModel{}
	err := r.reconcileDevices(context.Background(), "test-id", plan, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addedDevices) != 1 {
		t.Fatalf("expected 1 add, got %d", len(addedDevices))
	}
	if addedDevices[0]["dev_type"] != "DISK" {
		t.Errorf("expected dev_type 'DISK', got %v", addedDevices[0]["dev_type"])
	}
	if addedDevices[0]["name"] != "newdisk" {
		t.Errorf("expected name 'newdisk', got %v", addedDevices[0]["name"])
	}
	if addedDevices[0]["readonly"] != true {
		t.Errorf("expected readonly true, got %v", addedDevices[0]["readonly"])
	}
}

func TestVirtInstanceResource_reconcileDevices_AddNIC(t *testing.T) {
	var addedDevices []map[string]any
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.device_add" {
					p := params.([]any)
					addedDevices = append(addedDevices, p[1].(map[string]any))
				}
				return nil, nil
			},
		}},

	}
	plan := &VirtInstanceResourceModel{
		NICs: []NICModel{
			{Name: types.StringValue("eth0"), Network: types.StringValue("bridge0"), NICType: types.StringValue("BRIDGED"), Parent: types.StringNull()},
		},
	}
	state := &VirtInstanceResourceModel{}
	err := r.reconcileDevices(context.Background(), "test-id", plan, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addedDevices) != 1 {
		t.Fatalf("expected 1 add, got %d", len(addedDevices))
	}
	if addedDevices[0]["dev_type"] != "NIC" {
		t.Errorf("expected dev_type 'NIC', got %v", addedDevices[0]["dev_type"])
	}
	if addedDevices[0]["network"] != "bridge0" {
		t.Errorf("expected network 'bridge0', got %v", addedDevices[0]["network"])
	}
}

func TestVirtInstanceResource_reconcileDevices_AddNIC_WithParent(t *testing.T) {
	var addedDevices []map[string]any
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.device_add" {
					p := params.([]any)
					addedDevices = append(addedDevices, p[1].(map[string]any))
				}
				return nil, nil
			},
		}},

	}
	plan := &VirtInstanceResourceModel{
		NICs: []NICModel{
			{Name: types.StringValue("macvlan0"), Network: types.StringNull(), NICType: types.StringValue("MACVLAN"), Parent: types.StringValue("enp0s3")},
		},
	}
	state := &VirtInstanceResourceModel{}
	err := r.reconcileDevices(context.Background(), "test-id", plan, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addedDevices) != 1 {
		t.Fatalf("expected 1 add, got %d", len(addedDevices))
	}
	if addedDevices[0]["parent"] != "enp0s3" {
		t.Errorf("expected parent 'enp0s3', got %v", addedDevices[0]["parent"])
	}
}

func TestVirtInstanceResource_reconcileDevices_AddProxy(t *testing.T) {
	var addedDevices []map[string]any
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.device_add" {
					p := params.([]any)
					addedDevices = append(addedDevices, p[1].(map[string]any))
				}
				return nil, nil
			},
		}},

	}
	plan := &VirtInstanceResourceModel{
		Proxies: []ProxyModel{
			{Name: types.StringValue("http"), SourceProto: types.StringValue("TCP"), SourcePort: types.Int64Value(8080), DestProto: types.StringValue("TCP"), DestPort: types.Int64Value(80)},
		},
	}
	state := &VirtInstanceResourceModel{}
	err := r.reconcileDevices(context.Background(), "test-id", plan, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addedDevices) != 1 {
		t.Fatalf("expected 1 add, got %d", len(addedDevices))
	}
	if addedDevices[0]["dev_type"] != "PROXY" {
		t.Errorf("expected dev_type 'PROXY', got %v", addedDevices[0]["dev_type"])
	}
	if addedDevices[0]["source_port"] != int64(8080) {
		t.Errorf("expected source_port 8080, got %v", addedDevices[0]["source_port"])
	}
}

func TestVirtInstanceResource_reconcileDevices_DeleteError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.device_delete" {
					return nil, errors.New("delete failed")
				}
				return nil, nil
			},
		}},

	}
	plan := &VirtInstanceResourceModel{}
	state := &VirtInstanceResourceModel{
		Disks: []DiskModel{{Name: types.StringValue("disk1")}},
	}
	err := r.reconcileDevices(context.Background(), "test-id", plan, state)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "delete failed") {
		t.Errorf("expected error to contain 'delete failed', got %v", err)
	}
}

func TestVirtInstanceResource_reconcileDevices_AddDiskError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.device_add" {
					return nil, errors.New("add disk failed")
				}
				return nil, nil
			},
		}},

	}
	plan := &VirtInstanceResourceModel{
		Disks: []DiskModel{{Name: types.StringValue("disk1"), Source: types.StringValue("/src"), Destination: types.StringValue("/dst")}},
	}
	state := &VirtInstanceResourceModel{}
	err := r.reconcileDevices(context.Background(), "test-id", plan, state)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "add disk failed") {
		t.Errorf("expected error to contain 'add disk failed', got %v", err)
	}
}

func TestVirtInstanceResource_reconcileDevices_AddNICError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.device_add" {
					return nil, errors.New("add NIC failed")
				}
				return nil, nil
			},
		}},

	}
	plan := &VirtInstanceResourceModel{
		NICs: []NICModel{{Name: types.StringValue("eth0"), Network: types.StringValue("bridge0")}},
	}
	state := &VirtInstanceResourceModel{}
	err := r.reconcileDevices(context.Background(), "test-id", plan, state)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "add NIC failed") {
		t.Errorf("expected error to contain 'add NIC failed', got %v", err)
	}
}

func TestVirtInstanceResource_reconcileDevices_AddProxyError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.device_add" {
					return nil, errors.New("add proxy failed")
				}
				return nil, nil
			},
		}},

	}
	plan := &VirtInstanceResourceModel{
		Proxies: []ProxyModel{{Name: types.StringValue("http"), SourceProto: types.StringValue("TCP"), SourcePort: types.Int64Value(80), DestProto: types.StringValue("TCP"), DestPort: types.Int64Value(80)}},
	}
	state := &VirtInstanceResourceModel{}
	err := r.reconcileDevices(context.Background(), "test-id", plan, state)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "add proxy failed") {
		t.Errorf("expected error to contain 'add proxy failed', got %v", err)
	}
}

func TestVirtInstanceResource_reconcileDevices_SkipsEmptyNames(t *testing.T) {
	callCount := 0
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				callCount++
				return nil, nil
			},
		}},

	}
	plan := &VirtInstanceResourceModel{
		Disks: []DiskModel{{Name: types.StringValue(""), Source: types.StringValue("/src"), Destination: types.StringValue("/dst")}}, // Empty name
		NICs: []NICModel{{Name: types.StringValue(""), Network: types.StringValue("bridge0")}}, // Empty name
		Proxies: []ProxyModel{{Name: types.StringValue(""), SourceProto: types.StringValue("TCP"), SourcePort: types.Int64Value(80), DestProto: types.StringValue("TCP"), DestPort: types.Int64Value(80)}}, // Empty name
	}
	state := &VirtInstanceResourceModel{}
	err := r.reconcileDevices(context.Background(), "test-id", plan, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should not add any devices because names are empty
	if callCount != 0 {
		t.Errorf("expected 0 API calls for empty names, got %d", callCount)
	}
}

// Tests for queryDevices
func TestVirtInstanceResource_queryDevices_Success(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.device_list" {
					return json.RawMessage(`[
						{"dev_type": "DISK", "name": "data", "source": "/mnt/tank/data", "destination": "/data"},
						{"dev_type": "NIC", "name": "eth0", "network": "bridge0"}
					]`), nil
				}
				return nil, nil
			},
		}},

	}
	devices, err := r.queryDevices(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}
}

func TestVirtInstanceResource_queryDevices_APIError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("API error")
			},
		}},

	}
	_, err := r.queryDevices(context.Background(), "test-id")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVirtInstanceResource_queryDevices_InvalidJSON(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`invalid json`), nil
			},
		}},

	}
	_, err := r.queryDevices(context.Background(), "test-id")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to unmarshal") {
		t.Errorf("expected unmarshal error, got %v", err)
	}
}

// Tests for reconcileDesiredState error paths
func TestVirtInstanceResource_reconcileDesiredState_StartError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.start" {
					return nil, errors.New("start failed")
				}
				return nil, nil
			},
		}},

	}
	schemaResp := getVirtInstanceResourceSchema(t)
	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}
	err := r.reconcileDesiredState(context.Background(), "test", "test-id", VirtInstanceStateStopped, VirtInstanceStateRunning, 30*time.Second, 30, resp)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "start failed") {
		t.Errorf("expected 'start failed', got %v", err)
	}
}

func TestVirtInstanceResource_reconcileDesiredState_StopError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.stop" {
					return nil, errors.New("stop failed")
				}
				return nil, nil
			},
		}},

	}
	schemaResp := getVirtInstanceResourceSchema(t)
	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}
	err := r.reconcileDesiredState(context.Background(), "test", "test-id", VirtInstanceStateRunning, VirtInstanceStateStopped, 30*time.Second, 30, resp)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "stop failed") {
		t.Errorf("expected 'stop failed', got %v", err)
	}
}

func TestVirtInstanceResource_reconcileDesiredState_WrongFinalState(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// Return STOPPED instead of RUNNING
				return json.RawMessage(`{"id": "test-id", "name": "test", "status": "STOPPED", "storage_pool": "tank", "autostart": false, "aliases": [], "image": {"os": "ubuntu", "release": "24.04", "architecture": "amd64", "description": "", "variant": ""}}`), nil
			},
		}},

	}
	schemaResp := getVirtInstanceResourceSchema(t)
	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}
	err := r.reconcileDesiredState(context.Background(), "test", "test-id", VirtInstanceStateStopped, VirtInstanceStateRunning, 30*time.Second, 30, resp)
	if err == nil {
		t.Fatal("expected error for wrong final state")
	}
	if !strings.Contains(err.Error(), "reached state STOPPED instead of desired RUNNING") {
		t.Errorf("expected wrong state error, got %v", err)
	}
}

// Tests for Update edge cases
func TestVirtInstanceResource_Update_QueryStateError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("query state failed")
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
		State:        "RUNNING",
	})
	planValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
		Autostart:    true,
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
		t.Fatal("expected error for query state failure")
	}
}

func TestVirtInstanceResource_Update_ReconcileDevicesError(t *testing.T) {
	queryCount := 0
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.device_delete" {
					return nil, errors.New("device delete failed")
				}
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				queryCount++
				return mockVirtInstanceResponse("test-container", "RUNNING", false), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
		State:        "RUNNING",
		Disks: []diskParams{
			{Name: "disk1", Source: "/src", Destination: "/dst"},
		},
	})
	planValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
		// No disks in plan - should trigger delete
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
		t.Fatal("expected error for device reconcile failure")
	}
}

// Test Delete error paths
func TestVirtInstanceResource_Delete_StopError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.stop" {
					return nil, errors.New("stop failed")
				}
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return mockVirtInstanceResponse("test-container", "RUNNING", false), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
		State:        "RUNNING",
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
		t.Fatal("expected error for stop failure")
	}
}

func TestVirtInstanceResource_Delete_QueryStateError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("query failed")
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
		State:        "RUNNING",
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
		t.Fatal("expected error for query failure")
	}
}

// Test Create with stop error when desired_state is STOPPED
func TestVirtInstanceResource_Create_StopError(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.stop" {
					return nil, errors.New("stop failed after create")
				}
				return json.RawMessage(`1`), nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return mockVirtInstanceResponse("test-container", "RUNNING", false), nil
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	planValue := createVirtInstanceModelValue(virtInstanceModelParams{
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "STOPPED",
		StateTimeout: float64(90),
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
		t.Fatal("expected error for stop failure after create")
	}
}

// Test for isVirtInstanceStableState
func TestIsVirtInstanceStableState(t *testing.T) {
	tests := []struct {
		state    string
		expected bool
	}{
		{VirtInstanceStateRunning, true},
		{VirtInstanceStateStopped, true},
		{VirtInstanceStateStarting, false},
		{VirtInstanceStateStopping, false},
		{"UNKNOWN", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.state, func(t *testing.T) {
			result := isVirtInstanceStableState(tc.state)
			if result != tc.expected {
				t.Errorf("isVirtInstanceStableState(%q) = %v, expected %v", tc.state, result, tc.expected)
			}
		})
	}
}

// Test for isNotFoundError
func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"does not exist", errors.New("Instance does not exist"), true},
		{"not found", errors.New("Container not found"), true},
		{"no such instance", errors.New("No such instance: test"), true},
		{"case insensitive", errors.New("DOES NOT EXIST"), true},
		{"unrelated error", errors.New("connection failed"), false},
		{"api error", errors.New("API returned 500"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isNotFoundError(tc.err)
			if result != tc.expected {
				t.Errorf("isNotFoundError(%v) = %v, expected %v", tc.err, result, tc.expected)
			}
		})
	}
}

// Test for mapAliasesToAddresses
func TestMapAliasesToAddresses(t *testing.T) {
	t.Run("empty aliases", func(t *testing.T) {
		result := mapAliasesToAddresses(nil)
		if result.IsNull() {
			t.Error("expected non-null list")
		}
		elements := result.Elements()
		if len(elements) != 0 {
			t.Errorf("expected empty list, got %d items", len(elements))
		}
	})

	t.Run("single IPv4 address", func(t *testing.T) {
		netmask := int64(24)
		aliases := []virtInstanceAlias{
			{Type: "INET", Address: "192.168.1.100", Netmask: &netmask},
		}
		result := mapAliasesToAddresses(aliases)
		elements := result.Elements()
		if len(elements) != 1 {
			t.Fatalf("expected 1 address, got %d", len(elements))
		}
		obj := elements[0].(types.Object)
		attrs := obj.Attributes()
		if attrs["type"].(types.String).ValueString() != "INET" {
			t.Errorf("expected type INET, got %s", attrs["type"].(types.String).ValueString())
		}
		if attrs["address"].(types.String).ValueString() != "192.168.1.100" {
			t.Errorf("expected address 192.168.1.100, got %s", attrs["address"].(types.String).ValueString())
		}
		if attrs["netmask"].(types.Int64).ValueInt64() != 24 {
			t.Errorf("expected netmask 24, got %d", attrs["netmask"].(types.Int64).ValueInt64())
		}
	})

	t.Run("multiple addresses", func(t *testing.T) {
		netmask24 := int64(24)
		netmask64 := int64(64)
		aliases := []virtInstanceAlias{
			{Type: "INET", Address: "192.168.1.100", Netmask: &netmask24},
			{Type: "INET6", Address: "2001:db8::1", Netmask: &netmask64},
		}
		result := mapAliasesToAddresses(aliases)
		elements := result.Elements()
		if len(elements) != 2 {
			t.Fatalf("expected 2 addresses, got %d", len(elements))
		}
		obj0 := elements[0].(types.Object)
		if obj0.Attributes()["type"].(types.String).ValueString() != "INET" {
			t.Errorf("expected first type INET, got %s", obj0.Attributes()["type"].(types.String).ValueString())
		}
		obj1 := elements[1].(types.Object)
		if obj1.Attributes()["type"].(types.String).ValueString() != "INET6" {
			t.Errorf("expected second type INET6, got %s", obj1.Attributes()["type"].(types.String).ValueString())
		}
	})

	t.Run("nil netmask", func(t *testing.T) {
		aliases := []virtInstanceAlias{
			{Type: "INET", Address: "192.168.1.100", Netmask: nil},
		}
		result := mapAliasesToAddresses(aliases)
		elements := result.Elements()
		if len(elements) != 1 {
			t.Fatalf("expected 1 address, got %d", len(elements))
		}
		obj := elements[0].(types.Object)
		netmaskVal := obj.Attributes()["netmask"].(types.Int64)
		if !netmaskVal.IsNull() {
			t.Errorf("expected netmask to be null, got %d", netmaskVal.ValueInt64())
		}
	})
}

// Test addresses populated when running
func TestVirtInstanceResource_Read_WithAddresses(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.get_instance" {
					return mockVirtInstanceResponseWithAliases(
						"test-container",
						"RUNNING",
						false,
						[]map[string]interface{}{
							{"type": "INET", "address": "192.168.1.100", "netmask": 24},
							{"type": "INET6", "address": "2001:db8::1", "netmask": 64},
						},
					), nil
				}
				return nil, errors.New("unexpected method: " + method)
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "RUNNING",
		StateTimeout: float64(90),
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

	// Extract model from state to verify addresses
	var model VirtInstanceResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get model: %v", diags)
	}

	elements := model.Addresses.Elements()
	if len(elements) != 2 {
		t.Fatalf("expected 2 addresses, got %d", len(elements))
	}
	obj0 := elements[0].(types.Object)
	if obj0.Attributes()["type"].(types.String).ValueString() != "INET" {
		t.Errorf("expected first address type INET, got %s", obj0.Attributes()["type"].(types.String).ValueString())
	}
	if obj0.Attributes()["address"].(types.String).ValueString() != "192.168.1.100" {
		t.Errorf("expected first address 192.168.1.100, got %s", obj0.Attributes()["address"].(types.String).ValueString())
	}
}

// Test addresses empty when stopped
func TestVirtInstanceResource_Read_AddressesEmptyWhenStopped(t *testing.T) {
	r := &VirtInstanceResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4, Patch: 0, Build: 0},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "virt.instance.get_instance" {
					// Stopped container has no aliases
					return mockVirtInstanceResponse("test-container", "STOPPED", false), nil
				}
				return nil, errors.New("unexpected method: " + method)
			},
		}},

	}

	schemaResp := getVirtInstanceResourceSchema(t)
	stateValue := createVirtInstanceModelValue(virtInstanceModelParams{
		ID:           "test-container",
		Name:         "test-container",
		StoragePool:  "tank",
		ImageName:    "ubuntu",
		ImageVersion: "24.04",
		DesiredState: "STOPPED",
		StateTimeout: float64(90),
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

	// Extract model from state to verify addresses are empty
	var model VirtInstanceResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get model: %v", diags)
	}

	elements := model.Addresses.Elements()
	if len(elements) != 0 {
		t.Errorf("expected 0 addresses when stopped, got %d", len(elements))
	}
}
