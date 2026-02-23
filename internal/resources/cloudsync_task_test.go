package resources

import (
	"context"
	"errors"
	"math/big"
	"testing"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewCloudSyncTaskResource(t *testing.T) {
	r := NewCloudSyncTaskResource()
	if r == nil {
		t.Fatal("NewCloudSyncTaskResource returned nil")
	}

	_, ok := r.(*CloudSyncTaskResource)
	if !ok {
		t.Fatalf("expected *CloudSyncTaskResource, got %T", r)
	}

	// Verify interface implementations
	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(r.(*CloudSyncTaskResource))
	_ = resource.ResourceWithImportState(r.(*CloudSyncTaskResource))
}

func TestCloudSyncTaskResource_Metadata(t *testing.T) {
	r := NewCloudSyncTaskResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_cloudsync_task" {
		t.Errorf("expected TypeName 'truenas_cloudsync_task', got %q", resp.TypeName)
	}
}

func TestCloudSyncTaskResource_Configure_Success(t *testing.T) {
	r := NewCloudSyncTaskResource().(*CloudSyncTaskResource)

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

func TestCloudSyncTaskResource_Configure_NilProviderData(t *testing.T) {
	r := NewCloudSyncTaskResource().(*CloudSyncTaskResource)

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestCloudSyncTaskResource_Configure_WrongType(t *testing.T) {
	r := NewCloudSyncTaskResource().(*CloudSyncTaskResource)

	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

func TestCloudSyncTaskResource_Schema(t *testing.T) {
	r := NewCloudSyncTaskResource()

	ctx := context.Background()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}

	r.Schema(ctx, schemaReq, schemaResp)

	if schemaResp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify required attributes exist
	attrs := schemaResp.Schema.Attributes
	if attrs["id"] == nil {
		t.Error("expected 'id' attribute")
	}
	if attrs["description"] == nil {
		t.Error("expected 'description' attribute")
	}
	if attrs["path"] == nil {
		t.Error("expected 'path' attribute")
	}
	if attrs["credentials"] == nil {
		t.Error("expected 'credentials' attribute")
	}
	if attrs["direction"] == nil {
		t.Error("expected 'direction' attribute")
	}
	if attrs["transfer_mode"] == nil {
		t.Error("expected 'transfer_mode' attribute")
	}

	// Verify sync_on_change attribute
	if attrs["sync_on_change"] == nil {
		t.Error("expected 'sync_on_change' attribute")
	}

	// Verify include attribute
	if attrs["include"] == nil {
		t.Error("expected 'include' attribute")
	}

	// Verify blocks exist
	blocks := schemaResp.Schema.Blocks
	if blocks["schedule"] == nil {
		t.Error("expected 'schedule' block")
	}
	if blocks["encryption"] == nil {
		t.Error("expected 'encryption' block")
	}
	if blocks["s3"] == nil {
		t.Error("expected 's3' block")
	}
	if blocks["b2"] == nil {
		t.Error("expected 'b2' block")
	}
	if blocks["gcs"] == nil {
		t.Error("expected 'gcs' block")
	}
	if blocks["azure"] == nil {
		t.Error("expected 'azure' block")
	}
}

// Test helpers

func getCloudSyncTaskResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewCloudSyncTaskResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("failed to get schema: %v", schemaResp.Diagnostics)
	}
	return *schemaResp
}

// cloudSyncTaskModelParams holds parameters for creating test model values.
type cloudSyncTaskModelParams struct {
	ID                 interface{}
	Description        interface{}
	Path               interface{}
	Credentials        int64
	Direction          interface{}
	TransferMode       interface{}
	Snapshot           bool
	Transfers          int64
	BWLimit            interface{}
	Exclude            []string
	Include            []string
	FollowSymlinks     bool
	CreateEmptySrcDirs bool
	Enabled            bool
	SyncOnChange       bool
	Schedule           *scheduleBlockParams
	Encryption         *encryptionBlockParams
	S3                 *taskS3BlockParams
	B2                 *taskB2BlockParams
	GCS                *taskGCSBlockParams
	Azure              *taskAzureBlockParams
}

type scheduleBlockParams struct {
	Minute interface{}
	Hour   interface{}
	Dom    interface{}
	Month  interface{}
	Dow    interface{}
}

type encryptionBlockParams struct {
	Password interface{}
	Salt     interface{}
}

type taskS3BlockParams struct {
	Bucket interface{}
	Folder interface{}
}

type taskB2BlockParams struct {
	Bucket interface{}
	Folder interface{}
}

type taskGCSBlockParams struct {
	Bucket interface{}
	Folder interface{}
}

type taskAzureBlockParams struct {
	Container interface{}
	Folder    interface{}
}

