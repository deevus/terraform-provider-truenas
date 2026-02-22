package resources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"testing"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/truenas-go/client"
	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewFileResource(t *testing.T) {
	r := NewFileResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	// Verify it implements the required interfaces
	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(r.(*FileResource))
	_ = resource.ResourceWithImportState(r.(*FileResource))
	_ = resource.ResourceWithValidateConfig(r.(*FileResource))
}

func TestFileResource_Metadata(t *testing.T) {
	r := NewFileResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_file" {
		t.Errorf("expected TypeName 'truenas_file', got %q", resp.TypeName)
	}
}

func TestFileResource_Schema(t *testing.T) {
	r := NewFileResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify required attributes
	contentAttr, ok := resp.Schema.Attributes["content"]
	if !ok {
		t.Fatal("expected 'content' attribute")
	}
	if !contentAttr.IsRequired() {
		t.Error("expected 'content' to be required")
	}

	// Verify optional attributes
	for _, attr := range []string{"host_path", "relative_path", "path", "mode", "uid", "gid"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Fatalf("expected '%s' attribute", attr)
		}
		if !a.IsOptional() {
			t.Errorf("expected '%s' to be optional", attr)
		}
	}

	// Verify computed attributes
	for _, attr := range []string{"id", "checksum"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Fatalf("expected '%s' attribute", attr)
		}
		if !a.IsComputed() {
			t.Errorf("expected '%s' to be computed", attr)
		}
	}
}

func TestFileResource_ValidateConfig_HostPathAndRelativePath(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Valid: host_path + relative_path
	configValue := createFileResourceModel(nil, "/mnt/storage/apps/myapp", "config/app.conf", nil, "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFileResource_ValidateConfig_StandalonePath(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Valid: standalone path
	configValue := createFileResourceModel(nil, nil, nil, "/mnt/storage/apps/myapp/config.txt", "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFileResource_ValidateConfig_BothHostPathAndPath(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: both host_path and path specified
	configValue := createFileResourceModel(nil, "/mnt/storage/apps/myapp", "config.txt", "/mnt/other/path", "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when both host_path and path are specified")
	}
}

func TestFileResource_ValidateConfig_NeitherHostPathNorPath(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: neither host_path nor path specified
	configValue := createFileResourceModel(nil, nil, nil, nil, "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when neither host_path nor path is specified")
	}
}

func TestFileResource_ValidateConfig_RelativePathWithoutHostPath(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: relative_path without host_path
	configValue := createFileResourceModel(nil, nil, "config.txt", nil, "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when relative_path is specified without host_path")
	}
}

func TestFileResource_ValidateConfig_RelativePathStartsWithSlash(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: relative_path starts with /
	configValue := createFileResourceModel(nil, "/mnt/storage/apps", "/config.txt", nil, "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when relative_path starts with /")
	}
}

func TestFileResource_ValidateConfig_RelativePathContainsDoubleDot(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: relative_path contains ..
	configValue := createFileResourceModel(nil, "/mnt/storage/apps", "../etc/passwd", nil, "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when relative_path contains ..")
	}
}

func TestFileResource_ValidateConfig_PathNotAbsolute(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: path is not absolute
	configValue := createFileResourceModel(nil, nil, nil, "relative/path.txt", "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when path is not absolute")
	}
}

func TestFileResource_ValidateConfig_PathContainsDoubleDot(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: path contains .. (path traversal)
	configValue := createFileResourceModel(nil, nil, nil, "/mnt/storage/../etc/passwd", "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when path contains .. (path traversal)")
	}
}

func TestFileResource_ValidateConfig_HostPathWithoutRelativePath(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Invalid: host_path without relative_path
	configValue := createFileResourceModel(nil, "/mnt/storage/apps", nil, nil, "content", nil, nil, nil, nil)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when host_path is specified without relative_path")
	}
}

// Configure tests

