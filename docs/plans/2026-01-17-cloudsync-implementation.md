# Cloud Sync Resources Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement `truenas_cloudsync_credentials` resource, `truenas_cloudsync_task` resource, and `data.truenas_cloudsync_credentials` data source with 100% test coverage using TDD.

**Architecture:** Provider-specific nested blocks (s3, b2, gcs, azure) for both credentials and tasks. Schedule as nested block with cron fields. Fire-and-forget async sync on change. All sensitive fields properly marked.

**Tech Stack:** Go, Terraform Plugin Framework, TrueNAS midclt API

---

## Phase 1: API Response Types

### Task 1.1: Create CloudSync API Types

**Files:**
- Create: `internal/api/cloudsync.go`

**Step 1: Create the API types file**

```go
package api

// CloudSyncCredentialResponse represents a cloud sync credential from the API.
type CloudSyncCredentialResponse struct {
	ID         int64             `json:"id"`
	Name       string            `json:"name"`
	Provider   string            `json:"provider"`
	Attributes map[string]string `json:"attributes"`
}

// CloudSyncTaskResponse represents a cloud sync task from the API.
type CloudSyncTaskResponse struct {
	ID                  int64             `json:"id"`
	Description         string            `json:"description"`
	Path                string            `json:"path"`
	Credentials         int64             `json:"credentials"`
	Attributes          map[string]string `json:"attributes"`
	Schedule            ScheduleResponse  `json:"schedule"`
	Direction           string            `json:"direction"`
	TransferMode        string            `json:"transfer_mode"`
	Encryption          bool              `json:"encryption"`
	EncryptionPassword  string            `json:"encryption_password,omitempty"`
	EncryptionSalt      string            `json:"encryption_salt,omitempty"`
	Snapshot            bool              `json:"snapshot"`
	Transfers           int64             `json:"transfers"`
	BWLimit             string            `json:"bwlimit,omitempty"`
	Exclude             []string          `json:"exclude"`
	FollowSymlinks      bool              `json:"follow_symlinks"`
	CreateEmptySrcDirs  bool              `json:"create_empty_src_dirs"`
	Enabled             bool              `json:"enabled"`
	Job                 *JobStatus        `json:"job,omitempty"`
}

// ScheduleResponse represents a cron schedule from the API.
type ScheduleResponse struct {
	Minute string `json:"minute"`
	Hour   string `json:"hour"`
	Dom    string `json:"dom"`
	Month  string `json:"month"`
	Dow    string `json:"dow"`
}

// JobStatus represents the last job status for a task.
type JobStatus struct {
	ID    int64  `json:"id"`
	State string `json:"state"`
}
```

**Step 2: Run build to verify syntax**

Run: `go build ./internal/api/...`
Expected: SUCCESS (no errors)

**Step 3: Commit**

```bash
git add internal/api/cloudsync.go
git commit -m "feat(api): add cloud sync API response types"
```

---

## Phase 2: CloudSync Credentials Resource

### Task 2.1: Create Credentials Resource Skeleton and Constructor Test

**Files:**
- Create: `internal/resources/cloudsync_credentials.go`
- Create: `internal/resources/cloudsync_credentials_test.go`

**Step 1: Write the failing constructor test**

```go
package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/resources -run TestNewCloudSyncCredentialsResource -v`
Expected: FAIL with "undefined: NewCloudSyncCredentialsResource"

**Step 3: Write minimal implementation**

```go
package resources

import (
	"context"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ resource.Resource = &CloudSyncCredentialsResource{}
var _ resource.ResourceWithConfigure = &CloudSyncCredentialsResource{}
var _ resource.ResourceWithImportState = &CloudSyncCredentialsResource{}

// CloudSyncCredentialsResource defines the resource implementation.
type CloudSyncCredentialsResource struct {
	client client.Client
}

// NewCloudSyncCredentialsResource creates a new CloudSyncCredentialsResource.
func NewCloudSyncCredentialsResource() resource.Resource {
	return &CloudSyncCredentialsResource{}
}

func (r *CloudSyncCredentialsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloudsync_credentials"
}

func (r *CloudSyncCredentialsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resources -run TestNewCloudSyncCredentialsResource -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/cloudsync_credentials.go internal/resources/cloudsync_credentials_test.go
git commit -m "feat(resources): add cloud sync credentials resource skeleton"
```