func createCloudSyncTaskModelValue(p cloudSyncTaskModelParams) tftypes.Value {
	// Define type structures
	scheduleType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"minute": tftypes.String,
			"hour":   tftypes.String,
			"dom":    tftypes.String,
			"month":  tftypes.String,
			"dow":    tftypes.String,
		},
	}

	encryptionType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"password": tftypes.String,
			"salt":     tftypes.String,
		},
	}

	bucketFolderType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"bucket": tftypes.String,
			"folder": tftypes.String,
		},
	}

	containerFolderType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"container": tftypes.String,
			"folder":    tftypes.String,
		},
	}

	// Build the values map
	values := map[string]tftypes.Value{
		"id":                    tftypes.NewValue(tftypes.String, p.ID),
		"description":           tftypes.NewValue(tftypes.String, p.Description),
		"path":                  tftypes.NewValue(tftypes.String, p.Path),
		"credentials":           tftypes.NewValue(tftypes.Number, big.NewFloat(float64(p.Credentials))),
		"direction":             tftypes.NewValue(tftypes.String, p.Direction),
		"transfer_mode":         tftypes.NewValue(tftypes.String, p.TransferMode),
		"snapshot":              tftypes.NewValue(tftypes.Bool, p.Snapshot),
		"transfers":             tftypes.NewValue(tftypes.Number, big.NewFloat(float64(p.Transfers))),
		"bwlimit":               tftypes.NewValue(tftypes.String, p.BWLimit),
		"follow_symlinks":       tftypes.NewValue(tftypes.Bool, p.FollowSymlinks),
		"create_empty_src_dirs": tftypes.NewValue(tftypes.Bool, p.CreateEmptySrcDirs),
		"enabled":               tftypes.NewValue(tftypes.Bool, p.Enabled),
		"sync_on_change":        tftypes.NewValue(tftypes.Bool, p.SyncOnChange),
	}

	// Handle exclude list
	if len(p.Exclude) > 0 {
		excludeValues := make([]tftypes.Value, len(p.Exclude))
		for i, e := range p.Exclude {
			excludeValues[i] = tftypes.NewValue(tftypes.String, e)
		}
		values["exclude"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, excludeValues)
	} else {
		values["exclude"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil)
	}

	// Handle include list
	if len(p.Include) > 0 {
		includeValues := make([]tftypes.Value, len(p.Include))
		for i, e := range p.Include {
			includeValues[i] = tftypes.NewValue(tftypes.String, e)
		}
		values["include"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, includeValues)
	} else {
		values["include"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil)
	}

	// Handle schedule block
	if p.Schedule != nil {
		values["schedule"] = tftypes.NewValue(scheduleType, map[string]tftypes.Value{
			"minute": tftypes.NewValue(tftypes.String, p.Schedule.Minute),
			"hour":   tftypes.NewValue(tftypes.String, p.Schedule.Hour),
			"dom":    tftypes.NewValue(tftypes.String, p.Schedule.Dom),
			"month":  tftypes.NewValue(tftypes.String, p.Schedule.Month),
			"dow":    tftypes.NewValue(tftypes.String, p.Schedule.Dow),
		})
	} else {
		values["schedule"] = tftypes.NewValue(scheduleType, nil)
	}

	// Handle encryption block
	if p.Encryption != nil {
		values["encryption"] = tftypes.NewValue(encryptionType, map[string]tftypes.Value{
			"password": tftypes.NewValue(tftypes.String, p.Encryption.Password),
			"salt":     tftypes.NewValue(tftypes.String, p.Encryption.Salt),
		})
	} else {
		values["encryption"] = tftypes.NewValue(encryptionType, nil)
	}

	// Handle S3 block
	if p.S3 != nil {
		values["s3"] = tftypes.NewValue(bucketFolderType, map[string]tftypes.Value{
			"bucket": tftypes.NewValue(tftypes.String, p.S3.Bucket),
			"folder": tftypes.NewValue(tftypes.String, p.S3.Folder),
		})
	} else {
		values["s3"] = tftypes.NewValue(bucketFolderType, nil)
	}

	// Handle B2 block
	if p.B2 != nil {
		values["b2"] = tftypes.NewValue(bucketFolderType, map[string]tftypes.Value{
			"bucket": tftypes.NewValue(tftypes.String, p.B2.Bucket),
			"folder": tftypes.NewValue(tftypes.String, p.B2.Folder),
		})
	} else {
		values["b2"] = tftypes.NewValue(bucketFolderType, nil)
	}

	// Handle GCS block
	if p.GCS != nil {
		values["gcs"] = tftypes.NewValue(bucketFolderType, map[string]tftypes.Value{
			"bucket": tftypes.NewValue(tftypes.String, p.GCS.Bucket),
			"folder": tftypes.NewValue(tftypes.String, p.GCS.Folder),
		})
	} else {
		values["gcs"] = tftypes.NewValue(bucketFolderType, nil)
	}

	// Handle Azure block
	if p.Azure != nil {
		values["azure"] = tftypes.NewValue(containerFolderType, map[string]tftypes.Value{
			"container": tftypes.NewValue(tftypes.String, p.Azure.Container),
			"folder":    tftypes.NewValue(tftypes.String, p.Azure.Folder),
		})
	} else {
		values["azure"] = tftypes.NewValue(containerFolderType, nil)
	}

	// Create object type matching the schema
	objectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                    tftypes.String,
			"description":           tftypes.String,
			"path":                  tftypes.String,
			"credentials":           tftypes.Number,
			"direction":             tftypes.String,
			"transfer_mode":         tftypes.String,
			"snapshot":              tftypes.Bool,
			"transfers":             tftypes.Number,
			"bwlimit":               tftypes.String,
			"exclude":               tftypes.List{ElementType: tftypes.String},
			"include":               tftypes.List{ElementType: tftypes.String},
			"follow_symlinks":       tftypes.Bool,
			"create_empty_src_dirs": tftypes.Bool,
			"enabled":               tftypes.Bool,
			"sync_on_change":        tftypes.Bool,
			"schedule":              scheduleType,
			"encryption":            encryptionType,
			"s3":                    bucketFolderType,
			"b2":                    bucketFolderType,
			"gcs":                   bucketFolderType,
			"azure":                 containerFolderType,
		},
	}

	return tftypes.NewValue(objectType, values)
}

