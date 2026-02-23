package resources

import (
	"context"
	"fmt"
	"strconv"

	truenas "github.com/deevus/truenas-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// VM state constants matching TrueNAS API values.
const (
	VMStateRunning = "RUNNING"
	VMStateStopped = "STOPPED"
)

var (
	_ resource.Resource                = &VMResource{}
	_ resource.ResourceWithConfigure   = &VMResource{}
	_ resource.ResourceWithImportState = &VMResource{}
)

// VMResourceModel describes the resource data model.
type VMResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	VCPUs            types.Int64  `tfsdk:"vcpus"`
	Cores            types.Int64  `tfsdk:"cores"`
	Threads          types.Int64  `tfsdk:"threads"`
	Memory           types.Int64  `tfsdk:"memory"`
	MinMemory        types.Int64  `tfsdk:"min_memory"`
	Autostart        types.Bool   `tfsdk:"autostart"`
	Time             types.String `tfsdk:"time"`
	Bootloader       types.String `tfsdk:"bootloader"`
	BootloaderOVMF   types.String `tfsdk:"bootloader_ovmf"`
	CPUMode          types.String `tfsdk:"cpu_mode"`
	CPUModel         types.String `tfsdk:"cpu_model"`
	ShutdownTimeout  types.Int64  `tfsdk:"shutdown_timeout"`
	CommandLineArgs  types.String `tfsdk:"command_line_args"`
	State            types.String `tfsdk:"state"`
	DisplayAvailable types.Bool   `tfsdk:"display_available"`
	// Device blocks
	Disks    []VMDiskModel    `tfsdk:"disk"`
	Raws     []VMRawModel     `tfsdk:"raw"`
	CDROMs   []VMCDROMModel   `tfsdk:"cdrom"`
	NICs     []VMNICModel     `tfsdk:"nic"`
	Displays []VMDisplayModel `tfsdk:"display"`
	PCIs     []VMPCIModel     `tfsdk:"pci"`
	USBs     []VMUSBModel     `tfsdk:"usb"`
}

// VMDiskModel represents a DISK device.
type VMDiskModel struct {
	DeviceID           types.Int64  `tfsdk:"device_id"`
	Path               types.String `tfsdk:"path"`
	Type               types.String `tfsdk:"type"`
	LogicalSectorSize  types.Int64  `tfsdk:"logical_sectorsize"`
	PhysicalSectorSize types.Int64  `tfsdk:"physical_sectorsize"`
	IOType             types.String `tfsdk:"iotype"`
	Serial             types.String `tfsdk:"serial"`
	Order              types.Int64  `tfsdk:"order"`
}

// VMRawModel represents a RAW device.
type VMRawModel struct {
	DeviceID           types.Int64  `tfsdk:"device_id"`
	Path               types.String `tfsdk:"path"`
	Type               types.String `tfsdk:"type"`
	Boot               types.Bool   `tfsdk:"boot"`
	Exists             types.Bool   `tfsdk:"exists"`
	Size               types.Int64  `tfsdk:"size"`
	LogicalSectorSize  types.Int64  `tfsdk:"logical_sectorsize"`
	PhysicalSectorSize types.Int64  `tfsdk:"physical_sectorsize"`
	IOType             types.String `tfsdk:"iotype"`
	Serial             types.String `tfsdk:"serial"`
	Order              types.Int64  `tfsdk:"order"`
}

// VMCDROMModel represents a CDROM device.
type VMCDROMModel struct {
	DeviceID types.Int64  `tfsdk:"device_id"`
	Path     types.String `tfsdk:"path"`
	Order    types.Int64  `tfsdk:"order"`
}

// VMNICModel represents a NIC device.
type VMNICModel struct {
	DeviceID            types.Int64  `tfsdk:"device_id"`
	Type                types.String `tfsdk:"type"`
	NICAttach           types.String `tfsdk:"nic_attach"`
	MAC                 types.String `tfsdk:"mac"`
	TrustGuestRXFilters types.Bool   `tfsdk:"trust_guest_rx_filters"`
	Order               types.Int64  `tfsdk:"order"`
}

