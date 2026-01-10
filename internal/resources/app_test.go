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

func TestNewAppResource(t *testing.T) {
	r := NewAppResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	// Verify it implements the required interfaces
	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*AppResource)
	var _ resource.ResourceWithImportState = r.(*AppResource)
}

func TestAppResource_Metadata(t *testing.T) {
	r := NewAppResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_app" {
		t.Errorf("expected TypeName 'truenas_app', got %q", resp.TypeName)
	}
}

func TestAppResource_Schema(t *testing.T) {
	r := NewAppResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify schema has description
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

	// Verify name attribute exists and is required
	nameAttr, ok := resp.Schema.Attributes["name"]
	if !ok {
		t.Fatal("expected 'name' attribute in schema")
	}
	if !nameAttr.IsRequired() {
		t.Error("expected 'name' attribute to be required")
	}

	// Verify custom_app attribute exists and is required
	customAppAttr, ok := resp.Schema.Attributes["custom_app"]
	if !ok {
		t.Fatal("expected 'custom_app' attribute in schema")
	}
	if !customAppAttr.IsRequired() {
		t.Error("expected 'custom_app' attribute to be required")
	}

	// Verify compose_config attribute exists and is optional
	composeConfigAttr, ok := resp.Schema.Attributes["compose_config"]
	if !ok {
		t.Fatal("expected 'compose_config' attribute in schema")
	}
	if !composeConfigAttr.IsOptional() {
		t.Error("expected 'compose_config' attribute to be optional")
	}

	// Verify labels attribute exists and is optional
	labelsAttr, ok := resp.Schema.Attributes["labels"]
	if !ok {
		t.Fatal("expected 'labels' attribute in schema")
	}
	if !labelsAttr.IsOptional() {
		t.Error("expected 'labels' attribute to be optional")
	}

	// Verify state attribute exists and is computed
	stateAttr, ok := resp.Schema.Attributes["state"]
	if !ok {
		t.Fatal("expected 'state' attribute in schema")
	}
	if !stateAttr.IsComputed() {
		t.Error("expected 'state' attribute to be computed")
	}

	// Verify storage block exists
	storageBlock, ok := resp.Schema.Blocks["storage"]
	if !ok {
		t.Fatal("expected 'storage' block in schema")
	}
	if storageBlock == nil {
		t.Error("expected 'storage' block to be non-nil")
	}

	// Verify network block exists
	networkBlock, ok := resp.Schema.Blocks["network"]
	if !ok {
		t.Fatal("expected 'network' block in schema")
	}
	if networkBlock == nil {
		t.Error("expected 'network' block to be non-nil")
	}
}

func TestAppResource_Configure_Success(t *testing.T) {
	r := NewAppResource().(*AppResource)

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

func TestAppResource_Configure_NilProviderData(t *testing.T) {
	r := NewAppResource().(*AppResource)

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	// Should not error - nil ProviderData is valid during schema validation
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestAppResource_Configure_WrongType(t *testing.T) {
	r := NewAppResource().(*AppResource)

	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

// getAppResourceSchema returns the schema for the app resource
func getAppResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewAppResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	return *schemaResp
}

// createAppResourceModelValue creates a tftypes.Value for the app resource model
func createAppResourceModelValue(
	id, name interface{},
	customApp interface{},
	composeConfig interface{},
	labels interface{},
	state interface{},
	storage interface{},
	network interface{},
) tftypes.Value {
	// Storage nested block type
	storageBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"volume_name":      tftypes.String,
			"type":             tftypes.String,
			"host_path":        tftypes.String,
			"acl_enable":       tftypes.Bool,
			"auto_permissions": tftypes.Bool,
		},
	}

	// Network nested block type
	networkBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"port_name":   tftypes.String,
			"bind_mode":   tftypes.String,
			"host_ips":    tftypes.List{ElementType: tftypes.String},
			"port_number": tftypes.Number,
		},
	}

	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":             tftypes.String,
			"name":           tftypes.String,
			"custom_app":     tftypes.Bool,
			"compose_config": tftypes.String,
			"labels":         tftypes.List{ElementType: tftypes.String},
			"state":          tftypes.String,
			"storage":        tftypes.List{ElementType: storageBlockType},
			"network":        tftypes.List{ElementType: networkBlockType},
		},
	}, map[string]tftypes.Value{
		"id":             tftypes.NewValue(tftypes.String, id),
		"name":           tftypes.NewValue(tftypes.String, name),
		"custom_app":     tftypes.NewValue(tftypes.Bool, customApp),
		"compose_config": tftypes.NewValue(tftypes.String, composeConfig),
		"labels":         tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, labels),
		"state":          tftypes.NewValue(tftypes.String, state),
		"storage":        tftypes.NewValue(tftypes.List{ElementType: storageBlockType}, storage),
		"network":        tftypes.NewValue(tftypes.List{ElementType: networkBlockType}, network),
	})
}