---

### Task 2.2: Implement Metadata Test

**Files:**
- Modify: `internal/resources/cloudsync_credentials_test.go`

**Step 1: Write the failing metadata test**

```go
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
```

**Step 2: Run test to verify it passes (already implemented)**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource_Metadata -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/resources/cloudsync_credentials_test.go
git commit -m "test(resources): add cloud sync credentials metadata test"
```

---

### Task 2.3: Implement Schema with Provider Blocks

**Files:**
- Modify: `internal/resources/cloudsync_credentials.go`
- Modify: `internal/resources/cloudsync_credentials_test.go`

**Step 1: Write the failing schema test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource_Schema -v`
Expected: FAIL

**Step 3: Write the schema implementation**

```go
func (r *CloudSyncCredentialsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages cloud sync credentials for backup tasks.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Credential ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Credential name.",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"s3": schema.SingleNestedBlock{
				Description: "S3-compatible storage credentials.",
				Attributes: map[string]schema.Attribute{
					"access_key_id": schema.StringAttribute{
						Description: "Access key ID.",
						Required:    true,
						Sensitive:   true,
					},
					"secret_access_key": schema.StringAttribute{
						Description: "Secret access key.",
						Required:    true,
						Sensitive:   true,
					},
					"endpoint": schema.StringAttribute{
						Description: "Custom endpoint URL for S3-compatible storage.",
						Optional:    true,
					},
					"region": schema.StringAttribute{
						Description: "Region.",
						Optional:    true,
					},
				},
			},
			"b2": schema.SingleNestedBlock{
				Description: "Backblaze B2 credentials.",
				Attributes: map[string]schema.Attribute{
					"account": schema.StringAttribute{
						Description: "Account ID.",
						Required:    true,
						Sensitive:   true,
					},
					"key": schema.StringAttribute{
						Description: "Application key.",
						Required:    true,
						Sensitive:   true,
					},
				},
			},
			"gcs": schema.SingleNestedBlock{
				Description: "Google Cloud Storage credentials.",
				Attributes: map[string]schema.Attribute{
					"service_account_credentials": schema.StringAttribute{
						Description: "Service account JSON credentials.",
						Required:    true,
						Sensitive:   true,
					},
				},
			},
			"azure": schema.SingleNestedBlock{
				Description: "Azure Blob Storage credentials.",
				Attributes: map[string]schema.Attribute{
					"account": schema.StringAttribute{
						Description: "Storage account name.",
						Required:    true,
						Sensitive:   true,
					},
					"key": schema.StringAttribute{
						Description: "Account key.",
						Required:    true,
						Sensitive:   true,
					},
				},
			},
		},
	}
}
```

Add imports:
```go
import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource_Schema -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/cloudsync_credentials.go internal/resources/cloudsync_credentials_test.go
git commit -m "feat(resources): implement cloud sync credentials schema with provider blocks"
```

---

### Task 2.4: Implement Configure Tests

**Files:**
- Modify: `internal/resources/cloudsync_credentials.go`
- Modify: `internal/resources/cloudsync_credentials_test.go`

**Step 1: Write the failing configure tests**

```go
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
```

Add import for client package in test file.

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource_Configure -v`
Expected: FAIL

**Step 3: Implement Configure**

```go
func (r *CloudSyncCredentialsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}
```

Add `"fmt"` to imports.

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource_Configure -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/cloudsync_credentials.go internal/resources/cloudsync_credentials_test.go
git commit -m "feat(resources): implement cloud sync credentials Configure"
```

---

### Task 2.5: Add Resource Model

**Files:**
- Modify: `internal/resources/cloudsync_credentials.go`

**Step 1: Add the model types**

```go
// CloudSyncCredentialsResourceModel describes the resource data model.
type CloudSyncCredentialsResourceModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	S3    *S3Block     `tfsdk:"s3"`
	B2    *B2Block     `tfsdk:"b2"`
	GCS   *GCSBlock    `tfsdk:"gcs"`
	Azure *AzureBlock  `tfsdk:"azure"`
}

// S3Block represents S3 credentials.
type S3Block struct {
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
	Endpoint        types.String `tfsdk:"endpoint"`
	Region          types.String `tfsdk:"region"`
}

// B2Block represents Backblaze B2 credentials.
type B2Block struct {
	Account types.String `tfsdk:"account"`
	Key     types.String `tfsdk:"key"`
}

// GCSBlock represents Google Cloud Storage credentials.
type GCSBlock struct {
	ServiceAccountCredentials types.String `tfsdk:"service_account_credentials"`
}

// AzureBlock represents Azure Blob Storage credentials.
type AzureBlock struct {
	Account types.String `tfsdk:"account"`
	Key     types.String `tfsdk:"key"`
}
```

