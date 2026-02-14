package resources

import (
	"context"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
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
	DeviceID              types.Int64  `tfsdk:"device_id"`
	Type                  types.String `tfsdk:"type"`
	NICAttach             types.String `tfsdk:"nic_attach"`
	MAC                   types.String `tfsdk:"mac"`
	TrustGuestRXFilters   types.Bool   `tfsdk:"trust_guest_rx_filters"`
	Order                 types.Int64  `tfsdk:"order"`
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
	client client.Client
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
				Description: "Whether a display device is available.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"disk": schema.ListNestedBlock{
				Description: "DISK devices (zvol block devices).",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{
							Description: "Device ID assigned by TrueNAS.",
							Computed:    true,
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
							Description: "Disk serial number.",
							Optional:    true,
						},
						"order": schema.Int64Attribute{
							Description: "Device boot/load order.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"raw": schema.ListNestedBlock{
				Description: "RAW file devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{
							Description: "Device ID assigned by TrueNAS.",
							Computed:    true,
						},
						"path": schema.StringAttribute{
							Description: "Path to raw file.",
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
						"boot": schema.BoolAttribute{
							Description: "Bootable device. Defaults to false.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"size": schema.Int64Attribute{
							Description: "File size in bytes (for creation).",
							Optional:    true,
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
							Description: "Disk serial number.",
							Optional:    true,
						},
						"order": schema.Int64Attribute{
							Description: "Device boot/load order.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"cdrom": schema.ListNestedBlock{
				Description: "CD-ROM/ISO devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{
							Description: "Device ID assigned by TrueNAS.",
							Computed:    true,
						},
						"path": schema.StringAttribute{
							Description: "Path to ISO file (must start with /mnt/).",
							Required:    true,
						},
						"order": schema.Int64Attribute{
							Description: "Device boot/load order.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"nic": schema.ListNestedBlock{
				Description: "Network interface devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{
							Description: "Device ID assigned by TrueNAS.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "NIC emulation type: E1000 or VIRTIO. Defaults to E1000.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("E1000"),
							Validators: []validator.String{
								stringvalidator.OneOf("E1000", "VIRTIO"),
							},
						},
						"nic_attach": schema.StringAttribute{
							Description: "Host interface to attach to.",
							Optional:    true,
						},
						"mac": schema.StringAttribute{
							Description: "MAC address (auto-generated if not set).",
							Optional:    true,
							Computed:    true,
						},
						"trust_guest_rx_filters": schema.BoolAttribute{
							Description: "Trust guest RX filters. Defaults to false.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"order": schema.Int64Attribute{
							Description: "Device boot/load order.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"display": schema.ListNestedBlock{
				Description: "SPICE display devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{
							Description: "Device ID assigned by TrueNAS.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "Display protocol. Currently only SPICE.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("SPICE"),
							Validators: []validator.String{
								stringvalidator.OneOf("SPICE"),
							},
						},
						"resolution": schema.StringAttribute{
							Description: "Screen resolution. Defaults to 1024x768.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("1024x768"),
							Validators: []validator.String{
								stringvalidator.OneOf(
									"1920x1200", "1920x1080", "1600x1200", "1600x900",
									"1400x1050", "1280x1024", "1280x720", "1024x768",
									"800x600", "640x480",
								),
							},
						},
						"port": schema.Int64Attribute{
							Description: "SPICE port (auto-assigned if not set). Range 5900-65535.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.Int64{
								int64validator.Between(5900, 65535),
							},
						},
						"web_port": schema.Int64Attribute{
							Description: "Web client port (auto-assigned if not set).",
							Optional:    true,
							Computed:    true,
						},
						"bind": schema.StringAttribute{
							Description: "Bind address. Defaults to 127.0.0.1.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("127.0.0.1"),
						},
						"wait": schema.BoolAttribute{
							Description: "Wait for client before booting. Defaults to false.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"password": schema.StringAttribute{
							Description: "Connection password.",
							Optional:    true,
							Sensitive:   true,
						},
						"web": schema.BoolAttribute{
							Description: "Enable web client. Defaults to true.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(true),
						},
						"order": schema.Int64Attribute{
							Description: "Device boot/load order.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"pci": schema.ListNestedBlock{
				Description: "PCI passthrough devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{
							Description: "Device ID assigned by TrueNAS.",
							Computed:    true,
						},
						"pptdev": schema.StringAttribute{
							Description: "PCI device address (e.g., 0000:01:00.0).",
							Required:    true,
						},
						"order": schema.Int64Attribute{
							Description: "Device boot/load order.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"usb": schema.ListNestedBlock{
				Description: "USB passthrough devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{
							Description: "Device ID assigned by TrueNAS.",
							Computed:    true,
						},
						"controller_type": schema.StringAttribute{
							Description: "USB controller type. Defaults to nec-xhci.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("nec-xhci"),
							Validators: []validator.String{
								stringvalidator.OneOf(
									"piix3-uhci", "piix4-uhci", "ehci", "ich9-ehci1",
									"vt82c686b-uhci", "pci-ohci", "nec-xhci", "qemu-xhci",
								),
							},
						},
						"device": schema.StringAttribute{
							Description: "USB device identifier.",
							Optional:    true,
						},
						"order": schema.Int64Attribute{
							Description: "Device boot/load order.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *VMResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *VMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("Not Implemented", "Create is not yet implemented")
}

func (r *VMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("Not Implemented", "Read is not yet implemented")
}

func (r *VMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Not Implemented", "Update is not yet implemented")
}

func (r *VMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("Not Implemented", "Delete is not yet implemented")
}

func (r *VMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
