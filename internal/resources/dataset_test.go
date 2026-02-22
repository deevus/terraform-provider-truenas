package resources

import (
	"context"
	"errors"
	"testing"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewDatasetResource(t *testing.T) {
	r := NewDatasetResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	// Verify it implements the required interfaces
	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(r.(*DatasetResource))
	_ = resource.ResourceWithImportState(r.(*DatasetResource))
	_ = resource.ResourceWithValidateConfig(r.(*DatasetResource))
}

func TestDatasetResource_ValidateConfig_ModeRequiredWithUID(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// uid set without mode - should fail validation
	configValue := createDatasetResourceModelWithPerms(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil, nil, int64(1000), nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected validation error when uid is set without mode")
	}

	// Verify error message
	found := false
	for _, diag := range resp.Diagnostics.Errors() {
		if diag.Summary() == "Mode Required with UID/GID" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error with summary 'Mode Required with UID/GID'")
	}
}

func TestDatasetResource_ValidateConfig_ModeRequiredWithGID(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// gid set without mode - should fail validation
	configValue := createDatasetResourceModelWithPerms(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, int64(1000))

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected validation error when gid is set without mode")
	}

	// Verify error message
	found := false
	for _, diag := range resp.Diagnostics.Errors() {
		if diag.Summary() == "Mode Required with UID/GID" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error with summary 'Mode Required with UID/GID'")
	}
}

func TestDatasetResource_ValidateConfig_ModeRequiredWithBoth(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// uid and gid set without mode - should fail validation
	configValue := createDatasetResourceModelWithPerms(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil, nil, int64(1000), int64(1000))

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected validation error when uid and gid are set without mode")
	}

	// Verify error message
	found := false
	for _, diag := range resp.Diagnostics.Errors() {
		if diag.Summary() == "Mode Required with UID/GID" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error with summary 'Mode Required with UID/GID'")
	}
}

func TestDatasetResource_ValidateConfig_ModeWithUID_OK(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// uid with mode - should pass validation
	configValue := createDatasetResourceModelWithPerms(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil, "755", int64(1000), nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected validation error: %v", resp.Diagnostics)
	}
}

func TestDatasetResource_ValidateConfig_ModeWithGID_OK(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// gid with mode - should pass validation
	configValue := createDatasetResourceModelWithPerms(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil, "755", nil, int64(1000))

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected validation error: %v", resp.Diagnostics)
	}
}

func TestDatasetResource_ValidateConfig_ModeWithBoth_OK(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// uid and gid with mode - should pass validation
	configValue := createDatasetResourceModelWithPerms(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil, "755", int64(1000), int64(1000))

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected validation error: %v", resp.Diagnostics)
	}
}

func TestDatasetResource_ValidateConfig_ModeOnly_OK(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// mode alone - should pass validation
	configValue := createDatasetResourceModelWithPerms(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil, "755", nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected validation error: %v", resp.Diagnostics)
	}
}

func TestDatasetResource_ValidateConfig_NoPermissions_OK(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// no permissions - should pass validation
	configValue := createDatasetResourceModel(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected validation error: %v", resp.Diagnostics)
	}
}

func TestDatasetResource_Metadata(t *testing.T) {
	r := NewDatasetResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_dataset" {
		t.Errorf("expected TypeName 'truenas_dataset', got %q", resp.TypeName)
	}
}

func TestDatasetResource_Schema(t *testing.T) {
	r := NewDatasetResource()

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

	// Verify pool attribute exists and is optional
	poolAttr, ok := resp.Schema.Attributes["pool"]
	if !ok {
		t.Fatal("expected 'pool' attribute in schema")
	}
	if !poolAttr.IsOptional() {
		t.Error("expected 'pool' attribute to be optional")
	}

	// Verify path attribute exists and is optional
	pathAttr, ok := resp.Schema.Attributes["path"]
	if !ok {
		t.Fatal("expected 'path' attribute in schema")
	}
	if !pathAttr.IsOptional() {
		t.Error("expected 'path' attribute to be optional")
	}

	// Verify parent attribute exists and is optional
	parentAttr, ok := resp.Schema.Attributes["parent"]
	if !ok {
		t.Fatal("expected 'parent' attribute in schema")
	}
	if !parentAttr.IsOptional() {
		t.Error("expected 'parent' attribute to be optional")
	}

	// Verify name attribute exists and is optional
	nameAttr, ok := resp.Schema.Attributes["name"]
	if !ok {
		t.Fatal("expected 'name' attribute in schema")
	}
	if !nameAttr.IsOptional() {
		t.Error("expected 'name' attribute to be optional")
	}

	// Verify mount_path attribute exists and is computed
	mountPathAttr, ok := resp.Schema.Attributes["mount_path"]
	if !ok {
		t.Fatal("expected 'mount_path' attribute in schema")
	}
	if !mountPathAttr.IsComputed() {
		t.Error("expected 'mount_path' attribute to be computed")
	}

	// Verify compression attribute exists and is optional
	compressionAttr, ok := resp.Schema.Attributes["compression"]
	if !ok {
		t.Fatal("expected 'compression' attribute in schema")
	}
	if !compressionAttr.IsOptional() {
		t.Error("expected 'compression' attribute to be optional")
	}

	// Verify quota attribute exists and is optional
	quotaAttr, ok := resp.Schema.Attributes["quota"]
	if !ok {
		t.Fatal("expected 'quota' attribute in schema")
	}
	if !quotaAttr.IsOptional() {
		t.Error("expected 'quota' attribute to be optional")
	}

	// Verify refquota attribute exists and is optional
	refquotaAttr, ok := resp.Schema.Attributes["refquota"]
	if !ok {
		t.Fatal("expected 'refquota' attribute in schema")
	}
	if !refquotaAttr.IsOptional() {
		t.Error("expected 'refquota' attribute to be optional")
	}

	// Verify atime attribute exists and is optional
	atimeAttr, ok := resp.Schema.Attributes["atime"]
	if !ok {
		t.Fatal("expected 'atime' attribute in schema")
	}
	if !atimeAttr.IsOptional() {
		t.Error("expected 'atime' attribute to be optional")
	}

	// Verify force_destroy attribute exists and is optional
	forceDestroyAttr, ok := resp.Schema.Attributes["force_destroy"]
	if !ok {
		t.Fatal("expected 'force_destroy' attribute in schema")
	}
	if !forceDestroyAttr.IsOptional() {
		t.Error("expected 'force_destroy' attribute to be optional")
	}

	// Verify snapshot_id attribute exists and is optional
	snapshotIDAttr, ok := resp.Schema.Attributes["snapshot_id"]
	if !ok {
		t.Fatal("expected 'snapshot_id' attribute in schema")
	}
	if !snapshotIDAttr.IsOptional() {
		t.Error("expected 'snapshot_id' attribute to be optional")
	}
}

