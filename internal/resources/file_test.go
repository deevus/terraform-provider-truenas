package resources

import (
	"context"
	"testing"

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