func TestFileResource_Configure_Success(t *testing.T) {
	r := NewFileResource().(*FileResource)

	req := resource.ConfigureRequest{
		ProviderData: &services.TrueNASServices{},
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFileResource_Configure_NilProviderData(t *testing.T) {
	r := NewFileResource().(*FileResource)

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	// Should not error - nil ProviderData is valid during schema validation
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFileResource_Configure_WrongType(t *testing.T) {
	r := NewFileResource().(*FileResource)

	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

// Helper functions

func getFileResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewFileResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	return *schemaResp
}

func createFileResourceModel(id, hostPath, relativePath, path, content, mode, uid, gid, checksum interface{}) tftypes.Value {
	return createFileResourceModelWithForceDestroy(id, hostPath, relativePath, path, content, mode, uid, gid, checksum, nil)
}

func createFileResourceModelWithForceDestroy(id, hostPath, relativePath, path, content, mode, uid, gid, checksum, forceDestroy interface{}) tftypes.Value {
	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"host_path":     tftypes.String,
			"relative_path": tftypes.String,
			"path":          tftypes.String,
			"content":       tftypes.String,
			"mode":          tftypes.String,
			"uid":           tftypes.Number,
			"gid":           tftypes.Number,
			"checksum":      tftypes.String,
			"force_destroy": tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, id),
		"host_path":     tftypes.NewValue(tftypes.String, hostPath),
		"relative_path": tftypes.NewValue(tftypes.String, relativePath),
		"path":          tftypes.NewValue(tftypes.String, path),
		"content":       tftypes.NewValue(tftypes.String, content),
		"mode":          tftypes.NewValue(tftypes.String, mode),
		"uid":           tftypes.NewValue(tftypes.Number, uid),
		"gid":           tftypes.NewValue(tftypes.Number, gid),
		"checksum":      tftypes.NewValue(tftypes.String, checksum),
		"force_destroy": tftypes.NewValue(tftypes.Bool, forceDestroy),
	})
}

// Create operation tests

func TestFileResource_Create_WithHostPath(t *testing.T) {
	var writtenPath string
	var writtenContent []byte
	var mkdirPath string

	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						MkdirAllFunc: func(ctx context.Context, path string, mode fs.FileMode) error {
							mkdirPath = path
							return nil
						},
						WriteFileFunc: func(ctx context.Context, path string, params truenas.WriteFileParams) error {
							writtenPath = path
							writtenContent = params.Content
							return nil
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	planValue := createFileResourceModel(nil, "/mnt/storage/apps/myapp", "config/app.conf", nil, "hello world", "0644", 0, 0, nil)

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

	// Verify mkdir was called for parent directory
	expectedMkdir := "/mnt/storage/apps/myapp/config"
	if mkdirPath != expectedMkdir {
		t.Errorf("expected mkdir path %q, got %q", expectedMkdir, mkdirPath)
	}

	// Verify file was written
	expectedPath := "/mnt/storage/apps/myapp/config/app.conf"
	if writtenPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, writtenPath)
	}

	if string(writtenContent) != "hello world" {
		t.Errorf("expected content 'hello world', got %q", string(writtenContent))
	}

	// Verify state
	var model FileResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.Path.ValueString() != expectedPath {
		t.Errorf("expected state path %q, got %q", expectedPath, model.Path.ValueString())
	}
}