// VMDisplayModel represents a DISPLAY device.
type VMDisplayModel struct {
	DeviceID   types.Int64  `tfsdk:"device_id"`
	Type       types.String `tfsdk:"type"`
	Resolution types.String `tfsdk:"resolution"`
	Port       types.Int64  `tfsdk:"port"`
	WebPort    types.Int64  `tfsdk:"web_port"`
	Bind       types.String `tfsdk:"bind"`
	Wait       types.Bool   `tfsdk:"wait"`
	Password   types.String `tfsdk:"password"`
	Web        types.Bool   `tfsdk:"web"`
	Order      types.Int64  `tfsdk:"order"`
}

// VMPCIModel represents a PCI passthrough device.
type VMPCIModel struct {
	DeviceID types.Int64  `tfsdk:"device_id"`
	PPTDev   types.String `tfsdk:"pptdev"`
	Order    types.Int64  `tfsdk:"order"`
}

// VMUSBModel represents a USB passthrough device.
type VMUSBModel struct {
	DeviceID       types.Int64  `tfsdk:"device_id"`
	ControllerType types.String `tfsdk:"controller_type"`
	Device         types.String `tfsdk:"device"`
	Order          types.Int64  `tfsdk:"order"`
}

// VMResource defines the resource implementation.
type VMResource struct {
	BaseResource
}

// NewVMResource creates a new VMResource.
func NewVMResource() resource.Resource {
	return &VMResource{}
}

