package resources

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
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
	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*CloudSyncTaskResource)
	var _ resource.ResourceWithImportState = r.(*CloudSyncTaskResource)
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

func TestCloudSyncTaskResource_Create_S3_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &CloudSyncTaskResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.create" {
					capturedMethod = method
					capturedParams = params
					return json.RawMessage(`{"id": 10}`), nil
				}
				if method == "cloudsync.query" {
					return json.RawMessage(`[{
						"id": 10,
						"description": "Daily Backup",
						"path": "/mnt/tank/data",
						"credentials": 5,
						"attributes": {"bucket": "my-bucket", "folder": "/backups/"},
						"schedule": {"minute": "0", "hour": "3", "dom": "*", "month": "*", "dow": "*"},
						"direction": "PUSH",
						"transfer_mode": "SYNC",
						"encryption": false,
						"snapshot": false,
						"transfers": 4,
						"follow_symlinks": false,
						"create_empty_src_dirs": false,
						"enabled": true
					}]`), nil
				}
				return nil, nil
			},
		},
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

	if capturedMethod != "cloudsync.create" {
		t.Errorf("expected method 'cloudsync.create', got %q", capturedMethod)
	}

	// Verify params
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["description"] != "Daily Backup" {
		t.Errorf("expected description 'Daily Backup', got %v", params["description"])
	}
	if params["path"] != "/mnt/tank/data" {
		t.Errorf("expected path '/mnt/tank/data', got %v", params["path"])
	}
	if params["direction"] != "PUSH" {
		t.Errorf("expected direction 'PUSH', got %v", params["direction"])
	}
	if params["transfer_mode"] != "SYNC" {
		t.Errorf("expected transfer_mode 'SYNC', got %v", params["transfer_mode"])
	}

	// Verify schedule
	schedule, ok := params["schedule"].(map[string]any)
	if !ok {
		t.Fatalf("expected schedule to be map[string]any, got %T", params["schedule"])
	}
	if schedule["minute"] != "0" {
		t.Errorf("expected schedule minute '0', got %v", schedule["minute"])
	}
	if schedule["hour"] != "3" {
		t.Errorf("expected schedule hour '3', got %v", schedule["hour"])
	}

	// Verify attributes (bucket/folder)
	attributes, ok := params["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected attributes to be map[string]any, got %T", params["attributes"])
	}
	if attributes["bucket"] != "my-bucket" {
		t.Errorf("expected attributes bucket 'my-bucket', got %v", attributes["bucket"])
	}
	if attributes["folder"] != "/backups/" {
		t.Errorf("expected attributes folder '/backups/', got %v", attributes["folder"])
	}

	// Verify state was set
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "10" {
		t.Errorf("expected ID '10', got %q", resultData.ID.ValueString())
	}
}

func TestCloudSyncTaskResource_Create_B2_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &CloudSyncTaskResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.create" {
					capturedMethod = method
					capturedParams = params
					return json.RawMessage(`{"id": 11}`), nil
				}
				if method == "cloudsync.query" {
					return json.RawMessage(`[{
						"id": 11,
						"description": "B2 Backup",
						"path": "/mnt/tank/b2data",
						"credentials": 6,
						"attributes": {"bucket": "b2-bucket", "folder": "/b2-backups/"},
						"schedule": {"minute": "30", "hour": "2", "dom": "*", "month": "*", "dow": "*"},
						"direction": "PUSH",
						"transfer_mode": "COPY",
						"encryption": false,
						"snapshot": false,
						"transfers": 8,
						"follow_symlinks": false,
						"create_empty_src_dirs": false,
						"enabled": true
					}]`), nil
				}
				return nil, nil
			},
		},
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

	if capturedMethod != "cloudsync.create" {
		t.Errorf("expected method 'cloudsync.create', got %q", capturedMethod)
	}

	// Verify params
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["description"] != "B2 Backup" {
		t.Errorf("expected description 'B2 Backup', got %v", params["description"])
	}
	if params["path"] != "/mnt/tank/b2data" {
		t.Errorf("expected path '/mnt/tank/b2data', got %v", params["path"])
	}
	if params["direction"] != "PUSH" {
		t.Errorf("expected direction 'PUSH', got %v", params["direction"])
	}
	if params["transfer_mode"] != "COPY" {
		t.Errorf("expected transfer_mode 'COPY', got %v", params["transfer_mode"])
	}

	// Verify schedule
	schedule, ok := params["schedule"].(map[string]any)
	if !ok {
		t.Fatalf("expected schedule to be map[string]any, got %T", params["schedule"])
	}
	if schedule["minute"] != "30" {
		t.Errorf("expected schedule minute '30', got %v", schedule["minute"])
	}
	if schedule["hour"] != "2" {
		t.Errorf("expected schedule hour '2', got %v", schedule["hour"])
	}

	// Verify attributes (bucket/folder for B2)
	attributes, ok := params["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected attributes to be map[string]any, got %T", params["attributes"])
	}
	if attributes["bucket"] != "b2-bucket" {
		t.Errorf("expected attributes bucket 'b2-bucket', got %v", attributes["bucket"])
	}
	if attributes["folder"] != "/b2-backups/" {
		t.Errorf("expected attributes folder '/b2-backups/', got %v", attributes["folder"])
	}

	// Verify state was set
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "11" {
		t.Errorf("expected ID '11', got %q", resultData.ID.ValueString())
	}
}

