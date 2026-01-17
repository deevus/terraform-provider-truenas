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

func TestNewCloudSyncCredentialsResource(t *testing.T) {
	r := NewCloudSyncCredentialsResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*CloudSyncCredentialsResource)
	var _ resource.ResourceWithImportState = r.(*CloudSyncCredentialsResource)
}

func TestCloudSyncCredentialsResource_Metadata(t *testing.T) {
	r := NewCloudSyncCredentialsResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_cloudsync_credentials" {
		t.Errorf("expected TypeName 'truenas_cloudsync_credentials', got %q", resp.TypeName)
	}
}

func TestCloudSyncCredentialsResource_Schema(t *testing.T) {
	r := NewCloudSyncCredentialsResource()

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

	// Verify name attribute exists and is required
	nameAttr, ok := resp.Schema.Attributes["name"]
	if !ok {
		t.Fatal("expected 'name' attribute in schema")
	}
	if !nameAttr.IsRequired() {
		t.Error("expected 'name' attribute to be required")
	}

	// Verify provider blocks exist
	for _, block := range []string{"s3", "b2", "gcs", "azure"} {
		_, ok := resp.Schema.Blocks[block]
		if !ok {
			t.Errorf("expected '%s' block in schema", block)
		}
	}
}

func TestCloudSyncCredentialsResource_Configure_Success(t *testing.T) {
	r := NewCloudSyncCredentialsResource().(*CloudSyncCredentialsResource)

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

func TestCloudSyncCredentialsResource_Configure_NilProviderData(t *testing.T) {
	r := NewCloudSyncCredentialsResource().(*CloudSyncCredentialsResource)

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestCloudSyncCredentialsResource_Configure_WrongType(t *testing.T) {
	r := NewCloudSyncCredentialsResource().(*CloudSyncCredentialsResource)

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

func getCloudSyncCredentialsResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewCloudSyncCredentialsResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	return *schemaResp
}

// cloudSyncCredentialsModelParams holds parameters for creating test model values.
type cloudSyncCredentialsModelParams struct {
	ID    interface{}
	Name  interface{}
	S3    *s3BlockParams
	B2    *b2BlockParams
	GCS   *gcsBlockParams
	Azure *azureBlockParams
}

type s3BlockParams struct {
	AccessKeyID     interface{}
	SecretAccessKey interface{}
	Endpoint        interface{}
	Region          interface{}
}

type b2BlockParams struct {
	Account interface{}
	Key     interface{}
}

type gcsBlockParams struct {
	ServiceAccountCredentials interface{}
}

type azureBlockParams struct {
	Account interface{}
	Key     interface{}
}

func createCloudSyncCredentialsModelValue(p cloudSyncCredentialsModelParams) tftypes.Value {
	s3Value := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"access_key_id":     tftypes.String,
			"secret_access_key": tftypes.String,
			"endpoint":          tftypes.String,
			"region":            tftypes.String,
		},
	}, nil)
	if p.S3 != nil {
		s3Value = tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"access_key_id":     tftypes.String,
				"secret_access_key": tftypes.String,
				"endpoint":          tftypes.String,
				"region":            tftypes.String,
			},
		}, map[string]tftypes.Value{
			"access_key_id":     tftypes.NewValue(tftypes.String, p.S3.AccessKeyID),
			"secret_access_key": tftypes.NewValue(tftypes.String, p.S3.SecretAccessKey),
			"endpoint":          tftypes.NewValue(tftypes.String, p.S3.Endpoint),
			"region":            tftypes.NewValue(tftypes.String, p.S3.Region),
		})
	}

	b2Value := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"account": tftypes.String,
			"key":     tftypes.String,
		},
	}, nil)
	if p.B2 != nil {
		b2Value = tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"account": tftypes.String,
				"key":     tftypes.String,
			},
		}, map[string]tftypes.Value{
			"account": tftypes.NewValue(tftypes.String, p.B2.Account),
			"key":     tftypes.NewValue(tftypes.String, p.B2.Key),
		})
	}

	gcsValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"service_account_credentials": tftypes.String,
		},
	}, nil)
	if p.GCS != nil {
		gcsValue = tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"service_account_credentials": tftypes.String,
			},
		}, map[string]tftypes.Value{
			"service_account_credentials": tftypes.NewValue(tftypes.String, p.GCS.ServiceAccountCredentials),
		})
	}

	azureValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"account": tftypes.String,
			"key":     tftypes.String,
		},
	}, nil)
	if p.Azure != nil {
		azureValue = tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"account": tftypes.String,
				"key":     tftypes.String,
			},
		}, map[string]tftypes.Value{
			"account": tftypes.NewValue(tftypes.String, p.Azure.Account),
			"key":     tftypes.NewValue(tftypes.String, p.Azure.Key),
		})
	}

	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":   tftypes.String,
			"name": tftypes.String,
			"s3": tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"access_key_id":     tftypes.String,
					"secret_access_key": tftypes.String,
					"endpoint":          tftypes.String,
					"region":            tftypes.String,
				},
			},
			"b2": tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"account": tftypes.String,
					"key":     tftypes.String,
				},
			},
			"gcs": tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"service_account_credentials": tftypes.String,
				},
			},
			"azure": tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"account": tftypes.String,
					"key":     tftypes.String,
				},
			},
		},
	}, map[string]tftypes.Value{
		"id":    tftypes.NewValue(tftypes.String, p.ID),
		"name":  tftypes.NewValue(tftypes.String, p.Name),
		"s3":    s3Value,
		"b2":    b2Value,
		"gcs":   gcsValue,
		"azure": azureValue,
	})
}