func TestAppResource_Create_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedParams = params
				return json.RawMessage(`{
					"name": "myapp",
					"state": "RUNNING"
				}`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	planValue := createAppResourceModelValue(nil, "myapp", true, nil, nil, nil, []tftypes.Value{}, []tftypes.Value{})

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

	// Verify app.create was called
	if capturedMethod != "app.create" {
		t.Errorf("expected method 'app.create', got %q", capturedMethod)
	}

	// Verify params include app_name
	params, ok := capturedParams.(client.AppCreateParams)
	if !ok {
		t.Fatalf("expected params to be AppCreateParams, got %T", capturedParams)
	}

	if params.AppName != "myapp" {
		t.Errorf("expected app_name 'myapp', got %q", params.AppName)
	}

	if !params.CustomApp {
		t.Error("expected custom_app to be true")
	}

	// Verify state was set
	var model AppResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "myapp" {
		t.Errorf("expected ID 'myapp', got %q", model.ID.ValueString())
	}

	if model.State.ValueString() != "RUNNING" {
		t.Errorf("expected State 'RUNNING', got %q", model.State.ValueString())
	}
}

func TestAppResource_Create_WithComposeConfig(t *testing.T) {
	var capturedParams any

	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`{
					"name": "myapp",
					"state": "RUNNING"
				}`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	composeYAML := "version: '3'\nservices:\n  web:\n    image: nginx"
	planValue := createAppResourceModelValue(nil, "myapp", true, composeYAML, nil, nil, []tftypes.Value{}, []tftypes.Value{})

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

	// Verify params include compose config
	params, ok := capturedParams.(client.AppCreateParams)
	if !ok {
		t.Fatalf("expected params to be AppCreateParams, got %T", capturedParams)
	}

	if params.CustomComposeConfigString != composeYAML {
		t.Errorf("expected compose config %q, got %q", composeYAML, params.CustomComposeConfigString)
	}
}

func TestAppResource_Create_WithStorage(t *testing.T) {
	var capturedParams any

	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`{
					"name": "myapp",
					"state": "RUNNING"
				}`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Create storage block value
	storageBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"volume_name":      tftypes.String,
			"type":             tftypes.String,
			"host_path":        tftypes.String,
			"acl_enable":       tftypes.Bool,
			"auto_permissions": tftypes.Bool,
		},
	}

	storageValue := tftypes.NewValue(storageBlockType, map[string]tftypes.Value{
		"volume_name":      tftypes.NewValue(tftypes.String, "data"),
		"type":             tftypes.NewValue(tftypes.String, "host_path"),
		"host_path":        tftypes.NewValue(tftypes.String, "/mnt/tank/apps/myapp"),
		"acl_enable":       tftypes.NewValue(tftypes.Bool, false),
		"auto_permissions": tftypes.NewValue(tftypes.Bool, true),
	})

	planValue := createAppResourceModelValue(nil, "myapp", true, nil, nil, nil, []tftypes.Value{storageValue}, []tftypes.Value{})

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

	// Verify params include storage
	params, ok := capturedParams.(client.AppCreateParams)
	if !ok {
		t.Fatalf("expected params to be AppCreateParams, got %T", capturedParams)
	}

	storageConfig, ok := params.Values.Storage["data"]
	if !ok {
		t.Fatal("expected storage config for 'data' key")
	}

	if storageConfig.Type != "host_path" {
		t.Errorf("expected storage type 'host_path', got %q", storageConfig.Type)
	}

	if storageConfig.HostPathConfig.Path != "/mnt/tank/apps/myapp" {
		t.Errorf("expected host_path '/mnt/tank/apps/myapp', got %q", storageConfig.HostPathConfig.Path)
	}

	if storageConfig.HostPathConfig.AutoPermissions != true {
		t.Error("expected auto_permissions to be true")
	}
}

func TestAppResource_Create_WithNetwork(t *testing.T) {
	var capturedParams any

	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`{
					"name": "myapp",
					"state": "RUNNING"
				}`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Create network block value
	networkBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"port_name":   tftypes.String,
			"bind_mode":   tftypes.String,
			"host_ips":    tftypes.List{ElementType: tftypes.String},
			"port_number": tftypes.Number,
		},
	}

	networkValue := tftypes.NewValue(networkBlockType, map[string]tftypes.Value{
		"port_name":   tftypes.NewValue(tftypes.String, "http"),
		"bind_mode":   tftypes.NewValue(tftypes.String, "published"),
		"host_ips":    tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "0.0.0.0")}),
		"port_number": tftypes.NewValue(tftypes.Number, 8080),
	})

	planValue := createAppResourceModelValue(nil, "myapp", true, nil, nil, nil, []tftypes.Value{}, []tftypes.Value{networkValue})

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

	// Verify params include network
	params, ok := capturedParams.(client.AppCreateParams)
	if !ok {
		t.Fatalf("expected params to be AppCreateParams, got %T", capturedParams)
	}

	networkConfig, ok := params.Values.Network["http"]
	if !ok {
		t.Fatal("expected network config for 'http' key")
	}

	if networkConfig.BindMode != "published" {
		t.Errorf("expected bind_mode 'published', got %q", networkConfig.BindMode)
	}

	if networkConfig.PortNumber != 8080 {
		t.Errorf("expected port_number 8080, got %d", networkConfig.PortNumber)
	}
}

