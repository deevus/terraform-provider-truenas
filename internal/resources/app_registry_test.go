package resources

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/deevus/truenas-go/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewAppRegistryResource(t *testing.T) {
	r := NewAppRegistryResource()
	if r == nil {
		t.Fatal("NewAppRegistryResource returned nil")
	}

	_, ok := r.(*AppRegistryResource)
	if !ok {
		t.Fatalf("expected *AppRegistryResource, got %T", r)
	}

	// Verify interface implementations
	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(r.(*AppRegistryResource))
	_ = resource.ResourceWithImportState(r.(*AppRegistryResource))
}

func TestAppRegistryResource_Metadata(t *testing.T) {
	r := NewAppRegistryResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_app_registry" {
		t.Errorf("expected TypeName 'truenas_app_registry', got %q", resp.TypeName)
	}
}

func TestAppRegistryResource_Configure_Success(t *testing.T) {
	r := NewAppRegistryResource().(*AppRegistryResource)

	mockClient := &client.MockClient{}

	req := resource.ConfigureRequest{
		ProviderData: mockClient,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if r.client == nil {
		t.Error("expected client to be set")
	}
}

func TestAppRegistryResource_Configure_NilProviderData(t *testing.T) {
	r := NewAppRegistryResource().(*AppRegistryResource)

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestAppRegistryResource_Configure_WrongType(t *testing.T) {
	r := NewAppRegistryResource().(*AppRegistryResource)

	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

func TestAppRegistryResource_Schema(t *testing.T) {
	r := NewAppRegistryResource()

	ctx := context.Background()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}

	r.Schema(ctx, schemaReq, schemaResp)

	if schemaResp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify attributes exist
	attrs := schemaResp.Schema.Attributes
	if attrs["id"] == nil {
		t.Error("expected 'id' attribute")
	}
	if attrs["name"] == nil {
		t.Error("expected 'name' attribute")
	}
	if attrs["description"] == nil {
		t.Error("expected 'description' attribute")
	}
	if attrs["username"] == nil {
		t.Error("expected 'username' attribute")
	}
	if attrs["password"] == nil {
		t.Error("expected 'password' attribute")
	}
	if attrs["uri"] == nil {
		t.Error("expected 'uri' attribute")
	}
}

// Test helpers

func getAppRegistryResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewAppRegistryResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("failed to get schema: %v", schemaResp.Diagnostics)
	}
	return *schemaResp
}

// appRegistryModelParams holds parameters for creating test model values.
type appRegistryModelParams struct {
	ID          interface{}
	Name        interface{}
	Description interface{}
	Username    interface{}
	Password    interface{}
	URI         interface{}
}

func createAppRegistryModelValue(p appRegistryModelParams) tftypes.Value {
	// Build the values map
	values := map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, p.ID),
		"name":        tftypes.NewValue(tftypes.String, p.Name),
		"description": tftypes.NewValue(tftypes.String, p.Description),
		"username":    tftypes.NewValue(tftypes.String, p.Username),
		"password":    tftypes.NewValue(tftypes.String, p.Password),
		"uri":         tftypes.NewValue(tftypes.String, p.URI),
	}

	// Create object type matching the schema
	objectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":          tftypes.String,
			"name":        tftypes.String,
			"description": tftypes.String,
			"username":    tftypes.String,
			"password":    tftypes.String,
			"uri":         tftypes.String,
		},
	}

	return tftypes.NewValue(objectType, values)
}

