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

// -- Test helpers --

func getVMResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewVMResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("failed to get schema: %v", resp.Diagnostics)
	}
	return *resp
}

// Block type helpers for tftypes.

func vmDiskBlockType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"device_id":           tftypes.Number,
		"path":                tftypes.String,
		"type":                tftypes.String,
		"logical_sectorsize":  tftypes.Number,
		"physical_sectorsize": tftypes.Number,
		"iotype":              tftypes.String,
		"serial":              tftypes.String,
		"order":               tftypes.Number,
	}}
}

func vmRawBlockType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"device_id":           tftypes.Number,
		"path":                tftypes.String,
		"type":                tftypes.String,
		"boot":                tftypes.Bool,
		"exists":              tftypes.Bool,
		"size":                tftypes.Number,
		"logical_sectorsize":  tftypes.Number,
		"physical_sectorsize": tftypes.Number,
		"iotype":              tftypes.String,
		"serial":              tftypes.String,
		"order":               tftypes.Number,
	}}
}

func vmCDROMBlockType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"device_id": tftypes.Number,
		"path":      tftypes.String,
		"order":     tftypes.Number,
	}}
}

func vmNICBlockType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"device_id":              tftypes.Number,
		"type":                   tftypes.String,
		"nic_attach":             tftypes.String,
		"mac":                    tftypes.String,
		"trust_guest_rx_filters": tftypes.Bool,
		"order":                  tftypes.Number,
	}}
}

func vmDisplayBlockType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"device_id":  tftypes.Number,
		"type":       tftypes.String,
		"resolution": tftypes.String,
		"port":       tftypes.Number,
		"web_port":   tftypes.Number,
		"bind":       tftypes.String,
		"wait":       tftypes.Bool,
		"password":   tftypes.String,
		"web":        tftypes.Bool,
		"order":      tftypes.Number,
	}}
}

func vmPCIBlockType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"device_id": tftypes.Number,
		"pptdev":    tftypes.String,
		"order":     tftypes.Number,
	}}
}

func vmUSBBlockType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"device_id":       tftypes.Number,
		"controller_type": tftypes.String,
		"device":          tftypes.String,
		"order":           tftypes.Number,
	}}
}

// vmObjectType returns the full tftypes.Object type for the VM resource model.
func vmObjectType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                tftypes.String,
			"name":              tftypes.String,
			"description":       tftypes.String,
			"vcpus":             tftypes.Number,
			"cores":             tftypes.Number,
			"threads":           tftypes.Number,
			"memory":            tftypes.Number,
			"min_memory":        tftypes.Number,
			"autostart":         tftypes.Bool,
			"time":              tftypes.String,
			"bootloader":        tftypes.String,
			"bootloader_ovmf":   tftypes.String,
			"cpu_mode":          tftypes.String,
			"cpu_model":         tftypes.String,
			"shutdown_timeout":  tftypes.Number,
			"command_line_args": tftypes.String,
			"state":             tftypes.String,
			"display_available": tftypes.Bool,
			"disk":              tftypes.List{ElementType: vmDiskBlockType()},
			"raw":               tftypes.List{ElementType: vmRawBlockType()},
			"cdrom":             tftypes.List{ElementType: vmCDROMBlockType()},
			"nic":               tftypes.List{ElementType: vmNICBlockType()},
			"display":           tftypes.List{ElementType: vmDisplayBlockType()},
			"pci":               tftypes.List{ElementType: vmPCIBlockType()},
			"usb":               tftypes.List{ElementType: vmUSBBlockType()},
		},
	}
}

type vmModelParams struct {
	ID               interface{}
	Name             interface{}
	Description      interface{}
	VCPUs            interface{}
	Cores            interface{}
	Threads          interface{}
	Memory           interface{}
	MinMemory        interface{}
	Autostart        interface{}
	Time             interface{}
	Bootloader       interface{}
	BootloaderOVMF   interface{}
	CPUMode          interface{}
	CPUModel         interface{}
	ShutdownTimeout  interface{}
	CommandLineArgs  interface{}
	State            interface{}
	DisplayAvailable interface{}
	Disks            []vmDiskParams
	NICs             []vmNICParams
	CDROMs           []vmCDROMParams
	Displays         []vmDisplayParams
}

type vmDiskParams struct {
	DeviceID           interface{}
	Path               interface{}
	Type               interface{}
	LogicalSectorSize  interface{}
	PhysicalSectorSize interface{}
	IOType             interface{}
	Serial             interface{}
	Order              interface{}
}

type vmNICParams struct {
	DeviceID            interface{}
	Type                interface{}
	NICAttach           interface{}
	MAC                 interface{}
	TrustGuestRXFilters interface{}
	Order               interface{}
}

type vmCDROMParams struct {
	DeviceID interface{}
	Path     interface{}
	Order    interface{}
}

type vmDisplayParams struct {
	DeviceID   interface{}
	Type       interface{}
	Resolution interface{}
	Port       interface{}
	WebPort    interface{}
	Bind       interface{}
	Wait       interface{}
	Password   interface{}
	Web        interface{}
	Order      interface{}
}

func emptyBlockList(elemType tftypes.Object) tftypes.Value {
	return tftypes.NewValue(tftypes.List{ElementType: elemType}, []tftypes.Value{})
}

func createVMModelValue(p vmModelParams) tftypes.Value {
	// Build disk block values
	var diskValues []tftypes.Value
	for _, d := range p.Disks {
		diskValues = append(diskValues, tftypes.NewValue(vmDiskBlockType(), map[string]tftypes.Value{
			"device_id":           tftypes.NewValue(tftypes.Number, d.DeviceID),
			"path":                tftypes.NewValue(tftypes.String, d.Path),
			"type":                tftypes.NewValue(tftypes.String, d.Type),
			"logical_sectorsize":  tftypes.NewValue(tftypes.Number, d.LogicalSectorSize),
			"physical_sectorsize": tftypes.NewValue(tftypes.Number, d.PhysicalSectorSize),
			"iotype":              tftypes.NewValue(tftypes.String, d.IOType),
			"serial":              tftypes.NewValue(tftypes.String, d.Serial),
			"order":               tftypes.NewValue(tftypes.Number, d.Order),
		}))
	}
	diskList := emptyBlockList(vmDiskBlockType())
	if len(diskValues) > 0 {
		diskList = tftypes.NewValue(tftypes.List{ElementType: vmDiskBlockType()}, diskValues)
	}

	// Build NIC block values
	var nicValues []tftypes.Value
	for _, n := range p.NICs {
		nicValues = append(nicValues, tftypes.NewValue(vmNICBlockType(), map[string]tftypes.Value{
			"device_id":              tftypes.NewValue(tftypes.Number, n.DeviceID),
			"type":                   tftypes.NewValue(tftypes.String, n.Type),
			"nic_attach":             tftypes.NewValue(tftypes.String, n.NICAttach),
			"mac":                    tftypes.NewValue(tftypes.String, n.MAC),
			"trust_guest_rx_filters": tftypes.NewValue(tftypes.Bool, n.TrustGuestRXFilters),
			"order":                  tftypes.NewValue(tftypes.Number, n.Order),
		}))
	}
	nicList := emptyBlockList(vmNICBlockType())
	if len(nicValues) > 0 {
		nicList = tftypes.NewValue(tftypes.List{ElementType: vmNICBlockType()}, nicValues)
	}

	// Build CDROM block values
	var cdromValues []tftypes.Value
	for _, c := range p.CDROMs {
		cdromValues = append(cdromValues, tftypes.NewValue(vmCDROMBlockType(), map[string]tftypes.Value{
			"device_id": tftypes.NewValue(tftypes.Number, c.DeviceID),
			"path":      tftypes.NewValue(tftypes.String, c.Path),
			"order":     tftypes.NewValue(tftypes.Number, c.Order),
		}))
	}
	cdromList := emptyBlockList(vmCDROMBlockType())
	if len(cdromValues) > 0 {
		cdromList = tftypes.NewValue(tftypes.List{ElementType: vmCDROMBlockType()}, cdromValues)
	}

	// Build Display block values
	var displayValues []tftypes.Value
	for _, d := range p.Displays {
		displayValues = append(displayValues, tftypes.NewValue(vmDisplayBlockType(), map[string]tftypes.Value{
			"device_id":  tftypes.NewValue(tftypes.Number, d.DeviceID),
			"type":       tftypes.NewValue(tftypes.String, d.Type),
			"resolution": tftypes.NewValue(tftypes.String, d.Resolution),
			"port":       tftypes.NewValue(tftypes.Number, d.Port),
			"web_port":   tftypes.NewValue(tftypes.Number, d.WebPort),
			"bind":       tftypes.NewValue(tftypes.String, d.Bind),
			"wait":       tftypes.NewValue(tftypes.Bool, d.Wait),
			"password":   tftypes.NewValue(tftypes.String, d.Password),
			"web":        tftypes.NewValue(tftypes.Bool, d.Web),
			"order":      tftypes.NewValue(tftypes.Number, d.Order),
		}))
	}
	displayList := emptyBlockList(vmDisplayBlockType())
	if len(displayValues) > 0 {
		displayList = tftypes.NewValue(tftypes.List{ElementType: vmDisplayBlockType()}, displayValues)
	}

	values := map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, p.ID),
		"name":              tftypes.NewValue(tftypes.String, p.Name),
		"description":       tftypes.NewValue(tftypes.String, p.Description),
		"vcpus":             tftypes.NewValue(tftypes.Number, p.VCPUs),
		"cores":             tftypes.NewValue(tftypes.Number, p.Cores),
		"threads":           tftypes.NewValue(tftypes.Number, p.Threads),
		"memory":            tftypes.NewValue(tftypes.Number, p.Memory),
		"min_memory":        tftypes.NewValue(tftypes.Number, p.MinMemory),
		"autostart":         tftypes.NewValue(tftypes.Bool, p.Autostart),
		"time":              tftypes.NewValue(tftypes.String, p.Time),
		"bootloader":        tftypes.NewValue(tftypes.String, p.Bootloader),
		"bootloader_ovmf":   tftypes.NewValue(tftypes.String, p.BootloaderOVMF),
		"cpu_mode":          tftypes.NewValue(tftypes.String, p.CPUMode),
		"cpu_model":         tftypes.NewValue(tftypes.String, p.CPUModel),
		"shutdown_timeout":  tftypes.NewValue(tftypes.Number, p.ShutdownTimeout),
		"command_line_args": tftypes.NewValue(tftypes.String, p.CommandLineArgs),
		"state":             tftypes.NewValue(tftypes.String, p.State),
		"display_available": tftypes.NewValue(tftypes.Bool, p.DisplayAvailable),
		"disk":              diskList,
		"raw":               emptyBlockList(vmRawBlockType()),
		"cdrom":             cdromList,
		"nic":               nicList,
		"display":           displayList,
		"pci":               emptyBlockList(vmPCIBlockType()),
		"usb":               emptyBlockList(vmUSBBlockType()),
	}

	return tftypes.NewValue(vmObjectType(), values)
}

// mockVM generates a valid *truenas.VM for testing.
func mockVM(id int64, name string, memory int64, state string) *truenas.VM {
	return &truenas.VM{
		ID:              id,
		Name:            name,
		Description:     "",
		VCPUs:           1,
		Cores:           1,
		Threads:         1,
		Memory:          memory,
		MinMemory:       nil,
		Autostart:       true,
		Time:            "LOCAL",
		Bootloader:      "UEFI",
		BootloaderOVMF:  "OVMF_CODE.fd",
		CPUMode:         "CUSTOM",
		CPUModel:        "",
		ShutdownTimeout: 90,
		CommandLineArgs: "",
		State:           state,
	}
}

// defaultVMPlanParams returns params for a minimal VM plan.
func defaultVMPlanParams() vmModelParams {
	return vmModelParams{
		Name:            "test-vm",
		Description:     "",
		VCPUs:           float64(1),
		Cores:           float64(1),
		Threads:         float64(1),
		Memory:          float64(2048),
		MinMemory:       nil,
		Autostart:       true,
		Time:            "LOCAL",
		Bootloader:      "UEFI",
		BootloaderOVMF:  "OVMF_CODE.fd",
		CPUMode:         "CUSTOM",
		CPUModel:        nil,
		ShutdownTimeout: float64(90),
		CommandLineArgs: "",
		State:           "STOPPED",
		DisplayAvailable: nil,
	}
}