Add import for types:
```go
"github.com/hashicorp/terraform-plugin-framework/types"
```

**Step 2: Run build to verify**

Run: `go build ./internal/resources/...`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/resources/cloudsync_credentials.go
git commit -m "feat(resources): add cloud sync credentials model types"
```

---

### Task 2.6: Add Test Helpers

**Files:**
- Modify: `internal/resources/cloudsync_credentials_test.go`

**Step 1: Add test helpers**

```go
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
```

Add imports:
```go
"github.com/hashicorp/terraform-plugin-go/tftypes"
```

**Step 2: Run build to verify**

Run: `go build ./internal/resources/...`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/resources/cloudsync_credentials_test.go
git commit -m "test(resources): add cloud sync credentials test helpers"
```

---

### Task 2.7: Implement Create with S3

**Files:**
- Modify: `internal/resources/cloudsync_credentials.go`
- Modify: `internal/resources/cloudsync_credentials_test.go`

**Step 1: Write the failing create test**

```go
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
}
```

Add imports:
```go
"encoding/json"
"github.com/hashicorp/terraform-plugin-framework/tfsdk"
"github.com/deevus/terraform-provider-truenas/internal/client"
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource_Create_S3_Success -v`
Expected: FAIL

**Step 3: Implement Create**

```go
// queryCredential queries a credential by ID and returns the response.
func (r *CloudSyncCredentialsResource) queryCredential(ctx context.Context, id int64) (*api.CloudSyncCredentialResponse, error) {
	filter := [][]any{{"id", "=", id}}
	result, err := r.client.Call(ctx, "cloudsync.credentials.query", filter)
	if err != nil {
		return nil, err
	}

	var credentials []api.CloudSyncCredentialResponse
	if err := json.Unmarshal(result, &credentials); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(credentials) == 0 {
		return nil, nil
	}

	return &credentials[0], nil
}

// getProviderAndAttributes extracts provider type and attributes from the model.
func getProviderAndAttributes(data *CloudSyncCredentialsResourceModel) (string, map[string]any) {
	if data.S3 != nil {
		attrs := map[string]any{
			"access_key_id":     data.S3.AccessKeyID.ValueString(),
			"secret_access_key": data.S3.SecretAccessKey.ValueString(),
		}
		if !data.S3.Endpoint.IsNull() {
			attrs["endpoint"] = data.S3.Endpoint.ValueString()
		}
		if !data.S3.Region.IsNull() {
			attrs["region"] = data.S3.Region.ValueString()
		}
		return "S3", attrs
	}
	if data.B2 != nil {
		return "B2", map[string]any{
			"account": data.B2.Account.ValueString(),
			"key":     data.B2.Key.ValueString(),
		}
	}
	if data.GCS != nil {
		return "GOOGLE_CLOUD_STORAGE", map[string]any{
			"service_account_credentials": data.GCS.ServiceAccountCredentials.ValueString(),
		}
	}
	if data.Azure != nil {
		return "AZUREBLOB", map[string]any{
			"account": data.Azure.Account.ValueString(),
			"key":     data.Azure.Key.ValueString(),
		}
	}
	return "", nil
}

func (r *CloudSyncCredentialsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CloudSyncCredentialsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	provider, attributes := getProviderAndAttributes(&data)
	if provider == "" {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Exactly one provider block (s3, b2, gcs, or azure) must be specified.",
		)
		return
	}

	params := map[string]any{
		"name":       data.Name.ValueString(),
		"provider":   provider,
		"attributes": attributes,
	}

	result, err := r.client.Call(ctx, "cloudsync.credentials.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Cloud Sync Credentials",
			fmt.Sprintf("Unable to create credentials: %s", err.Error()),
		)
		return
	}

	// Parse response to get ID
	var createResp struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(result, &createResp); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Response",
			fmt.Sprintf("Unable to parse create response: %s", err.Error()),
		)
		return
	}

	// Query to get full state
	cred, err := r.queryCredential(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Credentials",
			fmt.Sprintf("Credentials created but unable to read: %s", err.Error()),
		)
		return
	}

	if cred == nil {
		resp.Diagnostics.AddError(
			"Credentials Not Found",
			"Credentials were created but could not be found.",
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d", cred.ID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

Add import:
```go
"encoding/json"
"github.com/deevus/terraform-provider-truenas/internal/api"
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource_Create_S3_Success -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/cloudsync_credentials.go internal/resources/cloudsync_credentials_test.go
git commit -m "feat(resources): implement cloud sync credentials Create for S3"
```

---

### Task 2.8: Implement Read

**Files:**
- Modify: `internal/resources/cloudsync_credentials.go`
- Modify: `internal/resources/cloudsync_credentials_test.go`

**Step 1: Write the failing read test**

```go
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
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource_Read -v`
Expected: FAIL

**Step 3: Implement Read**

```go
func (r *CloudSyncCredentialsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CloudSyncCredentialsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	cred, err := r.queryCredential(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Credentials",
			fmt.Sprintf("Unable to read credentials %q: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	if cred == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(cred.Name)
	// Note: We preserve the existing block values since the API returns
	// sensitive data that we don't want to overwrite from state

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

Add import:
```go
"strconv"
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource_Read -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/cloudsync_credentials.go internal/resources/cloudsync_credentials_test.go
git commit -m "feat(resources): implement cloud sync credentials Read"
```

---

### Task 2.9: Implement Update

**Files:**
- Modify: `internal/resources/cloudsync_credentials.go`
- Modify: `internal/resources/cloudsync_credentials_test.go`

**Step 1: Write the failing update test**

```go
func TestCloudSyncCredentialsResource_Update_Success(t *testing.T) {
	var capturedMethod string
	var capturedID int64

	r := &CloudSyncCredentialsResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.credentials.update" {
					capturedMethod = method
					// params is []any{id, updateData}
					paramsSlice := params.([]any)
					capturedID = paramsSlice[0].(int64)
					return json.RawMessage(`{"id": 5}`), nil
				}
				if method == "cloudsync.credentials.query" {
					return json.RawMessage(`[{
						"id": 5,
						"name": "Scaleway Updated",
						"provider": "S3",
						"attributes": {}
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
		},
	})

	planValue := createCloudSyncCredentialsModelValue(cloudSyncCredentialsModelParams{
		ID:   "5",
		Name: "Scaleway Updated",
		S3: &s3BlockParams{
			AccessKeyID:     "AKIATEST",
			SecretAccessKey: "secret123",
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
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource_Update_Success -v`
Expected: FAIL

**Step 3: Implement Update**

```go
func (r *CloudSyncCredentialsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state CloudSyncCredentialsResourceModel
	var plan CloudSyncCredentialsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	provider, attributes := getProviderAndAttributes(&plan)
	if provider == "" {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Exactly one provider block (s3, b2, gcs, or azure) must be specified.",
		)
		return
	}

	updateData := map[string]any{
		"name":       plan.Name.ValueString(),
		"provider":   provider,
		"attributes": attributes,
	}

	_, err = r.client.Call(ctx, "cloudsync.credentials.update", []any{id, updateData})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Credentials",
			fmt.Sprintf("Unable to update credentials: %s", err.Error()),
		)
		return
	}

	// Query to refresh state
	cred, err := r.queryCredential(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Credentials",
			fmt.Sprintf("Credentials updated but unable to read: %s", err.Error()),
		)
		return
	}

	if cred == nil {
		resp.Diagnostics.AddError(
			"Credentials Not Found",
			"Credentials were updated but could not be found.",
		)
		return
	}

	plan.ID = state.ID
	plan.Name = types.StringValue(cred.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource_Update_Success -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/cloudsync_credentials.go internal/resources/cloudsync_credentials_test.go
git commit -m "feat(resources): implement cloud sync credentials Update"
```

---

### Task 2.10: Implement Delete

**Files:**
- Modify: `internal/resources/cloudsync_credentials.go`
- Modify: `internal/resources/cloudsync_credentials_test.go`

**Step 1: Write the failing delete test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource_Delete_Success -v`
Expected: FAIL

**Step 3: Implement Delete**

```go
func (r *CloudSyncCredentialsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CloudSyncCredentialsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	_, err = r.client.Call(ctx, "cloudsync.credentials.delete", id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Credentials",
			fmt.Sprintf("Unable to delete credentials: %s", err.Error()),
		)
		return
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource_Delete_Success -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/cloudsync_credentials.go internal/resources/cloudsync_credentials_test.go
git commit -m "feat(resources): implement cloud sync credentials Delete"
```

---

### Task 2.11: Add Error Path Tests

**Files:**
- Modify: `internal/resources/cloudsync_credentials_test.go`

**Step 1: Write error path tests**

```go
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
```

Add import:
```go
"errors"
```

**Step 2: Run tests to verify they pass**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource -v`
Expected: All PASS

**Step 3: Commit**

```bash
git add internal/resources/cloudsync_credentials_test.go
git commit -m "test(resources): add cloud sync credentials error path tests"
```

---

### Task 2.12: Add B2, GCS, Azure Create Tests

**Files:**
- Modify: `internal/resources/cloudsync_credentials_test.go`

**Step 1: Write provider-specific create tests**

```go
func TestCloudSyncCredentialsResource_Create_B2_Success(t *testing.T) {
	var capturedParams any

	r := &CloudSyncCredentialsResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.credentials.create" {
					capturedParams = params
					return json.RawMessage(`{"id": 6}`), nil
				}
				if method == "cloudsync.credentials.query" {
					return json.RawMessage(`[{"id": 6, "name": "Backblaze", "provider": "B2", "attributes": {}}]`), nil
				}
				return nil, nil
			},
		},
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

	params := capturedParams.(map[string]any)
	if params["provider"] != "B2" {
		t.Errorf("expected provider 'B2', got %v", params["provider"])
	}
}

