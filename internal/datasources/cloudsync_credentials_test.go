package datasources

import (
	"context"
	"errors"
	"testing"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewCloudSyncCredentialsDataSource(t *testing.T) {
	ds := NewCloudSyncCredentialsDataSource()
	if ds == nil {
		t.Fatal("expected non-nil data source")
	}

	// Verify it implements the required interfaces
	_ = datasource.DataSource(ds)
	var _ datasource.DataSourceWithConfigure = ds.(*CloudSyncCredentialsDataSource)
}

func TestCloudSyncCredentialsDataSource_Metadata(t *testing.T) {
	ds := NewCloudSyncCredentialsDataSource()

	req := datasource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_cloudsync_credentials" {
		t.Errorf("expected TypeName 'truenas_cloudsync_credentials', got %q", resp.TypeName)
	}
}

func TestCloudSyncCredentialsDataSource_Schema(t *testing.T) {
	ds := NewCloudSyncCredentialsDataSource()

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), req, resp)

	// Verify schema has description
	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify name attribute exists and is required
	nameAttr, ok := resp.Schema.Attributes["name"]
	if !ok {
		t.Fatal("expected 'name' attribute in schema")
	}
	if !nameAttr.IsRequired() {
		t.Error("expected 'name' attribute to be required")
	}

	// Verify id attribute exists and is computed
	idAttr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal("expected 'id' attribute in schema")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' attribute to be computed")
	}

	// Verify provider_type attribute exists and is computed
	providerTypeAttr, ok := resp.Schema.Attributes["provider_type"]
	if !ok {
		t.Fatal("expected 'provider_type' attribute in schema")
	}
	if !providerTypeAttr.IsComputed() {
		t.Error("expected 'provider_type' attribute to be computed")
	}
}

func TestCloudSyncCredentialsDataSource_Configure_Success(t *testing.T) {
	ds := NewCloudSyncCredentialsDataSource().(*CloudSyncCredentialsDataSource)

	svc := &services.TrueNASServices{}

	req := datasource.ConfigureRequest{
		ProviderData: svc,
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestCloudSyncCredentialsDataSource_Configure_NilProviderData(t *testing.T) {
	ds := NewCloudSyncCredentialsDataSource().(*CloudSyncCredentialsDataSource)

	req := datasource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	// Should not error - nil ProviderData is valid during schema validation
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestCloudSyncCredentialsDataSource_Configure_WrongType(t *testing.T) {
	ds := NewCloudSyncCredentialsDataSource().(*CloudSyncCredentialsDataSource)

	req := datasource.ConfigureRequest{
		ProviderData: "not a services",
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

// createCloudSyncCredentialsTestReadRequest creates a datasource.ReadRequest with the given name
func createCloudSyncCredentialsTestReadRequest(t *testing.T, name string) datasource.ReadRequest {
	t.Helper()

	// Get the schema
	ds := NewCloudSyncCredentialsDataSource()
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	// Build config value
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"name":          tftypes.String,
			"provider_type": tftypes.String,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, nil),
		"name":          tftypes.NewValue(tftypes.String, name),
		"provider_type": tftypes.NewValue(tftypes.String, nil),
	})

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    configValue,
	}

	return datasource.ReadRequest{
		Config: config,
	}
}

func TestCloudSyncCredentialsDataSource_Read_Success_S3(t *testing.T) {
	ds := &CloudSyncCredentialsDataSource{
		services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				ListCredentialsFunc: func(ctx context.Context) ([]truenas.CloudSyncCredential, error) {
					return []truenas.CloudSyncCredential{
						{
							ID:           5,
							Name:         "Scaleway",
							ProviderType: "S3",
							Attributes:   map[string]string{"access_key_id": "AKIATEST"},
						},
					}, nil
				},
			},
		},
	}

	req := createCloudSyncCredentialsTestReadRequest(t, "Scaleway")

	// Get the schema for the state
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify the state was set correctly
	var model CloudSyncCredentialsDataSourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "5" {
		t.Errorf("expected ID '5', got %q", model.ID.ValueString())
	}
	if model.Name.ValueString() != "Scaleway" {
		t.Errorf("expected Name 'Scaleway', got %q", model.Name.ValueString())
	}
	if model.ProviderType.ValueString() != "s3" {
		t.Errorf("expected ProviderType 's3', got %q", model.ProviderType.ValueString())
	}
}

func TestCloudSyncCredentialsDataSource_Read_Success_B2(t *testing.T) {
	ds := &CloudSyncCredentialsDataSource{
		services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				ListCredentialsFunc: func(ctx context.Context) ([]truenas.CloudSyncCredential, error) {
					return []truenas.CloudSyncCredential{
						{ID: 6, Name: "Backblaze", ProviderType: "B2"},
					}, nil
				},
			},
		},
	}

	req := createCloudSyncCredentialsTestReadRequest(t, "Backblaze")

	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model CloudSyncCredentialsDataSourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "6" {
		t.Errorf("expected ID '6', got %q", model.ID.ValueString())
	}
	if model.ProviderType.ValueString() != "b2" {
		t.Errorf("expected ProviderType 'b2', got %q", model.ProviderType.ValueString())
	}
}