// -- Scaffold tests --

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

	for _, name := range []string{"name", "memory"} {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("expected %q attribute", name)
		}
		if !attr.IsRequired() {
			t.Errorf("expected %q to be required", name)
		}
	}

	for _, name := range []string{"id", "display_available"} {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("expected %q attribute", name)
		}
		if !attr.IsComputed() {
			t.Errorf("expected %q to be computed", name)
		}
	}

	for _, name := range []string{
		"description", "vcpus", "cores", "threads", "autostart", "time",
		"bootloader", "bootloader_ovmf", "cpu_mode", "cpu_model",
		"shutdown_timeout", "command_line_args", "state",
	} {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("expected %q attribute", name)
		}
		if !attr.IsOptional() {
			t.Errorf("expected %q to be optional", name)
		}
	}

	blocks := schemaResp.Schema.Blocks
	for _, name := range []string{"disk", "raw", "cdrom", "nic", "display", "pci", "usb"} {
		if _, ok := blocks[name]; !ok {
			t.Errorf("expected %q block", name)
		}
	}
}

// -- buildCreateOpts tests --

func TestVMResource_buildCreateOpts(t *testing.T) {
	r := &VMResource{}

	t.Run("minimal", func(t *testing.T) {
		data := &VMResourceModel{
			Name:   types.StringValue("test-vm"),
			Memory: types.Int64Value(2048),
		}
		opts := r.buildCreateOpts(data)
		if opts.Name != "test-vm" {
			t.Errorf("expected name 'test-vm', got %v", opts.Name)
		}
		if opts.Memory != 2048 {
			t.Errorf("expected memory 2048, got %v", opts.Memory)
		}
	})

	t.Run("with optional fields", func(t *testing.T) {
		data := &VMResourceModel{
			Name:            types.StringValue("test-vm"),
			Memory:          types.Int64Value(4096),
			Description:     types.StringValue("A test VM"),
			VCPUs:           types.Int64Value(2),
			Cores:           types.Int64Value(2),
			Threads:         types.Int64Value(2),
			Autostart:       types.BoolValue(false),
			Time:            types.StringValue("UTC"),
			Bootloader:      types.StringValue("UEFI"),
			CPUMode:         types.StringValue("HOST-PASSTHROUGH"),
			CommandLineArgs: types.StringValue("-cpu host"),
		}
		opts := r.buildCreateOpts(data)
		if opts.Description != "A test VM" {
			t.Errorf("expected description, got %v", opts.Description)
		}
		if opts.VCPUs != 2 {
			t.Errorf("expected vcpus 2, got %v", opts.VCPUs)
		}
		if opts.Autostart != false {
			t.Errorf("expected autostart false, got %v", opts.Autostart)
		}
		if opts.Time != "UTC" {
			t.Errorf("expected time UTC, got %v", opts.Time)
		}
		if opts.CPUMode != "HOST-PASSTHROUGH" {
			t.Errorf("expected cpu_mode HOST-PASSTHROUGH, got %v", opts.CPUMode)
		}
		if opts.CommandLineArgs != "-cpu host" {
			t.Errorf("expected command_line_args '-cpu host', got %v", opts.CommandLineArgs)
		}
	})

	t.Run("null optional fields omitted", func(t *testing.T) {
		data := &VMResourceModel{
			Name:      types.StringValue("test-vm"),
			Memory:    types.Int64Value(2048),
			CPUModel:  types.StringNull(),
			MinMemory: types.Int64Null(),
		}
		opts := r.buildCreateOpts(data)
		if opts.CPUModel != "" {
			t.Errorf("null cpu_model should be empty, got %v", opts.CPUModel)
		}
		if opts.MinMemory != nil {
			t.Errorf("null min_memory should be nil, got %v", opts.MinMemory)
		}
	})
}

// -- Create tests --

func TestVMResource_Create_Success(t *testing.T) {
	var capturedOpts truenas.CreateVMOpts

	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				capturedOpts = opts
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return nil, nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	planValue := createVMModelValue(defaultVMPlanParams())
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if capturedOpts.Name != "test-vm" {
		t.Errorf("expected name 'test-vm', got %v", capturedOpts.Name)
	}
	if capturedOpts.Memory != 2048 {
		t.Errorf("expected memory 2048, got %v", capturedOpts.Memory)
	}

	var model VMResourceModel
	resp.State.Get(context.Background(), &model)
	if model.ID.ValueString() != "1" {
		t.Errorf("expected ID '1', got %q", model.ID.ValueString())
	}
	if model.State.ValueString() != "STOPPED" {
		t.Errorf("expected state STOPPED, got %q", model.State.ValueString())
	}
}

func TestVMResource_Create_WithStateRunning(t *testing.T) {
	startCalled := false

	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "RUNNING"), nil
			},
			StartVMFunc: func(ctx context.Context, id int64) error {
				startCalled = true
				return nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return nil, nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.State = "RUNNING"
	planValue := createVMModelValue(p)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify vm.start was called
	if !startCalled {
		t.Error("expected vm.start to be called")
	}
}

func TestVMResource_Create_APIError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return nil, errors.New("vm already exists")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	planValue := createVMModelValue(defaultVMPlanParams())
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestVMResource_Create_WithDevices(t *testing.T) {
	var deviceCreateCalls []truenas.CreateVMDeviceOpts

	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
				deviceCreateCalls = append(deviceCreateCalls, opts)
				return &truenas.VMDevice{ID: int64(len(deviceCreateCalls) + 100)}, nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return []truenas.VMDevice{
					{ID: 101, VM: 1, Order: 1000, DeviceType: truenas.DeviceTypeDisk,
						Disk: &truenas.DiskDevice{Path: "/dev/zvol/tank/vms/disk0", Type: "VIRTIO"}},
					{ID: 102, VM: 1, Order: 1001, DeviceType: truenas.DeviceTypeNIC,
						NIC: &truenas.NICDevice{Type: "VIRTIO", NICAttach: "br0", MAC: "00:11:22:33:44:55"}},
				}, nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.Disks = []vmDiskParams{{
		DeviceID: nil, Path: "/dev/zvol/tank/vms/disk0", Type: "VIRTIO",
		LogicalSectorSize: nil, PhysicalSectorSize: nil, IOType: "THREADS",
		Serial: nil, Order: nil,
	}}
	p.NICs = []vmNICParams{{
		DeviceID: nil, Type: "VIRTIO", NICAttach: "br0", MAC: nil,
		TrustGuestRXFilters: false, Order: nil,
	}}
	planValue := createVMModelValue(p)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if len(deviceCreateCalls) != 2 {
		t.Fatalf("expected 2 device create calls, got %d", len(deviceCreateCalls))
	}
}

// -- Read tests --

func TestVMResource_Read_Success(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return nil, nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "1"
	stateValue := createVMModelValue(p)
	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model VMResourceModel
	resp.State.Get(context.Background(), &model)
	if model.ID.ValueString() != "1" {
		t.Errorf("expected ID '1', got %q", model.ID.ValueString())
	}
	if model.Name.ValueString() != "test-vm" {
		t.Errorf("expected name 'test-vm', got %q", model.Name.ValueString())
	}
}

func TestVMResource_Read_NotFound(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return nil, errors.New("does not exist")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "999"
	stateValue := createVMModelValue(p)
	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// State should be removed (resource deleted externally)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed")
	}
}

func TestVMResource_Read_WithDevices(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "RUNNING"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return []truenas.VMDevice{
					{ID: 10, VM: 1, Order: 1000, DeviceType: truenas.DeviceTypeDisk,
						Disk: &truenas.DiskDevice{Path: "/dev/zvol/tank/vms/disk0", Type: "VIRTIO"}},
					{ID: 11, VM: 1, Order: 1001, DeviceType: truenas.DeviceTypeNIC,
						NIC: &truenas.NICDevice{Type: "VIRTIO", NICAttach: "br0", MAC: "00:aa:bb:cc:dd:ee"}},
				}, nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "1"
	p.State = "RUNNING"
	p.Disks = []vmDiskParams{{
		DeviceID: float64(10), Path: "/dev/zvol/tank/vms/disk0", Type: "VIRTIO",
		LogicalSectorSize: nil, PhysicalSectorSize: nil, IOType: "THREADS",
		Serial: nil, Order: float64(1000),
	}}
	p.NICs = []vmNICParams{{
		DeviceID: float64(11), Type: "VIRTIO", NICAttach: "br0", MAC: "00:aa:bb:cc:dd:ee",
		TrustGuestRXFilters: false, Order: float64(1001),
	}}
	stateValue := createVMModelValue(p)
	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model VMResourceModel
	resp.State.Get(context.Background(), &model)
	if len(model.Disks) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(model.Disks))
	}
	if model.Disks[0].Path.ValueString() != "/dev/zvol/tank/vms/disk0" {
		t.Errorf("expected disk path '/dev/zvol/tank/vms/disk0', got %q", model.Disks[0].Path.ValueString())
	}
	if len(model.NICs) != 1 {
		t.Fatalf("expected 1 NIC, got %d", len(model.NICs))
	}
}

// -- Update tests --

func TestVMResource_Update_TopLevel(t *testing.T) {
	var capturedUpdateOpts truenas.UpdateVMOpts
	var updateCalled bool

	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			UpdateVMFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMOpts) (*truenas.VM, error) {
				capturedUpdateOpts = opts
				updateCalled = true
				return mockVM(1, "test-vm", 4096, "STOPPED"), nil
			},
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 4096, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return nil, nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)

	stateParams := defaultVMPlanParams()
	stateParams.ID = "1"
	stateParams.Memory = float64(2048)
	stateValue := createVMModelValue(stateParams)

	planParams := defaultVMPlanParams()
	planParams.ID = "1"
	planParams.Memory = float64(4096)
	planParams.Description = "Updated"
	planValue := createVMModelValue(planParams)

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !updateCalled {
		t.Fatal("expected vm.update to be called")
	}
	if capturedUpdateOpts.Memory != 4096 {
		t.Errorf("expected memory 4096, got %v", capturedUpdateOpts.Memory)
	}
}

// -- Delete tests --

func TestVMResource_Delete_Stopped(t *testing.T) {
	var methods []string

	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			DeleteVMFunc: func(ctx context.Context, id int64) error {
				return nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "1"
	stateValue := createVMModelValue(p)
	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Should not call vm.stop
	for _, m := range methods {
		if m == "vm.stop" {
			t.Error("should not call vm.stop for stopped VM")
		}
	}
}

func TestVMResource_Delete_Running(t *testing.T) {
	var methods []string

	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "RUNNING"), nil
			},
			DeleteVMFunc: func(ctx context.Context, id int64) error {
				return nil
			},
			StopVMFunc: func(ctx context.Context, id int64, opts truenas.StopVMOpts) error {
				methods = append(methods, "vm.stop")
				return nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "1"
	p.State = "RUNNING"
	stateValue := createVMModelValue(p)
	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Should call vm.stop then vm.delete
	foundStop := false
	for _, m := range methods {
		if m == "vm.stop" {
			foundStop = true
		}
	}
	if !foundStop {
		t.Errorf("expected vm.stop to be called for running VM, got: %v", methods)
	}
}

// -- ImportState tests --

func TestVMResource_ImportState(t *testing.T) {
	r := NewVMResource().(*VMResource)

	schemaResp := getVMResourceSchema(t)
	emptyState := createVMModelValue(defaultVMPlanParams())

	req := resource.ImportStateRequest{ID: "42"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: emptyState},
	}

	r.ImportState(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model VMResourceModel
	resp.State.Get(context.Background(), &model)
	if model.ID.ValueString() != "42" {
		t.Errorf("expected ID '42', got %q", model.ID.ValueString())
	}
}

// -- Device mapping tests --