func TestFileResource_Create_WithStandalonePath(t *testing.T) {
	var writtenPath string

	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						WriteFileFunc: func(ctx context.Context, path string, params truenas.WriteFileParams) error {
							writtenPath = path
							return nil
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	planValue := createFileResourceModel(nil, nil, nil, "/mnt/storage/existing/config.txt", "content", "0644", 0, 0, nil)

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

	if writtenPath != "/mnt/storage/existing/config.txt" {
		t.Errorf("expected path '/mnt/storage/existing/config.txt', got %q", writtenPath)
	}
}

func TestFileResource_Create_WriteError(t *testing.T) {
	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						MkdirAllFunc: func(ctx context.Context, path string, mode fs.FileMode) error {
							return nil
						},
						WriteFileFunc: func(ctx context.Context, path string, params truenas.WriteFileParams) error {
							return errors.New("permission denied")
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	planValue := createFileResourceModel(nil, "/mnt/storage/apps", "config.txt", nil, "content", "0644", 0, 0, nil)

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
		t.Fatal("expected error for write failure")
	}
}

// Read operation tests

// Helper to compute checksum in tests
func computeChecksumForTest(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

func TestFileResource_Read_Success(t *testing.T) {
	content := "file content"
	checksum := computeChecksumForTest(content)

	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						FileExistsFunc: func(ctx context.Context, path string) (bool, error) {
							return true, nil
						},
						ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
							return []byte(content), nil
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", content, "0644", 0, 0, checksum)

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

	var model FileResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.Checksum.ValueString() != checksum {
		t.Errorf("expected checksum %q, got %q", checksum, model.Checksum.ValueString())
	}
}

func TestFileResource_Read_FileNotFound(t *testing.T) {
	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						FileExistsFunc: func(ctx context.Context, path string) (bool, error) {
							return false, nil
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "content", "0644", 0, 0, "checksum")

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

	// Should not error, just remove from state
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// State should be null (removed)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed when file not found")
	}
}

func TestFileResource_Read_DriftDetection(t *testing.T) {
	// Remote content is different from state
	remoteContent := "modified content"

	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						FileExistsFunc: func(ctx context.Context, path string) (bool, error) {
							return true, nil
						},
						ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
							return []byte(remoteContent), nil
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	// State has old content/checksum
	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "old content", "0644", 0, 0, "old-checksum")

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

	var model FileResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// Checksum should be updated to match remote
	expectedChecksum := computeChecksumForTest(remoteContent)
	if model.Checksum.ValueString() != expectedChecksum {
		t.Errorf("expected checksum %q, got %q", expectedChecksum, model.Checksum.ValueString())
	}
}

func TestFileResource_Read_FileExistsError(t *testing.T) {
	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						FileExistsFunc: func(ctx context.Context, path string) (bool, error) {
							return false, errors.New("connection failed")
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "content", "0644", 0, 0, "checksum")

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
		t.Fatal("expected error when FileExists fails")
	}
}

func TestFileResource_Read_ReadFileError(t *testing.T) {
	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						FileExistsFunc: func(ctx context.Context, path string) (bool, error) {
							return true, nil
						},
						ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
							return nil, errors.New("read error")
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "content", "0644", 0, 0, "checksum")

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
		t.Fatal("expected error when ReadFile fails")
	}
}

// Update operation tests

func TestFileResource_Update_ContentChange(t *testing.T) {
	var writtenContent []byte

	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						WriteFileFunc: func(ctx context.Context, path string, params truenas.WriteFileParams) error {
							writtenContent = params.Content
							return nil
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	oldChecksum := computeChecksumForTest("old content")
	newChecksum := computeChecksumForTest("new content")

	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "old content", "0644", 0, 0, oldChecksum)
	planValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "new content", "0644", 0, 0, nil)

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

	if string(writtenContent) != "new content" {
		t.Errorf("expected content 'new content', got %q", string(writtenContent))
	}

	var model FileResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "/mnt/storage/test.txt" {
		t.Errorf("expected ID '/mnt/storage/test.txt', got %q", model.ID.ValueString())
	}

	if model.Path.ValueString() != "/mnt/storage/test.txt" {
		t.Errorf("expected path '/mnt/storage/test.txt', got %q", model.Path.ValueString())
	}

	if model.Checksum.ValueString() != newChecksum {
		t.Errorf("expected checksum %q, got %q", newChecksum, model.Checksum.ValueString())
	}
}