// testCloudSyncTask returns a standard test task for S3.
func testCloudSyncTask(id int64, description string) *truenas.CloudSyncTask {
	return &truenas.CloudSyncTask{
		ID:           id,
		Description:  description,
		Path:         "/mnt/tank/data",
		CredentialID: 5,
		Direction:    "PUSH",
		TransferMode: "SYNC",
		Snapshot:     false,
		Transfers:    4,
		Attributes: map[string]any{
			"bucket": "my-bucket",
			"folder": "/backups/",
		},
		Schedule: truenas.Schedule{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		FollowSymlinks:     false,
		CreateEmptySrcDirs: false,
		Enabled:            true,
	}
}

func TestCloudSyncTaskResource_Create_S3_Success(t *testing.T) {
	var capturedOpts truenas.CreateCloudSyncTaskOpts

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateTaskFunc: func(ctx context.Context, opts truenas.CreateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					capturedOpts = opts
					return testCloudSyncTask(10, "Daily Backup"), nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		Description:  "Daily Backup",
		Path:         "/mnt/tank/data",
		Credentials:  5,
		Direction:    "push",
		TransferMode: "sync",
		Transfers:    4,
		Enabled:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/backups/",
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
	if capturedOpts.Description != "Daily Backup" {
		t.Errorf("expected description 'Daily Backup', got %q", capturedOpts.Description)
	}
	if capturedOpts.Path != "/mnt/tank/data" {
		t.Errorf("expected path '/mnt/tank/data', got %q", capturedOpts.Path)
	}
	if capturedOpts.Direction != "PUSH" {
		t.Errorf("expected direction 'PUSH', got %q", capturedOpts.Direction)
	}
	if capturedOpts.TransferMode != "SYNC" {
		t.Errorf("expected transfer_mode 'SYNC', got %q", capturedOpts.TransferMode)
	}
	if capturedOpts.CredentialID != 5 {
		t.Errorf("expected credential ID 5, got %d", capturedOpts.CredentialID)
	}

	// Verify schedule
	if capturedOpts.Schedule.Minute != "0" {
		t.Errorf("expected schedule minute '0', got %q", capturedOpts.Schedule.Minute)
	}
	if capturedOpts.Schedule.Hour != "3" {
		t.Errorf("expected schedule hour '3', got %q", capturedOpts.Schedule.Hour)
	}

	// Verify attributes (bucket/folder)
	if capturedOpts.Attributes["bucket"] != "my-bucket" {
		t.Errorf("expected attributes bucket 'my-bucket', got %v", capturedOpts.Attributes["bucket"])
	}
	if capturedOpts.Attributes["folder"] != "/backups/" {
		t.Errorf("expected attributes folder '/backups/', got %v", capturedOpts.Attributes["folder"])
	}

	// Verify state was set
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "10" {
		t.Errorf("expected ID '10', got %q", resultData.ID.ValueString())
	}
}

func TestCloudSyncTaskResource_Create_B2_Success(t *testing.T) {
	var capturedOpts truenas.CreateCloudSyncTaskOpts

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateTaskFunc: func(ctx context.Context, opts truenas.CreateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					capturedOpts = opts
					return &truenas.CloudSyncTask{
						ID:           11,
						Description:  "B2 Backup",
						Path:         "/mnt/tank/b2data",
						CredentialID: 6,
						Direction:    "PUSH",
						TransferMode: "COPY",
						Transfers:    8,
						Enabled:      true,
						Attributes:   map[string]any{"bucket": "b2-bucket", "folder": "/b2-backups/"},
						Schedule:     truenas.Schedule{Minute: "30", Hour: "2", Dom: "*", Month: "*", Dow: "*"},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		Description:  "B2 Backup",
		Path:         "/mnt/tank/b2data",
		Credentials:  6,
		Direction:    "push",
		TransferMode: "copy",
		Transfers:    8,
		Enabled:      true,
		Schedule: &scheduleBlockParams{
			Minute: "30",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		B2: &taskB2BlockParams{
			Bucket: "b2-bucket",
			Folder: "/b2-backups/",
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

	// Verify opts
	if capturedOpts.Description != "B2 Backup" {
		t.Errorf("expected description 'B2 Backup', got %q", capturedOpts.Description)
	}
	if capturedOpts.Path != "/mnt/tank/b2data" {
		t.Errorf("expected path '/mnt/tank/b2data', got %q", capturedOpts.Path)
	}
	if capturedOpts.Direction != "PUSH" {
		t.Errorf("expected direction 'PUSH', got %q", capturedOpts.Direction)
	}
	if capturedOpts.TransferMode != "COPY" {
		t.Errorf("expected transfer_mode 'COPY', got %q", capturedOpts.TransferMode)
	}

	// Verify schedule
	if capturedOpts.Schedule.Minute != "30" {
		t.Errorf("expected schedule minute '30', got %q", capturedOpts.Schedule.Minute)
	}
	if capturedOpts.Schedule.Hour != "2" {
		t.Errorf("expected schedule hour '2', got %q", capturedOpts.Schedule.Hour)
	}

	// Verify attributes (bucket/folder for B2)
	if capturedOpts.Attributes["bucket"] != "b2-bucket" {
		t.Errorf("expected attributes bucket 'b2-bucket', got %v", capturedOpts.Attributes["bucket"])
	}
	if capturedOpts.Attributes["folder"] != "/b2-backups/" {
		t.Errorf("expected attributes folder '/b2-backups/', got %v", capturedOpts.Attributes["folder"])
	}

	// Verify state was set
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "11" {
		t.Errorf("expected ID '11', got %q", resultData.ID.ValueString())
	}
}

func TestCloudSyncTaskResource_Create_GCS_Success(t *testing.T) {
	var capturedOpts truenas.CreateCloudSyncTaskOpts

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateTaskFunc: func(ctx context.Context, opts truenas.CreateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					capturedOpts = opts
					return &truenas.CloudSyncTask{
						ID:           12,
						Description:  "GCS Backup",
						Path:         "/mnt/tank/gcsdata",
						CredentialID: 7,
						Direction:    "PULL",
						TransferMode: "MOVE",
						Transfers:    2,
						Enabled:      true,
						Attributes:   map[string]any{"bucket": "gcs-bucket", "folder": "/gcs-backups/"},
						Schedule:     truenas.Schedule{Minute: "15", Hour: "4", Dom: "*", Month: "*", Dow: "*"},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		Description:  "GCS Backup",
		Path:         "/mnt/tank/gcsdata",
		Credentials:  7,
		Direction:    "pull",
		TransferMode: "move",
		Transfers:    2,
		Enabled:      true,
		Schedule: &scheduleBlockParams{
			Minute: "15",
			Hour:   "4",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		GCS: &taskGCSBlockParams{
			Bucket: "gcs-bucket",
			Folder: "/gcs-backups/",
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

	// Verify opts
	if capturedOpts.Description != "GCS Backup" {
		t.Errorf("expected description 'GCS Backup', got %q", capturedOpts.Description)
	}
	if capturedOpts.Path != "/mnt/tank/gcsdata" {
		t.Errorf("expected path '/mnt/tank/gcsdata', got %q", capturedOpts.Path)
	}
	if capturedOpts.Direction != "PULL" {
		t.Errorf("expected direction 'PULL', got %q", capturedOpts.Direction)
	}
	if capturedOpts.TransferMode != "MOVE" {
		t.Errorf("expected transfer_mode 'MOVE', got %q", capturedOpts.TransferMode)
	}

	// Verify schedule
	if capturedOpts.Schedule.Minute != "15" {
		t.Errorf("expected schedule minute '15', got %q", capturedOpts.Schedule.Minute)
	}
	if capturedOpts.Schedule.Hour != "4" {
		t.Errorf("expected schedule hour '4', got %q", capturedOpts.Schedule.Hour)
	}

	// Verify attributes (bucket/folder for GCS)
	if capturedOpts.Attributes["bucket"] != "gcs-bucket" {
		t.Errorf("expected attributes bucket 'gcs-bucket', got %v", capturedOpts.Attributes["bucket"])
	}
	if capturedOpts.Attributes["folder"] != "/gcs-backups/" {
		t.Errorf("expected attributes folder '/gcs-backups/', got %v", capturedOpts.Attributes["folder"])
	}

	// Verify state was set
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "12" {
		t.Errorf("expected ID '12', got %q", resultData.ID.ValueString())
	}
}

func TestCloudSyncTaskResource_Create_Azure_Success(t *testing.T) {
	var capturedOpts truenas.CreateCloudSyncTaskOpts

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateTaskFunc: func(ctx context.Context, opts truenas.CreateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					capturedOpts = opts
					return &truenas.CloudSyncTask{
						ID:           13,
						Description:  "Azure Backup",
						Path:         "/mnt/tank/azuredata",
						CredentialID: 8,
						Direction:    "PUSH",
						TransferMode: "SYNC",
						Snapshot:     true,
						Transfers:    6,
						Enabled:      true,
						Attributes:   map[string]any{"container": "azure-container", "folder": "/azure-backups/"},
						Schedule:     truenas.Schedule{Minute: "45", Hour: "6", Dom: "*", Month: "*", Dow: "*"},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		Description:  "Azure Backup",
		Path:         "/mnt/tank/azuredata",
		Credentials:  8,
		Direction:    "push",
		TransferMode: "sync",
		Transfers:    6,
		Snapshot:     true,
		Enabled:      true,
		Schedule: &scheduleBlockParams{
			Minute: "45",
			Hour:   "6",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		Azure: &taskAzureBlockParams{
			Container: "azure-container",
			Folder:    "/azure-backups/",
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

	// Verify opts
	if capturedOpts.Description != "Azure Backup" {
		t.Errorf("expected description 'Azure Backup', got %q", capturedOpts.Description)
	}
	if capturedOpts.Path != "/mnt/tank/azuredata" {
		t.Errorf("expected path '/mnt/tank/azuredata', got %q", capturedOpts.Path)
	}
	if capturedOpts.Direction != "PUSH" {
		t.Errorf("expected direction 'PUSH', got %q", capturedOpts.Direction)
	}
	if capturedOpts.TransferMode != "SYNC" {
		t.Errorf("expected transfer_mode 'SYNC', got %q", capturedOpts.TransferMode)
	}
	if capturedOpts.Snapshot != true {
		t.Errorf("expected snapshot true, got %v", capturedOpts.Snapshot)
	}

	// Verify schedule
	if capturedOpts.Schedule.Minute != "45" {
		t.Errorf("expected schedule minute '45', got %q", capturedOpts.Schedule.Minute)
	}
	if capturedOpts.Schedule.Hour != "6" {
		t.Errorf("expected schedule hour '6', got %q", capturedOpts.Schedule.Hour)
	}

	// Verify attributes (container/folder for Azure)
	if capturedOpts.Attributes["container"] != "azure-container" {
		t.Errorf("expected attributes container 'azure-container', got %v", capturedOpts.Attributes["container"])
	}
	if capturedOpts.Attributes["folder"] != "/azure-backups/" {
		t.Errorf("expected attributes folder '/azure-backups/', got %v", capturedOpts.Attributes["folder"])
	}

	// Verify state was set
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "13" {
		t.Errorf("expected ID '13', got %q", resultData.ID.ValueString())
	}
}

func TestCloudSyncTaskResource_Read_Success(t *testing.T) {
	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				GetTaskFunc: func(ctx context.Context, id int64) (*truenas.CloudSyncTask, error) {
					return testCloudSyncTask(10, "Daily Backup"), nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	stateValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:          "10",
		Description: "Daily Backup",
		Path:        "/mnt/tank/data",
		Credentials: 5,
		Direction:   "push",
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/backups/",
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

	// Verify state was updated
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.Description.ValueString() != "Daily Backup" {
		t.Errorf("expected description 'Daily Backup', got %q", resultData.Description.ValueString())
	}
}

func TestCloudSyncTaskResource_Read_NotFound(t *testing.T) {
	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				GetTaskFunc: func(ctx context.Context, id int64) (*truenas.CloudSyncTask, error) {
					return nil, nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	stateValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:          "10",
		Description: "Deleted Task",
		Path:        "/mnt/tank/data",
		Credentials: 5,
		Direction:   "push",
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/backups/",
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

	// State should be removed (resource not found)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed when resource not found")
	}
}

func TestCloudSyncTaskResource_Update_Success(t *testing.T) {
	var capturedID int64
	var capturedOpts truenas.UpdateCloudSyncTaskOpts

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				UpdateTaskFunc: func(ctx context.Context, id int64, opts truenas.UpdateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					capturedID = id
					capturedOpts = opts
					return &truenas.CloudSyncTask{
						ID:           10,
						Description:  "Updated Backup",
						Path:         "/mnt/tank/data",
						CredentialID: 5,
						Direction:    "PUSH",
						TransferMode: "SYNC",
						Transfers:    4,
						Enabled:      true,
						Attributes:   map[string]any{"bucket": "my-bucket", "folder": "/new-folder/"},
						Schedule:     truenas.Schedule{Minute: "30", Hour: "4", Dom: "*", Month: "*", Dow: "*"},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)

	// Current state
	stateValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:          "10",
		Description: "Daily Backup",
		Path:        "/mnt/tank/data",
		Credentials: 5,
		Direction:   "push",
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/backups/",
		},
	})

	// Updated plan
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:          "10",
		Description: "Updated Backup",
		Path:        "/mnt/tank/data",
		Credentials: 5,
		Direction:   "push",
		Schedule: &scheduleBlockParams{
			Minute: "30",
			Hour:   "4",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/new-folder/",
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

	if capturedID != 10 {
		t.Errorf("expected ID 10, got %d", capturedID)
	}

	if capturedOpts.Description != "Updated Backup" {
		t.Errorf("expected description 'Updated Backup', got %q", capturedOpts.Description)
	}

	// Verify state was set
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.Description.ValueString() != "Updated Backup" {
		t.Errorf("expected description 'Updated Backup', got %q", resultData.Description.ValueString())
	}
}

func TestCloudSyncTaskResource_Update_SyncOnChange(t *testing.T) {
	var syncCalled bool
	var syncID int64

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				UpdateTaskFunc: func(ctx context.Context, id int64, opts truenas.UpdateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					return &truenas.CloudSyncTask{
						ID:           10,
						Description:  "Updated Backup",
						Path:         "/mnt/tank/data",
						CredentialID: 5,
						Direction:    "PUSH",
						TransferMode: "SYNC",
						Transfers:    4,
						Enabled:      true,
						Attributes:   map[string]any{"bucket": "my-bucket", "folder": "/backups/"},
						Schedule:     truenas.Schedule{Minute: "0", Hour: "3", Dom: "*", Month: "*", Dow: "*"},
					}, nil
				},
				SyncFunc: func(ctx context.Context, id int64) error {
					syncCalled = true
					syncID = id
					return nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)

	stateValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:           "10",
		Description:  "Daily Backup",
		Path:         "/mnt/tank/data",
		Credentials:  5,
		Direction:    "push",
		SyncOnChange: true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/backups/",
		},
	})

	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:           "10",
		Description:  "Updated Backup",
		Path:         "/mnt/tank/data",
		Credentials:  5,
		Direction:    "push",
		SyncOnChange: true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/backups/",
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

	if !syncCalled {
		t.Error("expected Sync to be called when sync_on_change is true")
	}

	if syncID != 10 {
		t.Errorf("expected sync to be called with ID 10, got %d", syncID)
	}
}

func TestCloudSyncTaskResource_Delete_Success(t *testing.T) {
	var capturedID int64

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				DeleteTaskFunc: func(ctx context.Context, id int64) error {
					capturedID = id
					return nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	stateValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:          "10",
		Description: "Daily Backup",
		Path:        "/mnt/tank/data",
		Credentials: 5,
		Direction:   "push",
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/backups/",
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

	if capturedID != 10 {
		t.Errorf("expected ID 10, got %d", capturedID)
	}
}

func TestCloudSyncTaskResource_Create_APIError(t *testing.T) {
	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateTaskFunc: func(ctx context.Context, opts truenas.CreateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					return nil, errors.New("connection refused")
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		Description: "Daily Backup",
		Path:        "/mnt/tank/data",
		Credentials: 5,
		Direction:   "push",
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/backups/",
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

func TestCloudSyncTaskResource_Create_NoProviderBlock(t *testing.T) {
	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		Description: "No Provider",
		Path:        "/mnt/tank/data",
		Credentials: 5,
		Direction:   "push",
		// No S3, B2, GCS, or Azure block
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

func TestCloudSyncTaskResource_Read_APIError(t *testing.T) {
	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				GetTaskFunc: func(ctx context.Context, id int64) (*truenas.CloudSyncTask, error) {
					return nil, errors.New("connection refused")
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	stateValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:          "10",
		Description: "Daily Backup",
		Path:        "/mnt/tank/data",
		Credentials: 5,
		Direction:   "push",
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/backups/",
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

func TestCloudSyncTaskResource_Update_APIError(t *testing.T) {
	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				UpdateTaskFunc: func(ctx context.Context, id int64, opts truenas.UpdateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					return nil, errors.New("connection refused")
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	stateValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:          "10",
		Description: "Daily Backup",
		Path:        "/mnt/tank/data",
		Credentials: 5,
		Direction:   "push",
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/backups/",
		},
	})

	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:          "10",
		Description: "Updated Backup",
		Path:        "/mnt/tank/data",
		Credentials: 5,
		Direction:   "push",
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/new-folder/",
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

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestCloudSyncTaskResource_Delete_APIError(t *testing.T) {
	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				DeleteTaskFunc: func(ctx context.Context, id int64) error {
					return errors.New("task in use by active job")
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	stateValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:          "10",
		Description: "Daily Backup",
		Path:        "/mnt/tank/data",
		Credentials: 5,
		Direction:   "push",
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/backups/",
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

func TestCloudSyncTaskResource_Create_MultipleProviderBlocks(t *testing.T) {
	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		Description: "Multiple Providers",
		Path:        "/mnt/tank/data",
		Credentials: 5,
		Direction:   "push",
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/backups/",
		},
		B2: &taskB2BlockParams{
			Bucket: "another-bucket",
			Folder: "/other/",
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
		t.Fatal("expected error when multiple provider blocks specified")
	}
}

func TestCloudSyncTaskResource_Create_ScheduleWithWildcards(t *testing.T) {
	var capturedOpts truenas.CreateCloudSyncTaskOpts

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateTaskFunc: func(ctx context.Context, opts truenas.CreateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					capturedOpts = opts
					return &truenas.CloudSyncTask{
						ID:           20,
						Description:  "Schedule Wildcards Test",
						Path:         "/mnt/tank/data",
						CredentialID: 5,
						Direction:    "PUSH",
						TransferMode: "SYNC",
						Transfers:    4,
						Enabled:      true,
						Attributes:   map[string]any{"bucket": "test-bucket", "folder": "/"},
						Schedule:     truenas.Schedule{Minute: "0", Hour: "3", Dom: "*", Month: "*", Dow: "*"},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	// Test schedule with explicit wildcard values for dom, month, and dow
	// This verifies wildcards are correctly passed through to the API
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		Description:  "Schedule Wildcards Test",
		Path:         "/mnt/tank/data",
		Credentials:  5,
		Direction:    "push",
		TransferMode: "sync",
		Transfers:    4,
		Enabled:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		S3: &taskS3BlockParams{
			Bucket: "test-bucket",
			Folder: "/",
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

	// Verify schedule opts
	if capturedOpts.Schedule.Minute != "0" {
		t.Errorf("expected schedule minute '0', got %q", capturedOpts.Schedule.Minute)
	}
	if capturedOpts.Schedule.Hour != "3" {
		t.Errorf("expected schedule hour '3', got %q", capturedOpts.Schedule.Hour)
	}

	// Verify wildcard fields are passed correctly
	if capturedOpts.Schedule.Dom != "*" {
		t.Errorf("expected schedule dom '*', got %q", capturedOpts.Schedule.Dom)
	}
	if capturedOpts.Schedule.Month != "*" {
		t.Errorf("expected schedule month '*', got %q", capturedOpts.Schedule.Month)
	}
	if capturedOpts.Schedule.Dow != "*" {
		t.Errorf("expected schedule dow '*', got %q", capturedOpts.Schedule.Dow)
	}

	// Verify state was set correctly
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "20" {
		t.Errorf("expected ID '20', got %q", resultData.ID.ValueString())
	}
	if resultData.Schedule == nil {
		t.Fatal("expected schedule block to be set")
	}
	if resultData.Schedule.Dom.ValueString() != "*" {
		t.Errorf("expected state schedule dom '*', got %q", resultData.Schedule.Dom.ValueString())
	}
}

func TestCloudSyncTaskResource_Create_CustomSchedule(t *testing.T) {
	var capturedOpts truenas.CreateCloudSyncTaskOpts

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateTaskFunc: func(ctx context.Context, opts truenas.CreateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					capturedOpts = opts
					return &truenas.CloudSyncTask{
						ID:           21,
						Description:  "Custom Schedule Test",
						Path:         "/mnt/tank/data",
						CredentialID: 5,
						Direction:    "PUSH",
						TransferMode: "SYNC",
						Transfers:    4,
						Enabled:      true,
						Attributes:   map[string]any{"bucket": "test-bucket", "folder": "/"},
						Schedule:     truenas.Schedule{Minute: "*/5", Hour: "9-17", Dom: "1,15", Month: "1-6", Dow: "1-5"},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	// Custom schedule: every 5 minutes during business hours (9-17), weekdays only (1-5),
	// on 1st and 15th of months Jan-June
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		Description:  "Custom Schedule Test",
		Path:         "/mnt/tank/data",
		Credentials:  5,
		Direction:    "push",
		TransferMode: "sync",
		Transfers:    4,
		Enabled:      true,
		Schedule: &scheduleBlockParams{
			Minute: "*/5",  // Every 5 minutes
			Hour:   "9-17", // Business hours (9am-5pm)
			Dom:    "1,15", // 1st and 15th of month
			Month:  "1-6",  // January through June
			Dow:    "1-5",  // Monday through Friday
		},
		S3: &taskS3BlockParams{
			Bucket: "test-bucket",
			Folder: "/",
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

	// Verify all custom cron expressions are passed correctly
	if capturedOpts.Schedule.Minute != "*/5" {
		t.Errorf("expected schedule minute '*/5', got %q", capturedOpts.Schedule.Minute)
	}
	if capturedOpts.Schedule.Hour != "9-17" {
		t.Errorf("expected schedule hour '9-17', got %q", capturedOpts.Schedule.Hour)
	}
	if capturedOpts.Schedule.Dom != "1,15" {
		t.Errorf("expected schedule dom '1,15', got %q", capturedOpts.Schedule.Dom)
	}
	if capturedOpts.Schedule.Month != "1-6" {
		t.Errorf("expected schedule month '1-6', got %q", capturedOpts.Schedule.Month)
	}
	if capturedOpts.Schedule.Dow != "1-5" {
		t.Errorf("expected schedule dow '1-5', got %q", capturedOpts.Schedule.Dow)
	}

	// Verify state was set correctly
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "21" {
		t.Errorf("expected ID '21', got %q", resultData.ID.ValueString())
	}
	if resultData.Schedule == nil {
		t.Fatal("expected schedule block to be set")
	}
	if resultData.Schedule.Minute.ValueString() != "*/5" {
		t.Errorf("expected state schedule minute '*/5', got %q", resultData.Schedule.Minute.ValueString())
	}
	if resultData.Schedule.Hour.ValueString() != "9-17" {
		t.Errorf("expected state schedule hour '9-17', got %q", resultData.Schedule.Hour.ValueString())
	}
	if resultData.Schedule.Dom.ValueString() != "1,15" {
		t.Errorf("expected state schedule dom '1,15', got %q", resultData.Schedule.Dom.ValueString())
	}
	if resultData.Schedule.Month.ValueString() != "1-6" {
		t.Errorf("expected state schedule month '1-6', got %q", resultData.Schedule.Month.ValueString())
	}
	if resultData.Schedule.Dow.ValueString() != "1-5" {
		t.Errorf("expected state schedule dow '1-5', got %q", resultData.Schedule.Dow.ValueString())
	}
}

func TestCloudSyncTaskResource_Update_ScheduleOnly(t *testing.T) {
	var capturedID int64
	var capturedOpts truenas.UpdateCloudSyncTaskOpts

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				UpdateTaskFunc: func(ctx context.Context, id int64, opts truenas.UpdateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					capturedID = id
					capturedOpts = opts
					return &truenas.CloudSyncTask{
						ID:           22,
						Description:  "Schedule Update Test",
						Path:         "/mnt/tank/data",
						CredentialID: 5,
						Direction:    "PUSH",
						TransferMode: "SYNC",
						Transfers:    4,
						Enabled:      true,
						Attributes:   map[string]any{"bucket": "test-bucket", "folder": "/backups/"},
						Schedule:     truenas.Schedule{Minute: "*/15", Hour: "0", Dom: "*", Month: "*", Dow: "0,6"},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)

	// Current state: daily at 3am
	stateValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:           "22",
		Description:  "Schedule Update Test",
		Path:         "/mnt/tank/data",
		Credentials:  5,
		Direction:    "push",
		TransferMode: "sync",
		Transfers:    4,
		Enabled:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		S3: &taskS3BlockParams{
			Bucket: "test-bucket",
			Folder: "/backups/",
		},
	})

	// Updated plan: every 15 minutes at midnight on weekends only
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:           "22",
		Description:  "Schedule Update Test",
		Path:         "/mnt/tank/data",
		Credentials:  5,
		Direction:    "push",
		TransferMode: "sync",
		Transfers:    4,
		Enabled:      true,
		Schedule: &scheduleBlockParams{
			Minute: "*/15", // Every 15 minutes
			Hour:   "0",    // Midnight
			Dom:    "*",
			Month:  "*",
			Dow:    "0,6", // Saturday and Sunday
		},
		S3: &taskS3BlockParams{
			Bucket: "test-bucket",
			Folder: "/backups/",
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

	if capturedID != 22 {
		t.Errorf("expected ID 22, got %d", capturedID)
	}

	// Verify that description remains unchanged
	if capturedOpts.Description != "Schedule Update Test" {
		t.Errorf("expected description 'Schedule Update Test', got %q", capturedOpts.Description)
	}

	// Verify schedule was updated
	if capturedOpts.Schedule.Minute != "*/15" {
		t.Errorf("expected schedule minute '*/15', got %q", capturedOpts.Schedule.Minute)
	}
	if capturedOpts.Schedule.Hour != "0" {
		t.Errorf("expected schedule hour '0', got %q", capturedOpts.Schedule.Hour)
	}
	if capturedOpts.Schedule.Dow != "0,6" {
		t.Errorf("expected schedule dow '0,6', got %q", capturedOpts.Schedule.Dow)
	}

	// Verify state was updated correctly
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.Schedule == nil {
		t.Fatal("expected schedule block to be set")
	}
	if resultData.Schedule.Minute.ValueString() != "*/15" {
		t.Errorf("expected state schedule minute '*/15', got %q", resultData.Schedule.Minute.ValueString())
	}
	if resultData.Schedule.Hour.ValueString() != "0" {
		t.Errorf("expected state schedule hour '0', got %q", resultData.Schedule.Hour.ValueString())
	}
	if resultData.Schedule.Dow.ValueString() != "0,6" {
		t.Errorf("expected state schedule dow '0,6', got %q", resultData.Schedule.Dow.ValueString())
	}
}

func TestCloudSyncTaskResource_Create_WithEncryption(t *testing.T) {
	var capturedOpts truenas.CreateCloudSyncTaskOpts

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateTaskFunc: func(ctx context.Context, opts truenas.CreateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					capturedOpts = opts
					return &truenas.CloudSyncTask{
						ID:                 30,
						Description:        "Encrypted Backup",
						Path:               "/mnt/tank/secure",
						CredentialID:       5,
						Direction:          "PUSH",
						TransferMode:       "SYNC",
						Transfers:          4,
						Enabled:            true,
						Encryption:         true,
						EncryptionPassword: "my-secret-password",
						EncryptionSalt:     "random-salt-value",
						Attributes:         map[string]any{"bucket": "secure-bucket", "folder": "/encrypted/"},
						Schedule:           truenas.Schedule{Minute: "0", Hour: "2", Dom: "*", Month: "*", Dow: "*"},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		Description:  "Encrypted Backup",
		Path:         "/mnt/tank/secure",
		Credentials:  5,
		Direction:    "push",
		TransferMode: "sync",
		Transfers:    4,
		Enabled:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		Encryption: &encryptionBlockParams{
			Password: "my-secret-password",
			Salt:     "random-salt-value",
		},
		S3: &taskS3BlockParams{
			Bucket: "secure-bucket",
			Folder: "/encrypted/",
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

	// Verify encryption opts are passed to service
	if capturedOpts.Encryption != true {
		t.Errorf("expected encryption true, got %v", capturedOpts.Encryption)
	}
	if capturedOpts.EncryptionPassword != "my-secret-password" {
		t.Errorf("expected encryption_password 'my-secret-password', got %q", capturedOpts.EncryptionPassword)
	}
	if capturedOpts.EncryptionSalt != "random-salt-value" {
		t.Errorf("expected encryption_salt 'random-salt-value', got %q", capturedOpts.EncryptionSalt)
	}

	// Verify state was set
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "30" {
		t.Errorf("expected ID '30', got %q", resultData.ID.ValueString())
	}
	// Encryption block should be preserved from plan
	if resultData.Encryption == nil {
		t.Fatal("expected encryption block to be set in state")
	}
	if resultData.Encryption.Password.ValueString() != "my-secret-password" {
		t.Errorf("expected encryption password 'my-secret-password', got %q", resultData.Encryption.Password.ValueString())
	}
	if resultData.Encryption.Salt.ValueString() != "random-salt-value" {
		t.Errorf("expected encryption salt 'random-salt-value', got %q", resultData.Encryption.Salt.ValueString())
	}
}

func TestCloudSyncTaskResource_Create_WithEncryption_NoSalt(t *testing.T) {
	var capturedOpts truenas.CreateCloudSyncTaskOpts

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateTaskFunc: func(ctx context.Context, opts truenas.CreateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					capturedOpts = opts
					return &truenas.CloudSyncTask{
						ID:                 31,
						Description:        "Encrypted No Salt",
						Path:               "/mnt/tank/secure",
						CredentialID:       5,
						Direction:          "PUSH",
						TransferMode:       "SYNC",
						Transfers:          4,
						Enabled:            true,
						Encryption:         true,
						EncryptionPassword: "password-only",
						Attributes:         map[string]any{"bucket": "secure-bucket", "folder": "/encrypted/"},
						Schedule:           truenas.Schedule{Minute: "0", Hour: "2", Dom: "*", Month: "*", Dow: "*"},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	// Create with encryption but no salt specified
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		Description:  "Encrypted No Salt",
		Path:         "/mnt/tank/secure",
		Credentials:  5,
		Direction:    "push",
		TransferMode: "sync",
		Transfers:    4,
		Enabled:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		Encryption: &encryptionBlockParams{
			Password: "password-only",
			// Salt is nil (not provided)
		},
		S3: &taskS3BlockParams{
			Bucket: "secure-bucket",
			Folder: "/encrypted/",
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

	// Verify encryption is enabled
	if capturedOpts.Encryption != true {
		t.Errorf("expected encryption true, got %v", capturedOpts.Encryption)
	}
	if capturedOpts.EncryptionPassword != "password-only" {
		t.Errorf("expected encryption_password 'password-only', got %q", capturedOpts.EncryptionPassword)
	}
	// Salt should be empty when not provided
	if capturedOpts.EncryptionSalt != "" {
		t.Errorf("expected encryption_salt to be empty when not provided, but got %q", capturedOpts.EncryptionSalt)
	}

	// Verify state
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.Encryption == nil {
		t.Fatal("expected encryption block to be set in state")
	}
	if resultData.Encryption.Password.ValueString() != "password-only" {
		t.Errorf("expected encryption password 'password-only', got %q", resultData.Encryption.Password.ValueString())
	}
}

func TestCloudSyncTaskResource_Update_EnableEncryption(t *testing.T) {
	var capturedID int64
	var capturedOpts truenas.UpdateCloudSyncTaskOpts

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				UpdateTaskFunc: func(ctx context.Context, id int64, opts truenas.UpdateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					capturedID = id
					capturedOpts = opts
					return &truenas.CloudSyncTask{
						ID:                 32,
						Description:        "Now Encrypted",
						Path:               "/mnt/tank/data",
						CredentialID:       5,
						Direction:          "PUSH",
						TransferMode:       "SYNC",
						Transfers:          4,
						Enabled:            true,
						Encryption:         true,
						EncryptionPassword: "new-encryption-pass",
						EncryptionSalt:     "new-salt",
						Attributes:         map[string]any{"bucket": "my-bucket", "folder": "/backups/"},
						Schedule:           truenas.Schedule{Minute: "0", Hour: "3", Dom: "*", Month: "*", Dow: "*"},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)

	// Current state: no encryption
	stateValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:           "32",
		Description:  "Unencrypted Backup",
		Path:         "/mnt/tank/data",
		Credentials:  5,
		Direction:    "push",
		TransferMode: "sync",
		Transfers:    4,
		Enabled:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		// No encryption block
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/backups/",
		},
	})

	// Updated plan: enable encryption
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:           "32",
		Description:  "Now Encrypted",
		Path:         "/mnt/tank/data",
		Credentials:  5,
		Direction:    "push",
		TransferMode: "sync",
		Transfers:    4,
		Enabled:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		Encryption: &encryptionBlockParams{
			Password: "new-encryption-pass",
			Salt:     "new-salt",
		},
		S3: &taskS3BlockParams{
			Bucket: "my-bucket",
			Folder: "/backups/",
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

	if capturedID != 32 {
		t.Errorf("expected ID 32, got %d", capturedID)
	}

	// Verify encryption was added
	if capturedOpts.Encryption != true {
		t.Errorf("expected encryption true, got %v", capturedOpts.Encryption)
	}
	if capturedOpts.EncryptionPassword != "new-encryption-pass" {
		t.Errorf("expected encryption_password 'new-encryption-pass', got %q", capturedOpts.EncryptionPassword)
	}
	if capturedOpts.EncryptionSalt != "new-salt" {
		t.Errorf("expected encryption_salt 'new-salt', got %q", capturedOpts.EncryptionSalt)
	}

	// Verify state was updated
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.Description.ValueString() != "Now Encrypted" {
		t.Errorf("expected description 'Now Encrypted', got %q", resultData.Description.ValueString())
	}
	if resultData.Encryption == nil {
		t.Fatal("expected encryption block to be set in state")
	}
	if resultData.Encryption.Password.ValueString() != "new-encryption-pass" {
		t.Errorf("expected encryption password 'new-encryption-pass', got %q", resultData.Encryption.Password.ValueString())
	}
}

func TestCloudSyncTaskResource_Read_WithEncryption(t *testing.T) {
	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				GetTaskFunc: func(ctx context.Context, id int64) (*truenas.CloudSyncTask, error) {
					return &truenas.CloudSyncTask{
						ID:                 33,
						Description:        "Encrypted Task",
						Path:               "/mnt/tank/encrypted",
						CredentialID:       5,
						Direction:          "PUSH",
						TransferMode:       "SYNC",
						Transfers:          4,
						Enabled:            true,
						Encryption:         true,
						EncryptionPassword: "stored-password",
						EncryptionSalt:     "stored-salt",
						Attributes:         map[string]any{"bucket": "encrypted-bucket", "folder": "/secure/"},
						Schedule:           truenas.Schedule{Minute: "0", Hour: "4", Dom: "*", Month: "*", Dow: "*"},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	// Initial state with encryption block populated
	stateValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		ID:          "33",
		Description: "Encrypted Task",
		Path:        "/mnt/tank/encrypted",
		Credentials: 5,
		Direction:   "push",
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "4",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		Encryption: &encryptionBlockParams{
			Password: "stored-password",
			Salt:     "stored-salt",
		},
		S3: &taskS3BlockParams{
			Bucket: "encrypted-bucket",
			Folder: "/secure/",
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

	// Verify state was updated correctly
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.Description.ValueString() != "Encrypted Task" {
		t.Errorf("expected description 'Encrypted Task', got %q", resultData.Description.ValueString())
	}

	// Verify encryption block is preserved from state
	// (since mapTaskToModel preserves encryption block from plan/state)
	if resultData.Encryption == nil {
		t.Fatal("expected encryption block to be preserved in state")
	}
	if resultData.Encryption.Password.ValueString() != "stored-password" {
		t.Errorf("expected encryption password 'stored-password', got %q", resultData.Encryption.Password.ValueString())
	}
	if resultData.Encryption.Salt.ValueString() != "stored-salt" {
		t.Errorf("expected encryption salt 'stored-salt', got %q", resultData.Encryption.Salt.ValueString())
	}
}

func TestCloudSyncTaskResource_Create_WithIncludePatterns(t *testing.T) {
	var capturedOpts truenas.CreateCloudSyncTaskOpts

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateTaskFunc: func(ctx context.Context, opts truenas.CreateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					capturedOpts = opts
					return &truenas.CloudSyncTask{
						ID:           40,
						Description:  "Include Test",
						Path:         "/mnt/tank/data",
						CredentialID: 5,
						Direction:    "PUSH",
						TransferMode: "SYNC",
						Transfers:    4,
						Enabled:      true,
						Include:      []string{"/photos/**", "/docs/**"},
						Attributes:   map[string]any{"bucket": "test-bucket", "folder": "/"},
						Schedule:     truenas.Schedule{Minute: "0", Hour: "2", Dom: "*", Month: "*", Dow: "*"},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		Description:  "Include Test",
		Path:         "/mnt/tank/data",
		Credentials:  5,
		Direction:    "push",
		TransferMode: "sync",
		Transfers:    4,
		Enabled:      true,
		Include:      []string{"/photos/**", "/docs/**"},
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		S3: &taskS3BlockParams{
			Bucket: "test-bucket",
			Folder: "/",
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

	if len(capturedOpts.Include) != 2 {
		t.Errorf("expected 2 include patterns, got %d", len(capturedOpts.Include))
	}
	if capturedOpts.Include[0] != "/photos/**" {
		t.Errorf("expected first include pattern '/photos/**', got %q", capturedOpts.Include[0])
	}
	if capturedOpts.Include[1] != "/docs/**" {
		t.Errorf("expected second include pattern '/docs/**', got %q", capturedOpts.Include[1])
	}
}

func TestCloudSyncTaskResource_Create_WithIncludeAndExclude(t *testing.T) {
	var capturedOpts truenas.CreateCloudSyncTaskOpts

	r := &CloudSyncTaskResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			CloudSync: &truenas.MockCloudSyncService{
				CreateTaskFunc: func(ctx context.Context, opts truenas.CreateCloudSyncTaskOpts) (*truenas.CloudSyncTask, error) {
					capturedOpts = opts
					return &truenas.CloudSyncTask{
						ID:           41,
						Description:  "Include and Exclude Test",
						Path:         "/mnt/tank/data",
						CredentialID: 5,
						Direction:    "PUSH",
						TransferMode: "SYNC",
						Transfers:    4,
						Enabled:      true,
						Include:      []string{"/photos/**"},
						Exclude:      []string{"*.tmp", "thumbs.db"},
						Attributes:   map[string]any{"bucket": "test-bucket", "folder": "/"},
						Schedule:     truenas.Schedule{Minute: "0", Hour: "2", Dom: "*", Month: "*", Dow: "*"},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCloudSyncTaskResourceSchema(t)
	planValue := createCloudSyncTaskModelValue(cloudSyncTaskModelParams{
		Description:  "Include and Exclude Test",
		Path:         "/mnt/tank/data",
		Credentials:  5,
		Direction:    "push",
		TransferMode: "sync",
		Transfers:    4,
		Enabled:      true,
		Include:      []string{"/photos/**"},
		Exclude:      []string{"*.tmp", "thumbs.db"},
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
		S3: &taskS3BlockParams{
			Bucket: "test-bucket",
			Folder: "/",
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

	// Verify include
	if len(capturedOpts.Include) != 1 || capturedOpts.Include[0] != "/photos/**" {
		t.Errorf("expected include ['/photos/**'], got %v", capturedOpts.Include)
	}

	// Verify exclude
	if len(capturedOpts.Exclude) != 2 {
		t.Errorf("expected 2 exclude patterns, got %d", len(capturedOpts.Exclude))
	}
}