func TestCloudSyncCredentialsDataSource_Read_Success_GCS(t *testing.T) {
	ds := &CloudSyncCredentialsDataSource{
		services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				ListCredentialsFunc: func(ctx context.Context) ([]truenas.CloudSyncCredential, error) {
					return []truenas.CloudSyncCredential{
						{ID: 7, Name: "GCS", ProviderType: "GOOGLE_CLOUD_STORAGE"},
					}, nil
				},
			},
		},
	}

	req := createCloudSyncCredentialsTestReadRequest(t, "GCS")

	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model CloudSyncCredentialsDataSourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "7" {
		t.Errorf("expected ID '7', got %q", model.ID.ValueString())
	}
	if model.ProviderType.ValueString() != "gcs" {
		t.Errorf("expected ProviderType 'gcs', got %q", model.ProviderType.ValueString())
	}
}

func TestCloudSyncCredentialsDataSource_Read_Success_Azure(t *testing.T) {
	ds := &CloudSyncCredentialsDataSource{
		services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				ListCredentialsFunc: func(ctx context.Context) ([]truenas.CloudSyncCredential, error) {
					return []truenas.CloudSyncCredential{
						{ID: 8, Name: "Azure", ProviderType: "AZUREBLOB"},
					}, nil
				},
			},
		},
	}

	req := createCloudSyncCredentialsTestReadRequest(t, "Azure")

	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model CloudSyncCredentialsDataSourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "8" {
		t.Errorf("expected ID '8', got %q", model.ID.ValueString())
	}
	if model.ProviderType.ValueString() != "azure" {
		t.Errorf("expected ProviderType 'azure', got %q", model.ProviderType.ValueString())
	}
}

func TestCloudSyncCredentialsDataSource_Read_NotFound(t *testing.T) {
	ds := &CloudSyncCredentialsDataSource{
		services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				ListCredentialsFunc: func(ctx context.Context) ([]truenas.CloudSyncCredential, error) {
					return []truenas.CloudSyncCredential{}, nil
				},
			},
		},
	}

	req := createCloudSyncCredentialsTestReadRequest(t, "nonexistent")

	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for credentials not found")
	}
}

func TestCloudSyncCredentialsDataSource_Read_MultipleCredentials_FindsMatch(t *testing.T) {
	ds := &CloudSyncCredentialsDataSource{
		services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				ListCredentialsFunc: func(ctx context.Context) ([]truenas.CloudSyncCredential, error) {
					return []truenas.CloudSyncCredential{
						{ID: 1, Name: "First", ProviderType: "S3"},
						{ID: 2, Name: "Target", ProviderType: "B2"},
						{ID: 3, Name: "Third", ProviderType: "AZUREBLOB"},
					}, nil
				},
			},
		},
	}

	req := createCloudSyncCredentialsTestReadRequest(t, "Target")

	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model CloudSyncCredentialsDataSourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "2" {
		t.Errorf("expected ID '2', got %q", model.ID.ValueString())
	}
	if model.Name.ValueString() != "Target" {
		t.Errorf("expected Name 'Target', got %q", model.Name.ValueString())
	}
	if model.ProviderType.ValueString() != "b2" {
		t.Errorf("expected ProviderType 'b2', got %q", model.ProviderType.ValueString())
	}
}

func TestCloudSyncCredentialsDataSource_Read_APIError(t *testing.T) {
	ds := &CloudSyncCredentialsDataSource{
		services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				ListCredentialsFunc: func(ctx context.Context) ([]truenas.CloudSyncCredential, error) {
					return nil, errors.New("connection failed")
				},
			},
		},
	}

	req := createCloudSyncCredentialsTestReadRequest(t, "Scaleway")

	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestCloudSyncCredentialsDataSource_Read_UnknownProvider(t *testing.T) {
	ds := &CloudSyncCredentialsDataSource{
		services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				ListCredentialsFunc: func(ctx context.Context) ([]truenas.CloudSyncCredential, error) {
					return []truenas.CloudSyncCredential{
						{ID: 9, Name: "Unknown", ProviderType: "DROPBOX"},
					}, nil
				},
			},
		},
	}

	req := createCloudSyncCredentialsTestReadRequest(t, "Unknown")

	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model CloudSyncCredentialsDataSourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// Unknown providers should be returned as-is in lowercase
	if model.ProviderType.ValueString() != "dropbox" {
		t.Errorf("expected ProviderType 'dropbox', got %q", model.ProviderType.ValueString())
	}
}

// Test that CloudSyncCredentialsDataSource implements the DataSource interface
func TestCloudSyncCredentialsDataSource_ImplementsInterfaces(t *testing.T) {
	ds := NewCloudSyncCredentialsDataSource()

	_ = datasource.DataSource(ds)
	_ = datasource.DataSourceWithConfigure(ds.(*CloudSyncCredentialsDataSource))
}