func TestCloudSyncCredentialsResource_Create_GCS_Success(t *testing.T) {
	var capturedParams any

	r := &CloudSyncCredentialsResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.credentials.create" {
					capturedParams = params
					return json.RawMessage(`{"id": 7}`), nil
				}
				if method == "cloudsync.credentials.query" {
					return json.RawMessage(`[{"id": 7, "name": "GCS", "provider": "GOOGLE_CLOUD_STORAGE", "attributes": {}}]`), nil
				}
				return nil, nil
			},
		},
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

	params := capturedParams.(map[string]any)
	if params["provider"] != "GOOGLE_CLOUD_STORAGE" {
		t.Errorf("expected provider 'GOOGLE_CLOUD_STORAGE', got %v", params["provider"])
	}
}

func TestCloudSyncCredentialsResource_Create_Azure_Success(t *testing.T) {
	var capturedParams any

	r := &CloudSyncCredentialsResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.credentials.create" {
					capturedParams = params
					return json.RawMessage(`{"id": 8}`), nil
				}
				if method == "cloudsync.credentials.query" {
					return json.RawMessage(`[{"id": 8, "name": "Azure", "provider": "AZUREBLOB", "attributes": {}}]`), nil
				}
				return nil, nil
			},
		},
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

	params := capturedParams.(map[string]any)
	if params["provider"] != "AZUREBLOB" {
		t.Errorf("expected provider 'AZUREBLOB', got %v", params["provider"])
	}
}
```

**Step 2: Run tests to verify they pass**

Run: `go test ./internal/resources -run "TestCloudSyncCredentialsResource_Create_(B2|GCS|Azure)" -v`
Expected: All PASS

**Step 3: Commit**

```bash
git add internal/resources/cloudsync_credentials_test.go
git commit -m "test(resources): add B2, GCS, Azure provider create tests"
```

---

### Task 2.13: Add Import Test

**Files:**
- Modify: `internal/resources/cloudsync_credentials_test.go`

**Step 1: Write the import test**

```go
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
```

**Step 2: Run test to verify it passes**

Run: `go test ./internal/resources -run TestCloudSyncCredentialsResource_ImportState -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/resources/cloudsync_credentials_test.go
git commit -m "test(resources): add cloud sync credentials import test"
```

---

### Task 2.14: Register Resource in Provider

**Files:**
- Modify: `internal/provider/provider.go`

**Step 1: Add resource to provider**

Find the `Resources` function and add:

```go
resources.NewCloudSyncCredentialsResource,
```

**Step 2: Run build to verify**

Run: `go build ./...`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/provider/provider.go
git commit -m "feat(provider): register cloud sync credentials resource"
```