func TestAppResource_Create_APIError(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("app already exists")
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	planValue := createAppResourceModelValue(nil, "myapp", true, nil, nil, nil, []tftypes.Value{}, []tftypes.Value{})

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

func TestAppResource_Read_Success(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method != "app.query" {
					t.Errorf("expected method 'app.query', got %q", method)
				}
				return json.RawMessage(`[{
					"name": "myapp",
					"state": "RUNNING"
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "STOPPED", []tftypes.Value{}, []tftypes.Value{})

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
	var model AppResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "myapp" {
		t.Errorf("expected ID 'myapp', got %q", model.ID.ValueString())
	}

	if model.State.ValueString() != "RUNNING" {
		t.Errorf("expected State 'RUNNING', got %q", model.State.ValueString())
	}
}

func TestAppResource_Read_NotFound(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// Return empty array - app not found
				return json.RawMessage(`[]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "RUNNING", []tftypes.Value{}, []tftypes.Value{})

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

	// Should NOT have errors - just remove from state
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// State should be empty (removed)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed (null) when app not found")
	}
}

func TestAppResource_Read_APIError(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection failed")
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "RUNNING", []tftypes.Value{}, []tftypes.Value{})

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

func TestAppResource_Update_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedParams = params
				return json.RawMessage(`{
					"name": "myapp",
					"state": "RUNNING"
				}`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Current state
	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "STOPPED", []tftypes.Value{}, []tftypes.Value{})

	// Plan with new compose config
	composeYAML := "version: '3'\nservices:\n  web:\n    image: nginx:latest"
	planValue := createAppResourceModelValue("myapp", "myapp", true, composeYAML, nil, nil, []tftypes.Value{}, []tftypes.Value{})

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

	// Verify app.update was called
	if capturedMethod != "app.update" {
		t.Errorf("expected method 'app.update', got %q", capturedMethod)
	}

	// Verify params is an array [name, updateParams]
	paramsSlice, ok := capturedParams.([]any)
	if !ok {
		t.Fatalf("expected params to be []any, got %T", capturedParams)
	}

	if len(paramsSlice) < 2 {
		t.Fatalf("expected params to have at least 2 elements, got %d", len(paramsSlice))
	}

	if paramsSlice[0] != "myapp" {
		t.Errorf("expected first param 'myapp', got %v", paramsSlice[0])
	}
}

func TestAppResource_Update_APIError(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("update failed")
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "STOPPED", []tftypes.Value{}, []tftypes.Value{})
	planValue := createAppResourceModelValue("myapp", "myapp", true, "new: config", nil, nil, []tftypes.Value{}, []tftypes.Value{})

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

func TestAppResource_Delete_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedParams = params
				return json.RawMessage(`null`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "RUNNING", []tftypes.Value{}, []tftypes.Value{})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify app.delete was called
	if capturedMethod != "app.delete" {
		t.Errorf("expected method 'app.delete', got %q", capturedMethod)
	}

	// Verify the app name was passed
	if capturedParams != "myapp" {
		t.Errorf("expected params 'myapp', got %v", capturedParams)
	}
}