func TestCloudSyncTaskResource_Create_GCS_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &CloudSyncTaskResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.create" {
					capturedMethod = method
					capturedParams = params
					return json.RawMessage(`{"id": 12}`), nil
				}
				if method == "cloudsync.query" {
					return json.RawMessage(`[{
						"id": 12,
						"description": "GCS Backup",
						"path": "/mnt/tank/gcsdata",
						"credentials": 7,
						"attributes": {"bucket": "gcs-bucket", "folder": "/gcs-backups/"},
						"schedule": {"minute": "15", "hour": "4", "dom": "*", "month": "*", "dow": "*"},
						"direction": "PULL",
						"transfer_mode": "MOVE",
						"encryption": false,
						"snapshot": false,
						"transfers": 2,
						"follow_symlinks": false,
						"create_empty_src_dirs": false,
						"enabled": true
					}]`), nil
				}
				return nil, nil
			},
		},
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

	if capturedMethod != "cloudsync.create" {
		t.Errorf("expected method 'cloudsync.create', got %q", capturedMethod)
	}

	// Verify params
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["description"] != "GCS Backup" {
		t.Errorf("expected description 'GCS Backup', got %v", params["description"])
	}
	if params["path"] != "/mnt/tank/gcsdata" {
		t.Errorf("expected path '/mnt/tank/gcsdata', got %v", params["path"])
	}
	if params["direction"] != "PULL" {
		t.Errorf("expected direction 'PULL', got %v", params["direction"])
	}
	if params["transfer_mode"] != "MOVE" {
		t.Errorf("expected transfer_mode 'MOVE', got %v", params["transfer_mode"])
	}

	// Verify schedule
	schedule, ok := params["schedule"].(map[string]any)
	if !ok {
		t.Fatalf("expected schedule to be map[string]any, got %T", params["schedule"])
	}
	if schedule["minute"] != "15" {
		t.Errorf("expected schedule minute '15', got %v", schedule["minute"])
	}
	if schedule["hour"] != "4" {
		t.Errorf("expected schedule hour '4', got %v", schedule["hour"])
	}

	// Verify attributes (bucket/folder for GCS)
	attributes, ok := params["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected attributes to be map[string]any, got %T", params["attributes"])
	}
	if attributes["bucket"] != "gcs-bucket" {
		t.Errorf("expected attributes bucket 'gcs-bucket', got %v", attributes["bucket"])
	}
	if attributes["folder"] != "/gcs-backups/" {
		t.Errorf("expected attributes folder '/gcs-backups/', got %v", attributes["folder"])
	}

	// Verify state was set
	var resultData CloudSyncTaskResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "12" {
		t.Errorf("expected ID '12', got %q", resultData.ID.ValueString())
	}
}