---

## Phase 3: CloudSync Task Resource

The task resource follows the same pattern as credentials but with more fields. Due to document length, I'll outline the remaining tasks at a higher level.

### Task 3.1-3.15: CloudSync Task Resource

Follow the same TDD pattern as credentials:

1. Create skeleton with constructor test
2. Implement metadata
3. Implement schema with provider blocks (s3, b2, gcs, azure), schedule block, encryption block
4. Implement configure
5. Add model types (including schedule and encryption models)
6. Add test helpers
7. Implement Create with sync_on_change support
8. Implement Read
9. Implement Update
10. Implement Delete
11. Add error path tests
12. Add provider-specific tests
13. Add schedule validation tests
14. Add encryption tests
15. Register in provider

**Key differences from credentials:**
- Schedule nested block with minute, hour, dom, month, dow
- Encryption nested block with password, salt
- Scalar fields: transfers, bwlimit, exclude, follow_symlinks, create_empty_src_dirs, snapshot, sync_on_change
- Direction/transfer_mode with lowercase -> uppercase mapping
- sync_on_change triggers `cloudsync.sync` call after create/update

---

## Phase 4: CloudSync Credentials Data Source

### Task 4.1-4.6: CloudSync Credentials Data Source

1. Create skeleton with constructor test
2. Implement metadata (`truenas_cloudsync_credentials`)
3. Implement schema (name required, id/provider computed)
4. Implement configure
5. Implement Read with name lookup
6. Add tests for not found, unsupported provider
7. Register in provider