func TestFileResource_Update_WriteError(t *testing.T) {
	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						WriteFileFunc: func(ctx context.Context, path string, params truenas.WriteFileParams) error {
							return errors.New("permission denied")
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "old content", "0644", 0, 0, "checksum")
	planValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "new content", "0644", 0, 0, nil)

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
		t.Fatal("expected error for update write failure")
	}
}

// Delete operation tests

func TestFileResource_Delete_Success(t *testing.T) {
	var deletedPath string

	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						DeleteFileFunc: func(ctx context.Context, path string) error {
							deletedPath = path
							return nil
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "content", "0644", 0, 0, "checksum")

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

	if deletedPath != "/mnt/storage/test.txt" {
		t.Errorf("expected path '/mnt/storage/test.txt', got %q", deletedPath)
	}
}

func TestFileResource_Delete_Error(t *testing.T) {
	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						DeleteFileFunc: func(ctx context.Context, path string) error {
							return errors.New("permission denied")
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	stateValue := createFileResourceModel("/mnt/storage/test.txt", nil, nil, "/mnt/storage/test.txt", "content", "0644", 0, 0, "checksum")

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
		t.Fatal("expected error for delete failure")
	}
}

// Bug fix tests

// TestFileResource_ValidateConfig_UnknownHostPath tests that validation passes
// when host_path is unknown (e.g., referencing another resource's output).
// During terraform plan, referenced values are unknown until apply time.
func TestFileResource_ValidateConfig_UnknownHostPath(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Create config with unknown host_path (simulates: host_path = truenas_host_path.foo.id)
	configValue := createFileResourceModelWithUnknown(
		nil,                          // id
		tftypes.UnknownValue,         // host_path - unknown during plan
		"config/app.conf",            // relative_path - known
		nil,                          // path
		"content",                    // content
		nil,                          // mode
		nil,                          // uid
		nil,                          // gid
		nil,                          // checksum
	)

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}
	resp := &resource.ValidateConfigResponse{}

	r.ValidateConfig(context.Background(), req, resp)

	// Validation should pass - unknown values should be allowed through
	// and validated at apply time when values are known
	if resp.Diagnostics.HasError() {
		t.Fatalf("validation should pass with unknown host_path, got errors: %v", resp.Diagnostics)
	}
}

// TestFileResource_Read_UsesIDWhenPathIsNull tests that Read uses the ID
// attribute when path is null (as happens during terraform import).
func TestFileResource_Read_UsesIDWhenPathIsNull(t *testing.T) {
	content := "file content"
	var checkedPath string

	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						FileExistsFunc: func(ctx context.Context, path string) (bool, error) {
							checkedPath = path
							return true, nil
						},
						ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
							return []byte(content), nil
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	// Simulate state after ImportStatePassthroughID: only ID is set, path is null
	// This is what happens during: terraform import truenas_file.example /mnt/storage/test.txt
	stateValue := createFileResourceModel(
		"/mnt/storage/test.txt", // id - set by import
		nil,                     // host_path
		nil,                     // relative_path
		nil,                     // path - NULL during import
		nil,                     // content - null during import
		nil,                     // mode
		nil,                     // uid
		nil,                     // gid
		nil,                     // checksum
	)

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

	// Read should have used the ID to check the file, not an empty string
	expectedPath := "/mnt/storage/test.txt"
	if checkedPath != expectedPath {
		t.Errorf("expected FileExists to be called with %q (from ID), got %q", expectedPath, checkedPath)
	}
}

// Helper that supports tftypes.UnknownValue for creating unknown attribute values
func createFileResourceModelWithUnknown(id, hostPath, relativePath, path, content, mode, uid, gid, checksum interface{}) tftypes.Value {
	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"host_path":     tftypes.String,
			"relative_path": tftypes.String,
			"path":          tftypes.String,
			"content":       tftypes.String,
			"mode":          tftypes.String,
			"uid":           tftypes.Number,
			"gid":           tftypes.Number,
			"checksum":      tftypes.String,
			"force_destroy": tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":            newStringOrUnknown(id),
		"host_path":     newStringOrUnknown(hostPath),
		"relative_path": newStringOrUnknown(relativePath),
		"path":          newStringOrUnknown(path),
		"content":       newStringOrUnknown(content),
		"mode":          newStringOrUnknown(mode),
		"uid":           newNumberOrUnknown(uid),
		"gid":           newNumberOrUnknown(gid),
		"checksum":      newStringOrUnknown(checksum),
		"force_destroy": tftypes.NewValue(tftypes.Bool, nil),
	})
}