func TestVMResource_mapDevicesToModel(t *testing.T) {
	r := &VMResource{}

	size := int64(10737418240)
	devices := []truenas.VMDevice{
		{ID: 10, VM: 1, Order: 1000, DeviceType: truenas.DeviceTypeDisk,
			Disk: &truenas.DiskDevice{Path: "/dev/zvol/tank/vms/disk0", Type: "VIRTIO"}},
		{ID: 11, VM: 1, Order: 1001, DeviceType: truenas.DeviceTypeRaw,
			Raw: &truenas.RawDevice{Path: "/mnt/tank/vms/raw.img", Type: "AHCI", Boot: true, Size: &size}},
		{ID: 12, VM: 1, Order: 1002, DeviceType: truenas.DeviceTypeCDROM,
			CDROM: &truenas.CDROMDevice{Path: "/mnt/tank/iso/ubuntu.iso"}},
		{ID: 13, VM: 1, Order: 1003, DeviceType: truenas.DeviceTypeNIC,
			NIC: &truenas.NICDevice{Type: "VIRTIO", NICAttach: "br0", MAC: "00:aa:bb:cc:dd:ee"}},
		{ID: 14, VM: 1, Order: 1004, DeviceType: truenas.DeviceTypeDisplay,
			Display: &truenas.DisplayDevice{Type: "SPICE", Resolution: "1920x1080", Port: 5900, Bind: "0.0.0.0", Web: true}},
		{ID: 15, VM: 1, Order: 1005, DeviceType: truenas.DeviceTypePCI,
			PCI: &truenas.PCIDevice{PPTDev: "0000:01:00.0"}},
		{ID: 16, VM: 1, Order: 1006, DeviceType: truenas.DeviceTypeUSB,
			USB: &truenas.USBDevice{ControllerType: "qemu-xhci", Device: "usb_0001"}},
	}

	data := &VMResourceModel{}
	r.mapDevicesToModel(devices, data)

	if len(data.Disks) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(data.Disks))
	}
	if data.Disks[0].Path.ValueString() != "/dev/zvol/tank/vms/disk0" {
		t.Errorf("disk path: got %q", data.Disks[0].Path.ValueString())
	}
	if data.Disks[0].DeviceID.ValueInt64() != 10 {
		t.Errorf("disk device_id: got %d", data.Disks[0].DeviceID.ValueInt64())
	}

	if len(data.Raws) != 1 {
		t.Fatalf("expected 1 raw, got %d", len(data.Raws))
	}
	if data.Raws[0].Path.ValueString() != "/mnt/tank/vms/raw.img" {
		t.Errorf("raw path: got %q", data.Raws[0].Path.ValueString())
	}

	if len(data.CDROMs) != 1 {
		t.Fatalf("expected 1 cdrom, got %d", len(data.CDROMs))
	}
	if data.CDROMs[0].Path.ValueString() != "/mnt/tank/iso/ubuntu.iso" {
		t.Errorf("cdrom path: got %q", data.CDROMs[0].Path.ValueString())
	}

	if len(data.NICs) != 1 {
		t.Fatalf("expected 1 nic, got %d", len(data.NICs))
	}
	if data.NICs[0].MAC.ValueString() != "00:aa:bb:cc:dd:ee" {
		t.Errorf("nic mac: got %q", data.NICs[0].MAC.ValueString())
	}

	if len(data.Displays) != 1 {
		t.Fatalf("expected 1 display, got %d", len(data.Displays))
	}
	if data.Displays[0].Resolution.ValueString() != "1920x1080" {
		t.Errorf("display resolution: got %q", data.Displays[0].Resolution.ValueString())
	}

	if len(data.PCIs) != 1 {
		t.Fatalf("expected 1 pci, got %d", len(data.PCIs))
	}
	if data.PCIs[0].PPTDev.ValueString() != "0000:01:00.0" {
		t.Errorf("pci pptdev: got %q", data.PCIs[0].PPTDev.ValueString())
	}

	if len(data.USBs) != 1 {
		t.Fatalf("expected 1 usb, got %d", len(data.USBs))
	}
	if data.USBs[0].ControllerType.ValueString() != "qemu-xhci" {
		t.Errorf("usb controller_type: got %q", data.USBs[0].ControllerType.ValueString())
	}
}

// -- buildDeviceOpts tests --

func TestVMResource_buildDiskDeviceOpts(t *testing.T) {
	disk := &VMDiskModel{
		Path:   types.StringValue("/dev/zvol/tank/vms/disk0"),
		Type:   types.StringValue("VIRTIO"),
		IOType: types.StringValue("THREADS"),
	}
	opts := buildDiskDeviceOpts(disk, 1)
	if opts.VM != 1 {
		t.Errorf("expected vm=1, got %v", opts.VM)
	}
	if opts.DeviceType != truenas.DeviceTypeDisk {
		t.Errorf("expected dtype=DISK, got %v", opts.DeviceType)
	}
	if opts.Disk == nil || opts.Disk.Path != "/dev/zvol/tank/vms/disk0" {
		t.Errorf("expected path /dev/zvol/tank/vms/disk0, got %v", opts.Disk)
	}
}

func TestVMResource_buildNICDeviceOpts(t *testing.T) {
	nic := &VMNICModel{
		Type:      types.StringValue("VIRTIO"),
		NICAttach: types.StringValue("br0"),
	}
	opts := buildNICDeviceOpts(nic, 1)
	if opts.DeviceType != truenas.DeviceTypeNIC {
		t.Errorf("expected dtype=NIC, got %v", opts.DeviceType)
	}
	if opts.NIC == nil || opts.NIC.Type != "VIRTIO" {
		t.Errorf("expected type=VIRTIO, got %v", opts.NIC)
	}
}

func TestVMResource_buildCDROMDeviceOpts(t *testing.T) {
	cdrom := &VMCDROMModel{
		Path: types.StringValue("/mnt/tank/iso/ubuntu.iso"),
	}
	opts := buildCDROMDeviceOpts(cdrom, 1)
	if opts.DeviceType != truenas.DeviceTypeCDROM {
		t.Errorf("expected dtype=CDROM, got %v", opts.DeviceType)
	}
	if opts.CDROM == nil || opts.CDROM.Path != "/mnt/tank/iso/ubuntu.iso" {
		t.Errorf("expected path /mnt/tank/iso/ubuntu.iso, got %v", opts.CDROM)
	}
}

func TestVMResource_buildDisplayDeviceOpts(t *testing.T) {
	display := &VMDisplayModel{
		Type:       types.StringValue("SPICE"),
		Resolution: types.StringValue("1920x1080"),
		Bind:       types.StringValue("0.0.0.0"),
		Web:        types.BoolValue(true),
	}
	opts := buildDisplayDeviceOpts(display, 1)
	if opts.DeviceType != truenas.DeviceTypeDisplay {
		t.Errorf("expected dtype=DISPLAY, got %v", opts.DeviceType)
	}
	if opts.Display == nil || opts.Display.Resolution != "1920x1080" {
		t.Errorf("expected resolution=1920x1080, got %v", opts.Display)
	}
}

func TestVMResource_buildPCIDeviceOpts(t *testing.T) {
	pci := &VMPCIModel{
		PPTDev: types.StringValue("0000:01:00.0"),
	}
	opts := buildPCIDeviceOpts(pci, 1)
	if opts.DeviceType != truenas.DeviceTypePCI {
		t.Errorf("expected dtype=PCI, got %v", opts.DeviceType)
	}
	if opts.PCI == nil || opts.PCI.PPTDev != "0000:01:00.0" {
		t.Errorf("expected pptdev 0000:01:00.0, got %v", opts.PCI)
	}
}

func TestVMResource_buildUSBDeviceOpts(t *testing.T) {
	usb := &VMUSBModel{
		ControllerType: types.StringValue("qemu-xhci"),
		Device:         types.StringValue("usb_0001"),
	}
	opts := buildUSBDeviceOpts(usb, 1)
	if opts.DeviceType != truenas.DeviceTypeUSB {
		t.Errorf("expected dtype=USB, got %v", opts.DeviceType)
	}
	if opts.USB == nil || opts.USB.ControllerType != "qemu-xhci" {
		t.Errorf("expected controller_type=qemu-xhci, got %v", opts.USB)
	}
}

// -- Device reconciliation tests --

func TestVMResource_reconcileDevices(t *testing.T) {
	t.Run("create new device", func(t *testing.T) {
		var createdDevices []truenas.CreateVMDeviceOpts
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					createdDevices = append(createdDevices, opts)
					return &truenas.VMDevice{ID: 100}, nil
				},
			}}},
		}

		plan := &VMResourceModel{
			Disks: []VMDiskModel{{
				Path: types.StringValue("/dev/zvol/tank/vms/new-disk"),
				Type: types.StringValue("VIRTIO"),
			}},
		}
		state := &VMResourceModel{}

		err := r.reconcileDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(createdDevices) != 1 {
			t.Fatalf("expected 1 device create, got %d", len(createdDevices))
		}
	})

	t.Run("delete removed device", func(t *testing.T) {
		var deletedIDs []int64
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				DeleteDeviceFunc: func(ctx context.Context, id int64) error {
					deletedIDs = append(deletedIDs, id)
					return nil
				},
			}}},
		}

		plan := &VMResourceModel{}
		state := &VMResourceModel{
			Disks: []VMDiskModel{{
				DeviceID: types.Int64Value(50),
				Path:     types.StringValue("/dev/zvol/tank/vms/old-disk"),
				Type:     types.StringValue("VIRTIO"),
			}},
		}

		err := r.reconcileDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(deletedIDs) != 1 || deletedIDs[0] != 50 {
			t.Fatalf("expected device 50 to be deleted, got %v", deletedIDs)
		}
	})

	t.Run("update existing device", func(t *testing.T) {
		var updatedParams []any
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				UpdateDeviceFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMDeviceOpts) (*truenas.VMDevice, error) {
					updatedParams = append(updatedParams, opts)
					return &truenas.VMDevice{ID: 50}, nil
				},
			}}},
		}

		plan := &VMResourceModel{
			Disks: []VMDiskModel{{
				DeviceID: types.Int64Value(50),
				Path:     types.StringValue("/dev/zvol/tank/vms/disk0"),
				Type:     types.StringValue("VIRTIO"), // changed from AHCI
			}},
		}
		state := &VMResourceModel{
			Disks: []VMDiskModel{{
				DeviceID: types.Int64Value(50),
				Path:     types.StringValue("/dev/zvol/tank/vms/disk0"),
				Type:     types.StringValue("AHCI"),
			}},
		}

		err := r.reconcileDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(updatedParams) != 1 {
			t.Fatalf("expected 1 device update, got %d", len(updatedParams))
		}
	})

	t.Run("no changes no calls", func(t *testing.T) {
		callCount := 0
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					callCount++
					return &truenas.VMDevice{ID: 1}, nil
				},
				UpdateDeviceFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMDeviceOpts) (*truenas.VMDevice, error) {
					callCount++
					return &truenas.VMDevice{ID: id}, nil
				},
				DeleteDeviceFunc: func(ctx context.Context, id int64) error {
					callCount++
					return nil
				},
			}}},
		}

		disk := VMDiskModel{
			DeviceID: types.Int64Value(50),
			Path:     types.StringValue("/dev/zvol/tank/vms/disk0"),
			Type:     types.StringValue("VIRTIO"),
		}
		plan := &VMResourceModel{Disks: []VMDiskModel{disk}}
		state := &VMResourceModel{Disks: []VMDiskModel{disk}}

		err := r.reconcileDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if callCount != 0 {
			t.Errorf("expected no API calls, got %d", callCount)
		}
	})
}

// -- State management tests --

func TestVMResource_reconcileState(t *testing.T) {
	t.Run("stopped to running calls vm.start", func(t *testing.T) {
		var calledMethod string
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				StartVMFunc: func(ctx context.Context, id int64) error {
					calledMethod = "vm.start"
					return nil
				},
			}}},
		}

		err := r.reconcileState(context.Background(), 1, "STOPPED", "RUNNING")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calledMethod != "vm.start" {
			t.Errorf("expected vm.start, got %q", calledMethod)
		}
	})

	t.Run("running to stopped calls vm.stop via CallAndWait", func(t *testing.T) {
		var calledMethod string
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				StopVMFunc: func(ctx context.Context, id int64, opts truenas.StopVMOpts) error {
					calledMethod = "vm.stop"
					return nil
				},
			}}},
		}

		err := r.reconcileState(context.Background(), 1, "RUNNING", "STOPPED")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calledMethod != "vm.stop" {
			t.Errorf("expected vm.stop, got %q", calledMethod)
		}
	})

	t.Run("same state is no-op", func(t *testing.T) {
		callCount := 0
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
					callCount++
					return nil, nil
				},
			}}},
		}

		err := r.reconcileState(context.Background(), 1, "RUNNING", "RUNNING")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if callCount != 0 {
			t.Errorf("expected no calls for same state, got %d", callCount)
		}
	})
}

// -- Additional helper types for raw/pci/usb model values --

