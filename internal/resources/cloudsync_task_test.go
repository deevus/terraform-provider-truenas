package resources

import (
	"context"
	"math/big"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
