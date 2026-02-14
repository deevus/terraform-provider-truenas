package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestNewVMResource(t *testing.T) {
	r := NewVMResource()
	if r == nil {
		t.Fatal("NewVMResource returned nil")
	}

	vmResource, ok := r.(*VMResource)
	if !ok {
		t.Fatalf("expected *VMResource, got %T", r)
	}

	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(vmResource)
	_ = resource.ResourceWithImportState(vmResource)
}

func TestVMResource_Metadata(t *testing.T) {
	r := NewVMResource()
	req := resource.MetadataRequest{ProviderTypeName: "truenas"}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_vm" {
		t.Errorf("expected TypeName 'truenas_vm', got %q", resp.TypeName)
	}
}

func TestVMResource_Schema(t *testing.T) {
	r := NewVMResource()
	ctx := context.Background()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, schemaReq, schemaResp)

	if schemaResp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	attrs := schemaResp.Schema.Attributes

	// Required
	for _, name := range []string{"name", "memory"} {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("expected %q attribute", name)
		}
		if !attr.IsRequired() {
			t.Errorf("expected %q to be required", name)
		}
	}

	// Computed
	for _, name := range []string{"id", "display_available"} {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("expected %q attribute", name)
		}
		if !attr.IsComputed() {
			t.Errorf("expected %q to be computed", name)
		}
	}

	// Optional
	for _, name := range []string{
		"description", "vcpus", "cores", "threads", "autostart", "time",
		"bootloader", "bootloader_ovmf", "cpu_mode", "cpu_model",
		"shutdown_timeout", "state",
	} {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("expected %q attribute", name)
		}
		if !attr.IsOptional() {
			t.Errorf("expected %q to be optional", name)
		}
	}

	// Device blocks
	blocks := schemaResp.Schema.Blocks
	for _, name := range []string{"disk", "raw", "cdrom", "nic", "display", "pci", "usb"} {
		if _, ok := blocks[name]; !ok {
			t.Errorf("expected %q block", name)
		}
	}
}