func newStringOrUnknown(v interface{}) tftypes.Value {
	if v == tftypes.UnknownValue {
		return tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	}
	return tftypes.NewValue(tftypes.String, v)
}

func newNumberOrUnknown(v interface{}) tftypes.Value {
	if v == tftypes.UnknownValue {
		return tftypes.NewValue(tftypes.Number, tftypes.UnknownValue)
	}
	return tftypes.NewValue(tftypes.Number, v)
}

// TestFileResource_Update_SetsDefaultsForUnknownComputedAttributes tests that Update
// sets default values for mode, uid, and gid when they are unknown in the plan.
// This can happen when these optional+computed attributes are not specified in config
// and Terraform marks them as unknown during planning.
// Bug: Without this fix, Terraform errors with "provider still indicated an unknown
// value for truenas_file.*.mode/uid/gid. All values must be known after apply."
func TestFileResource_ImportState(t *testing.T) {
	r := NewFileResource().(*FileResource)

	schemaResp := getFileResourceSchema(t)

	// Create an initial empty state with the correct schema
	emptyState := createFileResourceModel(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	req := resource.ImportStateRequest{
		ID: "/mnt/storage/imported/file.txt",
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

	// Verify state has id set to the import ID
	var model FileResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.ID.ValueString() != "/mnt/storage/imported/file.txt" {
		t.Errorf("expected ID '/mnt/storage/imported/file.txt', got %q", model.ID.ValueString())
	}
}

func TestFileResource_Create_MkdirError(t *testing.T) {
	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						MkdirAllFunc: func(ctx context.Context, path string, mode fs.FileMode) error {
							return errors.New("permission denied")
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	// Using host_path + relative_path triggers MkdirAll
	planValue := createFileResourceModel(nil, "/mnt/storage/apps/myapp", "config/app.conf", nil, "content", "0644", 0, 0, nil)

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
		t.Fatal("expected error for MkdirAll failure")
	}
}

func TestFileResource_Update_SetsDefaultsForUnknownComputedAttributes(t *testing.T) {
	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						WriteFileFunc: func(ctx context.Context, path string, params truenas.WriteFileParams) error {
							return nil
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	// State has known values from previous Create
	stateValue := createFileResourceModel(
		"/mnt/storage/test.txt",
		nil,
		nil,
		"/mnt/storage/test.txt",
		"old content",
		"0644",
		int64(0),
		int64(0),
		computeChecksumForTest("old content"),
	)

	// Plan has unknown values for mode, uid, gid (they weren't specified in config,
	// so Terraform marks computed attributes as unknown during plan)
	planValue := createFileResourceModelWithUnknown(
		"/mnt/storage/test.txt",
		nil,
		nil,
		"/mnt/storage/test.txt",
		"new content",
		tftypes.UnknownValue, // mode unknown
		tftypes.UnknownValue, // uid unknown
		tftypes.UnknownValue, // gid unknown
		tftypes.UnknownValue, // checksum unknown (will be computed)
	)

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

	var model FileResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// Mode should be set to default "0644", not unknown
	if model.Mode.IsUnknown() {
		t.Error("mode should not be unknown after Update")
	}
	if model.Mode.ValueString() != "0644" {
		t.Errorf("expected mode '0644', got %q", model.Mode.ValueString())
	}

	// UID should be set to default 0, not unknown
	if model.UID.IsUnknown() {
		t.Error("uid should not be unknown after Update")
	}
	if model.UID.ValueInt64() != 0 {
		t.Errorf("expected uid 0, got %d", model.UID.ValueInt64())
	}

	// GID should be set to default 0, not unknown
	if model.GID.IsUnknown() {
		t.Error("gid should not be unknown after Update")
	}
	if model.GID.ValueInt64() != 0 {
		t.Errorf("expected gid 0, got %d", model.GID.ValueInt64())
	}
}

// Test Schema includes force_destroy attribute
func TestFileResource_Schema_ForceDestroy(t *testing.T) {
	r := NewFileResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify force_destroy attribute exists and is optional
	forceDestroyAttr, ok := resp.Schema.Attributes["force_destroy"]
	if !ok {
		t.Fatal("expected 'force_destroy' attribute in schema")
	}
	if !forceDestroyAttr.IsOptional() {
		t.Error("expected 'force_destroy' attribute to be optional")
	}
}

// Test Delete with force_destroy=true uses Chown before DeleteFile
func TestFileResource_Delete_ForceDestroy(t *testing.T) {
	var chownCalled bool
	var chownPath string
	var chownUID, chownGID int
	var deleteFileCalled bool
	var deletedPath string

	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						ChownFunc: func(ctx context.Context, path string, uid, gid int) error {
							chownCalled = true
							chownPath = path
							chownUID = uid
							chownGID = gid
							return nil
						},
						DeleteFileFunc: func(ctx context.Context, path string) error {
							deleteFileCalled = true
							deletedPath = path
							return nil
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	// State with force_destroy = true
	stateValue := createFileResourceModelWithForceDestroy(
		"/mnt/storage/apps/myapp/config.yaml",
		"/mnt/storage/apps/myapp",
		"config.yaml",
		"/mnt/storage/apps/myapp/config.yaml",
		"content",
		"0644",
		1000,
		1000,
		"abc123",
		true,
	)

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

	// Chown should be called when force_destroy is true
	if !chownCalled {
		t.Error("expected Chown to be called when force_destroy is true")
	}

	if chownPath != "/mnt/storage/apps/myapp/config.yaml" {
		t.Errorf("expected chown path '/mnt/storage/apps/myapp/config.yaml', got %q", chownPath)
	}

	// Should change ownership to root (0, 0)
	if chownUID != 0 || chownGID != 0 {
		t.Errorf("expected chown uid=0 gid=0, got uid=%d gid=%d", chownUID, chownGID)
	}

	// DeleteFile should also be called
	if !deleteFileCalled {
		t.Error("expected DeleteFile to be called")
	}

	if deletedPath != "/mnt/storage/apps/myapp/config.yaml" {
		t.Errorf("expected deleted path '/mnt/storage/apps/myapp/config.yaml', got %q", deletedPath)
	}
}

// Test Delete with force_destroy=false does not call Chown
func TestFileResource_Delete_NoForceDestroy(t *testing.T) {
	var chownCalled bool
	var deleteFileCalled bool
	var deletedPath string

	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						ChownFunc: func(ctx context.Context, path string, uid, gid int) error {
							chownCalled = true
							return nil
						},
						DeleteFileFunc: func(ctx context.Context, path string) error {
							deleteFileCalled = true
							deletedPath = path
							return nil
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	// State with force_destroy = false
	stateValue := createFileResourceModelWithForceDestroy(
		"/mnt/storage/apps/myapp/config.yaml",
		"/mnt/storage/apps/myapp",
		"config.yaml",
		"/mnt/storage/apps/myapp/config.yaml",
		"content",
		"0644",
		1000,
		1000,
		"abc123",
		false,
	)

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

	// Chown should NOT be called when force_destroy is false
	if chownCalled {
		t.Error("expected Chown NOT to be called when force_destroy is false")
	}

	// DeleteFile should still be called
	if !deleteFileCalled {
		t.Error("expected DeleteFile to be called")
	}

	if deletedPath != "/mnt/storage/apps/myapp/config.yaml" {
		t.Errorf("expected deleted path '/mnt/storage/apps/myapp/config.yaml', got %q", deletedPath)
	}
}

// Test Delete with force_destroy unset (nil) does not call Chown (default behavior)
func TestFileResource_Delete_ForceDestroyNil(t *testing.T) {
	var chownCalled bool
	var deleteFileCalled bool
	var deletedPath string

	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						ChownFunc: func(ctx context.Context, path string, uid, gid int) error {
							chownCalled = true
							return nil
						},
						DeleteFileFunc: func(ctx context.Context, path string) error {
							deleteFileCalled = true
							deletedPath = path
							return nil
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	// State with force_destroy = nil (not set) - uses the original helper
	stateValue := createFileResourceModel(
		"/mnt/storage/apps/myapp/config.yaml",
		"/mnt/storage/apps/myapp",
		"config.yaml",
		"/mnt/storage/apps/myapp/config.yaml",
		"content",
		"0644",
		1000,
		1000,
		"abc123",
	)

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

	// Chown should NOT be called when force_destroy is nil (default)
	if chownCalled {
		t.Error("expected Chown NOT to be called when force_destroy is nil")
	}

	// DeleteFile should still be called
	if !deleteFileCalled {
		t.Error("expected DeleteFile to be called")
	}

	if deletedPath != "/mnt/storage/apps/myapp/config.yaml" {
		t.Errorf("expected deleted path '/mnt/storage/apps/myapp/config.yaml', got %q", deletedPath)
	}
}

// parseMode tests

func TestParseMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected fs.FileMode
	}{
		{
			name:     "valid octal mode 0644",
			mode:     "0644",
			expected: fs.FileMode(0644),
		},
		{
			name:     "valid octal mode 0755",
			mode:     "0755",
			expected: fs.FileMode(0755),
		},
		{
			name:     "valid octal mode 0600",
			mode:     "0600",
			expected: fs.FileMode(0600),
		},
		{
			name:     "empty mode string defaults to 0644",
			mode:     "",
			expected: fs.FileMode(0644),
		},
		{
			name:     "invalid octal format defaults to 0644",
			mode:     "invalid",
			expected: fs.FileMode(0644),
		},
		{
			name:     "non-octal digits default to 0644",
			mode:     "0999",
			expected: fs.FileMode(0644),
		},
		{
			name:     "decimal number parsed as octal",
			mode:     "644",
			expected: fs.FileMode(0644),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseMode(tc.mode)
			if result != tc.expected {
				t.Errorf("parseMode(%q) = %o, expected %o", tc.mode, result, tc.expected)
			}
		})
	}
}

// Test Delete with force_destroy=true continues even if Chown fails (warning only)
func TestFileResource_Delete_ForceDestroy_ChownFailsContinues(t *testing.T) {
	var deleteFileCalled bool

	r := &FileResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Filesystem: &truenas.MockFilesystemService{
				ClientFunc: func() truenas.FileCaller {
					return &client.MockClient{
						ChownFunc: func(ctx context.Context, path string, uid, gid int) error {
							return errors.New("operation not permitted")
						},
						DeleteFileFunc: func(ctx context.Context, path string) error {
							deleteFileCalled = true
							return nil
						},
					}
				},
			},
		}},
	}

	schemaResp := getFileResourceSchema(t)

	stateValue := createFileResourceModelWithForceDestroy(
		"/mnt/storage/apps/myapp/config.yaml",
		"/mnt/storage/apps/myapp",
		"config.yaml",
		"/mnt/storage/apps/myapp/config.yaml",
		"content",
		"0644",
		1000,
		1000,
		"abc123",
		true,
	)

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

	// Should NOT have error - Chown failure is just a warning, deletion continues
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors (Chown failure should be warning only): %v", resp.Diagnostics)
	}

	// DeleteFile should still be called even if Chown fails
	if !deleteFileCalled {
		t.Error("expected DeleteFile to be called even when Chown fails")
	}
}
