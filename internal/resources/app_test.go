package resources

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	customtypes "github.com/deevus/terraform-provider-truenas/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

	// Verify compose_config uses YAMLStringType for semantic comparison
	stringAttr, ok := composeConfigAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("expected 'compose_config' to be a StringAttribute")
	}
	if _, ok := stringAttr.CustomType.(customtypes.YAMLStringType); !ok {
		t.Errorf("expected 'compose_config' to use YAMLStringType, got %T", stringAttr.CustomType)
	}

	// Verify state attribute exists and is computed
	stateAttr, ok := resp.Schema.Attributes["state"]
	if !ok {
		t.Fatal("expected 'state' attribute in schema")
	}
	if !stateAttr.IsComputed() {
		t.Error("expected 'state' attribute to be computed")
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
	desiredState interface{},
	stateTimeout interface{},
	state interface{},
) tftypes.Value {
	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":             tftypes.String,
			"name":           tftypes.String,
			"custom_app":     tftypes.Bool,
			"compose_config": tftypes.String,
			"desired_state":  tftypes.String,
			"state_timeout":  tftypes.Number,
			"state":          tftypes.String,
		},
	}, map[string]tftypes.Value{
		"id":             tftypes.NewValue(tftypes.String, id),
		"name":           tftypes.NewValue(tftypes.String, name),
		"custom_app":     tftypes.NewValue(tftypes.Bool, customApp),
		"compose_config": tftypes.NewValue(tftypes.String, composeConfig),
		"desired_state":  tftypes.NewValue(tftypes.String, desiredState),
		"state_timeout":  tftypes.NewValue(tftypes.Number, stateTimeout),
		"state":          tftypes.NewValue(tftypes.String, state),
	})
}