type vmRawParams struct {
	DeviceID           interface{}
	Path               interface{}
	Type               interface{}
	Boot               interface{}
	Exists             interface{}
	Size               interface{}
	LogicalSectorSize  interface{}
	PhysicalSectorSize interface{}
	IOType             interface{}
	Serial             interface{}
	Order              interface{}
}

type vmPCIParams struct {
	DeviceID interface{}
	PPTDev   interface{}
	Order    interface{}
}

type vmUSBParams struct {
	DeviceID       interface{}
	ControllerType interface{}
	Device         interface{}
	Order          interface{}
}

// createVMModelValueFull is like createVMModelValue but supports all device types.
func createVMModelValueFull(p vmModelParams, raws []vmRawParams, pcis []vmPCIParams, usbs []vmUSBParams) tftypes.Value {
	// Start with the base value parts from createVMModelValue, but override raw/pci/usb lists.

	// Build disk block values
	var diskValues []tftypes.Value
	for _, d := range p.Disks {
		diskValues = append(diskValues, tftypes.NewValue(vmDiskBlockType(), map[string]tftypes.Value{
			"device_id":           tftypes.NewValue(tftypes.Number, d.DeviceID),
			"path":                tftypes.NewValue(tftypes.String, d.Path),
			"type":                tftypes.NewValue(tftypes.String, d.Type),
			"logical_sectorsize":  tftypes.NewValue(tftypes.Number, d.LogicalSectorSize),
			"physical_sectorsize": tftypes.NewValue(tftypes.Number, d.PhysicalSectorSize),
			"iotype":              tftypes.NewValue(tftypes.String, d.IOType),
			"serial":              tftypes.NewValue(tftypes.String, d.Serial),
			"order":               tftypes.NewValue(tftypes.Number, d.Order),
		}))
	}
	diskList := emptyBlockList(vmDiskBlockType())
	if len(diskValues) > 0 {
		diskList = tftypes.NewValue(tftypes.List{ElementType: vmDiskBlockType()}, diskValues)
	}

	// Build NIC block values
	var nicValues []tftypes.Value
	for _, n := range p.NICs {
		nicValues = append(nicValues, tftypes.NewValue(vmNICBlockType(), map[string]tftypes.Value{
			"device_id":              tftypes.NewValue(tftypes.Number, n.DeviceID),
			"type":                   tftypes.NewValue(tftypes.String, n.Type),
			"nic_attach":             tftypes.NewValue(tftypes.String, n.NICAttach),
			"mac":                    tftypes.NewValue(tftypes.String, n.MAC),
			"trust_guest_rx_filters": tftypes.NewValue(tftypes.Bool, n.TrustGuestRXFilters),
			"order":                  tftypes.NewValue(tftypes.Number, n.Order),
		}))
	}
	nicList := emptyBlockList(vmNICBlockType())
	if len(nicValues) > 0 {
		nicList = tftypes.NewValue(tftypes.List{ElementType: vmNICBlockType()}, nicValues)
	}

	// Build CDROM block values
	var cdromValues []tftypes.Value
	for _, c := range p.CDROMs {
		cdromValues = append(cdromValues, tftypes.NewValue(vmCDROMBlockType(), map[string]tftypes.Value{
			"device_id": tftypes.NewValue(tftypes.Number, c.DeviceID),
			"path":      tftypes.NewValue(tftypes.String, c.Path),
			"order":     tftypes.NewValue(tftypes.Number, c.Order),
		}))
	}
	cdromList := emptyBlockList(vmCDROMBlockType())
	if len(cdromValues) > 0 {
		cdromList = tftypes.NewValue(tftypes.List{ElementType: vmCDROMBlockType()}, cdromValues)
	}

	// Build Display block values
	var displayValues []tftypes.Value
	for _, d := range p.Displays {
		displayValues = append(displayValues, tftypes.NewValue(vmDisplayBlockType(), map[string]tftypes.Value{
			"device_id":  tftypes.NewValue(tftypes.Number, d.DeviceID),
			"type":       tftypes.NewValue(tftypes.String, d.Type),
			"resolution": tftypes.NewValue(tftypes.String, d.Resolution),
			"port":       tftypes.NewValue(tftypes.Number, d.Port),
			"web_port":   tftypes.NewValue(tftypes.Number, d.WebPort),
			"bind":       tftypes.NewValue(tftypes.String, d.Bind),
			"wait":       tftypes.NewValue(tftypes.Bool, d.Wait),
			"password":   tftypes.NewValue(tftypes.String, d.Password),
			"web":        tftypes.NewValue(tftypes.Bool, d.Web),
			"order":      tftypes.NewValue(tftypes.Number, d.Order),
		}))
	}
	displayList := emptyBlockList(vmDisplayBlockType())
	if len(displayValues) > 0 {
		displayList = tftypes.NewValue(tftypes.List{ElementType: vmDisplayBlockType()}, displayValues)
	}

	// Build Raw block values
	var rawValues []tftypes.Value
	for _, r := range raws {
		rawValues = append(rawValues, tftypes.NewValue(vmRawBlockType(), map[string]tftypes.Value{
			"device_id":           tftypes.NewValue(tftypes.Number, r.DeviceID),
			"path":                tftypes.NewValue(tftypes.String, r.Path),
			"type":                tftypes.NewValue(tftypes.String, r.Type),
			"boot":                tftypes.NewValue(tftypes.Bool, r.Boot),
			"exists":              tftypes.NewValue(tftypes.Bool, r.Exists),
			"size":                tftypes.NewValue(tftypes.Number, r.Size),
			"logical_sectorsize":  tftypes.NewValue(tftypes.Number, r.LogicalSectorSize),
			"physical_sectorsize": tftypes.NewValue(tftypes.Number, r.PhysicalSectorSize),
			"iotype":              tftypes.NewValue(tftypes.String, r.IOType),
			"serial":              tftypes.NewValue(tftypes.String, r.Serial),
			"order":               tftypes.NewValue(tftypes.Number, r.Order),
		}))
	}
	rawList := emptyBlockList(vmRawBlockType())
	if len(rawValues) > 0 {
		rawList = tftypes.NewValue(tftypes.List{ElementType: vmRawBlockType()}, rawValues)
	}

	// Build PCI block values
	var pciValues []tftypes.Value
	for _, pc := range pcis {
		pciValues = append(pciValues, tftypes.NewValue(vmPCIBlockType(), map[string]tftypes.Value{
			"device_id": tftypes.NewValue(tftypes.Number, pc.DeviceID),
			"pptdev":    tftypes.NewValue(tftypes.String, pc.PPTDev),
			"order":     tftypes.NewValue(tftypes.Number, pc.Order),
		}))
	}
	pciList := emptyBlockList(vmPCIBlockType())
	if len(pciValues) > 0 {
		pciList = tftypes.NewValue(tftypes.List{ElementType: vmPCIBlockType()}, pciValues)
	}

	// Build USB block values
	var usbValues []tftypes.Value
	for _, u := range usbs {
		usbValues = append(usbValues, tftypes.NewValue(vmUSBBlockType(), map[string]tftypes.Value{
			"device_id":       tftypes.NewValue(tftypes.Number, u.DeviceID),
			"controller_type": tftypes.NewValue(tftypes.String, u.ControllerType),
			"device":          tftypes.NewValue(tftypes.String, u.Device),
			"order":           tftypes.NewValue(tftypes.Number, u.Order),
		}))
	}
	usbList := emptyBlockList(vmUSBBlockType())
	if len(usbValues) > 0 {
		usbList = tftypes.NewValue(tftypes.List{ElementType: vmUSBBlockType()}, usbValues)
	}

	values := map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, p.ID),
		"name":              tftypes.NewValue(tftypes.String, p.Name),
		"description":       tftypes.NewValue(tftypes.String, p.Description),
		"vcpus":             tftypes.NewValue(tftypes.Number, p.VCPUs),
		"cores":             tftypes.NewValue(tftypes.Number, p.Cores),
		"threads":           tftypes.NewValue(tftypes.Number, p.Threads),
		"memory":            tftypes.NewValue(tftypes.Number, p.Memory),
		"min_memory":        tftypes.NewValue(tftypes.Number, p.MinMemory),
		"autostart":         tftypes.NewValue(tftypes.Bool, p.Autostart),
		"time":              tftypes.NewValue(tftypes.String, p.Time),
		"bootloader":        tftypes.NewValue(tftypes.String, p.Bootloader),
		"bootloader_ovmf":   tftypes.NewValue(tftypes.String, p.BootloaderOVMF),
		"cpu_mode":          tftypes.NewValue(tftypes.String, p.CPUMode),
		"cpu_model":         tftypes.NewValue(tftypes.String, p.CPUModel),
		"shutdown_timeout":  tftypes.NewValue(tftypes.Number, p.ShutdownTimeout),
		"command_line_args": tftypes.NewValue(tftypes.String, p.CommandLineArgs),
		"state":             tftypes.NewValue(tftypes.String, p.State),
		"display_available": tftypes.NewValue(tftypes.Bool, p.DisplayAvailable),
		"disk":              diskList,
		"raw":               rawList,
		"cdrom":             cdromList,
		"nic":               nicList,
		"display":           displayList,
		"pci":               pciList,
		"usb":               usbList,
	}

	return tftypes.NewValue(vmObjectType(), values)
}

// -- Create error path tests --

func TestVMResource_Create_DeviceCreateError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
				return nil, errors.New("device creation failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.Disks = []vmDiskParams{{
		DeviceID: nil, Path: "/dev/zvol/tank/vms/disk0", Type: "VIRTIO",
		LogicalSectorSize: nil, PhysicalSectorSize: nil, IOType: "THREADS",
		Serial: nil, Order: nil,
	}}
	planValue := createVMModelValue(p)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when device creation fails")
	}
}

func TestVMResource_Create_StartError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			StartVMFunc: func(ctx context.Context, id int64) error {
				return errors.New("vm.start failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.State = "RUNNING"
	planValue := createVMModelValue(p)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when vm.start fails")
	}
}

func TestVMResource_Create_ReadBackError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return nil, errors.New("read-back failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	planValue := createVMModelValue(defaultVMPlanParams())
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when read-back after create fails")
	}
}

func TestVMResource_Create_DeviceQueryError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return nil, errors.New("device query failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	planValue := createVMModelValue(defaultVMPlanParams())
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when device query after create fails")
	}
}

func TestVMResource_Create_WithCDROMDevice(t *testing.T) {
	deviceCreateCalled := false
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
				deviceCreateCalled = true
				return &truenas.VMDevice{ID: 101}, nil
			},
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return []truenas.VMDevice{
					{ID: 101, VM: 1, Order: 1000, DeviceType: truenas.DeviceTypeCDROM,
						CDROM: &truenas.CDROMDevice{Path: "/mnt/tank/iso/test.iso"}},
				}, nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.CDROMs = []vmCDROMParams{{
		DeviceID: nil, Path: "/mnt/tank/iso/test.iso", Order: nil,
	}}
	planValue := createVMModelValue(p)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !deviceCreateCalled {
		t.Error("expected vm.device.create to be called for CDROM device")
	}
}

func TestVMResource_Create_WithDisplayDevice(t *testing.T) {
	var deviceCreateCalls int
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
				deviceCreateCalls++
				return &truenas.VMDevice{ID: 101}, nil
			},
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return []truenas.VMDevice{
					{ID: 101, VM: 1, Order: 1000, DeviceType: truenas.DeviceTypeDisplay,
						Display: &truenas.DisplayDevice{Type: "SPICE", Resolution: "1920x1080", Bind: "0.0.0.0", Web: true, Port: 5900}},
				}, nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.Displays = []vmDisplayParams{{
		DeviceID: nil, Type: "SPICE", Resolution: "1920x1080", Port: float64(5900),
		WebPort: nil, Bind: "0.0.0.0", Wait: false, Password: nil, Web: true, Order: nil,
	}}
	planValue := createVMModelValue(p)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if deviceCreateCalls != 1 {
		t.Errorf("expected 1 device create call, got %d", deviceCreateCalls)
	}
}