func TestCloudSyncTaskResource_Create_Azure_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &CloudSyncTaskResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.create" {
					capturedMethod = method
					capturedParams = params
					return json.RawMessage(`{"id": 13}`), nil
				}
				if method == "cloudsync.query" {
					return json.RawMessage(`[{
						"id": 13,
						"description": "Azure Backup",
						"path": "/mnt/tank/azuredata",
						"credentials": 8,
						"attributes": {"container": "azure-container", "folder": "/azure-backups/"},
						"schedule": {"minute": "45", "hour": "6", "dom": "*", "month": "*", "dow": "*"},
						"direction": "PUSH",
						"transfer_mode": "SYNC",
						"encryption": false,
						"snapshot": true,
						"transfers": 6,
						"follow_symlinks": false,
						"create_empty_src_dirs": false,
						"enabled": true
					}]`), nil
				}
				return nil, nil
			},
		},
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

	if capturedMethod != "cloudsync.create" {
		t.Errorf("expected method 'cloudsync.create', got %q", capturedMethod)
	}

	// Verify params
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["description"] != "Azure Backup" {
		t.Errorf("expected description 'Azure Backup', got %v", params["description"])
	}
	if params["path"] != "/mnt/tank/azuredata" {
		t.Errorf("expected path '/mnt/tank/azuredata', got %v", params["path"])
	}
	if params["direction"] != "PUSH" {
		t.Errorf("expected direction 'PUSH', got %v", params["direction"])
	}
	if params["transfer_mode"] != "SYNC" {
		t.Errorf("expected transfer_mode 'SYNC', got %v", params["transfer_mode"])
	}
	if params["snapshot"] != true {
		t.Errorf("expected snapshot true, got %v", params["snapshot"])
	}

	// Verify schedule
	schedule, ok := params["schedule"].(map[string]any)
	if !ok {
		t.Fatalf("expected schedule to be map[string]any, got %T", params["schedule"])
	}
	if schedule["minute"] != "45" {
		t.Errorf("expected schedule minute '45', got %v", schedule["minute"])
	}
	if schedule["hour"] != "6" {
		t.Errorf("expected schedule hour '6', got %v", schedule["hour"])
	}

	// Verify attributes (container/folder for Azure)
	attributes, ok := params["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected attributes to be map[string]any, got %T", params["attributes"])
	}
	if attributes["container"] != "azure-container" {
		t.Errorf("expected attributes container 'azure-container', got %v", attributes["container"])
	}
	if attributes["folder"] != "/azure-backups/" {
		t.Errorf("expected attributes folder '/azure-backups/', got %v", attributes["folder"])
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
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": 10,
					"description": "Daily Backup",
					"path": "/mnt/tank/data",
					"credentials": 5,
					"attributes": {"bucket": "my-bucket", "folder": "/backups/"},
					"schedule": {"minute": "0", "hour": "3", "dom": "*", "month": "*", "dow": "*"},
					"direction": "PUSH",
					"transfer_mode": "SYNC",
					"encryption": false,
					"snapshot": false,
					"transfers": 4,
					"bwlimit": "",
					"exclude": [],
					"follow_symlinks": false,
					"create_empty_src_dirs": false,
					"enabled": true
				}]`), nil
			},
		},
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
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		},
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
	var capturedMethod string
	var capturedID int64
	var capturedUpdateData map[string]any

	r := &CloudSyncTaskResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.update" {
					capturedMethod = method
					args := params.([]any)
					capturedID = args[0].(int64)
					capturedUpdateData = args[1].(map[string]any)
					return json.RawMessage(`{"id": 10}`), nil
				}
				if method == "cloudsync.query" {
					return json.RawMessage(`[{
						"id": 10,
						"description": "Updated Backup",
						"path": "/mnt/tank/data",
						"credentials": 5,
						"attributes": {"bucket": "my-bucket", "folder": "/new-folder/"},
						"schedule": {"minute": "30", "hour": "4", "dom": "*", "month": "*", "dow": "*"},
						"direction": "PUSH",
						"transfer_mode": "SYNC",
						"encryption": false,
						"snapshot": false,
						"transfers": 4,
						"follow_symlinks": false,
						"create_empty_src_dirs": false,
						"enabled": true
					}]`), nil
				}
				return nil, nil
			},
		},
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

	if capturedMethod != "cloudsync.update" {
		t.Errorf("expected method 'cloudsync.update', got %q", capturedMethod)
	}

	if capturedID != 10 {
		t.Errorf("expected ID 10, got %d", capturedID)
	}

	if capturedUpdateData["description"] != "Updated Backup" {
		t.Errorf("expected description 'Updated Backup', got %v", capturedUpdateData["description"])
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
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cloudsync.update" {
					return json.RawMessage(`{"id": 10}`), nil
				}
				if method == "cloudsync.sync" {
					syncCalled = true
					syncID = params.(int64)
					return json.RawMessage(`1`), nil
				}
				if method == "cloudsync.query" {
					return json.RawMessage(`[{
						"id": 10,
						"description": "Updated Backup",
						"path": "/mnt/tank/data",
						"credentials": 5,
						"attributes": {"bucket": "my-bucket", "folder": "/backups/"},
						"schedule": {"minute": "0", "hour": "3", "dom": "*", "month": "*", "dow": "*"},
						"direction": "PUSH",
						"transfer_mode": "SYNC",
						"encryption": false,
						"snapshot": false,
						"transfers": 4,
						"follow_symlinks": false,
						"create_empty_src_dirs": false,
						"enabled": true
					}]`), nil
				}
				return nil, nil
			},
		},
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
		t.Error("expected cloudsync.sync to be called when sync_on_change is true")
	}

	if syncID != 10 {
		t.Errorf("expected sync to be called with ID 10, got %d", syncID)
	}
}

func TestCloudSyncTaskResource_Delete_Success(t *testing.T) {
	var capturedMethod string
	var capturedID int64

	r := &CloudSyncTaskResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedID = params.(int64)
				return json.RawMessage(`true`), nil
			},
		},
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

	if capturedMethod != "cloudsync.delete" {
		t.Errorf("expected method 'cloudsync.delete', got %q", capturedMethod)
	}

	if capturedID != 10 {
		t.Errorf("expected ID 10, got %d", capturedID)
	}
}

func TestCloudSyncTaskResource_Create_APIError(t *testing.T) {
	r := &CloudSyncTaskResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		},
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
		client: &client.MockClient{},
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
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		},
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
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		},
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
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("task in use by active job")
			},
		},
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
		client: &client.MockClient{},
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