func TestAppResource_Delete_APIError(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("app is running")
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "RUNNING", []tftypes.Value{}, []tftypes.Value{})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Delete(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestAppResource_ImportState(t *testing.T) {
	r := NewAppResource().(*AppResource)

	schemaResp := getAppResourceSchema(t)

	// Create an initial empty state with the correct schema
	emptyState := createAppResourceModelValue(nil, nil, nil, nil, nil, nil, nil, nil)

	req := resource.ImportStateRequest{
		ID: "imported-app",
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

	// Verify state has id set to the import ID
	var model AppResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "imported-app" {
		t.Errorf("expected ID 'imported-app', got %q", model.ID.ValueString())
	}

	if model.Name.ValueString() != "imported-app" {
		t.Errorf("expected Name 'imported-app', got %q", model.Name.ValueString())
	}
}

// Test interface compliance
func TestAppResource_ImplementsInterfaces(t *testing.T) {
	r := NewAppResource()

	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*AppResource)
	var _ resource.ResourceWithImportState = r.(*AppResource)
}

// Test Create with plan parsing error
func TestAppResource_Create_PlanParseError(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{},
	}

	schemaResp := getAppResourceSchema(t)

	// Create an invalid plan value with wrong type for name
	storageBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"volume_name":      tftypes.String,
			"type":             tftypes.String,
			"host_path":        tftypes.String,
			"acl_enable":       tftypes.Bool,
			"auto_permissions": tftypes.Bool,
		},
	}

	networkBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"port_name":   tftypes.String,
			"bind_mode":   tftypes.String,
			"host_ips":    tftypes.List{ElementType: tftypes.String},
			"port_number": tftypes.Number,
		},
	}

	planValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":             tftypes.String,
			"name":           tftypes.Number, // Wrong type!
			"custom_app":     tftypes.Bool,
			"compose_config": tftypes.String,
			"labels":         tftypes.List{ElementType: tftypes.String},
			"state":          tftypes.String,
			"storage":        tftypes.List{ElementType: storageBlockType},
			"network":        tftypes.List{ElementType: networkBlockType},
		},
	}, map[string]tftypes.Value{
		"id":             tftypes.NewValue(tftypes.String, nil),
		"name":           tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"custom_app":     tftypes.NewValue(tftypes.Bool, true),
		"compose_config": tftypes.NewValue(tftypes.String, nil),
		"labels":         tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"state":          tftypes.NewValue(tftypes.String, nil),
		"storage":        tftypes.NewValue(tftypes.List{ElementType: storageBlockType}, []tftypes.Value{}),
		"network":        tftypes.NewValue(tftypes.List{ElementType: networkBlockType}, []tftypes.Value{}),
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
		t.Fatal("expected error for plan parse error")
	}
}

// Test Read with state parsing error
func TestAppResource_Read_StateParseError(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{},
	}

	schemaResp := getAppResourceSchema(t)

	// Create an invalid state value with wrong type for id
	storageBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"volume_name":      tftypes.String,
			"type":             tftypes.String,
			"host_path":        tftypes.String,
			"acl_enable":       tftypes.Bool,
			"auto_permissions": tftypes.Bool,
		},
	}

	networkBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"port_name":   tftypes.String,
			"bind_mode":   tftypes.String,
			"host_ips":    tftypes.List{ElementType: tftypes.String},
			"port_number": tftypes.Number,
		},
	}

	stateValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":             tftypes.Number, // Wrong type!
			"name":           tftypes.String,
			"custom_app":     tftypes.Bool,
			"compose_config": tftypes.String,
			"labels":         tftypes.List{ElementType: tftypes.String},
			"state":          tftypes.String,
			"storage":        tftypes.List{ElementType: storageBlockType},
			"network":        tftypes.List{ElementType: networkBlockType},
		},
	}, map[string]tftypes.Value{
		"id":             tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"name":           tftypes.NewValue(tftypes.String, "myapp"),
		"custom_app":     tftypes.NewValue(tftypes.Bool, true),
		"compose_config": tftypes.NewValue(tftypes.String, nil),
		"labels":         tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"state":          tftypes.NewValue(tftypes.String, "RUNNING"),
		"storage":        tftypes.NewValue(tftypes.List{ElementType: storageBlockType}, []tftypes.Value{}),
		"network":        tftypes.NewValue(tftypes.List{ElementType: networkBlockType}, []tftypes.Value{}),
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
		t.Fatal("expected error for state parse error")
	}
}

// Test Update with plan parsing error
func TestAppResource_Update_PlanParseError(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{},
	}

	schemaResp := getAppResourceSchema(t)

	// Valid state
	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "RUNNING", []tftypes.Value{}, []tftypes.Value{})

	// Invalid plan with wrong type
	storageBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"volume_name":      tftypes.String,
			"type":             tftypes.String,
			"host_path":        tftypes.String,
			"acl_enable":       tftypes.Bool,
			"auto_permissions": tftypes.Bool,
		},
	}

	networkBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"port_name":   tftypes.String,
			"bind_mode":   tftypes.String,
			"host_ips":    tftypes.List{ElementType: tftypes.String},
			"port_number": tftypes.Number,
		},
	}

	planValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":             tftypes.String,
			"name":           tftypes.Number, // Wrong type!
			"custom_app":     tftypes.Bool,
			"compose_config": tftypes.String,
			"labels":         tftypes.List{ElementType: tftypes.String},
			"state":          tftypes.String,
			"storage":        tftypes.List{ElementType: storageBlockType},
			"network":        tftypes.List{ElementType: networkBlockType},
		},
	}, map[string]tftypes.Value{
		"id":             tftypes.NewValue(tftypes.String, "myapp"),
		"name":           tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"custom_app":     tftypes.NewValue(tftypes.Bool, true),
		"compose_config": tftypes.NewValue(tftypes.String, "new config"),
		"labels":         tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"state":          tftypes.NewValue(tftypes.String, nil),
		"storage":        tftypes.NewValue(tftypes.List{ElementType: storageBlockType}, []tftypes.Value{}),
		"network":        tftypes.NewValue(tftypes.List{ElementType: networkBlockType}, []tftypes.Value{}),
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
		t.Fatal("expected error for plan parse error")
	}
}