// getDatasetResourceSchema returns the schema for the dataset resource
func getDatasetResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewDatasetResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	return *schemaResp
}

// createDatasetResourceModel creates a tftypes.Value for the dataset resource model
func createDatasetResourceModel(id, pool, path, parent, name, mountPath, compression, quota, refquota, atime, forceDestroy interface{}) tftypes.Value {
	return createDatasetResourceModelWithPerms(id, pool, path, parent, name, mountPath, compression, quota, refquota, atime, forceDestroy, nil, nil, nil)
}

// createDatasetResourceModelWithPerms creates a tftypes.Value for the dataset resource model with permissions
func createDatasetResourceModelWithPerms(id, pool, path, parent, name, mountPath, compression, quota, refquota, atime, forceDestroy, mode, uid, gid interface{}) tftypes.Value {
	return createDatasetResourceModelFull(id, pool, path, parent, name, mountPath, mountPath, compression, quota, refquota, atime, forceDestroy, mode, uid, gid)
}

// createDatasetResourceModelFull creates a tftypes.Value for the dataset resource model with all fields
func createDatasetResourceModelFull(id, pool, path, parent, name, mountPath, fullPath, compression, quota, refquota, atime, forceDestroy, mode, uid, gid interface{}) tftypes.Value {
	return createDatasetResourceModelWithSnapshot(id, pool, path, parent, name, mountPath, fullPath, compression, quota, refquota, atime, forceDestroy, mode, uid, gid, nil)
}

// createDatasetResourceModelWithSnapshot creates a tftypes.Value for the dataset resource model with all fields including snapshot_id
func createDatasetResourceModelWithSnapshot(id, pool, path, parent, name, mountPath, fullPath, compression, quota, refquota, atime, forceDestroy, mode, uid, gid, snapshotID interface{}) tftypes.Value {
	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"pool":          tftypes.String,
			"path":          tftypes.String,
			"parent":        tftypes.String,
			"name":          tftypes.String,
			"mount_path":    tftypes.String,
			"full_path":     tftypes.String,
			"compression":   tftypes.String,
			"quota":         tftypes.String,
			"refquota":      tftypes.String,
			"atime":         tftypes.String,
			"mode":          tftypes.String,
			"uid":           tftypes.Number,
			"gid":           tftypes.Number,
			"force_destroy": tftypes.Bool,
			"snapshot_id":   tftypes.String,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, id),
		"pool":          tftypes.NewValue(tftypes.String, pool),
		"path":          tftypes.NewValue(tftypes.String, path),
		"parent":        tftypes.NewValue(tftypes.String, parent),
		"name":          tftypes.NewValue(tftypes.String, name),
		"mount_path":    tftypes.NewValue(tftypes.String, mountPath),
		"full_path":     tftypes.NewValue(tftypes.String, fullPath),
		"compression":   tftypes.NewValue(tftypes.String, compression),
		"quota":         tftypes.NewValue(tftypes.String, quota),
		"refquota":      tftypes.NewValue(tftypes.String, refquota),
		"atime":         tftypes.NewValue(tftypes.String, atime),
		"mode":          tftypes.NewValue(tftypes.String, mode),
		"uid":           tftypes.NewValue(tftypes.Number, uid),
		"gid":           tftypes.NewValue(tftypes.Number, gid),
		"force_destroy": tftypes.NewValue(tftypes.Bool, forceDestroy),
		"snapshot_id":   tftypes.NewValue(tftypes.String, snapshotID),
	})
}

// datasetModelParams holds parameters for creating test model values.
// Using a struct instead of many individual parameters per the 3-param rule.
type datasetModelParams struct {
	ID           interface{}
	Pool         interface{}
	Path         interface{}
	Parent       interface{}
	Name         interface{}
	MountPath    interface{}
	FullPath     interface{}
	Compression  interface{}
	Quota        interface{}
	RefQuota     interface{}
	Atime        interface{}
	ForceDestroy interface{}
	Mode         interface{}
	UID          interface{}
	GID          interface{}
	SnapshotID   interface{}
}

// createDatasetResourceModelValue creates a tftypes.Value from datasetModelParams
func createDatasetResourceModelValue(p datasetModelParams) tftypes.Value {
	// Use MountPath for FullPath if FullPath is not set
	fullPath := p.FullPath
	if fullPath == nil {
		fullPath = p.MountPath
	}
	return createDatasetResourceModelWithSnapshot(
		p.ID, p.Pool, p.Path, p.Parent, p.Name, p.MountPath, fullPath,
		p.Compression, p.Quota, p.RefQuota, p.Atime, p.ForceDestroy,
		p.Mode, p.UID, p.GID, p.SnapshotID,
	)
}

// defaultDataset returns a standard test Dataset for use in mocks.
func defaultDataset() *truenas.Dataset {
	return &truenas.Dataset{
		ID:          "storage/apps",
		Name:        "storage/apps",
		Mountpoint:  "/mnt/storage/apps",
		Compression: "lz4",
		Quota:       0,
		RefQuota:    0,
		Atime:       "on",
	}
}

func TestDatasetResource_Create_Success(t *testing.T) {
	var capturedOpts truenas.CreateDatasetOpts

	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				CreateDatasetFunc: func(ctx context.Context, opts truenas.CreateDatasetOpts) (*truenas.Dataset, error) {
					capturedOpts = opts
					return defaultDataset(), nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	planValue := createDatasetResourceModel(nil, "storage", "apps", nil, nil, nil, "lz4", nil, nil, nil, nil)

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

	// Verify the opts were set correctly
	if capturedOpts.Name != "storage/apps" {
		t.Errorf("expected name 'storage/apps', got %q", capturedOpts.Name)
	}

	if capturedOpts.Compression != "lz4" {
		t.Errorf("expected compression 'lz4', got %q", capturedOpts.Compression)
	}

	// Verify state was set
	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "storage/apps" {
		t.Errorf("expected ID 'storage/apps', got %q", model.ID.ValueString())
	}
	if model.MountPath.ValueString() != "/mnt/storage/apps" {
		t.Errorf("expected MountPath '/mnt/storage/apps', got %q", model.MountPath.ValueString())
	}
}

func TestDatasetResource_Create_InvalidConfig(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Neither pool/path nor parent/name provided
	planValue := createDatasetResourceModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

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
		t.Fatal("expected error for invalid config")
	}
}