---

## Phase 5: Documentation and Verification

### Task 5.1: Run Full Test Suite

**Step 1: Run all tests with coverage**

Run: `go test -cover ./internal/resources -run CloudSync`
Expected: All PASS, >90% coverage

**Step 2: Run lint**

Run: `golangci-lint run ./...`
Expected: No errors

**Step 3: Commit any fixes**

---

### Task 5.2: Update Resource Documentation

**Files:**
- Create: `docs/resources/cloudsync_credentials.md`
- Create: `docs/resources/cloudsync_task.md`
- Create: `docs/data-sources/cloudsync_credentials.md`

Create documentation following existing patterns in `docs/resources/`.

---

## Test Coverage Checklist

### CloudSync Credentials Resource (~25 tests)
- [ ] Constructor
- [ ] Metadata
- [ ] Schema (attributes, blocks)
- [ ] Configure (success, nil, wrong type)
- [ ] Create (S3, B2, GCS, Azure, no provider, API error, query error)
- [ ] Read (success, not found, API error)
- [ ] Update (success, API error)
- [ ] Delete (success, API error)
- [ ] Import

### CloudSync Task Resource (~45 tests)
- [ ] Constructor
- [ ] Metadata
- [ ] Schema (all attributes, blocks)
- [ ] Configure (success, nil, wrong type)
- [ ] Create (each provider, with encryption, with schedule, sync_on_change, errors)
- [ ] Read (success, not found, preserves state)
- [ ] Update (success, schedule only, enable encryption, errors)
- [ ] Delete (success, errors)
- [ ] Import
- [ ] Direction/transfer_mode case conversion
- [ ] Schedule defaults
- [ ] Exclude list handling

### CloudSync Credentials Data Source (~10 tests)
- [ ] Constructor
- [ ] Metadata
- [ ] Schema
- [ ] Configure
- [ ] Read (S3, B2, GCS, Azure, not found, unsupported provider)

**Total: ~80 tests for 100% coverage**

---

## Execution Commands Summary

```bash
# Run specific test
go test ./internal/resources -run TestCloudSyncCredentialsResource_Create_S3_Success -v

# Run all cloud sync tests
go test ./internal/resources -run CloudSync -v

# Run with coverage
go test -cover ./internal/resources -run CloudSync

# Generate coverage report
go test -coverprofile=coverage.out ./internal/resources -run CloudSync
go tool cover -html=coverage.out

# Build
go build ./...

# Lint
golangci-lint run ./...
```