func TestCloudSyncCredentialsResource_Create_S3_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &CloudSyncCredentialsResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.credentials.create" {
					capturedMethod = method
					capturedParams = params
					return json.RawMessage(`{"id": 5}`), nil
				}
				if method == "cloudsync.credentials.query" {
					return json.RawMessage(`[{
						"id": 5,
						"name": "Scaleway",
						"provider": "S3",
						"attributes": {
							"access_key_id": "AKIATEST",
							"secret_access_key": "secret123",
							"endpoint": "s3.nl-ams.scw.cloud",
							"region": "nl-ams"
						}
					}]`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)
	planValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		Name: "Scaleway",
		S3: &s3BlockParams{
			AccessKeyID:     "AKIATEST",
			SecretAccessKey: "secret123",
			Endpoint:        "s3.nl-ams.scw.cloud",
			Region:          "nl-ams",
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

	if capturedMethod != "cloudsync.credentials.create" {
		t.Errorf("expected method 'cloudsync.credentials.create', got %q", capturedMethod)
	}

	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["name"] != "Scaleway" {
		t.Errorf("expected name 'Scaleway', got %v", params["name"])
	}
	if params["provider"] != "S3" {
		t.Errorf("expected provider 'S3', got %v", params["provider"])
	}

	// Verify attributes were passed correctly
	attrs, ok := params["attributes"].(map[string]any)
	if !ok {
		t.Fatal("expected attributes to be map[string]any")
	}
	if attrs["access_key_id"] != "AKIATEST" {
		t.Errorf("expected access_key_id 'AKIATEST', got %v", attrs["access_key_id"])
	}
	if attrs["secret_access_key"] != "secret123" {
		t.Errorf("expected secret_access_key 'secret123', got %v", attrs["secret_access_key"])
	}
	if attrs["endpoint"] != "s3.nl-ams.scw.cloud" {
		t.Errorf("expected endpoint 's3.nl-ams.scw.cloud', got %v", attrs["endpoint"])
	}
	if attrs["region"] != "nl-ams" {
		t.Errorf("expected region 'nl-ams', got %v", attrs["region"])
	}

	// Verify state was set correctly
	var resultData CloudSyncCredentialsResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "5" {
		t.Errorf("expected ID '5', got %q", resultData.ID.ValueString())
	}
}

func TestCloudSyncCredentialsResource_Read_Success(t *testing.T) {
	r := &CloudSyncCredentialsResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": 5,
					"name": "Scaleway",
					"provider": "S3",
					"attributes": {
						"access_key_id": "AKIATEST",
						"secret_access_key": "secret123",
						"endpoint": "s3.nl-ams.scw.cloud",
						"region": "nl-ams"
					}
				}]`), nil
			},
		},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)
	stateValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		ID:   "5",
		Name: "Scaleway",
		S3: &s3BlockParams{
			AccessKeyID:     "AKIATEST",
			SecretAccessKey: "secret123",
			Endpoint:        "s3.nl-ams.scw.cloud",
			Region:          "nl-ams",
		},
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

	// Verify state was set correctly
	var resultData CloudSyncCredentialsResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "5" {
		t.Errorf("expected ID '5', got %q", resultData.ID.ValueString())
	}
	if resultData.Name.ValueString() != "Scaleway" {
		t.Errorf("expected Name 'Scaleway', got %q", resultData.Name.ValueString())
	}
}

func TestCloudSyncCredentialsResource_Read_NotFound(t *testing.T) {
	r := &CloudSyncCredentialsResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)
	stateValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		ID:   "5",
		Name: "Scaleway",
		S3: &s3BlockParams{
			AccessKeyID:     "AKIATEST",
			SecretAccessKey: "secret123",
		},
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

	// State should be null when credential not found
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be null when credential not found")
	}
}

func TestCloudSyncCredentialsResource_Update_Success(t *testing.T) {
	var capturedMethod string
	var capturedID int64
	var capturedUpdateData map[string]any

	r := &CloudSyncCredentialsResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.credentials.update" {
					capturedMethod = method
					// params is []any{id, updateData}
					paramsSlice := params.([]any)
					capturedID = paramsSlice[0].(int64)
					capturedUpdateData = paramsSlice[1].(map[string]any)
					return json.RawMessage(`{"id": 5}`), nil
				}
				if method == "cloudsync.credentials.query" {
					return json.RawMessage(`[{
						"id": 5,
						"name": "Scaleway Updated",
						"provider": "S3",
						"attributes": {
							"access_key_id": "AKIATEST-UPDATED",
							"secret_access_key": "newsecret456",
							"endpoint": "s3.fr-par.scw.cloud",
							"region": "fr-par"
						}
					}]`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)

	stateValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		ID:   "5",
		Name: "Scaleway",
		S3: &s3BlockParams{
			AccessKeyID:     "AKIATEST",
			SecretAccessKey: "secret123",
			Endpoint:        "s3.nl-ams.scw.cloud",
			Region:          "nl-ams",
		},
	})

	planValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		ID:   "5",
		Name: "Scaleway Updated",
		S3: &s3BlockParams{
			AccessKeyID:     "AKIATEST-UPDATED",
			SecretAccessKey: "newsecret456",
			Endpoint:        "s3.fr-par.scw.cloud",
			Region:          "fr-par",
		},
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

	if capturedMethod != "cloudsync.credentials.update" {
		t.Errorf("expected method 'cloudsync.credentials.update', got %q", capturedMethod)
	}

	if capturedID != 5 {
		t.Errorf("expected ID 5, got %d", capturedID)
	}

	// Verify updateData was formed correctly
	if capturedUpdateData["name"] != "Scaleway Updated" {
		t.Errorf("expected name 'Scaleway Updated', got %v", capturedUpdateData["name"])
	}
	if capturedUpdateData["provider"] != "S3" {
		t.Errorf("expected provider 'S3', got %v", capturedUpdateData["provider"])
	}

	// Verify attributes were passed correctly
	attrs, ok := capturedUpdateData["attributes"].(map[string]any)
	if !ok {
		t.Fatal("expected attributes to be map[string]any")
	}
	if attrs["access_key_id"] != "AKIATEST-UPDATED" {
		t.Errorf("expected access_key_id 'AKIATEST-UPDATED', got %v", attrs["access_key_id"])
	}
	if attrs["secret_access_key"] != "newsecret456" {
		t.Errorf("expected secret_access_key 'newsecret456', got %v", attrs["secret_access_key"])
	}
	if attrs["endpoint"] != "s3.fr-par.scw.cloud" {
		t.Errorf("expected endpoint 's3.fr-par.scw.cloud', got %v", attrs["endpoint"])
	}
	if attrs["region"] != "fr-par" {
		t.Errorf("expected region 'fr-par', got %v", attrs["region"])
	}

	// Verify state was set correctly after update
	var resultData CloudSyncCredentialsResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "5" {
		t.Errorf("expected ID '5', got %q", resultData.ID.ValueString())
	}
	if resultData.Name.ValueString() != "Scaleway Updated" {
		t.Errorf("expected Name 'Scaleway Updated', got %q", resultData.Name.ValueString())
	}
}

func TestCloudSyncCredentialsResource_Delete_Success(t *testing.T) {
	var capturedMethod string
	var capturedID int64

	r := &CloudSyncCredentialsResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedID = params.(int64)
				return json.RawMessage(`true`), nil
			},
		},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)
	stateValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		ID:   "5",
		Name: "Scaleway",
		S3: &s3BlockParams{
			AccessKeyID:     "AKIATEST",
			SecretAccessKey: "secret123",
		},
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

	if capturedMethod != "cloudsync.credentials.delete" {
		t.Errorf("expected method 'cloudsync.credentials.delete', got %q", capturedMethod)
	}

	if capturedID != 5 {
		t.Errorf("expected ID 5, got %d", capturedID)
	}
}

func TestCloudSyncCredentialsResource_Create_APIError(t *testing.T) {
	r := &CloudSyncCredentialsResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)
	planValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		Name: "Scaleway",
		S3: &s3BlockParams{
			AccessKeyID:     "AKIATEST",
			SecretAccessKey: "secret123",
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

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestCloudSyncCredentialsResource_Create_NoProviderBlock(t *testing.T) {
	r := &CloudSyncCredentialsResource{
		client: &client.MockClient{},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)
	planValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		Name: "No Provider",
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
		t.Fatal("expected error when no provider block specified")
	}
}

func TestCloudSyncCredentialsResource_Read_APIError(t *testing.T) {
	r := &CloudSyncCredentialsResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)
	stateValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		ID:   "5",
		Name: "Scaleway",
		S3: &s3BlockParams{
			AccessKeyID:     "AKIATEST",
			SecretAccessKey: "secret123",
		},
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

func TestCloudSyncCredentialsResource_Delete_APIError(t *testing.T) {
	r := &CloudSyncCredentialsResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("credentials in use")
			},
		},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)
	stateValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		ID:   "5",
		Name: "Scaleway",
		S3: &s3BlockParams{
			AccessKeyID:     "AKIATEST",
			SecretAccessKey: "secret123",
		},
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