func TestVMResource_Create_WithRawDevice(t *testing.T) {
	var deviceCreateCalls int
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
				deviceCreateCalls++
				return &truenas.VMDevice{ID: 101}, nil
			},
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return []truenas.VMDevice{
					{ID: 101, VM: 1, Order: 1000, DeviceType: truenas.DeviceTypeRaw,
						Raw: &truenas.RawDevice{Path: "/mnt/tank/vms/raw.img", Type: "AHCI", Boot: false}},
				}, nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	planValue := createVMModelValueFull(p,
		[]vmRawParams{{
			DeviceID: nil, Path: "/mnt/tank/vms/raw.img", Type: "AHCI",
			Boot: false, Size: nil, LogicalSectorSize: nil, PhysicalSectorSize: nil,
			IOType: "THREADS", Serial: nil, Order: nil,
		}},
		nil, nil,
	)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if deviceCreateCalls != 1 {
		t.Errorf("expected 1 device create call for raw device, got %d", deviceCreateCalls)
	}
}

func TestVMResource_Create_WithPCIDevice(t *testing.T) {
	var deviceCreateCalls int
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
				deviceCreateCalls++
				return &truenas.VMDevice{ID: 101}, nil
			},
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return []truenas.VMDevice{
					{ID: 101, VM: 1, Order: 1000, DeviceType: truenas.DeviceTypePCI,
						PCI: &truenas.PCIDevice{PPTDev: "0000:01:00.0"}},
				}, nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	planValue := createVMModelValueFull(p, nil,
		[]vmPCIParams{{DeviceID: nil, PPTDev: "0000:01:00.0", Order: nil}},
		nil,
	)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if deviceCreateCalls != 1 {
		t.Errorf("expected 1 device create call for PCI device, got %d", deviceCreateCalls)
	}
}

func TestVMResource_Create_WithUSBDevice(t *testing.T) {
	var deviceCreateCalls int
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
				deviceCreateCalls++
				return &truenas.VMDevice{ID: 101}, nil
			},
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return []truenas.VMDevice{
					{ID: 101, VM: 1, Order: 1000, DeviceType: truenas.DeviceTypeUSB,
						USB: &truenas.USBDevice{ControllerType: "qemu-xhci", Device: "usb_0001"}},
				}, nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	planValue := createVMModelValueFull(p, nil, nil,
		[]vmUSBParams{{DeviceID: nil, ControllerType: "qemu-xhci", Device: "usb_0001", Order: nil}},
	)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if deviceCreateCalls != 1 {
		t.Errorf("expected 1 device create call for USB device, got %d", deviceCreateCalls)
	}
}

func TestVMResource_Create_RawDeviceCreateError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
				return nil, errors.New("raw device creation failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	planValue := createVMModelValueFull(p,
		[]vmRawParams{{
			DeviceID: nil, Path: "/mnt/tank/vms/raw.img", Type: "AHCI",
			Boot: false, Size: nil, LogicalSectorSize: nil, PhysicalSectorSize: nil,
			IOType: "THREADS", Serial: nil, Order: nil,
		}},
		nil, nil,
	)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when raw device creation fails")
	}
}

func TestVMResource_Create_CDROMDeviceCreateError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
				return nil, errors.New("cdrom device creation failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.CDROMs = []vmCDROMParams{{DeviceID: nil, Path: "/mnt/tank/iso/test.iso", Order: nil}}
	planValue := createVMModelValue(p)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when CDROM device creation fails")
	}
}

func TestVMResource_Create_NICDeviceCreateError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
				return nil, errors.New("nic device creation failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.NICs = []vmNICParams{{DeviceID: nil, Type: "VIRTIO", NICAttach: "br0", MAC: nil, TrustGuestRXFilters: false, Order: nil}}
	planValue := createVMModelValue(p)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when NIC device creation fails")
	}
}

func TestVMResource_Create_DisplayDeviceCreateError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
				return nil, errors.New("display device creation failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.Displays = []vmDisplayParams{{
		DeviceID: nil, Type: "SPICE", Resolution: "1920x1080", Port: nil,
		WebPort: nil, Bind: "0.0.0.0", Wait: false, Password: nil, Web: true, Order: nil,
	}}
	planValue := createVMModelValue(p)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when display device creation fails")
	}
}

func TestVMResource_Create_PCIDeviceCreateError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
				return nil, errors.New("pci device creation failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	planValue := createVMModelValueFull(p, nil,
		[]vmPCIParams{{DeviceID: nil, PPTDev: "0000:01:00.0", Order: nil}},
		nil,
	)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when PCI device creation fails")
	}
}

func TestVMResource_Create_USBDeviceCreateError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			CreateVMFunc: func(ctx context.Context, opts truenas.CreateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
				return nil, errors.New("usb device creation failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	planValue := createVMModelValueFull(p, nil, nil,
		[]vmUSBParams{{DeviceID: nil, ControllerType: "qemu-xhci", Device: "usb_0001", Order: nil}},
	)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when USB device creation fails")
	}
}

// -- Read error path tests --

func TestVMResource_Read_NonNotFoundError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return nil, errors.New("internal server error")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "1"
	stateValue := createVMModelValue(p)
	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for non-not-found API error")
	}
}

func TestVMResource_Read_DeviceQueryError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return nil, errors.New("device query failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "1"
	stateValue := createVMModelValue(p)
	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when device query fails during read")
	}
}

// -- Update additional tests --

func TestVMResource_Update_WithDeviceReconciliation(t *testing.T) {
	deleteCalled := false
	createCalled := false
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			UpdateVMFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
				createCalled = true
				return &truenas.VMDevice{ID: 102}, nil
			},
			DeleteDeviceFunc: func(ctx context.Context, id int64) error {
				deleteCalled = true
				return nil
			},
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return []truenas.VMDevice{
					{ID: 102, VM: 1, Order: 1000, DeviceType: truenas.DeviceTypeDisk,
						Disk: &truenas.DiskDevice{Path: "/dev/zvol/tank/vms/new-disk", Type: "VIRTIO"}},
				}, nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)

	// State has a disk with device_id 50, plan has a new disk (no device_id)
	stateParams := defaultVMPlanParams()
	stateParams.ID = "1"
	stateParams.Disks = []vmDiskParams{{
		DeviceID: float64(50), Path: "/dev/zvol/tank/vms/old-disk", Type: "VIRTIO",
		LogicalSectorSize: nil, PhysicalSectorSize: nil, IOType: "THREADS",
		Serial: nil, Order: float64(1000),
	}}
	stateValue := createVMModelValue(stateParams)

	planParams := defaultVMPlanParams()
	planParams.ID = "1"
	planParams.Disks = []vmDiskParams{{
		DeviceID: nil, Path: "/dev/zvol/tank/vms/new-disk", Type: "VIRTIO",
		LogicalSectorSize: nil, PhysicalSectorSize: nil, IOType: "THREADS",
		Serial: nil, Order: nil,
	}}
	planValue := createVMModelValue(planParams)

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !deleteCalled {
		t.Error("expected vm.device.delete to be called for removed device")
	}
	if !createCalled {
		t.Error("expected vm.device.create to be called for new device")
	}
}

func TestVMResource_Update_StateTransition(t *testing.T) {
	startCalled := false
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			UpdateVMFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMOpts) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return nil, nil
			},
			StartVMFunc: func(ctx context.Context, id int64) error {
				startCalled = true
				return nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)

	stateParams := defaultVMPlanParams()
	stateParams.ID = "1"
	stateParams.State = "STOPPED"
	stateValue := createVMModelValue(stateParams)

	planParams := defaultVMPlanParams()
	planParams.ID = "1"
	planParams.State = "RUNNING"
	planValue := createVMModelValue(planParams)

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !startCalled {
		t.Error("expected vm.start to be called for state transition")
	}
}

func TestVMResource_Update_UpdateError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			UpdateVMFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMOpts) (*truenas.VM, error) {
				return nil, errors.New("update failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)

	stateParams := defaultVMPlanParams()
	stateParams.ID = "1"
	stateParams.Memory = float64(2048)
	stateValue := createVMModelValue(stateParams)

	planParams := defaultVMPlanParams()
	planParams.ID = "1"
	planParams.Memory = float64(4096)
	planValue := createVMModelValue(planParams)

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when vm.update fails")
	}
}

func TestVMResource_Update_DeviceReconcileError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			DeleteDeviceFunc: func(ctx context.Context, id int64) error {
				return errors.New("device delete failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)

	// State has a device, plan does not (triggers delete which will fail)
	stateParams := defaultVMPlanParams()
	stateParams.ID = "1"
	stateParams.Disks = []vmDiskParams{{
		DeviceID: float64(50), Path: "/dev/zvol/tank/vms/disk0", Type: "VIRTIO",
		LogicalSectorSize: nil, PhysicalSectorSize: nil, IOType: "THREADS",
		Serial: nil, Order: float64(1000),
	}}
	stateValue := createVMModelValue(stateParams)

	planParams := defaultVMPlanParams()
	planParams.ID = "1"
	planValue := createVMModelValue(planParams)

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when device reconciliation fails")
	}
}

func TestVMResource_Update_StateTransitionQueryError(t *testing.T) {
	callCount := 0
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				callCount++
				return nil, errors.New("query state failed")
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return nil, nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)

	stateParams := defaultVMPlanParams()
	stateParams.ID = "1"
	stateParams.State = "STOPPED"
	stateValue := createVMModelValue(stateParams)

	planParams := defaultVMPlanParams()
	planParams.ID = "1"
	planParams.State = "RUNNING"
	planValue := createVMModelValue(planParams)

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when query VM state fails during state transition")
	}
}

func TestVMResource_Update_StateReconcileError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return nil, nil
			},
			StartVMFunc: func(ctx context.Context, id int64) error {
				return errors.New("start failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)

	stateParams := defaultVMPlanParams()
	stateParams.ID = "1"
	stateParams.State = "STOPPED"
	stateValue := createVMModelValue(stateParams)

	planParams := defaultVMPlanParams()
	planParams.ID = "1"
	planParams.State = "RUNNING"
	planValue := createVMModelValue(planParams)

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when reconcileState fails during update")
	}
}

func TestVMResource_Update_ReadBackError(t *testing.T) {
	getInstanceCallCount := 0
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				getInstanceCallCount++
				if getInstanceCallCount >= 2 {
					return nil, errors.New("read-back failed")
				}
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return nil, nil
			},
			StartVMFunc: func(ctx context.Context, id int64) error {
				return nil
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)

	stateParams := defaultVMPlanParams()
	stateParams.ID = "1"
	stateParams.State = "STOPPED"
	stateValue := createVMModelValue(stateParams)

	planParams := defaultVMPlanParams()
	planParams.ID = "1"
	planParams.State = "RUNNING"
	planValue := createVMModelValue(planParams)

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when read-back after update fails")
	}
}

func TestVMResource_Update_DeviceQueryError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			ListDevicesFunc: func(ctx context.Context, vmID int64) ([]truenas.VMDevice, error) {
				return nil, errors.New("device query failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)

	stateParams := defaultVMPlanParams()
	stateParams.ID = "1"
	stateValue := createVMModelValue(stateParams)

	planParams := defaultVMPlanParams()
	planParams.ID = "1"
	planValue := createVMModelValue(planParams)

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when device query fails after update")
	}
}

// -- Delete error path tests --

func TestVMResource_Delete_StatusError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return nil, errors.New("internal server error")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "1"
	stateValue := createVMModelValue(p)
	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when vm.get_instance fails (non-not-found) during delete")
	}
}

func TestVMResource_Delete_StopError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "RUNNING"), nil
			},
			StopVMFunc: func(ctx context.Context, id int64, opts truenas.StopVMOpts) error {
				return errors.New("stop failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "1"
	p.State = "RUNNING"
	stateValue := createVMModelValue(p)
	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when vm.stop fails during delete")
	}
}

func TestVMResource_Delete_DeleteError(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return mockVM(1, "test-vm", 2048, "STOPPED"), nil
			},
			DeleteVMFunc: func(ctx context.Context, id int64) error {
				return errors.New("delete failed")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "1"
	stateValue := createVMModelValue(p)
	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when vm.delete fails")
	}
}

func TestVMResource_Delete_AlreadyDeleted(t *testing.T) {
	r := &VMResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
			GetVMFunc: func(ctx context.Context, id int64) (*truenas.VM, error) {
				return nil, errors.New("does not exist")
			},
		}}},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "1"
	stateValue := createVMModelValue(p)
	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no error when VM already deleted, got: %v", resp.Diagnostics)
	}
}

// -- buildUpdateOpts tests --