// Test Update with state parsing error
func TestAppResource_Update_StateParseError(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{},
	}

	schemaResp := getAppResourceSchema(t)

	// Invalid state with wrong type
	storageBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"volume_name":      tftypes.String,
			"type":             tftypes.String,
			"host_path":        tftypes.String,
			"acl_enable":       tftypes.Bool,
			"auto_permissions": tftypes.Bool,
		},
	}

	networkBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"port_name":   tftypes.String,
			"bind_mode":   tftypes.String,
			"host_ips":    tftypes.List{ElementType: tftypes.String},
			"port_number": tftypes.Number,
		},
	}

	stateValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":             tftypes.Number, // Wrong type!
			"name":           tftypes.String,
			"custom_app":     tftypes.Bool,
			"compose_config": tftypes.String,
			"labels":         tftypes.List{ElementType: tftypes.String},
			"state":          tftypes.String,
			"storage":        tftypes.List{ElementType: storageBlockType},
			"network":        tftypes.List{ElementType: networkBlockType},
		},
	}, map[string]tftypes.Value{
		"id":             tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"name":           tftypes.NewValue(tftypes.String, "myapp"),
		"custom_app":     tftypes.NewValue(tftypes.Bool, true),
		"compose_config": tftypes.NewValue(tftypes.String, nil),
		"labels":         tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"state":          tftypes.NewValue(tftypes.String, "RUNNING"),
		"storage":        tftypes.NewValue(tftypes.List{ElementType: storageBlockType}, []tftypes.Value{}),
		"network":        tftypes.NewValue(tftypes.List{ElementType: networkBlockType}, []tftypes.Value{}),
	})

	// Valid plan
	planValue := createAppResourceModelValue("myapp", "myapp", true, "new config", nil, nil, []tftypes.Value{}, []tftypes.Value{})

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
		t.Fatal("expected error for state parse error")
	}
}

// Test Delete with state parsing error
func TestAppResource_Delete_StateParseError(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{},
	}

	schemaResp := getAppResourceSchema(t)

	// Invalid state with wrong type
	storageBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"volume_name":      tftypes.String,
			"type":             tftypes.String,
			"host_path":        tftypes.String,
			"acl_enable":       tftypes.Bool,
			"auto_permissions": tftypes.Bool,
		},
	}

	networkBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"port_name":   tftypes.String,
			"bind_mode":   tftypes.String,
			"host_ips":    tftypes.List{ElementType: tftypes.String},
			"port_number": tftypes.Number,
		},
	}

	stateValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":             tftypes.Number, // Wrong type!
			"name":           tftypes.String,
			"custom_app":     tftypes.Bool,
			"compose_config": tftypes.String,
			"labels":         tftypes.List{ElementType: tftypes.String},
			"state":          tftypes.String,
			"storage":        tftypes.List{ElementType: storageBlockType},
			"network":        tftypes.List{ElementType: networkBlockType},
		},
	}, map[string]tftypes.Value{
		"id":             tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"name":           tftypes.NewValue(tftypes.String, "myapp"),
		"custom_app":     tftypes.NewValue(tftypes.Bool, true),
		"compose_config": tftypes.NewValue(tftypes.String, nil),
		"labels":         tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"state":          tftypes.NewValue(tftypes.String, "RUNNING"),
		"storage":        tftypes.NewValue(tftypes.List{ElementType: storageBlockType}, []tftypes.Value{}),
		"network":        tftypes.NewValue(tftypes.List{ElementType: networkBlockType}, []tftypes.Value{}),
	})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Delete(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for state parse error")
	}
}

// Test Read with invalid JSON response
func TestAppResource_Read_InvalidJSONResponse(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "RUNNING", []tftypes.Value{}, []tftypes.Value{})

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
		t.Fatal("expected error for invalid JSON response")
	}
}

// Test Create with labels
func TestAppResource_Create_WithLabels(t *testing.T) {
	var capturedParams any

	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`{
					"name": "myapp",
					"state": "RUNNING"
				}`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	labels := []tftypes.Value{
		tftypes.NewValue(tftypes.String, "env=production"),
		tftypes.NewValue(tftypes.String, "team=platform"),
	}
	planValue := createAppResourceModelValue(nil, "myapp", true, nil, labels, nil, []tftypes.Value{}, []tftypes.Value{})

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

	// Verify params include labels
	params, ok := capturedParams.(client.AppCreateParams)
	if !ok {
		t.Fatalf("expected params to be AppCreateParams, got %T", capturedParams)
	}

	if len(params.Values.Labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(params.Values.Labels))
	}

	if params.Values.Labels[0] != "env=production" {
		t.Errorf("expected first label 'env=production', got %q", params.Values.Labels[0])
	}
}