func (r *VMResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (r *VMResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a QEMU/KVM virtual machine on TrueNAS.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "VM ID (numeric, stored as string for Terraform compatibility).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "VM name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "VM description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"vcpus": schema.Int64Attribute{
				Description: "Number of virtual CPU sockets. Defaults to 1.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Validators: []validator.Int64{
					int64validator.Between(1, 16),
				},
			},
			"cores": schema.Int64Attribute{
				Description: "CPU cores per socket. Defaults to 1.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"threads": schema.Int64Attribute{
				Description: "Threads per core. Defaults to 1.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"memory": schema.Int64Attribute{
				Description: "Memory in MB (minimum 20).",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(20),
				},
			},
			"min_memory": schema.Int64Attribute{
				Description: "Minimum memory for ballooning in MB. Null to disable.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(20),
				},
			},
			"autostart": schema.BoolAttribute{
				Description: "Start VM on boot. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"time": schema.StringAttribute{
				Description: "Clock type: LOCAL or UTC. Defaults to LOCAL.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("LOCAL"),
				Validators: []validator.String{
					stringvalidator.OneOf("LOCAL", "UTC"),
				},
			},
			"bootloader": schema.StringAttribute{
				Description: "Bootloader type: UEFI or UEFI_CSM. Defaults to UEFI.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("UEFI"),
				Validators: []validator.String{
					stringvalidator.OneOf("UEFI", "UEFI_CSM"),
				},
			},
			"bootloader_ovmf": schema.StringAttribute{
				Description: "OVMF firmware file. Defaults to OVMF_CODE.fd.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("OVMF_CODE.fd"),
			},
			"cpu_mode": schema.StringAttribute{
				Description: "CPU mode: CUSTOM, HOST-MODEL, or HOST-PASSTHROUGH. Defaults to CUSTOM.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("CUSTOM"),
				Validators: []validator.String{
					stringvalidator.OneOf("CUSTOM", "HOST-MODEL", "HOST-PASSTHROUGH"),
				},
			},
			"cpu_model": schema.StringAttribute{
				Description: "CPU model name (when cpu_mode is CUSTOM).",
				Optional:    true,
			},
			"shutdown_timeout": schema.Int64Attribute{
				Description: "Shutdown timeout in seconds (5-300). Defaults to 90.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(90),
				Validators: []validator.Int64{
					int64validator.Between(5, 300),
				},
			},
			"command_line_args": schema.StringAttribute{
				Description: "Extra QEMU command line arguments.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"state": schema.StringAttribute{
				Description: "Desired VM power state: RUNNING or STOPPED. Defaults to STOPPED.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(VMStateStopped),
				Validators: []validator.String{
					stringvalidator.OneOf(VMStateRunning, VMStateStopped),
				},
			},
			"display_available": schema.BoolAttribute{
				Description:   "Whether a display device is available.",
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			"disk": schema.ListNestedBlock{
				Description: "DISK devices (zvol block devices).",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{
							Description:   "Device ID assigned by TrueNAS.",
							Computed:      true,
							PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
						},
						"path": schema.StringAttribute{
							Description: "Path to zvol device (e.g., /dev/zvol/tank/vms/disk0).",
							Required:    true,
						},
						"type": schema.StringAttribute{
							Description: "Disk bus type: AHCI or VIRTIO. Defaults to AHCI.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("AHCI"),
							Validators: []validator.String{
								stringvalidator.OneOf("AHCI", "VIRTIO"),
							},
						},
						"logical_sectorsize": schema.Int64Attribute{
							Description: "Logical sector size: 512 or 4096.",
							Optional:    true,
							Validators: []validator.Int64{
								int64validator.OneOf(512, 4096),
							},
						},
						"physical_sectorsize": schema.Int64Attribute{
							Description: "Physical sector size: 512 or 4096.",
							Optional:    true,
							Validators: []validator.Int64{
								int64validator.OneOf(512, 4096),
							},
						},
						"iotype": schema.StringAttribute{
							Description: "I/O type: NATIVE, THREADS, or IO_URING. Defaults to THREADS.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("THREADS"),
							Validators: []validator.String{
								stringvalidator.OneOf("NATIVE", "THREADS", "IO_URING"),
							},
						},
						"serial": schema.StringAttribute{
							Description:   "Disk serial number. Auto-generated if not set.",
							Optional:      true,
							Computed:      true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						},
						"order": schema.Int64Attribute{
							Description:   "Device boot/load order.",
							Optional:      true,
							Computed:      true,
							PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
						},
					},
				},
			},
			"raw": schema.ListNestedBlock{
				Description: "RAW file devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{Computed: true, Description: "Device ID assigned by TrueNAS.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
						"path":      schema.StringAttribute{Required: true, Description: "Path to raw file."},
						"type": schema.StringAttribute{
							Optional: true, Computed: true, Default: stringdefault.StaticString("AHCI"),
							Description: "Disk bus type: AHCI or VIRTIO. Defaults to AHCI.",
							Validators:  []validator.String{stringvalidator.OneOf("AHCI", "VIRTIO")},
						},
						"boot": schema.BoolAttribute{
							Optional: true, Computed: true, Default: booldefault.StaticBool(false),
							Description: "Bootable device. Defaults to false.",
						},
						"exists": schema.BoolAttribute{
							Optional:    true,
							Description: "Set to true when the file at path already exists. When false (default), the API creates the raw file.",
						},
						"size":                schema.Int64Attribute{Optional: true, Description: "File size in bytes (for creation)."},
						"logical_sectorsize":  schema.Int64Attribute{Optional: true, Description: "Logical sector size: 512 or 4096.", Validators: []validator.Int64{int64validator.OneOf(512, 4096)}},
						"physical_sectorsize": schema.Int64Attribute{Optional: true, Description: "Physical sector size: 512 or 4096.", Validators: []validator.Int64{int64validator.OneOf(512, 4096)}},
						"iotype": schema.StringAttribute{
							Optional: true, Computed: true, Default: stringdefault.StaticString("THREADS"),
							Description: "I/O type: NATIVE, THREADS, or IO_URING. Defaults to THREADS.",
							Validators:  []validator.String{stringvalidator.OneOf("NATIVE", "THREADS", "IO_URING")},
						},
						"serial": schema.StringAttribute{Optional: true, Computed: true, Description: "Disk serial number. Auto-generated if not set.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
						"order":  schema.Int64Attribute{Optional: true, Computed: true, Description: "Device boot/load order.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
					},
				},
			},
			"cdrom": schema.ListNestedBlock{
				Description: "CD-ROM/ISO devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{Computed: true, Description: "Device ID assigned by TrueNAS.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
						"path":      schema.StringAttribute{Required: true, Description: "Path to ISO file (must start with /mnt/)."},
						"order":     schema.Int64Attribute{Optional: true, Computed: true, Description: "Device boot/load order.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
					},
				},
			},
			"nic": schema.ListNestedBlock{
				Description: "Network interface devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{Computed: true, Description: "Device ID assigned by TrueNAS.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
						"type": schema.StringAttribute{
							Optional: true, Computed: true, Default: stringdefault.StaticString("E1000"),
							Description: "NIC emulation type: E1000 or VIRTIO. Defaults to E1000.",
							Validators:  []validator.String{stringvalidator.OneOf("E1000", "VIRTIO")},
						},
						"nic_attach":             schema.StringAttribute{Optional: true, Description: "Host interface to attach to."},
						"mac":                    schema.StringAttribute{Optional: true, Computed: true, Description: "MAC address (auto-generated if not set).", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
						"trust_guest_rx_filters": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Trust guest RX filters. Defaults to false."},
						"order":                  schema.Int64Attribute{Optional: true, Computed: true, Description: "Device boot/load order.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
					},
				},
			},
			"display": schema.ListNestedBlock{
				Description: "SPICE display devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{Computed: true, Description: "Device ID assigned by TrueNAS.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
						"type": schema.StringAttribute{
							Optional: true, Computed: true, Default: stringdefault.StaticString("SPICE"),
							Description: "Display protocol. Currently only SPICE.",
							Validators:  []validator.String{stringvalidator.OneOf("SPICE")},
						},
						"resolution": schema.StringAttribute{
							Optional: true, Computed: true, Default: stringdefault.StaticString("1024x768"),
							Description: "Screen resolution. Defaults to 1024x768.",
							Validators: []validator.String{stringvalidator.OneOf(
								"1920x1200", "1920x1080", "1600x1200", "1600x900",
								"1400x1050", "1280x1024", "1280x720", "1024x768",
								"800x600", "640x480",
							)},
						},
						"port":     schema.Int64Attribute{Optional: true, Computed: true, Description: "SPICE port (auto-assigned if not set). Range 5900-65535.", Validators: []validator.Int64{int64validator.Between(5900, 65535)}, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
						"web_port": schema.Int64Attribute{Optional: true, Computed: true, Description: "Web client port (auto-assigned if not set).", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
						"bind": schema.StringAttribute{
							Optional: true, Computed: true, Default: stringdefault.StaticString("127.0.0.1"),
							Description: "Bind address. Defaults to 127.0.0.1.",
						},
						"wait":     schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Wait for client before booting. Defaults to false."},
						"password": schema.StringAttribute{Required: true, Sensitive: true, Description: "Connection password. Required by TrueNAS for display devices."},
						"web":      schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Enable web client. Defaults to true."},
						"order":    schema.Int64Attribute{Optional: true, Computed: true, Description: "Device boot/load order.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
					},
				},
			},
			"pci": schema.ListNestedBlock{
				Description: "PCI passthrough devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{Computed: true, Description: "Device ID assigned by TrueNAS.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
						"pptdev":    schema.StringAttribute{Required: true, Description: "PCI device address (e.g., 0000:01:00.0)."},
						"order":     schema.Int64Attribute{Optional: true, Computed: true, Description: "Device boot/load order.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
					},
				},
			},
			"usb": schema.ListNestedBlock{
				Description: "USB passthrough devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{Computed: true, Description: "Device ID assigned by TrueNAS.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
						"controller_type": schema.StringAttribute{
							Optional: true, Computed: true, Default: stringdefault.StaticString("nec-xhci"),
							Description: "USB controller type. Defaults to nec-xhci.",
							Validators: []validator.String{stringvalidator.OneOf(
								"piix3-uhci", "piix4-uhci", "ehci", "ich9-ehci1",
								"vt82c686b-uhci", "pci-ohci", "nec-xhci", "qemu-xhci",
							)},
						},
						"device": schema.StringAttribute{Optional: true, Description: "USB device identifier."},
						"order":  schema.Int64Attribute{Optional: true, Computed: true, Description: "Device boot/load order.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
					},
				},
			},
		},
	}
}