func TestVMResource_buildUpdateOpts(t *testing.T) {
	r := &VMResource{}

	t.Run("no changes returns nil", func(t *testing.T) {
		data := &VMResourceModel{
			Name:   types.StringValue("test-vm"),
			Memory: types.Int64Value(2048),
		}
		opts, changed := r.buildUpdateOpts(data, data)
		if changed {
			t.Error("expected no changes")
		}
		if opts != nil {
			t.Errorf("expected nil opts for no changes, got %v", opts)
		}
	})

	t.Run("all fields changed", func(t *testing.T) {
		plan := &VMResourceModel{
			Name:            types.StringValue("new-name"),
			Description:     types.StringValue("new desc"),
			VCPUs:           types.Int64Value(4),
			Cores:           types.Int64Value(4),
			Threads:         types.Int64Value(2),
			Memory:          types.Int64Value(8192),
			MinMemory:       types.Int64Value(4096),
			Autostart:       types.BoolValue(false),
			Time:            types.StringValue("UTC"),
			Bootloader:      types.StringValue("UEFI_CSM"),
			BootloaderOVMF:  types.StringValue("OVMF_CODE_TPM.fd"),
			CPUMode:         types.StringValue("HOST-PASSTHROUGH"),
			CPUModel:        types.StringValue("Haswell"),
			ShutdownTimeout: types.Int64Value(120),
			CommandLineArgs: types.StringValue("-cpu host"),
		}
		state := &VMResourceModel{
			Name:            types.StringValue("test-vm"),
			Description:     types.StringValue("old desc"),
			VCPUs:           types.Int64Value(1),
			Cores:           types.Int64Value(1),
			Threads:         types.Int64Value(1),
			Memory:          types.Int64Value(2048),
			MinMemory:       types.Int64Null(),
			Autostart:       types.BoolValue(true),
			Time:            types.StringValue("LOCAL"),
			Bootloader:      types.StringValue("UEFI"),
			BootloaderOVMF:  types.StringValue("OVMF_CODE.fd"),
			CPUMode:         types.StringValue("CUSTOM"),
			CPUModel:        types.StringNull(),
			ShutdownTimeout: types.Int64Value(90),
			CommandLineArgs: types.StringValue(""),
		}
		opts, changed := r.buildUpdateOpts(plan, state)
		if !changed {
			t.Fatal("expected changes")
		}
		if opts == nil {
			t.Fatal("expected non-nil opts")
		}
		if opts.Name != "new-name" {
			t.Errorf("expected name new-name, got %v", opts.Name)
		}
		if opts.Description != "new desc" {
			t.Errorf("expected description 'new desc', got %v", opts.Description)
		}
		if opts.VCPUs != 4 {
			t.Errorf("expected vcpus 4, got %v", opts.VCPUs)
		}
		if opts.Memory != 8192 {
			t.Errorf("expected memory 8192, got %v", opts.Memory)
		}
		if opts.MinMemory == nil || *opts.MinMemory != 4096 {
			t.Errorf("expected min_memory 4096, got %v", opts.MinMemory)
		}
		if opts.Autostart != false {
			t.Errorf("expected autostart false, got %v", opts.Autostart)
		}
		if opts.Bootloader != "UEFI_CSM" {
			t.Errorf("expected bootloader UEFI_CSM, got %v", opts.Bootloader)
		}
		if opts.BootloaderOVMF != "OVMF_CODE_TPM.fd" {
			t.Errorf("expected bootloader_ovmf OVMF_CODE_TPM.fd, got %v", opts.BootloaderOVMF)
		}
		if opts.CPUMode != "HOST-PASSTHROUGH" {
			t.Errorf("expected cpu_mode HOST-PASSTHROUGH, got %v", opts.CPUMode)
		}
		if opts.CPUModel != "Haswell" {
			t.Errorf("expected cpu_model Haswell, got %v", opts.CPUModel)
		}
		if opts.ShutdownTimeout != 120 {
			t.Errorf("expected shutdown_timeout 120, got %v", opts.ShutdownTimeout)
		}
		if opts.CommandLineArgs != "-cpu host" {
			t.Errorf("expected command_line_args '-cpu host', got %v", opts.CommandLineArgs)
		}
	})

	t.Run("min_memory set to null", func(t *testing.T) {
		plan := &VMResourceModel{
			Name:      types.StringValue("test-vm"),
			MinMemory: types.Int64Null(),
		}
		state := &VMResourceModel{
			Name:      types.StringValue("test-vm"),
			MinMemory: types.Int64Value(1024),
		}
		opts, changed := r.buildUpdateOpts(plan, state)
		if !changed {
			t.Fatal("expected changes for min_memory change")
		}
		if opts.MinMemory != nil {
			t.Errorf("expected min_memory=nil, got %v", opts.MinMemory)
		}
	})

	t.Run("cpu_model set to null", func(t *testing.T) {
		plan := &VMResourceModel{
			Name:     types.StringValue("test-vm"),
			CPUModel: types.StringNull(),
		}
		state := &VMResourceModel{
			Name:     types.StringValue("test-vm"),
			CPUModel: types.StringValue("Haswell"),
		}
		opts, changed := r.buildUpdateOpts(plan, state)
		if !changed {
			t.Fatal("expected changes for cpu_model change")
		}
		if opts.CPUModel != "" {
			t.Errorf("expected cpu_model empty, got %v", opts.CPUModel)
		}
	})
}

// -- buildRawDeviceOpts tests --

func TestVMResource_buildRawDeviceOpts(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		raw := &VMRawModel{
			Path:   types.StringValue("/mnt/tank/vms/raw.img"),
			Type:   types.StringValue("AHCI"),
			Boot:   types.BoolValue(false),
			IOType: types.StringValue("THREADS"),
		}
		opts := buildRawDeviceOpts(raw, 1)
		if opts.VM != 1 {
			t.Errorf("expected vm=1, got %v", opts.VM)
		}
		if opts.DeviceType != truenas.DeviceTypeRaw {
			t.Errorf("expected dtype=RAW, got %v", opts.DeviceType)
		}
		if opts.Raw == nil {
			t.Fatal("expected raw device to be set")
		}
		if opts.Raw.Path != "/mnt/tank/vms/raw.img" {
			t.Errorf("expected path, got %v", opts.Raw.Path)
		}
		if opts.Raw.Type != "AHCI" {
			t.Errorf("expected type=AHCI, got %v", opts.Raw.Type)
		}
		if opts.Raw.Boot != false {
			t.Errorf("expected boot=false, got %v", opts.Raw.Boot)
		}
	})

	t.Run("with exists true", func(t *testing.T) {
		raw := &VMRawModel{
			Path:   types.StringValue("/dev/zvol/tank/vms/disk0"),
			Type:   types.StringValue("AHCI"),
			Boot:   types.BoolValue(false),
			Exists: types.BoolValue(true),
			IOType: types.StringValue("THREADS"),
		}
		opts := buildRawDeviceOpts(raw, 1)
		if opts.Raw == nil {
			t.Fatal("expected raw device to be set")
		}
		if opts.Raw.Exists != true {
			t.Errorf("expected exists=true, got %v", opts.Raw.Exists)
		}
	})

	t.Run("with optional fields", func(t *testing.T) {
		raw := &VMRawModel{
			Path:               types.StringValue("/mnt/tank/vms/raw.img"),
			Type:               types.StringValue("VIRTIO"),
			Boot:               types.BoolValue(true),
			Size:               types.Int64Value(10737418240),
			LogicalSectorSize:  types.Int64Value(512),
			PhysicalSectorSize: types.Int64Value(4096),
			IOType:             types.StringValue("NATIVE"),
			Serial:             types.StringValue("RAW001"),
			Order:              types.Int64Value(1000),
		}
		opts := buildRawDeviceOpts(raw, 2)
		if opts.Raw == nil {
			t.Fatal("expected raw device to be set")
		}
		if opts.Raw.Size == nil || *opts.Raw.Size != 10737418240 {
			t.Errorf("expected size=10737418240, got %v", opts.Raw.Size)
		}
		if opts.Raw.Logical_Sector_Size == nil || *opts.Raw.Logical_Sector_Size != 512 {
			t.Errorf("expected logical_sectorsize=512, got %v", opts.Raw.Logical_Sector_Size)
		}
		if opts.Raw.PhysicalSectorSize == nil || *opts.Raw.PhysicalSectorSize != 4096 {
			t.Errorf("expected physical_sectorsize=4096, got %v", opts.Raw.PhysicalSectorSize)
		}
		if opts.Raw.Serial != "RAW001" {
			t.Errorf("expected serial=RAW001, got %v", opts.Raw.Serial)
		}
		if opts.Order == nil || *opts.Order != 1000 {
			t.Errorf("expected order=1000, got %v", opts.Order)
		}
	})
}

// -- buildDiskDeviceOpts additional tests --

func TestVMResource_buildDiskDeviceOpts_WithOptionalFields(t *testing.T) {
	disk := &VMDiskModel{
		Path:               types.StringValue("/dev/zvol/tank/vms/disk0"),
		Type:               types.StringValue("VIRTIO"),
		IOType:             types.StringValue("THREADS"),
		LogicalSectorSize:  types.Int64Value(512),
		PhysicalSectorSize: types.Int64Value(4096),
		Serial:             types.StringValue("DISK001"),
		Order:              types.Int64Value(1000),
	}
	opts := buildDiskDeviceOpts(disk, 1)
	if opts.Disk == nil {
		t.Fatal("expected disk device to be set")
	}
	if opts.Disk.Logical_Sector_Size == nil || *opts.Disk.Logical_Sector_Size != 512 {
		t.Errorf("expected logical_sectorsize=512, got %v", opts.Disk.Logical_Sector_Size)
	}
	if opts.Disk.PhysicalSectorSize == nil || *opts.Disk.PhysicalSectorSize != 4096 {
		t.Errorf("expected physical_sectorsize=4096, got %v", opts.Disk.PhysicalSectorSize)
	}
	if opts.Disk.Serial != "DISK001" {
		t.Errorf("expected serial=DISK001, got %v", opts.Disk.Serial)
	}
	if opts.Order == nil || *opts.Order != 1000 {
		t.Errorf("expected order=1000, got %v", opts.Order)
	}
}

// -- buildDisplayDeviceOpts additional tests --

func TestVMResource_buildDisplayDeviceOpts_WithOptionalFields(t *testing.T) {
	display := &VMDisplayModel{
		Type:       types.StringValue("SPICE"),
		Resolution: types.StringValue("1920x1080"),
		Port:       types.Int64Value(5900),
		WebPort:    types.Int64Value(5901),
		Bind:       types.StringValue("0.0.0.0"),
		Wait:       types.BoolValue(true),
		Password:   types.StringValue("secret"),
		Web:        types.BoolValue(true),
		Order:      types.Int64Value(1000),
	}
	opts := buildDisplayDeviceOpts(display, 1)
	if opts.Display == nil {
		t.Fatal("expected display device to be set")
	}
	if opts.Display.Port != 5900 {
		t.Errorf("expected port=5900, got %v", opts.Display.Port)
	}
	if opts.Display.WebPort != 5901 {
		t.Errorf("expected web_port=5901, got %v", opts.Display.WebPort)
	}
	if opts.Display.Password != "secret" {
		t.Errorf("expected password=secret, got %v", opts.Display.Password)
	}
	if opts.Display.Wait != true {
		t.Errorf("expected wait=true, got %v", opts.Display.Wait)
	}
	if opts.Order == nil || *opts.Order != 1000 {
		t.Errorf("expected order=1000, got %v", opts.Order)
	}
}

// -- Reconcile device type tests --

