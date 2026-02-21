package resources

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/truenas-go/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewCloudSyncCredentialsResource(t *testing.T) {
	r := NewCloudSyncCredentialsResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	_ = resource.Resource(r)
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
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
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
						"provider": {
							"type": "S3",
							"access_key_id": "AKIATEST",
							"secret_access_key": "secret123",
							"endpoint": "s3.nl-ams.scw.cloud",
							"region": "nl-ams"
						}
					}]`), nil
				}
				return nil, nil
			},
		}},
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

	// Verify provider object was formed correctly (new format)
	providerObj, ok := params["provider"].(map[string]any)
	if !ok {
		t.Fatalf("expected provider to be map[string]any, got %T", params["provider"])
	}
	if providerObj["type"] != "S3" {
		t.Errorf("expected provider type 'S3', got %v", providerObj["type"])
	}
	if providerObj["access_key_id"] != "AKIATEST" {
		t.Errorf("expected access_key_id 'AKIATEST', got %v", providerObj["access_key_id"])
	}
	if providerObj["secret_access_key"] != "secret123" {
		t.Errorf("expected secret_access_key 'secret123', got %v", providerObj["secret_access_key"])
	}
	if providerObj["endpoint"] != "s3.nl-ams.scw.cloud" {
		t.Errorf("expected endpoint 's3.nl-ams.scw.cloud', got %v", providerObj["endpoint"])
	}
	if providerObj["region"] != "nl-ams" {
		t.Errorf("expected region 'nl-ams', got %v", providerObj["region"])
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
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": 5,
					"name": "Scaleway",
					"provider": {
						"type": "S3",
						"access_key_id": "AKIATEST",
						"secret_access_key": "secret123",
						"endpoint": "s3.nl-ams.scw.cloud",
						"region": "nl-ams"
					}
				}]`), nil
			},
		}},
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
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		}},
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
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
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
						"provider": {
							"type": "S3",
							"access_key_id": "AKIATEST-UPDATED",
							"secret_access_key": "newsecret456",
							"endpoint": "s3.fr-par.scw.cloud",
							"region": "fr-par"
						}
					}]`), nil
				}
				return nil, nil
			},
		}},
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

	// Verify provider object was formed correctly (new format)
	providerObj, ok := capturedUpdateData["provider"].(map[string]any)
	if !ok {
		t.Fatalf("expected provider to be map[string]any, got %T", capturedUpdateData["provider"])
	}
	if providerObj["type"] != "S3" {
		t.Errorf("expected provider type 'S3', got %v", providerObj["type"])
	}
	if providerObj["access_key_id"] != "AKIATEST-UPDATED" {
		t.Errorf("expected access_key_id 'AKIATEST-UPDATED', got %v", providerObj["access_key_id"])
	}
	if providerObj["secret_access_key"] != "newsecret456" {
		t.Errorf("expected secret_access_key 'newsecret456', got %v", providerObj["secret_access_key"])
	}
	if providerObj["endpoint"] != "s3.fr-par.scw.cloud" {
		t.Errorf("expected endpoint 's3.fr-par.scw.cloud', got %v", providerObj["endpoint"])
	}
	if providerObj["region"] != "fr-par" {
		t.Errorf("expected region 'fr-par', got %v", providerObj["region"])
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
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedID = params.(int64)
				return json.RawMessage(`true`), nil
			},
		}},
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
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		}},
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
		BaseResource: BaseResource{client: &client.MockClient{}},
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

func TestCloudSyncCredentialsResource_Create_S3_MissingRequiredFields(t *testing.T) {
	r := &CloudSyncCredentialsResource{
		BaseResource: BaseResource{client: &client.MockClient{}},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)
	// Create S3 block with missing required fields (access_key_id and secret_access_key are nil)
	planValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		Name: "Test S3 Missing Fields",
		S3: &s3BlockParams{
			// access_key_id and secret_access_key are nil - should fail validation
			Endpoint: "https://s3.example.com",
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
		t.Fatal("expected error when S3 block is missing required fields")
	}

	// Verify the error messages mention the missing fields
	errStr := resp.Diagnostics.Errors()[0].Detail()
	if errStr != "s3.access_key_id is required when s3 block is specified" &&
		errStr != "s3.secret_access_key is required when s3 block is specified" {
		t.Errorf("expected error about missing s3 required field, got: %s", errStr)
	}
}

func TestCloudSyncCredentialsResource_Read_APIError(t *testing.T) {
	r := &CloudSyncCredentialsResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		}},
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
		BaseResource: BaseResource{client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("credentials in use")
			},
		}},
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

func TestCloudSyncCredentialsResource_Create_B2_Success(t *testing.T) {
	var capturedParams any

	r := &CloudSyncCredentialsResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.credentials.create" {
					capturedParams = params
					return json.RawMessage(`{"id": 6}`), nil
				}
				if method == "cloudsync.credentials.query" {
					return json.RawMessage(`[{
						"id": 6,
						"name": "Backblaze",
						"provider": {
							"type": "B2",
							"account": "account123",
							"key": "key456"
						}
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)
	planValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		Name: "Backblaze",
		B2: &b2BlockParams{
			Account: "account123",
			Key:     "key456",
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

	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["name"] != "Backblaze" {
		t.Errorf("expected name 'Backblaze', got %v", params["name"])
	}

	// Verify provider object was formed correctly (new format)
	providerObj, ok := params["provider"].(map[string]any)
	if !ok {
		t.Fatalf("expected provider to be map[string]any, got %T", params["provider"])
	}
	if providerObj["type"] != "B2" {
		t.Errorf("expected provider type 'B2', got %v", providerObj["type"])
	}
	if providerObj["account"] != "account123" {
		t.Errorf("expected account 'account123', got %v", providerObj["account"])
	}
	if providerObj["key"] != "key456" {
		t.Errorf("expected key 'key456', got %v", providerObj["key"])
	}

	// Verify state was set correctly
	var resultData CloudSyncCredentialsResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "6" {
		t.Errorf("expected ID '6', got %q", resultData.ID.ValueString())
	}
}

func TestCloudSyncCredentialsResource_Create_GCS_Success(t *testing.T) {
	var capturedParams any

	r := &CloudSyncCredentialsResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.credentials.create" {
					capturedParams = params
					return json.RawMessage(`{"id": 7}`), nil
				}
				if method == "cloudsync.credentials.query" {
					return json.RawMessage(`[{
						"id": 7,
						"name": "GCS",
						"provider": {
							"type": "GOOGLE_CLOUD_STORAGE",
							"service_account_credentials": "{\"type\": \"service_account\"}"
						}
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)
	planValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		Name: "GCS",
		GCS: &gcsBlockParams{
			ServiceAccountCredentials: `{"type": "service_account"}`,
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

	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["name"] != "GCS" {
		t.Errorf("expected name 'GCS', got %v", params["name"])
	}

	// Verify provider object was formed correctly (new format)
	providerObj, ok := params["provider"].(map[string]any)
	if !ok {
		t.Fatalf("expected provider to be map[string]any, got %T", params["provider"])
	}
	if providerObj["type"] != "GOOGLE_CLOUD_STORAGE" {
		t.Errorf("expected provider type 'GOOGLE_CLOUD_STORAGE', got %v", providerObj["type"])
	}
	if providerObj["service_account_credentials"] != `{"type": "service_account"}` {
		t.Errorf("expected service_account_credentials '{\"type\": \"service_account\"}', got %v", providerObj["service_account_credentials"])
	}

	// Verify state was set correctly
	var resultData CloudSyncCredentialsResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "7" {
		t.Errorf("expected ID '7', got %q", resultData.ID.ValueString())
	}
}

func TestCloudSyncCredentialsResource_Create_Azure_Success(t *testing.T) {
	var capturedParams any

	r := &CloudSyncCredentialsResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 25, Minor: 4},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.credentials.create" {
					capturedParams = params
					return json.RawMessage(`{"id": 8}`), nil
				}
				if method == "cloudsync.credentials.query" {
					return json.RawMessage(`[{
						"id": 8,
						"name": "Azure",
						"provider": {
							"type": "AZUREBLOB",
							"account": "storageaccount",
							"key": "accountkey"
						}
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)
	planValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		Name: "Azure",
		Azure: &azureBlockParams{
			Account: "storageaccount",
			Key:     "accountkey",
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

	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["name"] != "Azure" {
		t.Errorf("expected name 'Azure', got %v", params["name"])
	}

	// Verify provider object was formed correctly (new format)
	providerObj, ok := params["provider"].(map[string]any)
	if !ok {
		t.Fatalf("expected provider to be map[string]any, got %T", params["provider"])
	}
	if providerObj["type"] != "AZUREBLOB" {
		t.Errorf("expected provider type 'AZUREBLOB', got %v", providerObj["type"])
	}
	if providerObj["account"] != "storageaccount" {
		t.Errorf("expected account 'storageaccount', got %v", providerObj["account"])
	}
	if providerObj["key"] != "accountkey" {
		t.Errorf("expected key 'accountkey', got %v", providerObj["key"])
	}

	// Verify state was set correctly
	var resultData CloudSyncCredentialsResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "8" {
		t.Errorf("expected ID '8', got %q", resultData.ID.ValueString())
	}
}

func TestCloudSyncCredentialsResource_ImportState_Success(t *testing.T) {
	r := NewCloudSyncCredentialsResource().(*CloudSyncCredentialsResource)

	schemaResp := getCloudSyncCredentialsResourceSchema(t)

	emptyState := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{})

	req := resource.ImportStateRequest{
		ID: "5",
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

	var data CloudSyncCredentialsResourceModel
	diags := resp.State.Get(context.Background(), &data)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if data.ID.ValueString() != "5" {
		t.Errorf("expected ID '5', got %q", data.ID.ValueString())
	}
}

func TestCloudSyncCredentialsResource_Create_S3_Legacy(t *testing.T) {
	var capturedParams any

	r := &CloudSyncCredentialsResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 24, Minor: 10},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.credentials.create" {
					capturedParams = params
					return json.RawMessage(`{"id": 10}`), nil
				}
				if method == "cloudsync.credentials.query" {
					return json.RawMessage(`[{
						"id": 10,
						"name": "Legacy S3",
						"provider": "S3",
						"attributes": {
							"access_key_id": "AKIALEGACY",
							"secret_access_key": "legacysecret"
						}
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)
	planValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		Name: "Legacy S3",
		S3: &s3BlockParams{
			AccessKeyID:     "AKIALEGACY",
			SecretAccessKey: "legacysecret",
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

	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	// Verify 24.x format: provider is string, attributes is separate
	provider, ok := params["provider"].(string)
	if !ok {
		t.Fatalf("expected provider to be string for 24.x, got %T", params["provider"])
	}
	if provider != "S3" {
		t.Errorf("expected provider 'S3', got %v", provider)
	}

	attributes, ok := params["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected attributes field for 24.x format, got %T", params["attributes"])
	}
	if attributes["access_key_id"] != "AKIALEGACY" {
		t.Errorf("expected access_key_id 'AKIALEGACY', got %v", attributes["access_key_id"])
	}
}

func TestCloudSyncCredentialsResource_Update_S3_Legacy(t *testing.T) {
	var capturedMethod string
	var capturedID int64
	var capturedUpdateData map[string]any

	r := &CloudSyncCredentialsResource{
		BaseResource: BaseResource{client: &client.MockClient{
			VersionVal: truenas.Version{Major: 24, Minor: 10},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.credentials.update" {
					capturedMethod = method
					// params is []any{id, updateData}
					paramsSlice := params.([]any)
					capturedID = paramsSlice[0].(int64)
					capturedUpdateData = paramsSlice[1].(map[string]any)
					return json.RawMessage(`{"id": 10}`), nil
				}
				if method == "cloudsync.credentials.query" {
					return json.RawMessage(`[{
						"id": 10,
						"name": "Legacy S3 Updated",
						"provider": "S3",
						"attributes": {
							"access_key_id": "AKIALEGACY-UPDATED",
							"secret_access_key": "updatedlegacysecret"
						}
					}]`), nil
				}
				return nil, nil
			},
		}},
	}

	schemaResp := getCloudSyncCredentialsResourceSchema(t)

	stateValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		ID:   "10",
		Name: "Legacy S3",
		S3: &s3BlockParams{
			AccessKeyID:     "AKIALEGACY",
			SecretAccessKey: "legacysecret",
		},
	})

	planValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		ID:   "10",
		Name: "Legacy S3 Updated",
		S3: &s3BlockParams{
			AccessKeyID:     "AKIALEGACY-UPDATED",
			SecretAccessKey: "updatedlegacysecret",
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

	if capturedID != 10 {
		t.Errorf("expected ID 10, got %d", capturedID)
	}

	// Verify 24.x format: provider is string, attributes is separate
	provider, ok := capturedUpdateData["provider"].(string)
	if !ok {
		t.Fatalf("expected provider to be string for 24.x, got %T", capturedUpdateData["provider"])
	}
	if provider != "S3" {
		t.Errorf("expected provider 'S3', got %v", provider)
	}

	attributes, ok := capturedUpdateData["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected attributes field for 24.x format, got %T", capturedUpdateData["attributes"])
	}
	if attributes["access_key_id"] != "AKIALEGACY-UPDATED" {
		t.Errorf("expected access_key_id 'AKIALEGACY-UPDATED', got %v", attributes["access_key_id"])
	}
	if attributes["secret_access_key"] != "updatedlegacysecret" {
		t.Errorf("expected secret_access_key 'updatedlegacysecret', got %v", attributes["secret_access_key"])
	}

	// Verify state was set correctly after update
	var resultData CloudSyncCredentialsResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "10" {
		t.Errorf("expected ID '10', got %q", resultData.ID.ValueString())
	}
	if resultData.Name.ValueString() != "Legacy S3 Updated" {
		t.Errorf("expected Name 'Legacy S3 Updated', got %q", resultData.Name.ValueString())
	}
}