// -- CRUD --

func (r *VMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VMResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := r.buildCreateOpts(&data)
	vm, err := r.services.VM.CreateVM(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create VM", fmt.Sprintf("Unable to create VM %q: %s", data.Name.ValueString(), err.Error()))
		return
	}
	vmID := vm.ID

	// Create devices
	for i := range data.Disks {
		dev, err := r.services.VM.CreateDevice(ctx, buildDiskDeviceOpts(&data.Disks[i], vmID))
		if err != nil {
			resp.Diagnostics.AddError("Unable to Create Disk Device", err.Error())
			return
		}
		data.Disks[i].DeviceID = types.Int64Value(dev.ID)
	}
	for i := range data.Raws {
		dev, err := r.services.VM.CreateDevice(ctx, buildRawDeviceOpts(&data.Raws[i], vmID))
		if err != nil {
			resp.Diagnostics.AddError("Unable to Create Raw Device", err.Error())
			return
		}
		data.Raws[i].DeviceID = types.Int64Value(dev.ID)
	}
	for i := range data.CDROMs {
		dev, err := r.services.VM.CreateDevice(ctx, buildCDROMDeviceOpts(&data.CDROMs[i], vmID))
		if err != nil {
			resp.Diagnostics.AddError("Unable to Create CDROM Device", err.Error())
			return
		}
		data.CDROMs[i].DeviceID = types.Int64Value(dev.ID)
	}
	for i := range data.NICs {
		dev, err := r.services.VM.CreateDevice(ctx, buildNICDeviceOpts(&data.NICs[i], vmID))
		if err != nil {
			resp.Diagnostics.AddError("Unable to Create NIC Device", err.Error())
			return
		}
		data.NICs[i].DeviceID = types.Int64Value(dev.ID)
	}
	for i := range data.Displays {
		dev, err := r.services.VM.CreateDevice(ctx, buildDisplayDeviceOpts(&data.Displays[i], vmID))
		if err != nil {
			resp.Diagnostics.AddError("Unable to Create Display Device", err.Error())
			return
		}
		data.Displays[i].DeviceID = types.Int64Value(dev.ID)
	}
	for i := range data.PCIs {
		dev, err := r.services.VM.CreateDevice(ctx, buildPCIDeviceOpts(&data.PCIs[i], vmID))
		if err != nil {
			resp.Diagnostics.AddError("Unable to Create PCI Device", err.Error())
			return
		}
		data.PCIs[i].DeviceID = types.Int64Value(dev.ID)
	}
	for i := range data.USBs {
		dev, err := r.services.VM.CreateDevice(ctx, buildUSBDeviceOpts(&data.USBs[i], vmID))
		if err != nil {
			resp.Diagnostics.AddError("Unable to Create USB Device", err.Error())
			return
		}
		data.USBs[i].DeviceID = types.Int64Value(dev.ID)
	}

	// Handle desired state
	desiredState := data.State.ValueString()
	if desiredState == VMStateRunning {
		if err := r.reconcileState(ctx, vmID, VMStateStopped, VMStateRunning); err != nil {
			resp.Diagnostics.AddError("Unable to Start VM", err.Error())
			return
		}
	}

	// Read back to get final state
	freshVM, err := r.services.VM.GetVM(ctx, vmID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read VM After Create", err.Error())
		return
	}
	r.mapVMToModel(freshVM, &data)

	// Read devices
	devices, err := r.services.VM.ListDevices(ctx, vmID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Query VM Devices", err.Error())
		return
	}
	priorRaws := data.Raws
	r.mapDevicesToModel(devices, &data)
	preserveRawExists(data.Raws, priorRaws)

	// Restore desired state (mapVMToModel sets state from API status)
	data.State = types.StringValue(desiredState)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VMResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid VM ID", fmt.Sprintf("Cannot parse VM ID %q: %s", data.ID.ValueString(), err.Error()))
		return
	}

	// Preserve user-specified desired state
	priorState := data.State

	vm, err := r.services.VM.GetVM(ctx, vmID)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to Read VM", err.Error())
		return
	}

	r.mapVMToModel(vm, &data)

	// Read devices
	devices, err := r.services.VM.ListDevices(ctx, vmID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Query VM Devices", err.Error())
		return
	}
	priorRaws := data.Raws
	r.mapDevicesToModel(devices, &data)
	preserveRawExists(data.Raws, priorRaws)

	// Restore desired state from prior state (user-specified)
	if !priorState.IsNull() && !priorState.IsUnknown() {
		data.State = priorState
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VMResourceModel
	var stateData VMResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid VM ID", err.Error())
		return
	}

	// Build update opts (check if anything changed)
	updateOpts, changed := r.buildUpdateOpts(&data, &stateData)
	if changed {
		_, err := r.services.VM.UpdateVM(ctx, vmID, *updateOpts)
		if err != nil {
			resp.Diagnostics.AddError("Unable to Update VM", err.Error())
			return
		}
	}

	// Reconcile devices
	if err := r.reconcileDevices(ctx, vmID, &data, &stateData); err != nil {
		resp.Diagnostics.AddError("Unable to Update VM Devices", err.Error())
		return
	}

	// Handle state transitions
	currentState := stateData.State.ValueString()
	desiredState := data.State.ValueString()
	if currentState != desiredState {
		// Get actual current state from API
		vm, err := r.services.VM.GetVM(ctx, vmID)
		if err != nil {
			resp.Diagnostics.AddError("Unable to Query VM State", err.Error())
			return
		}
		if err := r.reconcileState(ctx, vmID, vm.State, desiredState); err != nil {
			resp.Diagnostics.AddError("Unable to Reconcile VM State", err.Error())
			return
		}
	}

	// Read back fresh state
	freshVM, err := r.services.VM.GetVM(ctx, vmID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read VM After Update", err.Error())
		return
	}
	r.mapVMToModel(freshVM, &data)

	devices, err := r.services.VM.ListDevices(ctx, vmID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Query VM Devices", err.Error())
		return
	}
	priorRaws := data.Raws
	r.mapDevicesToModel(devices, &data)
	preserveRawExists(data.Raws, priorRaws)

	// Restore desired state
	data.State = types.StringValue(desiredState)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VMResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid VM ID", err.Error())
		return
	}

	// Check current state - if running, stop first
	vm, err := r.services.VM.GetVM(ctx, vmID)
	if err != nil {
		if isNotFoundError(err) {
			return // Already deleted
		}
		resp.Diagnostics.AddError("Unable to Query VM State", err.Error())
		return
	}

	if vm.State == VMStateRunning {
		err := r.services.VM.StopVM(ctx, vmID, truenas.StopVMOpts{Force: true, ForceAfterTimeout: true})
		if err != nil {
			resp.Diagnostics.AddError("Unable to Stop VM", fmt.Sprintf("Unable to stop VM before delete: %s", err.Error()))
			return
		}
	}

	// Delete the VM
	err = r.services.VM.DeleteVM(ctx, vmID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Delete VM", err.Error())
		return
	}
}