// Test Create with invalid JSON response
func TestAppResource_Create_InvalidJSONResponse(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	planValue := createAppResourceModelValue(nil, "myapp", true, nil, nil, nil, []tftypes.Value{}, []tftypes.Value{})

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
		t.Fatal("expected error for invalid JSON response")
	}
}

// Test Update with invalid JSON response
func TestAppResource_Update_InvalidJSONResponse(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "STOPPED", []tftypes.Value{}, []tftypes.Value{})
	planValue := createAppResourceModelValue("myapp", "myapp", true, "new: config", nil, nil, []tftypes.Value{}, []tftypes.Value{})

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
		t.Fatal("expected error for invalid JSON response")
	}
}

// Test Update with storage
func TestAppResource_Update_WithStorage(t *testing.T) {
	var capturedParams any

	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`{
					"name": "myapp",
					"state": "RUNNING"
				}`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Create storage block value
	storageBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"volume_name":      tftypes.String,
			"type":             tftypes.String,
			"host_path":        tftypes.String,
			"acl_enable":       tftypes.Bool,
			"auto_permissions": tftypes.Bool,
		},
	}

	storageValue := tftypes.NewValue(storageBlockType, map[string]tftypes.Value{
		"volume_name":      tftypes.NewValue(tftypes.String, "data"),
		"type":             tftypes.NewValue(tftypes.String, "host_path"),
		"host_path":        tftypes.NewValue(tftypes.String, "/mnt/tank/apps/myapp"),
		"acl_enable":       tftypes.NewValue(tftypes.Bool, false),
		"auto_permissions": tftypes.NewValue(tftypes.Bool, true),
	})

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "STOPPED", []tftypes.Value{}, []tftypes.Value{})
	planValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, nil, []tftypes.Value{storageValue}, []tftypes.Value{})

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

	// Verify params include storage
	paramsSlice, ok := capturedParams.([]any)
	if !ok {
		t.Fatalf("expected params to be []any, got %T", capturedParams)
	}

	if len(paramsSlice) < 2 {
		t.Fatalf("expected params to have at least 2 elements, got %d", len(paramsSlice))
	}

	updateParams, ok := paramsSlice[1].(map[string]any)
	if !ok {
		t.Fatalf("expected updateParams to be map[string]any, got %T", paramsSlice[1])
	}

	values, ok := updateParams["values"].(map[string]any)
	if !ok {
		t.Fatalf("expected values to be map[string]any, got %T", updateParams["values"])
	}

	storage, ok := values["storage"].(map[string]any)
	if !ok {
		t.Fatalf("expected storage to be map[string]any, got %T", values["storage"])
	}

	if _, exists := storage["data"]; !exists {
		t.Error("expected storage config for 'data' key")
	}
}

// Test Update with network
func TestAppResource_Update_WithNetwork(t *testing.T) {
	var capturedParams any

	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`{
					"name": "myapp",
					"state": "RUNNING"
				}`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Create network block value
	networkBlockType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"port_name":   tftypes.String,
			"bind_mode":   tftypes.String,
			"host_ips":    tftypes.List{ElementType: tftypes.String},
			"port_number": tftypes.Number,
		},
	}

	networkValue := tftypes.NewValue(networkBlockType, map[string]tftypes.Value{
		"port_name":   tftypes.NewValue(tftypes.String, "http"),
		"bind_mode":   tftypes.NewValue(tftypes.String, "published"),
		"host_ips":    tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "0.0.0.0")}),
		"port_number": tftypes.NewValue(tftypes.Number, 8080),
	})

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "STOPPED", []tftypes.Value{}, []tftypes.Value{})
	planValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, nil, []tftypes.Value{}, []tftypes.Value{networkValue})

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

	// Verify params include network
	paramsSlice, ok := capturedParams.([]any)
	if !ok {
		t.Fatalf("expected params to be []any, got %T", capturedParams)
	}

	updateParams, ok := paramsSlice[1].(map[string]any)
	if !ok {
		t.Fatalf("expected updateParams to be map[string]any, got %T", paramsSlice[1])
	}

	values, ok := updateParams["values"].(map[string]any)
	if !ok {
		t.Fatalf("expected values to be map[string]any, got %T", updateParams["values"])
	}

	network, ok := values["network"].(map[string]any)
	if !ok {
		t.Fatalf("expected network to be map[string]any, got %T", values["network"])
	}

	if _, exists := network["http"]; !exists {
		t.Error("expected network config for 'http' key")
	}
}

