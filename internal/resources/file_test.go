package resources

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
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
	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*FileResource)
	var _ resource.ResourceWithImportState = r.(*FileResource)
	var _ resource.ResourceWithValidateConfig = r.(*FileResource)
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
	})
}

// Create operation tests

func TestFileResource_Create_WithHostPath(t *testing.T) {
	var writtenPath string
	var writtenContent []byte
	var mkdirPath string

	r := &FileResource{
		client: &client.MockClient{
			MkdirAllFunc: func(ctx context.Context, path string, mode fs.FileMode) error {
				mkdirPath = path
				return nil
			},
			WriteFileFunc: func(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
				writtenPath = path
				writtenContent = content
				return nil
			},
		},
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
		client: &client.MockClient{
			WriteFileFunc: func(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
				writtenPath = path
				return nil
			},
		},
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
		client: &client.MockClient{
			MkdirAllFunc: func(ctx context.Context, path string, mode fs.FileMode) error {
				return nil
			},
			WriteFileFunc: func(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
				return errors.New("permission denied")
			},
		},
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