func TestAppRegistryResource_Create_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "app.registry.create" {
					capturedMethod = method
					capturedParams = params
					return json.RawMessage(`{"id": 1}`), nil
				}
				if method == "app.registry.query" {
					return json.RawMessage(`[{
						"id": 1,
						"name": "ghcr",
						"description": "GitHub Container Registry",
						"username": "github-user",
						"password": "ghp_token123",
						"uri": "https://ghcr.io"
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)
	planValue := createAppRegistryModelValue(appRegistryModelParams{
		Name:        "ghcr",
		Description: "GitHub Container Registry",
		Username:    "github-user",
		Password:    "ghp_token123",
		URI:         "https://ghcr.io",
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

	if capturedMethod != "app.registry.create" {
		t.Errorf("expected method 'app.registry.create', got %q", capturedMethod)
	}

	// Verify params
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["name"] != "ghcr" {
		t.Errorf("expected name 'ghcr', got %v", params["name"])
	}
	if params["username"] != "github-user" {
		t.Errorf("expected username 'github-user', got %v", params["username"])
	}
	if params["password"] != "ghp_token123" {
		t.Errorf("expected password 'ghp_token123', got %v", params["password"])
	}
	if params["uri"] != "https://ghcr.io" {
		t.Errorf("expected uri 'https://ghcr.io', got %v", params["uri"])
	}
	if params["description"] != "GitHub Container Registry" {
		t.Errorf("expected description 'GitHub Container Registry', got %v", params["description"])
	}

	// Verify state was set
	var resultData AppRegistryResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "1" {
		t.Errorf("expected ID '1', got %q", resultData.ID.ValueString())
	}
	if resultData.Name.ValueString() != "ghcr" {
		t.Errorf("expected name 'ghcr', got %q", resultData.Name.ValueString())
	}
	if resultData.Username.ValueString() != "github-user" {
		t.Errorf("expected username 'github-user', got %q", resultData.Username.ValueString())
	}
	if resultData.Password.ValueString() != "ghp_token123" {
		t.Errorf("expected password 'ghp_token123', got %q", resultData.Password.ValueString())
	}
	if resultData.URI.ValueString() != "https://ghcr.io" {
		t.Errorf("expected uri 'https://ghcr.io', got %q", resultData.URI.ValueString())
	}
	if resultData.Description.ValueString() != "GitHub Container Registry" {
		t.Errorf("expected description 'GitHub Container Registry', got %q", resultData.Description.ValueString())
	}
}

func TestAppRegistryResource_Create_MinimalFields(t *testing.T) {
	var capturedParams any

	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "app.registry.create" {
					capturedParams = params
					return json.RawMessage(`{"id": 2}`), nil
				}
				if method == "app.registry.query" {
					return json.RawMessage(`[{
						"id": 2,
						"name": "dockerhub",
						"description": null,
						"username": "docker-user",
						"password": "docker-token",
						"uri": "https://index.docker.io/v1/"
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)
	planValue := createAppRegistryModelValue(appRegistryModelParams{
		Name:        "dockerhub",
		Description: "",
		Username:    "docker-user",
		Password:    "docker-token",
		URI:         "https://index.docker.io/v1/",
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

	// Verify description is nil when empty
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["description"] != nil {
		t.Errorf("expected description nil for empty string, got %v", params["description"])
	}

	// Verify state was set with empty description
	var resultData AppRegistryResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.Description.ValueString() != "" {
		t.Errorf("expected description empty, got %q", resultData.Description.ValueString())
	}
}

func TestAppRegistryResource_Create_APIError(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)
	planValue := createAppRegistryModelValue(appRegistryModelParams{
		Name:        "test",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
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

func TestAppRegistryResource_Create_QueryError(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "app.registry.create" {
					return json.RawMessage(`{"id": 1}`), nil
				}
				if method == "app.registry.query" {
					return nil, errors.New("query failed")
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)
	planValue := createAppRegistryModelValue(appRegistryModelParams{
		Name:        "test",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
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
		t.Fatal("expected error for query error after create")
	}
}

func TestAppRegistryResource_Create_QueryNotFound(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "app.registry.create" {
					return json.RawMessage(`{"id": 1}`), nil
				}
				if method == "app.registry.query" {
					return json.RawMessage(`[]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)
	planValue := createAppRegistryModelValue(appRegistryModelParams{
		Name:        "test",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
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
		t.Fatal("expected error when registry not found after create")
	}
}

func TestAppRegistryResource_Read_Success(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": 1,
					"name": "ghcr",
					"description": "GitHub Container Registry",
					"username": "github-user",
					"password": "ghp_token123",
					"uri": "https://ghcr.io"
				}]`), nil
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)
	stateValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "1",
		Name:        "ghcr",
		Description: "GitHub Container Registry",
		Username:    "github-user",
		Password:    "ghp_token123",
		URI:         "https://ghcr.io",
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
	var resultData AppRegistryResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "1" {
		t.Errorf("expected ID '1', got %q", resultData.ID.ValueString())
	}
	if resultData.Name.ValueString() != "ghcr" {
		t.Errorf("expected name 'ghcr', got %q", resultData.Name.ValueString())
	}
	if resultData.Username.ValueString() != "github-user" {
		t.Errorf("expected username 'github-user', got %q", resultData.Username.ValueString())
	}
}

func TestAppRegistryResource_Read_NotFound(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)
	stateValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "1",
		Name:        "deleted",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
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

	// State should be removed (resource not found)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed when resource not found")
	}
}

func TestAppRegistryResource_Read_APIError(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)
	stateValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "1",
		Name:        "test",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
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

func TestAppRegistryResource_Read_InvalidID(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{}},
	}

	schemaResp := getAppRegistryResourceSchema(t)
	stateValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "not-a-number",
		Name:        "test",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
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
		t.Fatal("expected error for invalid ID")
	}
}

func TestAppRegistryResource_Update_Success(t *testing.T) {
	var capturedMethod string
	var capturedID int64
	var capturedUpdateData map[string]any

	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "app.registry.update" {
					capturedMethod = method
					args := params.([]any)
					capturedID = args[0].(int64)
					capturedUpdateData = args[1].(map[string]any)
					return json.RawMessage(`{"id": 1}`), nil
				}
				if method == "app.registry.query" {
					return json.RawMessage(`[{
						"id": 1,
						"name": "ghcr-updated",
						"description": "Updated Description",
						"username": "new-user",
						"password": "new-token",
						"uri": "https://ghcr.io"
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)

	// Current state
	stateValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "1",
		Name:        "ghcr",
		Description: "GitHub Container Registry",
		Username:    "github-user",
		Password:    "ghp_token123",
		URI:         "https://ghcr.io",
	})

	// Updated plan
	planValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "1",
		Name:        "ghcr-updated",
		Description: "Updated Description",
		Username:    "new-user",
		Password:    "new-token",
		URI:         "https://ghcr.io",
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

	if capturedMethod != "app.registry.update" {
		t.Errorf("expected method 'app.registry.update', got %q", capturedMethod)
	}

	if capturedID != 1 {
		t.Errorf("expected ID 1, got %d", capturedID)
	}

	// Verify update params
	if capturedUpdateData["name"] != "ghcr-updated" {
		t.Errorf("expected name 'ghcr-updated', got %v", capturedUpdateData["name"])
	}
	if capturedUpdateData["username"] != "new-user" {
		t.Errorf("expected username 'new-user', got %v", capturedUpdateData["username"])
	}
	if capturedUpdateData["password"] != "new-token" {
		t.Errorf("expected password 'new-token', got %v", capturedUpdateData["password"])
	}
	if capturedUpdateData["description"] != "Updated Description" {
		t.Errorf("expected description 'Updated Description', got %v", capturedUpdateData["description"])
	}

	// Verify state was set
	var resultData AppRegistryResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "1" {
		t.Errorf("expected ID '1', got %q", resultData.ID.ValueString())
	}
	if resultData.Name.ValueString() != "ghcr-updated" {
		t.Errorf("expected name 'ghcr-updated', got %q", resultData.Name.ValueString())
	}
	if resultData.Username.ValueString() != "new-user" {
		t.Errorf("expected username 'new-user', got %q", resultData.Username.ValueString())
	}
}

func TestAppRegistryResource_Update_APIError(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)

	stateValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "1",
		Name:        "test",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
	})

	planValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "1",
		Name:        "test-updated",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
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

func TestAppRegistryResource_Update_InvalidID(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{}},
	}

	schemaResp := getAppRegistryResourceSchema(t)

	stateValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "not-a-number",
		Name:        "test",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
	})

	planValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "not-a-number",
		Name:        "test-updated",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
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
		t.Fatal("expected error for invalid ID")
	}
}

func TestAppRegistryResource_Update_QueryError(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "app.registry.update" {
					return json.RawMessage(`{"id": 1}`), nil
				}
				if method == "app.registry.query" {
					return nil, errors.New("query failed")
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)

	stateValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "1",
		Name:        "test",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
	})

	planValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "1",
		Name:        "test-updated",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
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
		t.Fatal("expected error for query error after update")
	}
}

func TestAppRegistryResource_Update_QueryNotFound(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "app.registry.update" {
					return json.RawMessage(`{"id": 1}`), nil
				}
				if method == "app.registry.query" {
					return json.RawMessage(`[]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)

	stateValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "1",
		Name:        "test",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
	})

	planValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "1",
		Name:        "test-updated",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
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
		t.Fatal("expected error when registry not found after update")
	}
}