func TestVMResource_reconcileRawDevices(t *testing.T) {
	t.Run("create new raw device", func(t *testing.T) {
		var createdDevices []truenas.CreateVMDeviceOpts
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					createdDevices = append(createdDevices, opts)
					return &truenas.VMDevice{ID: 100}, nil
				},
			}}},
		}

		plan := []VMRawModel{{
			Path:   types.StringValue("/mnt/tank/vms/raw.img"),
			Type:   types.StringValue("AHCI"),
			Boot:   types.BoolValue(false),
			IOType: types.StringValue("THREADS"),
		}}

		err := r.reconcileRawDevices(context.Background(), 1, plan, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(createdDevices) != 1 {
			t.Fatalf("expected 1 device create, got %d", len(createdDevices))
		}
	})

	t.Run("update changed raw device", func(t *testing.T) {
		var updatedParams []any
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				UpdateDeviceFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMDeviceOpts) (*truenas.VMDevice, error) {
					updatedParams = append(updatedParams, opts)
					return &truenas.VMDevice{ID: 50}, nil
				},
			}}},
		}

		plan := []VMRawModel{{
			DeviceID: types.Int64Value(50),
			Path:     types.StringValue("/mnt/tank/vms/raw.img"),
			Type:     types.StringValue("VIRTIO"), // changed
			Boot:     types.BoolValue(false),
		}}
		state := []VMRawModel{{
			DeviceID: types.Int64Value(50),
			Path:     types.StringValue("/mnt/tank/vms/raw.img"),
			Type:     types.StringValue("AHCI"),
			Boot:     types.BoolValue(false),
		}}

		err := r.reconcileRawDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(updatedParams) != 1 {
			t.Fatalf("expected 1 device update, got %d", len(updatedParams))
		}
	})

	t.Run("create error", func(t *testing.T) {
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					return nil, errors.New("create failed")
				},
			}}},
		}

		plan := []VMRawModel{{
			Path: types.StringValue("/mnt/tank/vms/raw.img"),
			Type: types.StringValue("AHCI"),
		}}
		err := r.reconcileRawDevices(context.Background(), 1, plan, nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("update error", func(t *testing.T) {
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				UpdateDeviceFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMDeviceOpts) (*truenas.VMDevice, error) {
					return nil, errors.New("update failed")
				},
			}}},
		}

		plan := []VMRawModel{{
			DeviceID: types.Int64Value(50),
			Path:     types.StringValue("/mnt/tank/vms/raw.img"),
			Type:     types.StringValue("VIRTIO"),
			Boot:     types.BoolValue(true),
		}}
		state := []VMRawModel{{
			DeviceID: types.Int64Value(50),
			Path:     types.StringValue("/mnt/tank/vms/raw.img"),
			Type:     types.StringValue("AHCI"),
			Boot:     types.BoolValue(false),
		}}
		err := r.reconcileRawDevices(context.Background(), 1, plan, state)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestVMResource_reconcileCDROMDevices(t *testing.T) {
	t.Run("create new cdrom", func(t *testing.T) {
		var createdDevices int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					createdDevices++
					return &truenas.VMDevice{ID: 100}, nil
				},
			}}},
		}

		plan := []VMCDROMModel{{
			Path: types.StringValue("/mnt/tank/iso/test.iso"),
		}}
		err := r.reconcileCDROMDevices(context.Background(), 1, plan, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if createdDevices != 1 {
			t.Fatalf("expected 1 create, got %d", createdDevices)
		}
	})

	t.Run("update changed cdrom", func(t *testing.T) {
		var updatedCount int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				UpdateDeviceFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMDeviceOpts) (*truenas.VMDevice, error) {
					updatedCount++
					return &truenas.VMDevice{ID: 50}, nil
				},
			}}},
		}

		plan := []VMCDROMModel{{
			DeviceID: types.Int64Value(50),
			Path:     types.StringValue("/mnt/tank/iso/new.iso"),
		}}
		state := []VMCDROMModel{{
			DeviceID: types.Int64Value(50),
			Path:     types.StringValue("/mnt/tank/iso/old.iso"),
		}}
		err := r.reconcileCDROMDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updatedCount != 1 {
			t.Fatalf("expected 1 update, got %d", updatedCount)
		}
	})

	t.Run("create error", func(t *testing.T) {
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					return nil, errors.New("create failed")
				},
			}}},
		}

		plan := []VMCDROMModel{{Path: types.StringValue("/mnt/tank/iso/test.iso")}}
		err := r.reconcileCDROMDevices(context.Background(), 1, plan, nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("update error", func(t *testing.T) {
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				UpdateDeviceFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMDeviceOpts) (*truenas.VMDevice, error) {
					return nil, errors.New("update failed")
				},
			}}},
		}

		plan := []VMCDROMModel{{DeviceID: types.Int64Value(50), Path: types.StringValue("/mnt/tank/iso/new.iso")}}
		state := []VMCDROMModel{{DeviceID: types.Int64Value(50), Path: types.StringValue("/mnt/tank/iso/old.iso")}}
		err := r.reconcileCDROMDevices(context.Background(), 1, plan, state)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestVMResource_reconcileNICDevices(t *testing.T) {
	t.Run("create new nic", func(t *testing.T) {
		var createdDevices int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					createdDevices++
					return &truenas.VMDevice{ID: 100}, nil
				},
			}}},
		}

		plan := []VMNICModel{{
			Type:      types.StringValue("VIRTIO"),
			NICAttach: types.StringValue("br0"),
		}}
		err := r.reconcileNICDevices(context.Background(), 1, plan, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if createdDevices != 1 {
			t.Fatalf("expected 1 create, got %d", createdDevices)
		}
	})

	t.Run("update changed nic", func(t *testing.T) {
		var updatedCount int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				UpdateDeviceFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMDeviceOpts) (*truenas.VMDevice, error) {
					updatedCount++
					return &truenas.VMDevice{ID: 50}, nil
				},
			}}},
		}

		plan := []VMNICModel{{
			DeviceID: types.Int64Value(50),
			Type:     types.StringValue("E1000"), // changed
			NICAttach: types.StringValue("br0"),
		}}
		state := []VMNICModel{{
			DeviceID: types.Int64Value(50),
			Type:     types.StringValue("VIRTIO"),
			NICAttach: types.StringValue("br0"),
		}}
		err := r.reconcileNICDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updatedCount != 1 {
			t.Fatalf("expected 1 update, got %d", updatedCount)
		}
	})

	t.Run("create error", func(t *testing.T) {
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					return nil, errors.New("create failed")
				},
			}}},
		}

		plan := []VMNICModel{{Type: types.StringValue("VIRTIO"), NICAttach: types.StringValue("br0")}}
		err := r.reconcileNICDevices(context.Background(), 1, plan, nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("update error", func(t *testing.T) {
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				UpdateDeviceFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMDeviceOpts) (*truenas.VMDevice, error) {
					return nil, errors.New("update failed")
				},
			}}},
		}

		plan := []VMNICModel{{DeviceID: types.Int64Value(50), Type: types.StringValue("E1000"), NICAttach: types.StringValue("br0")}}
		state := []VMNICModel{{DeviceID: types.Int64Value(50), Type: types.StringValue("VIRTIO"), NICAttach: types.StringValue("br0")}}
		err := r.reconcileNICDevices(context.Background(), 1, plan, state)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestVMResource_reconcileDisplayDevices(t *testing.T) {
	t.Run("create new display", func(t *testing.T) {
		var createdDevices int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					createdDevices++
					return &truenas.VMDevice{ID: 100}, nil
				},
			}}},
		}

		plan := []VMDisplayModel{{
			Type:       types.StringValue("SPICE"),
			Resolution: types.StringValue("1920x1080"),
			Bind:       types.StringValue("0.0.0.0"),
			Web:        types.BoolValue(true),
		}}
		err := r.reconcileDisplayDevices(context.Background(), 1, plan, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if createdDevices != 1 {
			t.Fatalf("expected 1 create, got %d", createdDevices)
		}
	})

	t.Run("update changed display", func(t *testing.T) {
		var updatedCount int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				UpdateDeviceFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMDeviceOpts) (*truenas.VMDevice, error) {
					updatedCount++
					return &truenas.VMDevice{ID: 50}, nil
				},
			}}},
		}

		plan := []VMDisplayModel{{
			DeviceID:   types.Int64Value(50),
			Type:       types.StringValue("SPICE"),
			Resolution: types.StringValue("1920x1080"), // changed
			Bind:       types.StringValue("0.0.0.0"),
			Web:        types.BoolValue(true),
			Wait:       types.BoolValue(false),
			Port:       types.Int64Value(5900),
			WebPort:    types.Int64Value(5901),
		}}
		state := []VMDisplayModel{{
			DeviceID:   types.Int64Value(50),
			Type:       types.StringValue("SPICE"),
			Resolution: types.StringValue("1024x768"),
			Bind:       types.StringValue("0.0.0.0"),
			Web:        types.BoolValue(true),
			Wait:       types.BoolValue(false),
			Port:       types.Int64Value(5900),
			WebPort:    types.Int64Value(5901),
		}}
		err := r.reconcileDisplayDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updatedCount != 1 {
			t.Fatalf("expected 1 update, got %d", updatedCount)
		}
	})

	t.Run("create error", func(t *testing.T) {
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					return nil, errors.New("create failed")
				},
			}}},
		}

		plan := []VMDisplayModel{{Type: types.StringValue("SPICE"), Resolution: types.StringValue("1024x768"), Bind: types.StringValue("0.0.0.0")}}
		err := r.reconcileDisplayDevices(context.Background(), 1, plan, nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("update error", func(t *testing.T) {
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				UpdateDeviceFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMDeviceOpts) (*truenas.VMDevice, error) {
					return nil, errors.New("update failed")
				},
			}}},
		}

		plan := []VMDisplayModel{{DeviceID: types.Int64Value(50), Type: types.StringValue("SPICE"), Resolution: types.StringValue("1920x1080"), Bind: types.StringValue("0.0.0.0"), Web: types.BoolValue(true), Wait: types.BoolValue(false), Port: types.Int64Value(5900), WebPort: types.Int64Value(5901)}}
		state := []VMDisplayModel{{DeviceID: types.Int64Value(50), Type: types.StringValue("SPICE"), Resolution: types.StringValue("1024x768"), Bind: types.StringValue("0.0.0.0"), Web: types.BoolValue(true), Wait: types.BoolValue(false), Port: types.Int64Value(5900), WebPort: types.Int64Value(5901)}}
		err := r.reconcileDisplayDevices(context.Background(), 1, plan, state)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestVMResource_reconcilePCIDevices(t *testing.T) {
	t.Run("create new pci", func(t *testing.T) {
		var createdDevices int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					createdDevices++
					return &truenas.VMDevice{ID: 100}, nil
				},
			}}},
		}

		plan := []VMPCIModel{{PPTDev: types.StringValue("0000:01:00.0")}}
		err := r.reconcilePCIDevices(context.Background(), 1, plan, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if createdDevices != 1 {
			t.Fatalf("expected 1 create, got %d", createdDevices)
		}
	})

	t.Run("update changed pci", func(t *testing.T) {
		var updatedCount int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				UpdateDeviceFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMDeviceOpts) (*truenas.VMDevice, error) {
					updatedCount++
					return &truenas.VMDevice{ID: 50}, nil
				},
			}}},
		}

		plan := []VMPCIModel{{DeviceID: types.Int64Value(50), PPTDev: types.StringValue("0000:02:00.0")}}
		state := []VMPCIModel{{DeviceID: types.Int64Value(50), PPTDev: types.StringValue("0000:01:00.0")}}
		err := r.reconcilePCIDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updatedCount != 1 {
			t.Fatalf("expected 1 update, got %d", updatedCount)
		}
	})

	t.Run("create error", func(t *testing.T) {
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					return nil, errors.New("create failed")
				},
			}}},
		}

		plan := []VMPCIModel{{PPTDev: types.StringValue("0000:01:00.0")}}
		err := r.reconcilePCIDevices(context.Background(), 1, plan, nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("update error", func(t *testing.T) {
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				UpdateDeviceFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMDeviceOpts) (*truenas.VMDevice, error) {
					return nil, errors.New("update failed")
				},
			}}},
		}

		plan := []VMPCIModel{{DeviceID: types.Int64Value(50), PPTDev: types.StringValue("0000:02:00.0")}}
		state := []VMPCIModel{{DeviceID: types.Int64Value(50), PPTDev: types.StringValue("0000:01:00.0")}}
		err := r.reconcilePCIDevices(context.Background(), 1, plan, state)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestVMResource_reconcileUSBDevices(t *testing.T) {
	t.Run("create new usb", func(t *testing.T) {
		var createdDevices int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					createdDevices++
					return &truenas.VMDevice{ID: 100}, nil
				},
			}}},
		}

		plan := []VMUSBModel{{ControllerType: types.StringValue("qemu-xhci"), Device: types.StringValue("usb_0001")}}
		err := r.reconcileUSBDevices(context.Background(), 1, plan, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if createdDevices != 1 {
			t.Fatalf("expected 1 create, got %d", createdDevices)
		}
	})

	t.Run("update changed usb", func(t *testing.T) {
		var updatedCount int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				UpdateDeviceFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMDeviceOpts) (*truenas.VMDevice, error) {
					updatedCount++
					return &truenas.VMDevice{ID: 50}, nil
				},
			}}},
		}

		plan := []VMUSBModel{{DeviceID: types.Int64Value(50), ControllerType: types.StringValue("nec-xhci"), Device: types.StringValue("usb_0001")}}
		state := []VMUSBModel{{DeviceID: types.Int64Value(50), ControllerType: types.StringValue("qemu-xhci"), Device: types.StringValue("usb_0001")}}
		err := r.reconcileUSBDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updatedCount != 1 {
			t.Fatalf("expected 1 update, got %d", updatedCount)
		}
	})

	t.Run("create error", func(t *testing.T) {
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					return nil, errors.New("create failed")
				},
			}}},
		}

		plan := []VMUSBModel{{ControllerType: types.StringValue("qemu-xhci"), Device: types.StringValue("usb_0001")}}
		err := r.reconcileUSBDevices(context.Background(), 1, plan, nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("update error", func(t *testing.T) {
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				UpdateDeviceFunc: func(ctx context.Context, id int64, opts truenas.UpdateVMDeviceOpts) (*truenas.VMDevice, error) {
					return nil, errors.New("update failed")
				},
			}}},
		}

		plan := []VMUSBModel{{DeviceID: types.Int64Value(50), ControllerType: types.StringValue("nec-xhci"), Device: types.StringValue("usb_0001")}}
		state := []VMUSBModel{{DeviceID: types.Int64Value(50), ControllerType: types.StringValue("qemu-xhci"), Device: types.StringValue("usb_0001")}}
		err := r.reconcileUSBDevices(context.Background(), 1, plan, state)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// -- Equality function tests --