func TestDatasetResource_Create_WithParentName(t *testing.T) {
	var capturedOpts truenas.CreateDatasetOpts

	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				CreateDatasetFunc: func(ctx context.Context, opts truenas.CreateDatasetOpts) (*truenas.Dataset, error) {
					capturedOpts = opts
					return &truenas.Dataset{
						ID:          "tank/data/apps",
						Name:        "tank/data/apps",
						Mountpoint:  "/mnt/tank/data/apps",
						Compression: "lz4",
						Atime:       "on",
					}, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Use parent/name mode instead of pool/path
	planValue := createDatasetResourceModel(nil, nil, nil, "tank/data", "apps", nil, nil, nil, nil, nil, nil)

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

	// Verify opts include the full dataset name
	if capturedOpts.Name != "tank/data/apps" {
		t.Errorf("expected name 'tank/data/apps', got %q", capturedOpts.Name)
	}

	// Verify state was set
	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "tank/data/apps" {
		t.Errorf("expected ID 'tank/data/apps', got %q", model.ID.ValueString())
	}
}

func TestDatasetResource_Create_APIError(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				CreateDatasetFunc: func(ctx context.Context, opts truenas.CreateDatasetOpts) (*truenas.Dataset, error) {
					return nil, errors.New("dataset already exists")
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	planValue := createDatasetResourceModel(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil)

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

func TestDatasetResource_Create_DatasetNotFoundAfterCreate(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				CreateDatasetFunc: func(ctx context.Context, opts truenas.CreateDatasetOpts) (*truenas.Dataset, error) {
					return nil, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)
	planValue := createDatasetResourceModel(nil, "storage", "apps", nil, nil, nil, nil, nil, nil, nil, nil)

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
		t.Fatal("expected error when dataset not found after create")
	}
}

func TestDatasetResource_Read_Success(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					return &truenas.Dataset{
						ID:          "storage/apps",
						Name:        "storage/apps",
						Mountpoint:  "/mnt/storage/apps",
						Compression: "lz4",
						Quota:       10000000000,
						RefQuota:    5000000000,
						Atime:       "on",
					}, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// State has compression, quota, refquota, and atime set (user specified them) - they should sync from API
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", "5G", "2G", "off", nil)

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

	// Verify state was updated from API
	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "storage/apps" {
		t.Errorf("expected ID 'storage/apps', got %q", model.ID.ValueString())
	}
	if model.MountPath.ValueString() != "/mnt/storage/apps" {
		t.Errorf("expected MountPath '/mnt/storage/apps', got %q", model.MountPath.ValueString())
	}
	// Compression was set in state, so it syncs from API
	if model.Compression.ValueString() != "lz4" {
		t.Errorf("expected Compression 'lz4', got %q", model.Compression.ValueString())
	}
	// quota/refquota stored as bytes from API
	if model.Quota.ValueString() != "10000000000" {
		t.Errorf("expected Quota '10000000000', got %q", model.Quota.ValueString())
	}
	if model.RefQuota.ValueString() != "5000000000" {
		t.Errorf("expected RefQuota '5000000000', got %q", model.RefQuota.ValueString())
	}
	// Atime was set in state to "off", API returns "on", so it syncs to "on"
	if model.Atime.ValueString() != "on" {
		t.Errorf("expected Atime 'on', got %q", model.Atime.ValueString())
	}
}

func TestDatasetResource_Read_DatasetNotFound(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					return nil, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

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
		t.Error("expected state to be removed (null) when dataset not found")
	}
}

func TestDatasetResource_Read_APIError(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					return nil, errors.New("connection failed")
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

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

func TestDatasetResource_Update_Success(t *testing.T) {
	var capturedID string
	var capturedOpts truenas.UpdateDatasetOpts

	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				UpdateDatasetFunc: func(ctx context.Context, id string, opts truenas.UpdateDatasetOpts) (*truenas.Dataset, error) {
					capturedID = id
					capturedOpts = opts
					return &truenas.Dataset{
						ID:          "storage/apps",
						Name:        "storage/apps",
						Mountpoint:  "/mnt/storage/apps",
						Compression: "zstd",
						Quota:       10000000000,
						RefQuota:    5000000000,
						Atime:       "off",
					}, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Current state has lz4 compression
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

	// Plan has zstd compression (changed)
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "zstd", nil, nil, nil, nil)

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

	// Verify the service was called correctly
	if capturedID != "storage/apps" {
		t.Errorf("expected ID 'storage/apps', got %q", capturedID)
	}

	if capturedOpts.Compression != "zstd" {
		t.Errorf("expected compression 'zstd', got %q", capturedOpts.Compression)
	}

	// Verify state was updated from API response
	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// quota/refquota stored as bytes from API
	if model.Quota.ValueString() != "10000000000" {
		t.Errorf("expected Quota '10000000000', got %q", model.Quota.ValueString())
	}
	if model.RefQuota.ValueString() != "5000000000" {
		t.Errorf("expected RefQuota '5000000000', got %q", model.RefQuota.ValueString())
	}
	if model.Atime.ValueString() != "off" {
		t.Errorf("expected Atime 'off', got %q", model.Atime.ValueString())
	}
}

func TestDatasetResource_Update_NoChanges(t *testing.T) {
	apiCalled := false

	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				UpdateDatasetFunc: func(ctx context.Context, id string, opts truenas.UpdateDatasetOpts) (*truenas.Dataset, error) {
					apiCalled = true
					return nil, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Same state and plan (no changes)
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

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

	// API should NOT be called when there are no changes
	if apiCalled {
		t.Error("expected API not to be called when there are no changes")
	}
}

func TestDatasetResource_Update_APIError(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				UpdateDatasetFunc: func(ctx context.Context, id string, opts truenas.UpdateDatasetOpts) (*truenas.Dataset, error) {
					return nil, errors.New("update failed")
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "zstd", nil, nil, nil, nil)

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

func TestDatasetResource_Delete_Success(t *testing.T) {
	var capturedID string
	var capturedRecursive bool

	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				DeleteDatasetFunc: func(ctx context.Context, id string, recursive bool) error {
					capturedID = id
					capturedRecursive = recursive
					return nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

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

	// Verify the service was called correctly
	if capturedID != "storage/apps" {
		t.Errorf("expected ID 'storage/apps', got %q", capturedID)
	}

	if capturedRecursive {
		t.Error("expected recursive to be false")
	}
}

func TestDatasetResource_Delete_APIError(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				DeleteDatasetFunc: func(ctx context.Context, id string, recursive bool) error {
					return errors.New("dataset is busy")
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

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

func TestDatasetResource_Delete_WithForceDestroy(t *testing.T) {
	var capturedID string
	var capturedRecursive bool

	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				DeleteDatasetFunc: func(ctx context.Context, id string, recursive bool) error {
					capturedID = id
					capturedRecursive = recursive
					return nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// State with force_destroy = true
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, true)

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

	// Verify the service was called correctly
	if capturedID != "storage/apps" {
		t.Errorf("expected ID 'storage/apps', got %q", capturedID)
	}

	if !capturedRecursive {
		t.Error("expected recursive to be true")
	}
}

func TestDatasetResource_ImportState(t *testing.T) {
	r := NewDatasetResource().(*DatasetResource)

	schemaResp := getDatasetResourceSchema(t)

	// Initialize state with empty values (null)
	emptyStateValue := createDatasetResourceModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	req := resource.ImportStateRequest{
		ID: "storage/apps",
	}

	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    emptyStateValue,
		},
	}

	r.ImportState(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify the ID was set in state
	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "storage/apps" {
		t.Errorf("expected ID 'storage/apps', got %q", model.ID.ValueString())
	}
}

func TestGetFullName(t *testing.T) {
	tests := []struct {
		name           string
		model          DatasetResourceModel
		expectedResult string
	}{
		{
			name: "pool and path mode",
			model: DatasetResourceModel{
				Pool: stringValue("tank"),
				Path: stringValue("data/apps"),
			},
			expectedResult: "tank/data/apps",
		},
		{
			name: "parent and name mode",
			model: DatasetResourceModel{
				Parent: stringValue("tank/data"),
				Name:   stringValue("apps"),
			},
			expectedResult: "tank/data/apps",
		},
		{
			name: "pool only (invalid)",
			model: DatasetResourceModel{
				Pool: stringValue("tank"),
			},
			expectedResult: "",
		},
		{
			name: "path only (invalid)",
			model: DatasetResourceModel{
				Path: stringValue("data"),
			},
			expectedResult: "",
		},
		{
			name: "parent only (invalid)",
			model: DatasetResourceModel{
				Parent: stringValue("tank"),
			},
			expectedResult: "",
		},
		{
			name: "name only (invalid)",
			model: DatasetResourceModel{
				Name: stringValue("apps"),
			},
			expectedResult: "",
		},
		{
			name:           "all empty (invalid)",
			model:          DatasetResourceModel{},
			expectedResult: "",
		},
		{
			name: "both modes provided (pool/path takes precedence)",
			model: DatasetResourceModel{
				Pool:   stringValue("tank"),
				Path:   stringValue("data"),
				Parent: stringValue("other"),
				Name:   stringValue("name"),
			},
			expectedResult: "tank/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFullName(&tt.model)
			if result != tt.expectedResult {
				t.Errorf("expected %q, got %q", tt.expectedResult, result)
			}
		})
	}
}

func TestGetFullName_PathWithParent(t *testing.T) {
	// path with parent should work (new preferred way)
	model := DatasetResourceModel{
		Parent: stringValue("tank/data"),
		Path:   stringValue("apps"),
	}
	result := getFullName(&model)
	if result != "tank/data/apps" {
		t.Errorf("expected 'tank/data/apps', got %q", result)
	}
}

func TestGetFullName_PathOverName(t *testing.T) {
	// when both path and name are provided with parent, path takes precedence
	model := DatasetResourceModel{
		Parent: stringValue("tank/data"),
		Path:   stringValue("newpath"),
		Name:   stringValue("oldname"),
	}
	result := getFullName(&model)
	if result != "tank/data/newpath" {
		t.Errorf("expected 'tank/data/newpath', got %q", result)
	}
}

// stringValue is a helper to create a types.String with a value
func stringValue(s string) types.String {
	return types.StringValue(s)
}

// Test interface compliance
func TestDatasetResource_ImplementsInterfaces(t *testing.T) {
	r := NewDatasetResource()

	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(r.(*DatasetResource))
	_ = resource.ResourceWithImportState(r.(*DatasetResource))
}

// Additional test for Create with all optional parameters
func TestDatasetResource_Create_AllOptions(t *testing.T) {
	var capturedOpts truenas.CreateDatasetOpts

	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				CreateDatasetFunc: func(ctx context.Context, opts truenas.CreateDatasetOpts) (*truenas.Dataset, error) {
					capturedOpts = opts
					return &truenas.Dataset{
						ID:          "storage/apps",
						Name:        "storage/apps",
						Mountpoint:  "/mnt/storage/apps",
						Compression: "zstd",
						Quota:       10000000000,
						RefQuota:    5000000000,
						Atime:       "on",
					}, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	planValue := createDatasetResourceModel(nil, "storage", "apps", nil, nil, nil, "zstd", "10G", "5G", "on", nil)

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

	// Verify opts include all options
	if capturedOpts.Compression != "zstd" {
		t.Errorf("expected compression 'zstd', got %q", capturedOpts.Compression)
	}

	// quota and refquota are sent as int64 bytes (SI units: 1G = 1000^3)
	if capturedOpts.Quota != int64(10000000000) {
		t.Errorf("expected quota 10000000000 (10G in bytes), got %v", capturedOpts.Quota)
	}

	if capturedOpts.RefQuota != int64(5000000000) {
		t.Errorf("expected refquota 5000000000 (5G in bytes), got %v", capturedOpts.RefQuota)
	}

	if capturedOpts.Atime != "on" {
		t.Errorf("expected atime 'on', got %q", capturedOpts.Atime)
	}
}

// Test Read with parent/name mode
func TestDatasetResource_Read_WithParentName(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					return &truenas.Dataset{
						ID:          "tank/data/apps",
						Name:        "tank/data/apps",
						Mountpoint:  "/mnt/tank/data/apps",
						Compression: "lz4",
						Atime:       "on",
					}, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("tank/data/apps", nil, nil, "tank/data", "apps", "/mnt/tank/data/apps", "lz4", nil, nil, nil, nil)

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

	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "tank/data/apps" {
		t.Errorf("expected ID 'tank/data/apps', got %q", model.ID.ValueString())
	}
}

// Test Update with quota change
func TestDatasetResource_Update_QuotaChange(t *testing.T) {
	var capturedOpts truenas.UpdateDatasetOpts

	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				UpdateDatasetFunc: func(ctx context.Context, id string, opts truenas.UpdateDatasetOpts) (*truenas.Dataset, error) {
					capturedOpts = opts
					return &truenas.Dataset{
						ID:          "storage/apps",
						Name:        "storage/apps",
						Mountpoint:  "/mnt/storage/apps",
						Compression: "lz4",
						Quota:       10000000000,
						Atime:       "on",
					}, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Current state has no quota
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

	// Plan adds quota
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", "10G", nil, nil, nil)

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

	// Verify opts include the quota
	if capturedOpts.Quota == nil {
		t.Fatal("expected Quota to be set")
	}

	// quota is sent as int64 bytes (SI units: 1G = 1000^3)
	if *capturedOpts.Quota != int64(10000000000) {
		t.Errorf("expected quota 10000000000 (10G in bytes), got %v", *capturedOpts.Quota)
	}
}

// Test Update with refquota change
func TestDatasetResource_Update_RefQuotaChange(t *testing.T) {
	var capturedOpts truenas.UpdateDatasetOpts

	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				UpdateDatasetFunc: func(ctx context.Context, id string, opts truenas.UpdateDatasetOpts) (*truenas.Dataset, error) {
					capturedOpts = opts
					return &truenas.Dataset{
						ID:          "storage/apps",
						Name:        "storage/apps",
						Mountpoint:  "/mnt/storage/apps",
						Compression: "lz4",
						RefQuota:    5000000000,
						Atime:       "on",
					}, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, "5G", nil, nil)

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

	// Verify opts include refquota
	if capturedOpts.RefQuota == nil {
		t.Fatal("expected RefQuota to be set")
	}

	// refquota is sent as int64 bytes (SI units: 1G = 1000^3)
	if *capturedOpts.RefQuota != int64(5000000000) {
		t.Errorf("expected refquota 5000000000 (5G in bytes), got %v", *capturedOpts.RefQuota)
	}
}

// Test Update with atime change
func TestDatasetResource_Update_AtimeChange(t *testing.T) {
	var capturedOpts truenas.UpdateDatasetOpts

	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				UpdateDatasetFunc: func(ctx context.Context, id string, opts truenas.UpdateDatasetOpts) (*truenas.Dataset, error) {
					capturedOpts = opts
					return &truenas.Dataset{
						ID:          "storage/apps",
						Name:        "storage/apps",
						Mountpoint:  "/mnt/storage/apps",
						Compression: "lz4",
						Atime:       "off",
					}, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, "off", nil)

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

	if capturedOpts.Atime != "off" {
		t.Errorf("expected atime 'off', got %q", capturedOpts.Atime)
	}
}

// Test Create with plan parsing error
func TestDatasetResource_Create_PlanParseError(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Create an invalid plan value with wrong type
	planValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"pool":          tftypes.Number, // Wrong type!
			"path":          tftypes.String,
			"parent":        tftypes.String,
			"name":          tftypes.String,
			"mount_path":    tftypes.String,
			"full_path":     tftypes.String,
			"compression":   tftypes.String,
			"quota":         tftypes.String,
			"refquota":      tftypes.String,
			"atime":         tftypes.String,
			"mode":          tftypes.String,
			"uid":           tftypes.Number,
			"gid":           tftypes.Number,
			"force_destroy": tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, nil),
		"pool":          tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"path":          tftypes.NewValue(tftypes.String, "apps"),
		"parent":        tftypes.NewValue(tftypes.String, nil),
		"name":          tftypes.NewValue(tftypes.String, nil),
		"mount_path":    tftypes.NewValue(tftypes.String, nil),
		"full_path":     tftypes.NewValue(tftypes.String, nil),
		"compression":   tftypes.NewValue(tftypes.String, nil),
		"quota":         tftypes.NewValue(tftypes.String, nil),
		"refquota":      tftypes.NewValue(tftypes.String, nil),
		"atime":         tftypes.NewValue(tftypes.String, nil),
		"mode":          tftypes.NewValue(tftypes.String, nil),
		"uid":           tftypes.NewValue(tftypes.Number, nil),
		"gid":           tftypes.NewValue(tftypes.Number, nil),
		"force_destroy": tftypes.NewValue(tftypes.Bool, nil),
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
func TestDatasetResource_Read_StateParseError(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Create an invalid state value with wrong type
	stateValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.Number, // Wrong type!
			"pool":          tftypes.String,
			"path":          tftypes.String,
			"parent":        tftypes.String,
			"name":          tftypes.String,
			"mount_path":    tftypes.String,
			"full_path":     tftypes.String,
			"compression":   tftypes.String,
			"quota":         tftypes.String,
			"refquota":      tftypes.String,
			"atime":         tftypes.String,
			"mode":          tftypes.String,
			"uid":           tftypes.Number,
			"gid":           tftypes.Number,
			"force_destroy": tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"pool":          tftypes.NewValue(tftypes.String, "storage"),
		"path":          tftypes.NewValue(tftypes.String, "apps"),
		"parent":        tftypes.NewValue(tftypes.String, nil),
		"name":          tftypes.NewValue(tftypes.String, nil),
		"mount_path":    tftypes.NewValue(tftypes.String, "/mnt/storage/apps"),
		"full_path":     tftypes.NewValue(tftypes.String, nil),
		"compression":   tftypes.NewValue(tftypes.String, "lz4"),
		"quota":         tftypes.NewValue(tftypes.String, nil),
		"refquota":      tftypes.NewValue(tftypes.String, nil),
		"atime":         tftypes.NewValue(tftypes.String, nil),
		"mode":          tftypes.NewValue(tftypes.String, nil),
		"uid":           tftypes.NewValue(tftypes.Number, nil),
		"gid":           tftypes.NewValue(tftypes.Number, nil),
		"force_destroy": tftypes.NewValue(tftypes.Bool, nil),
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
func TestDatasetResource_Update_PlanParseError(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Valid state
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil)

	// Invalid plan with wrong type
	planValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"pool":          tftypes.Number, // Wrong type!
			"path":          tftypes.String,
			"parent":        tftypes.String,
			"name":          tftypes.String,
			"mount_path":    tftypes.String,
			"full_path":     tftypes.String,
			"compression":   tftypes.String,
			"quota":         tftypes.String,
			"refquota":      tftypes.String,
			"atime":         tftypes.String,
			"mode":          tftypes.String,
			"uid":           tftypes.Number,
			"gid":           tftypes.Number,
			"force_destroy": tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, "storage/apps"),
		"pool":          tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"path":          tftypes.NewValue(tftypes.String, "apps"),
		"parent":        tftypes.NewValue(tftypes.String, nil),
		"name":          tftypes.NewValue(tftypes.String, nil),
		"mount_path":    tftypes.NewValue(tftypes.String, "/mnt/storage/apps"),
		"full_path":     tftypes.NewValue(tftypes.String, nil),
		"compression":   tftypes.NewValue(tftypes.String, "zstd"),
		"quota":         tftypes.NewValue(tftypes.String, nil),
		"refquota":      tftypes.NewValue(tftypes.String, nil),
		"atime":         tftypes.NewValue(tftypes.String, nil),
		"mode":          tftypes.NewValue(tftypes.String, nil),
		"uid":           tftypes.NewValue(tftypes.Number, nil),
		"gid":           tftypes.NewValue(tftypes.Number, nil),
		"force_destroy": tftypes.NewValue(tftypes.Bool, nil),
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
func TestDatasetResource_Update_StateParseError(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Invalid state with wrong type
	stateValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.Number, // Wrong type!
			"pool":          tftypes.String,
			"path":          tftypes.String,
			"parent":        tftypes.String,
			"name":          tftypes.String,
			"mount_path":    tftypes.String,
			"full_path":     tftypes.String,
			"compression":   tftypes.String,
			"quota":         tftypes.String,
			"refquota":      tftypes.String,
			"atime":         tftypes.String,
			"mode":          tftypes.String,
			"uid":           tftypes.Number,
			"gid":           tftypes.Number,
			"force_destroy": tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"pool":          tftypes.NewValue(tftypes.String, "storage"),
		"path":          tftypes.NewValue(tftypes.String, "apps"),
		"parent":        tftypes.NewValue(tftypes.String, nil),
		"name":          tftypes.NewValue(tftypes.String, nil),
		"mount_path":    tftypes.NewValue(tftypes.String, "/mnt/storage/apps"),
		"full_path":     tftypes.NewValue(tftypes.String, nil),
		"compression":   tftypes.NewValue(tftypes.String, "lz4"),
		"quota":         tftypes.NewValue(tftypes.String, nil),
		"refquota":      tftypes.NewValue(tftypes.String, nil),
		"atime":         tftypes.NewValue(tftypes.String, nil),
		"mode":          tftypes.NewValue(tftypes.String, nil),
		"uid":           tftypes.NewValue(tftypes.Number, nil),
		"gid":           tftypes.NewValue(tftypes.Number, nil),
		"force_destroy": tftypes.NewValue(tftypes.Bool, nil),
	})

	// Valid plan
	planValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "zstd", nil, nil, nil, nil)

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

// Test that compression attribute is Optional+Computed with UseStateForUnknown
func TestDatasetResource_Schema_CompressionIsComputed(t *testing.T) {
	r := NewDatasetResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify compression attribute is computed (required for UseStateForUnknown)
	compressionAttr, ok := resp.Schema.Attributes["compression"]
	if !ok {
		t.Fatal("expected 'compression' attribute in schema")
	}
	if !compressionAttr.IsComputed() {
		t.Error("expected 'compression' attribute to be computed")
	}
}

// Test that atime attribute is Optional+Computed with UseStateForUnknown
func TestDatasetResource_Schema_AtimeIsComputed(t *testing.T) {
	r := NewDatasetResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify atime attribute is computed (required for UseStateForUnknown)
	atimeAttr, ok := resp.Schema.Attributes["atime"]
	if !ok {
		t.Fatal("expected 'atime' attribute in schema")
	}
	if !atimeAttr.IsComputed() {
		t.Error("expected 'atime' attribute to be computed")
	}
}

// Test that Read preserves null compression when not set in config
func TestDatasetResource_Read_PopulatesComputedAttributes(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					return &truenas.Dataset{
						ID:          "storage/apps",
						Name:        "storage/apps",
						Mountpoint:  "/mnt/storage/apps",
						Compression: "LZ4",
						Atime:       "OFF",
					}, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// State has null computed values (e.g., after import or first read)
	stateValue := createDatasetResourceModel("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", nil, nil, nil, nil, nil)

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

	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// All computed attributes should be populated from API response
	if model.Compression.ValueString() != "LZ4" {
		t.Errorf("expected compression 'LZ4', got %q", model.Compression.ValueString())
	}
	if model.Quota.ValueString() != "0" {
		t.Errorf("expected quota '0', got %q", model.Quota.ValueString())
	}
	if model.RefQuota.ValueString() != "0" {
		t.Errorf("expected refquota '0', got %q", model.RefQuota.ValueString())
	}
	if model.Atime.ValueString() != "OFF" {
		t.Errorf("expected atime 'OFF', got %q", model.Atime.ValueString())
	}
}

// Test Delete with state parsing error
func TestDatasetResource_Delete_StateParseError(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Invalid state with wrong type
	stateValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.Number, // Wrong type!
			"pool":          tftypes.String,
			"path":          tftypes.String,
			"parent":        tftypes.String,
			"name":          tftypes.String,
			"mount_path":    tftypes.String,
			"full_path":     tftypes.String,
			"compression":   tftypes.String,
			"quota":         tftypes.String,
			"refquota":      tftypes.String,
			"atime":         tftypes.String,
			"mode":          tftypes.String,
			"uid":           tftypes.Number,
			"gid":           tftypes.Number,
			"force_destroy": tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"pool":          tftypes.NewValue(tftypes.String, "storage"),
		"path":          tftypes.NewValue(tftypes.String, "apps"),
		"parent":        tftypes.NewValue(tftypes.String, nil),
		"name":          tftypes.NewValue(tftypes.String, nil),
		"mount_path":    tftypes.NewValue(tftypes.String, "/mnt/storage/apps"),
		"full_path":     tftypes.NewValue(tftypes.String, nil),
		"compression":   tftypes.NewValue(tftypes.String, "lz4"),
		"quota":         tftypes.NewValue(tftypes.String, nil),
		"refquota":      tftypes.NewValue(tftypes.String, nil),
		"atime":         tftypes.NewValue(tftypes.String, nil),
		"mode":          tftypes.NewValue(tftypes.String, nil),
		"uid":           tftypes.NewValue(tftypes.Number, nil),
		"gid":           tftypes.NewValue(tftypes.Number, nil),
		"force_destroy": tftypes.NewValue(tftypes.Bool, nil),
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

// Test Create with permissions
func TestDatasetResource_Create_WithPermissions(t *testing.T) {
	var setpermCalled bool
	var setpermOpts truenas.SetPermOpts

	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				CreateDatasetFunc: func(ctx context.Context, opts truenas.CreateDatasetOpts) (*truenas.Dataset, error) {
					return &truenas.Dataset{
						ID:          "storage/apps",
						Name:        "storage/apps",
						Mountpoint:  "/mnt/storage/apps",
						Compression: "lz4",
						Atime:       "off",
					}, nil
				},
			},
			Filesystem: &truenas.MockFilesystemService{
				SetPermissionsFunc: func(ctx context.Context, opts truenas.SetPermOpts) error {
					setpermCalled = true
					setpermOpts = opts
					return nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	planValue := createDatasetResourceModelWithPerms(nil, "storage", "apps", nil, nil, nil, "lz4", nil, nil, nil, nil, "755", int64(1000), int64(1000))

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

	if !setpermCalled {
		t.Fatal("expected filesystem.SetPermissions to be called")
	}

	if setpermOpts.Path != "/mnt/storage/apps" {
		t.Errorf("expected path '/mnt/storage/apps', got %v", setpermOpts.Path)
	}

	if setpermOpts.Mode != "755" {
		t.Errorf("expected mode '755', got %v", setpermOpts.Mode)
	}

	if setpermOpts.UID == nil || *setpermOpts.UID != int64(1000) {
		t.Errorf("expected uid 1000, got %v", setpermOpts.UID)
	}

	if setpermOpts.GID == nil || *setpermOpts.GID != int64(1000) {
		t.Errorf("expected gid 1000, got %v", setpermOpts.GID)
	}
}

// Test Create without permissions does not call setperm
func TestDatasetResource_Create_NoPermissions(t *testing.T) {
	var setpermCalled bool

	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				CreateDatasetFunc: func(ctx context.Context, opts truenas.CreateDatasetOpts) (*truenas.Dataset, error) {
					return &truenas.Dataset{
						ID:          "storage/apps",
						Name:        "storage/apps",
						Mountpoint:  "/mnt/storage/apps",
						Compression: "lz4",
						Atime:       "off",
					}, nil
				},
			},
			Filesystem: &truenas.MockFilesystemService{
				SetPermissionsFunc: func(ctx context.Context, opts truenas.SetPermOpts) error {
					setpermCalled = true
					return nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// No permissions specified (nil for mode, uid, gid)
	planValue := createDatasetResourceModel(nil, "storage", "apps", nil, nil, nil, "lz4", nil, nil, nil, nil)

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

	if setpermCalled {
		t.Fatal("filesystem.SetPermissions should not be called when no permissions specified")
	}
}

// Test Update with permission changes
func TestDatasetResource_Update_PermissionChange(t *testing.T) {
	var setpermCalled bool
	var setpermOpts truenas.SetPermOpts

	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{},
			Filesystem: &truenas.MockFilesystemService{
				SetPermissionsFunc: func(ctx context.Context, opts truenas.SetPermOpts) error {
					setpermCalled = true
					setpermOpts = opts
					return nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// State with mode 755, plan with mode 700
	stateValue := createDatasetResourceModelWithPerms("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil, "755", int64(0), int64(0))
	planValue := createDatasetResourceModelWithPerms("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil, "700", int64(0), int64(0))

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

	if !setpermCalled {
		t.Fatal("expected filesystem.SetPermissions to be called for permission change")
	}

	if setpermOpts.Mode != "700" {
		t.Errorf("expected mode '700', got %v", setpermOpts.Mode)
	}
}

func TestDatasetResource_Schema_FullPathExists(t *testing.T) {
	r := NewDatasetResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	fullPathAttr, ok := resp.Schema.Attributes["full_path"]
	if !ok {
		t.Fatal("expected 'full_path' attribute in schema")
	}
	if !fullPathAttr.IsComputed() {
		t.Error("expected 'full_path' attribute to be computed")
	}
}

// Test Read populates both mount_path and full_path from API response
func TestDatasetResource_Read_BothMountPathAndFullPath(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					return &truenas.Dataset{
						ID:          "storage/apps",
						Name:        "storage/apps",
						Mountpoint:  "/mnt/storage/apps",
						Compression: "lz4",
						Atime:       "on",
					}, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)
	// Start with empty/null full_path to verify it gets populated from API
	stateValue := createDatasetResourceModelFull("storage/apps", "storage", "apps", nil, nil, nil, nil, "lz4", nil, nil, nil, nil, nil, nil, nil)

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

	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.MountPath.ValueString() != "/mnt/storage/apps" {
		t.Errorf("expected MountPath '/mnt/storage/apps', got %q", model.MountPath.ValueString())
	}
	if model.FullPath.ValueString() != "/mnt/storage/apps" {
		t.Errorf("expected FullPath '/mnt/storage/apps', got %q", model.FullPath.ValueString())
	}
}

func TestDatasetResource_Schema_MountPathDeprecated(t *testing.T) {
	r := NewDatasetResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	mountPathAttr, ok := resp.Schema.Attributes["mount_path"]
	if !ok {
		t.Fatal("expected 'mount_path' attribute in schema")
	}
	if mountPathAttr.GetDeprecationMessage() == "" {
		t.Error("expected 'mount_path' attribute to have deprecation message")
	}
}

// Test Read reads permissions from filesystem.stat
func TestDatasetResource_Schema_NameDeprecated(t *testing.T) {
	r := NewDatasetResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	nameAttr, ok := resp.Schema.Attributes["name"]
	if !ok {
		t.Fatal("expected 'name' attribute in schema")
	}
	if nameAttr.GetDeprecationMessage() == "" {
		t.Error("expected 'name' attribute to have deprecation message")
	}
}

func TestDatasetResource_Read_WithPermissions(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					return &truenas.Dataset{
						ID:          "storage/apps",
						Name:        "storage/apps",
						Mountpoint:  "/mnt/storage/apps",
						Compression: "lz4",
						Atime:       "off",
					}, nil
				},
			},
			Filesystem: &truenas.MockFilesystemService{
				StatFunc: func(ctx context.Context, path string) (*truenas.StatResult, error) {
					// StatResult.Mode is already masked with 0o777 in truenas-go
					return &truenas.StatResult{Mode: 0o755, UID: 1000, GID: 1000}, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// State has permissions configured, so Read should update them from filesystem.Stat
	stateValue := createDatasetResourceModelWithPerms("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil, "700", int64(0), int64(0))

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

	// Verify the state was updated with new permission values
	var data DatasetResourceModel
	resp.State.Get(context.Background(), &data)

	// Mode is already masked in truenas-go, so 0o755 = "755"
	if data.Mode.ValueString() != "755" {
		t.Errorf("expected mode '755', got %v", data.Mode.ValueString())
	}

	if data.UID.ValueInt64() != 1000 {
		t.Errorf("expected uid 1000, got %v", data.UID.ValueInt64())
	}

	if data.GID.ValueInt64() != 1000 {
		t.Errorf("expected gid 1000, got %v", data.GID.ValueInt64())
	}
}

// Test Read after import populates pool/path from ID
func TestDatasetResource_Read_AfterImport_PopulatesPoolAndPath(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					return &truenas.Dataset{
						ID:          "tank/data/apps",
						Name:        "tank/data/apps",
						Mountpoint:  "/mnt/tank/data/apps",
						Compression: "lz4",
						Atime:       "on",
					}, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// After import, only ID is set - pool, path, parent, and name are all null
	stateValue := createDatasetResourceModel("tank/data/apps", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

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

	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// Pool and Path should be populated from the ID
	if model.Pool.ValueString() != "tank" {
		t.Errorf("expected Pool 'tank', got %q", model.Pool.ValueString())
	}
	if model.Path.ValueString() != "data/apps" {
		t.Errorf("expected Path 'data/apps', got %q", model.Path.ValueString())
	}
}

// Test Read after import with simple pool-level dataset
func TestDatasetResource_Read_AfterImport_SimpleDataset(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					return &truenas.Dataset{
						ID:          "tank/apps",
						Name:        "tank/apps",
						Mountpoint:  "/mnt/tank/apps",
						Compression: "lz4",
						Atime:       "on",
					}, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// After import, only ID is set
	stateValue := createDatasetResourceModel("tank/apps", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

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

	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// Pool and Path should be populated from the ID
	if model.Pool.ValueString() != "tank" {
		t.Errorf("expected Pool 'tank', got %q", model.Pool.ValueString())
	}
	if model.Path.ValueString() != "apps" {
		t.Errorf("expected Path 'apps', got %q", model.Path.ValueString())
	}
}

// Test Read does NOT override pool/path when already set (not import scenario)
func TestDatasetResource_Read_DoesNotOverridePoolPathWhenSet(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					return &truenas.Dataset{
						ID:          "tank/data/apps",
						Name:        "tank/data/apps",
						Mountpoint:  "/mnt/tank/data/apps",
						Compression: "lz4",
						Atime:       "on",
					}, nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// Normal read - pool and path are already set from config
	stateValue := createDatasetResourceModel("tank/data/apps", "tank", "data/apps", nil, nil, "/mnt/tank/data/apps", "lz4", nil, nil, nil, nil)

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

	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// Pool and Path should remain unchanged
	if model.Pool.ValueString() != "tank" {
		t.Errorf("expected Pool 'tank', got %q", model.Pool.ValueString())
	}
	if model.Path.ValueString() != "data/apps" {
		t.Errorf("expected Path 'data/apps', got %q", model.Path.ValueString())
	}
}

// Test Read with permissions returns warning on filesystem.stat error
func TestDatasetResource_Read_PermissionsStatError_ReturnsWarning(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					return &truenas.Dataset{
						ID:          "storage/apps",
						Name:        "storage/apps",
						Mountpoint:  "/mnt/storage/apps",
						Compression: "lz4",
						Atime:       "off",
					}, nil
				},
			},
			Filesystem: &truenas.MockFilesystemService{
				StatFunc: func(ctx context.Context, path string) (*truenas.StatResult, error) {
					return nil, errors.New("permission denied")
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)

	// State has permissions configured, so Read will try to call filesystem.Stat
	stateValue := createDatasetResourceModelWithPerms("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "lz4", nil, nil, nil, nil, "755", int64(1000), int64(1000))

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

	// Should NOT have errors - but should have a warning
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Should have a warning about permissions
	if resp.Diagnostics.WarningsCount() == 0 {
		t.Fatal("expected warning for filesystem.stat error")
	}

	// Verify warning message content
	var foundWarning bool
	for _, diag := range resp.Diagnostics {
		if diag.Severity().String() == "Warning" && diag.Summary() == "Unable to Read Mountpoint Permissions" {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Error("expected warning with summary 'Unable to Read Mountpoint Permissions'")
	}

	// State should still be set (read succeeded, just permissions reading failed)
	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "storage/apps" {
		t.Errorf("expected ID 'storage/apps', got %q", model.ID.ValueString())
	}
}

func TestDatasetResource_Create_WithSnapshotId(t *testing.T) {
	var cloneCalled bool
	var cloneSnapshot, cloneDatasetDst string
	var createCalled bool

	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Dataset: &truenas.MockDatasetService{
				CreateDatasetFunc: func(ctx context.Context, opts truenas.CreateDatasetOpts) (*truenas.Dataset, error) {
					createCalled = true
					return nil, nil
				},
				GetDatasetFunc: func(ctx context.Context, id string) (*truenas.Dataset, error) {
					return &truenas.Dataset{
						ID:          "tank/restored",
						Name:        "restored",
						Mountpoint:  "/mnt/tank/restored",
						Compression: "lz4",
						Atime:       "on",
					}, nil
				},
			},
			Snapshot: &truenas.MockSnapshotService{
				CloneFunc: func(ctx context.Context, snapshot, datasetDst string) error {
					cloneCalled = true
					cloneSnapshot = snapshot
					cloneDatasetDst = datasetDst
					return nil
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)
	planValue := createDatasetResourceModelValue(datasetModelParams{
		Pool:       "tank",
		Path:       "restored",
		SnapshotID: "tank/data@snap1",
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

	// Verify Snapshot.Clone was called
	if !cloneCalled {
		t.Error("expected Snapshot.Clone to be called")
	}

	// Verify Dataset.CreateDataset was NOT called
	if createCalled {
		t.Error("expected Dataset.CreateDataset to NOT be called when snapshot_id is set")
	}

	if cloneSnapshot != "tank/data@snap1" {
		t.Errorf("expected snapshot 'tank/data@snap1', got %v", cloneSnapshot)
	}

	if cloneDatasetDst != "tank/restored" {
		t.Errorf("expected dataset_dst 'tank/restored', got %v", cloneDatasetDst)
	}
}

func TestDatasetResource_Create_WithSnapshotId_APIError(t *testing.T) {
	r := &DatasetResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				CloneFunc: func(ctx context.Context, snapshot, datasetDst string) error {
					return errors.New("snapshot not found")
				},
			},
		}},
	}

	schemaResp := getDatasetResourceSchema(t)
	planValue := createDatasetResourceModelValue(datasetModelParams{
		Pool:       "tank",
		Path:       "restored",
		SnapshotID: "tank/data@nonexistent",
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
		t.Fatal("expected error for clone API error")
	}
}