func TestAppRegistryResource_Delete_Success(t *testing.T) {
	var capturedMethod string
	var capturedID int64

	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedID = params.(int64)
				return json.RawMessage(`true`), nil
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)
	stateValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "1",
		Name:        "ghcr",
		Description: "GitHub Container Registry",
		Username:    "github-user",
		Password:    "ghp_token123",
		URI:         "https://ghcr.io",
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

	if capturedMethod != "app.registry.delete" {
		t.Errorf("expected method 'app.registry.delete', got %q", capturedMethod)
	}

	if capturedID != 1 {
		t.Errorf("expected ID 1, got %d", capturedID)
	}
}

func TestAppRegistryResource_Delete_APIError(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("registry in use")
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)
	stateValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "1",
		Name:        "test",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
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

func TestAppRegistryResource_Delete_InvalidID(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{}},
	}

	schemaResp := getAppRegistryResourceSchema(t)
	stateValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "not-a-number",
		Name:        "test",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
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
		t.Fatal("expected error for invalid ID")
	}
}

func TestAppRegistryResource_Create_ParseError(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "app.registry.create" {
					return json.RawMessage(`not valid json`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)
	planValue := createAppRegistryModelValue(appRegistryModelParams{
		Name:        "test",
		Description: "",
		Username:    "user",
		Password:    "pass",
		URI:         "https://example.com",
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
		t.Fatal("expected error for parse error")
	}
}

func TestAppRegistryResource_Read_NullDescription(t *testing.T) {
	r := &AppRegistryResource{
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": 1,
					"name": "dockerhub",
					"description": null,
					"username": "docker-user",
					"password": "docker-token",
					"uri": "https://index.docker.io/v1/"
				}]`), nil
			},
		}},
	}

	schemaResp := getAppRegistryResourceSchema(t)
	stateValue := createAppRegistryModelValue(appRegistryModelParams{
		ID:          "1",
		Name:        "dockerhub",
		Description: "",
		Username:    "docker-user",
		Password:    "docker-token",
		URI:         "https://index.docker.io/v1/",
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

	// Verify description is empty string when API returns null
	var resultData AppRegistryResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.Description.ValueString() != "" {
		t.Errorf("expected description empty for null, got %q", resultData.Description.ValueString())
	}
}