// Test Update with labels
func TestAppResource_Update_WithLabels(t *testing.T) {
	var capturedParams any

	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedParams = params
				return json.RawMessage(`{
					"name": "myapp",
					"state": "RUNNING"
				}`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	labels := []tftypes.Value{
		tftypes.NewValue(tftypes.String, "env=production"),
		tftypes.NewValue(tftypes.String, "team=platform"),
	}

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "STOPPED", []tftypes.Value{}, []tftypes.Value{})
	planValue := createAppResourceModelValue("myapp", "myapp", true, nil, labels, nil, []tftypes.Value{}, []tftypes.Value{})

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

	// Verify params include labels
	paramsSlice, ok := capturedParams.([]any)
	if !ok {
		t.Fatalf("expected params to be []any, got %T", capturedParams)
	}

	updateParams, ok := paramsSlice[1].(map[string]any)
	if !ok {
		t.Fatalf("expected updateParams to be map[string]any, got %T", paramsSlice[1])
	}

	values, ok := updateParams["values"].(map[string]any)
	if !ok {
		t.Fatalf("expected values to be map[string]any, got %T", updateParams["values"])
	}

	labels_result, ok := values["labels"].([]string)
	if !ok {
		t.Fatalf("expected labels to be []string, got %T", values["labels"])
	}

	if len(labels_result) != 2 {
		t.Errorf("expected 2 labels, got %d", len(labels_result))
	}
}

// Test Read syncs all state fields from API
func TestAppResource_Read_SyncsAllStateFromAPI(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method != "app.query" {
					t.Errorf("expected method 'app.query', got %q", method)
				}
				// Return a complete API response with all fields
				return json.RawMessage(`[{
					"name": "myapp",
					"state": "RUNNING",
					"custom_app": true,
					"config": {
						"custom_compose_config_string": "version: '3'\nservices:\n  web:\n    image: nginx"
					},
					"active_workloads": {
						"workload1": {
							"storage": {
								"data": {
									"type": "host_path",
									"host_path_config": {
										"path": "/mnt/tank/apps/myapp",
										"acl_enable": false,
										"auto_permissions": true
									}
								}
							},
							"network": {
								"http": {
									"bind_mode": "published",
									"host_ips": ["0.0.0.0"],
									"port_number": 8080
								}
							},
							"labels": ["env=production", "team=platform"]
						}
					}
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Start with minimal state
	stateValue := createAppResourceModelValue("myapp", "myapp", false, nil, nil, "STOPPED", []tftypes.Value{}, []tftypes.Value{})

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

	// Verify all state was synced
	var model AppResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// Check basic fields
	if model.ID.ValueString() != "myapp" {
		t.Errorf("expected ID 'myapp', got %q", model.ID.ValueString())
	}

	if model.State.ValueString() != "RUNNING" {
		t.Errorf("expected State 'RUNNING', got %q", model.State.ValueString())
	}

	if !model.CustomApp.ValueBool() {
		t.Error("expected CustomApp to be true")
	}

	if model.ComposeConfig.ValueString() != "version: '3'\nservices:\n  web:\n    image: nginx" {
		t.Errorf("expected compose_config to be set, got %q", model.ComposeConfig.ValueString())
	}

	// Check storage
	if len(model.Storage) != 1 {
		t.Fatalf("expected 1 storage entry, got %d", len(model.Storage))
	}

	if model.Storage[0].VolumeName.ValueString() != "data" {
		t.Errorf("expected storage volume_name 'data', got %q", model.Storage[0].VolumeName.ValueString())
	}

	if model.Storage[0].Type.ValueString() != "host_path" {
		t.Errorf("expected storage type 'host_path', got %q", model.Storage[0].Type.ValueString())
	}

	if model.Storage[0].HostPath.ValueString() != "/mnt/tank/apps/myapp" {
		t.Errorf("expected host_path '/mnt/tank/apps/myapp', got %q", model.Storage[0].HostPath.ValueString())
	}

	if model.Storage[0].AutoPermissions.ValueBool() != true {
		t.Error("expected auto_permissions to be true")
	}

	// Check network
	if len(model.Network) != 1 {
		t.Fatalf("expected 1 network entry, got %d", len(model.Network))
	}

	if model.Network[0].PortName.ValueString() != "http" {
		t.Errorf("expected network port_name 'http', got %q", model.Network[0].PortName.ValueString())
	}

	if model.Network[0].BindMode.ValueString() != "published" {
		t.Errorf("expected bind_mode 'published', got %q", model.Network[0].BindMode.ValueString())
	}

	if model.Network[0].PortNumber.ValueInt64() != 8080 {
		t.Errorf("expected port_number 8080, got %d", model.Network[0].PortNumber.ValueInt64())
	}

	// Check labels
	var labels []string
	model.Labels.ElementsAs(context.Background(), &labels, false)

	if len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(labels))
	}
}

// Test Read with empty compose_config sets null
func TestAppResource_Read_EmptyComposeConfigSetsNull(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"name": "myapp",
					"state": "RUNNING",
					"custom_app": false,
					"config": {
						"custom_compose_config_string": ""
					},
					"active_workloads": {}
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, "old config", nil, "STOPPED", []tftypes.Value{}, []tftypes.Value{})

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

	var model AppResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// compose_config should be null when API returns empty string
	if !model.ComposeConfig.IsNull() {
		t.Errorf("expected compose_config to be null, got %q", model.ComposeConfig.ValueString())
	}

	// custom_app should be synced from API
	if model.CustomApp.ValueBool() {
		t.Error("expected CustomApp to be false (synced from API)")
	}
}

// Test Read with no storage/network/labels
func TestAppResource_Read_EmptyValues(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"name": "myapp",
					"state": "RUNNING",
					"custom_app": true,
					"config": {},
					"active_workloads": {}
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "STOPPED", []tftypes.Value{}, []tftypes.Value{})

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

	var model AppResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// Storage should be nil
	if len(model.Storage) != 0 {
		t.Errorf("expected no storage, got %d", len(model.Storage))
	}

	// Network should be nil
	if len(model.Network) != 0 {
		t.Errorf("expected no network, got %d", len(model.Network))
	}

	// Labels should be null
	if !model.Labels.IsNull() {
		t.Error("expected labels to be null")
	}
}

// Test ImportState followed by Read verifies the flow works
func TestAppResource_ImportState_FollowedByRead(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method != "app.query" {
					t.Errorf("expected method 'app.query', got %q", method)
				}
				return json.RawMessage(`[{
					"name": "imported-app",
					"state": "RUNNING",
					"custom_app": true,
					"config": {
						"custom_compose_config_string": "version: '3'"
					},
					"active_workloads": {
						"main": {
							"storage": {},
							"network": {},
							"labels": ["imported=true"]
						}
					}
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Step 1: Import state
	emptyState := createAppResourceModelValue(nil, nil, nil, nil, nil, nil, nil, nil)

	importReq := resource.ImportStateRequest{
		ID: "imported-app",
	}

	importResp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    emptyState,
		},
	}

	r.ImportState(context.Background(), importReq, importResp)

	if importResp.Diagnostics.HasError() {
		t.Fatalf("import state errors: %v", importResp.Diagnostics)
	}

	// Step 2: Read to refresh state from API
	readReq := resource.ReadRequest{
		State: importResp.State,
	}

	readResp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), readReq, readResp)

	if readResp.Diagnostics.HasError() {
		t.Fatalf("read errors: %v", readResp.Diagnostics)
	}

	// Verify all fields were populated from API
	var model AppResourceModel
	diags := readResp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "imported-app" {
		t.Errorf("expected ID 'imported-app', got %q", model.ID.ValueString())
	}

	if model.Name.ValueString() != "imported-app" {
		t.Errorf("expected Name 'imported-app', got %q", model.Name.ValueString())
	}

	if !model.CustomApp.ValueBool() {
		t.Error("expected CustomApp to be true (populated from API)")
	}

	if model.State.ValueString() != "RUNNING" {
		t.Errorf("expected State 'RUNNING', got %q", model.State.ValueString())
	}

	if model.ComposeConfig.ValueString() != "version: '3'" {
		t.Errorf("expected compose_config 'version: 3', got %q", model.ComposeConfig.ValueString())
	}

	// Check labels were synced
	var labels []string
	model.Labels.ElementsAs(context.Background(), &labels, false)

	if len(labels) != 1 || labels[0] != "imported=true" {
		t.Errorf("expected labels [imported=true], got %v", labels)
	}
}

