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
	var capturedOpts truenas.CreateCredentialOpts

	r := &CloudSyncCredentialsResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateCredentialFunc: func(ctx context.Context, opts truenas.CreateCredentialOpts) (*truenas.CloudSyncCredential, error) {
					capturedOpts = opts
					return &truenas.CloudSyncCredential{
						ID:           5,
						Name:         "Scaleway",
						ProviderType: "S3",
						Attributes: map[string]string{
							"access_key_id":     "AKIATEST",
							"secret_access_key": "secret123",
							"endpoint":          "s3.nl-ams.scw.cloud",
							"region":            "nl-ams",
						},
					}, nil
				},
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

	// Verify opts sent to service
	if capturedOpts.Name != "Scaleway" {
		t.Errorf("expected name 'Scaleway', got %q", capturedOpts.Name)
	}
	if capturedOpts.ProviderType != "S3" {
		t.Errorf("expected provider type 'S3', got %q", capturedOpts.ProviderType)
	}
	if capturedOpts.Attributes["access_key_id"] != "AKIATEST" {
		t.Errorf("expected access_key_id 'AKIATEST', got %v", capturedOpts.Attributes["access_key_id"])
	}
	if capturedOpts.Attributes["secret_access_key"] != "secret123" {
		t.Errorf("expected secret_access_key 'secret123', got %v", capturedOpts.Attributes["secret_access_key"])
	}
	if capturedOpts.Attributes["endpoint"] != "s3.nl-ams.scw.cloud" {
		t.Errorf("expected endpoint 's3.nl-ams.scw.cloud', got %v", capturedOpts.Attributes["endpoint"])
	}
	if capturedOpts.Attributes["region"] != "nl-ams" {
		t.Errorf("expected region 'nl-ams', got %v", capturedOpts.Attributes["region"])
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
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				GetCredentialFunc: func(ctx context.Context, id int64) (*truenas.CloudSyncCredential, error) {
					return &truenas.CloudSyncCredential{
						ID:           5,
						Name:         "Scaleway",
						ProviderType: "S3",
						Attributes: map[string]string{
							"access_key_id":     "AKIATEST",
							"secret_access_key": "secret123",
							"endpoint":          "s3.nl-ams.scw.cloud",
							"region":            "nl-ams",
						},
					}, nil
				},
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
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				GetCredentialFunc: func(ctx context.Context, id int64) (*truenas.CloudSyncCredential, error) {
					return nil, nil
				},
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
	var capturedID int64
	var capturedOpts truenas.UpdateCredentialOpts

	r := &CloudSyncCredentialsResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				UpdateCredentialFunc: func(ctx context.Context, id int64, opts truenas.UpdateCredentialOpts) (*truenas.CloudSyncCredential, error) {
					capturedID = id
					capturedOpts = opts
					return &truenas.CloudSyncCredential{
						ID:           5,
						Name:         "Scaleway Updated",
						ProviderType: "S3",
						Attributes: map[string]string{
							"access_key_id":     "AKIATEST-UPDATED",
							"secret_access_key": "newsecret456",
							"endpoint":          "s3.fr-par.scw.cloud",
							"region":            "fr-par",
						},
					}, nil
				},
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

	if capturedID != 5 {
		t.Errorf("expected ID 5, got %d", capturedID)
	}

	// Verify opts sent to service
	if capturedOpts.Name != "Scaleway Updated" {
		t.Errorf("expected name 'Scaleway Updated', got %q", capturedOpts.Name)
	}
	if capturedOpts.ProviderType != "S3" {
		t.Errorf("expected provider type 'S3', got %q", capturedOpts.ProviderType)
	}
	if capturedOpts.Attributes["access_key_id"] != "AKIATEST-UPDATED" {
		t.Errorf("expected access_key_id 'AKIATEST-UPDATED', got %v", capturedOpts.Attributes["access_key_id"])
	}
	if capturedOpts.Attributes["secret_access_key"] != "newsecret456" {
		t.Errorf("expected secret_access_key 'newsecret456', got %v", capturedOpts.Attributes["secret_access_key"])
	}
	if capturedOpts.Attributes["endpoint"] != "s3.fr-par.scw.cloud" {
		t.Errorf("expected endpoint 's3.fr-par.scw.cloud', got %v", capturedOpts.Attributes["endpoint"])
	}
	if capturedOpts.Attributes["region"] != "fr-par" {
		t.Errorf("expected region 'fr-par', got %v", capturedOpts.Attributes["region"])
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
	var capturedID int64

	r := &CloudSyncCredentialsResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				DeleteCredentialFunc: func(ctx context.Context, id int64) error {
					capturedID = id
					return nil
				},
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

	if capturedID != 5 {
		t.Errorf("expected ID 5, got %d", capturedID)
	}
}

func TestCloudSyncCredentialsResource_Create_APIError(t *testing.T) {
	r := &CloudSyncCredentialsResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateCredentialFunc: func(ctx context.Context, opts truenas.CreateCredentialOpts) (*truenas.CloudSyncCredential, error) {
					return nil, errors.New("connection refused")
				},
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
		BaseResource: BaseResource{services: &services.TrueNASServices{}},
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
		BaseResource: BaseResource{services: &services.TrueNASServices{}},
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
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				GetCredentialFunc: func(ctx context.Context, id int64) (*truenas.CloudSyncCredential, error) {
					return nil, errors.New("connection refused")
				},
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
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				DeleteCredentialFunc: func(ctx context.Context, id int64) error {
					return errors.New("credentials in use")
				},
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
	var capturedOpts truenas.CreateCredentialOpts

	r := &CloudSyncCredentialsResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateCredentialFunc: func(ctx context.Context, opts truenas.CreateCredentialOpts) (*truenas.CloudSyncCredential, error) {
					capturedOpts = opts
					return &truenas.CloudSyncCredential{
						ID:           6,
						Name:         "Backblaze",
						ProviderType: "B2",
						Attributes: map[string]string{
							"account": "account123",
							"key":     "key456",
						},
					}, nil
				},
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

	// Verify opts sent to service
	if capturedOpts.Name != "Backblaze" {
		t.Errorf("expected name 'Backblaze', got %q", capturedOpts.Name)
	}
	if capturedOpts.ProviderType != "B2" {
		t.Errorf("expected provider type 'B2', got %q", capturedOpts.ProviderType)
	}
	if capturedOpts.Attributes["account"] != "account123" {
		t.Errorf("expected account 'account123', got %v", capturedOpts.Attributes["account"])
	}
	if capturedOpts.Attributes["key"] != "key456" {
		t.Errorf("expected key 'key456', got %v", capturedOpts.Attributes["key"])
	}

	// Verify state was set correctly
	var resultData CloudSyncCredentialsResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "6" {
		t.Errorf("expected ID '6', got %q", resultData.ID.ValueString())
	}
}

func TestCloudSyncCredentialsResource_Create_GCS_Success(t *testing.T) {
	var capturedOpts truenas.CreateCredentialOpts

	r := &CloudSyncCredentialsResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateCredentialFunc: func(ctx context.Context, opts truenas.CreateCredentialOpts) (*truenas.CloudSyncCredential, error) {
					capturedOpts = opts
					return &truenas.CloudSyncCredential{
						ID:           7,
						Name:         "GCS",
						ProviderType: "GOOGLE_CLOUD_STORAGE",
						Attributes: map[string]string{
							"service_account_credentials": `{"type": "service_account"}`,
						},
					}, nil
				},
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

	// Verify opts sent to service
	if capturedOpts.Name != "GCS" {
		t.Errorf("expected name 'GCS', got %q", capturedOpts.Name)
	}
	if capturedOpts.ProviderType != "GOOGLE_CLOUD_STORAGE" {
		t.Errorf("expected provider type 'GOOGLE_CLOUD_STORAGE', got %q", capturedOpts.ProviderType)
	}
	if capturedOpts.Attributes["service_account_credentials"] != `{"type": "service_account"}` {
		t.Errorf("expected service_account_credentials '{\"type\": \"service_account\"}', got %v", capturedOpts.Attributes["service_account_credentials"])
	}

	// Verify state was set correctly
	var resultData CloudSyncCredentialsResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "7" {
		t.Errorf("expected ID '7', got %q", resultData.ID.ValueString())
	}
}

func TestCloudSyncCredentialsResource_Create_Azure_Success(t *testing.T) {
	var capturedOpts truenas.CreateCredentialOpts

	r := &CloudSyncCredentialsResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateCredentialFunc: func(ctx context.Context, opts truenas.CreateCredentialOpts) (*truenas.CloudSyncCredential, error) {
					capturedOpts = opts
					return &truenas.CloudSyncCredential{
						ID:           8,
						Name:         "Azure",
						ProviderType: "AZUREBLOB",
						Attributes: map[string]string{
							"account": "storageaccount",
							"key":     "accountkey",
						},
					}, nil
				},
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

	// Verify opts sent to service
	if capturedOpts.Name != "Azure" {
		t.Errorf("expected name 'Azure', got %q", capturedOpts.Name)
	}
	if capturedOpts.ProviderType != "AZUREBLOB" {
		t.Errorf("expected provider type 'AZUREBLOB', got %q", capturedOpts.ProviderType)
	}
	if capturedOpts.Attributes["account"] != "storageaccount" {
		t.Errorf("expected account 'storageaccount', got %v", capturedOpts.Attributes["account"])
	}
	if capturedOpts.Attributes["key"] != "accountkey" {
		t.Errorf("expected key 'accountkey', got %v", capturedOpts.Attributes["key"])
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

func TestCloudSyncCredentialsResource_Configure_Success(t *testing.T) {
	r := NewCloudSyncCredentialsResource().(*CloudSyncCredentialsResource)

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