func TestAppResource_Create_Success(t *testing.T) {
	var capturedCreateMethod string
	var capturedCreateParams any

	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedCreateMethod = method
				capturedCreateParams = params
				// app.create response is ignored
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// Return app.query response
				return json.RawMessage(`[{
					"name": "myapp",
					"state": "RUNNING"
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	planValue := createAppResourceModelValue(nil, "myapp", true, nil, "RUNNING", float64(120), nil)

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
	if capturedCreateMethod != "app.create" {
		t.Errorf("expected method 'app.create', got %q", capturedCreateMethod)
	}

	// Verify params include app_name
	params, ok := capturedCreateParams.(client.AppCreateParams)
	if !ok {
		t.Fatalf("expected params to be AppCreateParams, got %T", capturedCreateParams)
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
				// app.create response is ignored
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// Return app.query response
				return json.RawMessage(`[{
					"name": "myapp",
					"state": "RUNNING"
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	composeYAML := "version: '3'\nservices:\n  web:\n    image: nginx"
	planValue := createAppResourceModelValue(nil, "myapp", true, composeYAML, "RUNNING", float64(120), nil)

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

func TestAppResource_Create_APIError(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("app already exists")
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	planValue := createAppResourceModelValue(nil, "myapp", true, nil, "RUNNING", float64(120), nil)

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
				// API returns parsed compose config when retrieve_config: true
				return json.RawMessage(`[{
					"name": "myapp",
					"state": "RUNNING",
					"custom_app": true,
					"config": {
						"services": {
							"web": {
								"image": "nginx"
							}
						}
					}
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "STOPPED")

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

	if !model.CustomApp.ValueBool() {
		t.Error("expected CustomApp to be true")
	}

	// Config is returned as parsed YAML, then marshaled back - verify it contains expected content
	composeConfig := model.ComposeConfig.ValueString()
	if composeConfig == "" {
		t.Error("expected compose_config to be synced, got empty string")
	}
	if !strings.Contains(composeConfig, "services:") || !strings.Contains(composeConfig, "image: nginx") {
		t.Errorf("expected compose_config to contain services and nginx image, got %q", composeConfig)
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

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "RUNNING")

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

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "RUNNING")

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
	var capturedUpdateMethod string
	var capturedUpdateParams any
	var capturedQueryMethod string

	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedUpdateMethod = method
				capturedUpdateParams = params
				// app.update response is ignored, just return empty
				return json.RawMessage(`{}`), nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedQueryMethod = method
				// Return app.query response (array of apps)
				return json.RawMessage(`[{
					"name": "myapp",
					"state": "RUNNING"
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Current state
	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "STOPPED")

	// Plan with new compose config
	composeYAML := "version: '3'\nservices:\n  web:\n    image: nginx:latest"
	planValue := createAppResourceModelValue("myapp", "myapp", true, composeYAML, "RUNNING", float64(120), nil)

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
	if capturedUpdateMethod != "app.update" {
		t.Errorf("expected method 'app.update', got %q", capturedUpdateMethod)
	}

	// Verify app.query was called to get state after update
	if capturedQueryMethod != "app.query" {
		t.Errorf("expected query method 'app.query', got %q", capturedQueryMethod)
	}

	// Verify params is an array [name, updateParams]
	paramsSlice, ok := capturedUpdateParams.([]any)
	if !ok {
		t.Fatalf("expected params to be []any, got %T", capturedUpdateParams)
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

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "STOPPED")
	planValue := createAppResourceModelValue("myapp", "myapp", true, "new: config", "RUNNING", float64(120), nil)

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

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "RUNNING")

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

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "RUNNING")

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
	emptyState := createAppResourceModelValue(nil, nil, nil, nil, nil, nil, nil)

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

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "RUNNING")

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

	planValue := createAppResourceModelValue(nil, "myapp", true, nil, "RUNNING", float64(120), nil)

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

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "STOPPED")
	planValue := createAppResourceModelValue("myapp", "myapp", true, "new: config", "RUNNING", float64(120), nil)

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

// Test Read with empty compose_config sets null
func TestAppResource_Read_EmptyComposeConfigSetsNull(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// Empty config object means no compose config
				return json.RawMessage(`[{
					"name": "myapp",
					"state": "RUNNING",
					"custom_app": false,
					"config": {}
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, "old config", "RUNNING", float64(120), "STOPPED")

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

// Test Create with query error after create
func TestAppResource_Create_QueryErrorAfterCreate(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, nil // create succeeds
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("query failed")
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	planValue := createAppResourceModelValue(nil, "myapp", true, nil, "RUNNING", float64(120), nil)

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

// Test Create when app is not found after create
func TestAppResource_Create_AppNotFoundAfterCreate(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, nil // create succeeds
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil // empty array
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	planValue := createAppResourceModelValue(nil, "myapp", true, nil, "RUNNING", float64(120), nil)

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
		t.Fatal("expected error when app not found after create")
	}
}

// Test Update with query error after update
func TestAppResource_Update_QueryErrorAfterUpdate(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, nil // update succeeds
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("query failed")
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "STOPPED")
	planValue := createAppResourceModelValue("myapp", "myapp", true, "new: config", "RUNNING", float64(120), nil)

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
		t.Fatal("expected error when query fails after update")
	}
}

// Test Update when app is not found after update
func TestAppResource_Update_AppNotFoundAfterUpdate(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, nil // update succeeds
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil // empty array
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "STOPPED")
	planValue := createAppResourceModelValue("myapp", "myapp", true, "new: config", "RUNNING", float64(120), nil)

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
		t.Fatal("expected error when app not found after update")
	}
}

// Test Update with invalid JSON response from query
func TestAppResource_Update_QueryInvalidJSONResponse(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, nil // update succeeds
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "STOPPED")
	planValue := createAppResourceModelValue("myapp", "myapp", true, "new: config", "RUNNING", float64(120), nil)

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
		t.Fatal("expected error for invalid JSON response from query")
	}
}

// Test Create with invalid JSON response from query
func TestAppResource_Create_QueryInvalidJSONResponse(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, nil // create succeeds
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	planValue := createAppResourceModelValue(nil, "myapp", true, nil, "RUNNING", float64(120), nil)

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
		t.Fatal("expected error for invalid JSON response from query")
	}
}

func TestAppResource_Schema_DesiredStateAttribute(t *testing.T) {
	r := NewAppResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify desired_state attribute exists and is optional
	desiredStateAttr, ok := resp.Schema.Attributes["desired_state"]
	if !ok {
		t.Fatal("expected 'desired_state' attribute in schema")
	}
	if !desiredStateAttr.IsOptional() {
		t.Error("expected 'desired_state' attribute to be optional")
	}

	// Verify state_timeout attribute exists and is optional
	stateTimeoutAttr, ok := resp.Schema.Attributes["state_timeout"]
	if !ok {
		t.Fatal("expected 'state_timeout' attribute in schema")
	}
	if !stateTimeoutAttr.IsOptional() {
		t.Error("expected 'state_timeout' attribute to be optional")
	}
}

// Test ImportState followed by Read verifies the flow works
func TestAppResource_queryAppState_Success(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method != "app.query" {
					t.Errorf("expected method 'app.query', got %q", method)
				}
				return json.RawMessage(`[{"name": "myapp", "state": "RUNNING"}]`), nil
			},
		},
	}

	state, err := r.queryAppState(context.Background(), "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != "RUNNING" {
		t.Errorf("expected state RUNNING, got %q", state)
	}
}

func TestAppResource_queryAppState_NotFound(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		},
	}

	_, err := r.queryAppState(context.Background(), "myapp")
	if err == nil {
		t.Fatal("expected error for app not found")
	}
}

func TestAppResource_queryAppState_APIError(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection failed")
			},
		},
	}

	_, err := r.queryAppState(context.Background(), "myapp")
	if err == nil {
		t.Fatal("expected error for API error")
	}
}

func TestAppResource_queryAppState_InvalidJSON(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`not valid json`), nil
			},
		},
	}

	_, err := r.queryAppState(context.Background(), "myapp")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestAppResource_ImportState_FollowedByRead(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method != "app.query" {
					t.Errorf("expected method 'app.query', got %q", method)
				}
				// API returns parsed compose config when retrieve_config: true
				return json.RawMessage(`[{
					"name": "imported-app",
					"state": "RUNNING",
					"custom_app": true,
					"config": {
						"version": "3"
					}
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Step 1: Import state
	emptyState := createAppResourceModelValue(nil, nil, nil, nil, nil, nil, nil)

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

	// Config is returned as parsed YAML, then marshaled back
	composeConfig := model.ComposeConfig.ValueString()
	if !strings.Contains(composeConfig, "version:") || !strings.Contains(composeConfig, "\"3\"") {
		t.Errorf("expected compose_config to contain version: 3, got %q", composeConfig)
	}
}

func TestAppResource_reconcileDesiredState_StartApp(t *testing.T) {
	var calledMethod string
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				calledMethod = method
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{"name": "myapp", "state": "RUNNING"}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)
	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}
	err := r.reconcileDesiredState(context.Background(), "myapp", "STOPPED", "RUNNING", 30*time.Second, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledMethod != "app.start" {
		t.Errorf("expected app.start to be called, got %q", calledMethod)
	}
}

func TestAppResource_reconcileDesiredState_StopApp(t *testing.T) {
	var calledMethod string
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				calledMethod = method
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{"name": "myapp", "state": "STOPPED"}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)
	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}
	err := r.reconcileDesiredState(context.Background(), "myapp", "RUNNING", "STOPPED", 30*time.Second, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledMethod != "app.stop" {
		t.Errorf("expected app.stop to be called, got %q", calledMethod)
	}
}

func TestAppResource_reconcileDesiredState_NoChangeNeeded(t *testing.T) {
	callCount := 0
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				callCount++
				return nil, nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)
	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}
	err := r.reconcileDesiredState(context.Background(), "myapp", "RUNNING", "RUNNING", 30*time.Second, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 0 {
		t.Errorf("expected no API calls when state matches, got %d calls", callCount)
	}
}

func TestAppResource_Update_ReconcileStateFromStoppedToRunning(t *testing.T) {
	var methods []string
	queryCount := 0
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				queryCount++
				// First query: return STOPPED (simulating external change)
				// Second query (after start): return RUNNING
				if queryCount == 1 {
					return json.RawMessage(`[{"name": "myapp", "state": "STOPPED"}]`), nil
				}
				return json.RawMessage(`[{"name": "myapp", "state": "RUNNING"}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Current state: STOPPED, desired: RUNNING
	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "STOPPED")
	planValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), nil)

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

	// Verify app.start was called
	foundStart := false
	for _, m := range methods {
		if m == "app.start" {
			foundStart = true
			break
		}
	}
	if !foundStart {
		t.Errorf("expected app.start to be called, got methods: %v", methods)
	}

	// Verify warning was added about drift
	hasWarning := false
	for _, d := range resp.Diagnostics.Warnings() {
		if strings.Contains(d.Summary(), "externally changed") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected drift warning to be added")
	}
}

func TestAppResource_Read_PreservesDesiredState(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// API reports RUNNING state, but user wants it STOPPED
				return json.RawMessage(`[{
					"name": "myapp",
					"state": "RUNNING",
					"custom_app": true,
					"config": {}
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Prior state has desired_state = "STOPPED" (user intentionally wants it stopped)
	// but API returns state = "RUNNING" (maybe it was started externally)
	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "STOPPED", float64(180), "STOPPED")

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

	// desired_state should be preserved from prior state (not reset to current state)
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

func TestAppResource_Read_DefaultsDesiredStateWhenNull(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"name": "myapp",
					"state": "RUNNING",
					"custom_app": true,
					"config": {}
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Prior state has null desired_state (like after import)
	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, nil, nil, nil)

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

	// When desired_state is null, it should default to current state from API
	if model.DesiredState.ValueString() != "RUNNING" {
		t.Errorf("expected desired_state to default to 'RUNNING', got %q", model.DesiredState.ValueString())
	}

	// When state_timeout is null, it should default to 120
	if model.StateTimeout.ValueInt64() != 120 {
		t.Errorf("expected state_timeout to default to 120, got %d", model.StateTimeout.ValueInt64())
	}
}

func TestAppResource_Create_WithDesiredStateStopped(t *testing.T) {
	var methods []string
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// Return RUNNING initially, then STOPPED after stop is called
				if len(methods) == 1 {
					return json.RawMessage(`[{"name": "myapp", "state": "RUNNING"}]`), nil
				}
				return json.RawMessage(`[{"name": "myapp", "state": "STOPPED"}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Plan with desired_state = "stopped"
	planValue := createAppResourceModelValue(nil, "myapp", true, nil, "stopped", float64(120), nil)

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

	// Verify app.create was called, then app.stop
	if len(methods) < 2 {
		t.Fatalf("expected at least 2 API calls, got %d: %v", len(methods), methods)
	}
	if methods[0] != "app.create" {
		t.Errorf("expected first call to be app.create, got %q", methods[0])
	}
	if methods[1] != "app.stop" {
		t.Errorf("expected second call to be app.stop, got %q", methods[1])
	}

	// Verify final state
	var model AppResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.State.ValueString() != "STOPPED" {
		t.Errorf("expected final state STOPPED, got %q", model.State.ValueString())
	}
}