// Test Read with multiple workloads aggregates values
func TestAppResource_Read_MultipleWorkloads(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"name": "myapp",
					"state": "RUNNING",
					"custom_app": true,
					"config": {},
					"active_workloads": {
						"workload1": {
							"storage": {
								"data1": {
									"type": "host_path",
									"host_path_config": {
										"path": "/mnt/tank/data1",
										"acl_enable": false,
										"auto_permissions": false
									}
								}
							},
							"network": {},
							"labels": ["label1"]
						},
						"workload2": {
							"storage": {
								"data2": {
									"type": "host_path",
									"host_path_config": {
										"path": "/mnt/tank/data2",
										"acl_enable": true,
										"auto_permissions": true
									}
								}
							},
							"network": {
								"http": {
									"bind_mode": "published",
									"host_ips": [],
									"port_number": 80
								}
							},
							"labels": ["label2"]
						}
					}
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, "STOPPED", []tftypes.Value{}, []tftypes.Value{})

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

	var model AppResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// Should have 2 storage entries (aggregated from both workloads)
	if len(model.Storage) != 2 {
		t.Errorf("expected 2 storage entries, got %d", len(model.Storage))
	}

	// Should have 1 network entry
	if len(model.Network) != 1 {
		t.Errorf("expected 1 network entry, got %d", len(model.Network))
	}

	// Should have 2 labels (aggregated from both workloads)
	var labels []string
	model.Labels.ElementsAs(context.Background(), &labels, false)

	if len(labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(labels))
	}
}