func TestVMResource_rawEqual(t *testing.T) {
	a := VMRawModel{
		Path: types.StringValue("/mnt/tank/vms/raw.img"),
		Type: types.StringValue("AHCI"),
		Boot: types.BoolValue(false),
		Size: types.Int64Value(1024),
	}
	b := a

	if !rawEqual(a, b) {
		t.Error("expected equal raw models to return true")
	}

	c := a
	c.Path = types.StringValue("/mnt/tank/vms/other.img")
	if rawEqual(a, c) {
		t.Error("expected different path to return false")
	}

	d := a
	d.Type = types.StringValue("VIRTIO")
	if rawEqual(a, d) {
		t.Error("expected different type to return false")
	}

	e := a
	e.Boot = types.BoolValue(true)
	if rawEqual(a, e) {
		t.Error("expected different boot to return false")
	}

	f := a
	f.Size = types.Int64Value(2048)
	if rawEqual(a, f) {
		t.Error("expected different size to return false")
	}
}

func TestVMResource_nicEqual(t *testing.T) {
	a := VMNICModel{
		Type:                types.StringValue("VIRTIO"),
		NICAttach:           types.StringValue("br0"),
		MAC:                 types.StringValue("00:aa:bb:cc:dd:ee"),
		TrustGuestRXFilters: types.BoolValue(false),
	}
	b := a

	if !nicEqual(a, b) {
		t.Error("expected equal nic models to return true")
	}

	c := a
	c.Type = types.StringValue("E1000")
	if nicEqual(a, c) {
		t.Error("expected different type to return false")
	}

	d := a
	d.NICAttach = types.StringValue("br1")
	if nicEqual(a, d) {
		t.Error("expected different nic_attach to return false")
	}

	e := a
	e.MAC = types.StringValue("ff:ff:ff:ff:ff:ff")
	if nicEqual(a, e) {
		t.Error("expected different mac to return false")
	}

	f := a
	f.TrustGuestRXFilters = types.BoolValue(true)
	if nicEqual(a, f) {
		t.Error("expected different trust_guest_rx_filters to return false")
	}
}

func TestVMResource_displayEqual(t *testing.T) {
	a := VMDisplayModel{
		Type:       types.StringValue("SPICE"),
		Resolution: types.StringValue("1920x1080"),
		Bind:       types.StringValue("0.0.0.0"),
		Web:        types.BoolValue(true),
		Wait:       types.BoolValue(false),
		Port:       types.Int64Value(5900),
		WebPort:    types.Int64Value(5901),
	}
	b := a

	if !displayEqual(a, b) {
		t.Error("expected equal display models to return true")
	}

	c := a
	c.Resolution = types.StringValue("1024x768")
	if displayEqual(a, c) {
		t.Error("expected different resolution to return false")
	}

	d := a
	d.Port = types.Int64Value(5910)
	if displayEqual(a, d) {
		t.Error("expected different port to return false")
	}

	e := a
	e.WebPort = types.Int64Value(5911)
	if displayEqual(a, e) {
		t.Error("expected different web_port to return false")
	}

	f := a
	f.Web = types.BoolValue(false)
	if displayEqual(a, f) {
		t.Error("expected different web to return false")
	}

	g := a
	g.Wait = types.BoolValue(true)
	if displayEqual(a, g) {
		t.Error("expected different wait to return false")
	}

	h := a
	h.Bind = types.StringValue("127.0.0.1")
	if displayEqual(a, h) {
		t.Error("expected different bind to return false")
	}

	i := a
	i.Type = types.StringValue("VNC")
	if displayEqual(a, i) {
		t.Error("expected different type to return false")
	}
}

func TestVMResource_usbEqual(t *testing.T) {
	a := VMUSBModel{
		ControllerType: types.StringValue("qemu-xhci"),
		Device:         types.StringValue("usb_0001"),
	}
	b := a

	if !usbEqual(a, b) {
		t.Error("expected equal usb models to return true")
	}

	c := a
	c.ControllerType = types.StringValue("nec-xhci")
	if usbEqual(a, c) {
		t.Error("expected different controller_type to return false")
	}

	d := a
	d.Device = types.StringValue("usb_0002")
	if usbEqual(a, d) {
		t.Error("expected different device to return false")
	}
}

// -- collectDeviceIDs tests --

func TestVMResource_collectDeviceIDs_AllTypes(t *testing.T) {
	ids := make(map[int64]bool)
	data := &VMResourceModel{
		Disks: []VMDiskModel{
			{DeviceID: types.Int64Value(10)},
			{DeviceID: types.Int64Value(11)},
		},
		Raws: []VMRawModel{
			{DeviceID: types.Int64Value(20)},
		},
		CDROMs: []VMCDROMModel{
			{DeviceID: types.Int64Value(30)},
		},
		NICs: []VMNICModel{
			{DeviceID: types.Int64Value(40)},
		},
		Displays: []VMDisplayModel{
			{DeviceID: types.Int64Value(50)},
		},
		PCIs: []VMPCIModel{
			{DeviceID: types.Int64Value(60)},
		},
		USBs: []VMUSBModel{
			{DeviceID: types.Int64Value(70)},
		},
	}

	collectDeviceIDs(ids, data)

	expectedIDs := []int64{10, 11, 20, 30, 40, 50, 60, 70}
	for _, id := range expectedIDs {
		if !ids[id] {
			t.Errorf("expected device ID %d to be collected", id)
		}
	}
	if len(ids) != len(expectedIDs) {
		t.Errorf("expected %d IDs, got %d", len(expectedIDs), len(ids))
	}
}

func TestVMResource_collectDeviceIDs_SkipsNullAndUnknown(t *testing.T) {
	ids := make(map[int64]bool)
	data := &VMResourceModel{
		Disks: []VMDiskModel{
			{DeviceID: types.Int64Null()},
			{DeviceID: types.Int64Unknown()},
			{DeviceID: types.Int64Value(10)},
		},
	}

	collectDeviceIDs(ids, data)

	if len(ids) != 1 {
		t.Errorf("expected 1 ID, got %d", len(ids))
	}
	if !ids[10] {
		t.Error("expected device ID 10 to be collected")
	}
}


// -- reconcileDevices dispatching to non-disk types --

func TestVMResource_reconcileDevices_NonDiskTypes(t *testing.T) {
	t.Run("dispatches to raw reconciler", func(t *testing.T) {
		var createdDevices int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					createdDevices++
					return &truenas.VMDevice{ID: 100}, nil
				},
			}}},
		}

		plan := &VMResourceModel{
			Raws: []VMRawModel{{
				Path:   types.StringValue("/mnt/tank/vms/raw.img"),
				Type:   types.StringValue("AHCI"),
				Boot:   types.BoolValue(false),
				IOType: types.StringValue("THREADS"),
			}},
		}
		state := &VMResourceModel{}

		err := r.reconcileDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if createdDevices != 1 {
			t.Errorf("expected 1 raw device create, got %d", createdDevices)
		}
	})

	t.Run("dispatches to cdrom reconciler", func(t *testing.T) {
		var createdDevices int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					createdDevices++
					return &truenas.VMDevice{ID: 100}, nil
				},
			}}},
		}

		plan := &VMResourceModel{
			CDROMs: []VMCDROMModel{{Path: types.StringValue("/mnt/tank/iso/test.iso")}},
		}
		state := &VMResourceModel{}

		err := r.reconcileDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if createdDevices != 1 {
			t.Errorf("expected 1 cdrom device create, got %d", createdDevices)
		}
	})

	t.Run("dispatches to nic reconciler", func(t *testing.T) {
		var createdDevices int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					createdDevices++
					return &truenas.VMDevice{ID: 100}, nil
				},
			}}},
		}

		plan := &VMResourceModel{
			NICs: []VMNICModel{{Type: types.StringValue("VIRTIO"), NICAttach: types.StringValue("br0")}},
		}
		state := &VMResourceModel{}

		err := r.reconcileDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if createdDevices != 1 {
			t.Errorf("expected 1 nic device create, got %d", createdDevices)
		}
	})

	t.Run("dispatches to display reconciler", func(t *testing.T) {
		var createdDevices int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					createdDevices++
					return &truenas.VMDevice{ID: 100}, nil
				},
			}}},
		}

		plan := &VMResourceModel{
			Displays: []VMDisplayModel{{Type: types.StringValue("SPICE"), Resolution: types.StringValue("1024x768"), Bind: types.StringValue("0.0.0.0")}},
		}
		state := &VMResourceModel{}

		err := r.reconcileDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if createdDevices != 1 {
			t.Errorf("expected 1 display device create, got %d", createdDevices)
		}
	})

	t.Run("dispatches to pci reconciler", func(t *testing.T) {
		var createdDevices int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					createdDevices++
					return &truenas.VMDevice{ID: 100}, nil
				},
			}}},
		}

		plan := &VMResourceModel{
			PCIs: []VMPCIModel{{PPTDev: types.StringValue("0000:01:00.0")}},
		}
		state := &VMResourceModel{}

		err := r.reconcileDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if createdDevices != 1 {
			t.Errorf("expected 1 pci device create, got %d", createdDevices)
		}
	})

	t.Run("dispatches to usb reconciler", func(t *testing.T) {
		var createdDevices int
		r := &VMResource{
			BaseResource: BaseResource{services: &services.TrueNASServices{VM: &truenas.MockVMService{
				CreateDeviceFunc: func(ctx context.Context, opts truenas.CreateVMDeviceOpts) (*truenas.VMDevice, error) {
					createdDevices++
					return &truenas.VMDevice{ID: 100}, nil
				},
			}}},
		}

		plan := &VMResourceModel{
			USBs: []VMUSBModel{{ControllerType: types.StringValue("qemu-xhci"), Device: types.StringValue("usb_0001")}},
		}
		state := &VMResourceModel{}

		err := r.reconcileDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if createdDevices != 1 {
			t.Errorf("expected 1 usb device create, got %d", createdDevices)
		}
	})
}

// -- diskEqual tests --

func TestVMResource_diskEqual(t *testing.T) {
	a := VMDiskModel{
		Path:               types.StringValue("/dev/zvol/tank/vms/disk0"),
		Type:               types.StringValue("VIRTIO"),
		LogicalSectorSize:  types.Int64Null(),
		PhysicalSectorSize: types.Int64Null(),
		IOType:             types.StringValue("THREADS"),
		Serial:             types.StringNull(),
	}
	b := a

	if !diskEqual(a, b) {
		t.Error("expected equal disk models to return true")
	}

	c := a
	c.Path = types.StringValue("/dev/zvol/tank/vms/disk1")
	if diskEqual(a, c) {
		t.Error("expected different path to return false")
	}

	d := a
	d.IOType = types.StringValue("NATIVE")
	if diskEqual(a, d) {
		t.Error("expected different iotype to return false")
	}

	e := a
	e.LogicalSectorSize = types.Int64Value(512)
	if diskEqual(a, e) {
		t.Error("expected different logical_sectorsize to return false")
	}
}
