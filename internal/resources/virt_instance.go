package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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

// VirtInstance state constants matching TrueNAS API values.
const (
	VirtInstanceStateRunning  = "RUNNING"
	VirtInstanceStateStopped  = "STOPPED"
	VirtInstanceStateStarting = "STARTING"
	VirtInstanceStateStopping = "STOPPING"
)

var (
	_ resource.Resource                = &VirtInstanceResource{}
	_ resource.ResourceWithConfigure   = &VirtInstanceResource{}
	_ resource.ResourceWithImportState = &VirtInstanceResource{}
)

// VirtInstanceResourceModel describes the resource data model.
type VirtInstanceResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	StoragePool     types.String `tfsdk:"storage_pool"`
	ImageName       types.String `tfsdk:"image_name"`
	ImageVersion    types.String `tfsdk:"image_version"`
	Autostart       types.Bool   `tfsdk:"autostart"`
	DesiredState    types.String `tfsdk:"desired_state"`
	StateTimeout    types.Int64  `tfsdk:"state_timeout"`
	State           types.String `tfsdk:"state"`
	UUID            types.String `tfsdk:"uuid"`
	ShutdownTimeout types.Int64  `tfsdk:"shutdown_timeout"`
	Addresses       types.List   `tfsdk:"addresses"`
	Disks           []DiskModel  `tfsdk:"disk"`
	NICs            []NICModel   `tfsdk:"nic"`
	Proxies         []ProxyModel `tfsdk:"proxy"`
}

// DiskModel represents a disk device attachment.
type DiskModel struct {
	Name        types.String `tfsdk:"name"`
	Source      types.String `tfsdk:"source"`
	Destination types.String `tfsdk:"destination"`
	Readonly    types.Bool   `tfsdk:"readonly"`
}

// NICModel represents a network interface attachment.
type NICModel struct {
	Name    types.String `tfsdk:"name"`
	Network types.String `tfsdk:"network"`
	NICType types.String `tfsdk:"nic_type"`
	Parent  types.String `tfsdk:"parent"`
}

// ProxyModel represents a port proxy/forward.
type ProxyModel struct {
	Name        types.String `tfsdk:"name"`
	SourceProto types.String `tfsdk:"source_proto"`
	SourcePort  types.Int64  `tfsdk:"source_port"`
	DestProto   types.String `tfsdk:"dest_proto"`
	DestPort    types.Int64  `tfsdk:"dest_port"`
}

// AddressModel represents an IP address assigned to the instance.
type AddressModel struct {
	Type    types.String `tfsdk:"type"`    // "INET" or "INET6"
	Address types.String `tfsdk:"address"` // e.g., "192.168.1.203"
	Netmask types.Int64  `tfsdk:"netmask"` // e.g., 24
}

// virtInstanceAPIResponse represents the JSON response from virt.instance API calls (TrueNAS 25.0+).
type virtInstanceAPIResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	CPU         *string                `json:"cpu"`
	Memory      *int64                 `json:"memory"`
	Autostart   bool                   `json:"autostart"`
	Environment map[string]string      `json:"environment"`
	Aliases     []virtInstanceAlias    `json:"aliases"`
	Image       virtInstanceImage      `json:"image"`
	StoragePool string                 `json:"storage_pool"`
	VNCEnabled  bool                   `json:"vnc_enabled"`
	VNCPort     *int                   `json:"vnc_port"`
}

// virtInstanceAlias represents an IP address alias from the API.
type virtInstanceAlias struct {
	Type    string `json:"type"`    // "INET" or "INET6"
	Address string `json:"address"`
	Netmask *int64 `json:"netmask"` // nullable per API schema
}

type virtInstanceImage struct {
	Architecture string `json:"architecture"`
	Description  string `json:"description"`
	OS           string `json:"os"`
	Release      string `json:"release"`
	Variant      string `json:"variant"`
}

// deviceAPIResponse represents a device from virt.instance.device_list.
type deviceAPIResponse struct {
	DevType     string  `json:"dev_type"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Readonly    bool    `json:"readonly"`
	// DISK fields
	Source      *string `json:"source"`
	Destination *string `json:"destination"`
	// NIC fields
	Network *string `json:"network"`
	NICType *string `json:"nic_type"`
	Parent  *string `json:"parent"`
	// PROXY fields
	SourceProto *string `json:"source_proto"`
	SourcePort  *int64  `json:"source_port"`
	DestProto   *string `json:"dest_proto"`
	DestPort    *int64  `json:"dest_port"`
}

// VirtInstanceResource defines the resource implementation.
type VirtInstanceResource struct {
	BaseResource
}

// NewVirtInstanceResource creates a new VirtInstanceResource.
func NewVirtInstanceResource() resource.Resource {
	return &VirtInstanceResource{}
}

func (r *VirtInstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virt_instance"
}

func (r *VirtInstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Incus/LXC container on TrueNAS 25.0+.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Container ID (numeric).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Container name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"storage_pool": schema.StringAttribute{
				Description: "Storage pool for the container.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image_name": schema.StringAttribute{
				Description: "Container image name (e.g., 'ubuntu').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image_version": schema.StringAttribute{
				Description: "Container image version (e.g., '24.04').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"autostart": schema.BoolAttribute{
				Description: "Whether to start the container automatically on boot. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"desired_state": schema.StringAttribute{
				Description: "Desired container state: 'RUNNING' or 'STOPPED'. Defaults to 'RUNNING'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(VirtInstanceStateRunning),
				Validators: []validator.String{
					stringvalidator.OneOf(VirtInstanceStateRunning, VirtInstanceStateStopped),
				},
			},
			"state_timeout": schema.Int64Attribute{
				Description: "Timeout in seconds to wait for state transitions. Defaults to 90. Range: 30-600.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(90),
				Validators: []validator.Int64{
					int64validator.Between(30, 600),
				},
			},
			"state": schema.StringAttribute{
				Description: "Current container state (RUNNING, STOPPED, etc.).",
				Computed:    true,
			},
			"uuid": schema.StringAttribute{
				Description: "Container UUID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"shutdown_timeout": schema.Int64Attribute{
				Description: "Timeout in seconds for graceful shutdown. Defaults to 30.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
			},
			"addresses": schema.ListNestedAttribute{
				Description: "IP addresses assigned to the instance. Empty when stopped.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Address type: INET (IPv4) or INET6 (IPv6).",
							Computed:    true,
						},
						"address": schema.StringAttribute{
							Description: "IP address.",
							Computed:    true,
						},
						"netmask": schema.Int64Attribute{
							Description: "Network mask prefix length.",
							Computed:    true,
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"disk": schema.ListNestedBlock{
				Description: "Disk devices to attach to the container.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Device name (auto-generated if not specified).",
							Optional:    true,
							Computed:    true,
						},
						"source": schema.StringAttribute{
							Description: "Source path on the host (e.g., '/mnt/tank/data').",
							Required:    true,
						},
						"destination": schema.StringAttribute{
							Description: "Mount point inside the container (e.g., '/data').",
							Required:    true,
						},
						"readonly": schema.BoolAttribute{
							Description: "Mount as read-only. Defaults to false.",
							Optional:    true,
						},
					},
				},
			},
			"nic": schema.ListNestedBlock{
				Description: "Network interfaces to attach to the container.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Device name (auto-generated if not specified).",
							Optional:    true,
							Computed:    true,
						},
						"network": schema.StringAttribute{
							Description: "Network name to attach to.",
							Optional:    true,
						},
						"nic_type": schema.StringAttribute{
							Description: "NIC type: 'BRIDGED' or 'MACVLAN'.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("BRIDGED", "MACVLAN"),
							},
						},
						"parent": schema.StringAttribute{
							Description: "Parent interface name.",
							Optional:    true,
						},
					},
				},
			},
			"proxy": schema.ListNestedBlock{
				Description: "Port proxies/forwards for the container.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Device name (auto-generated if not specified).",
							Optional:    true,
							Computed:    true,
						},
						"source_proto": schema.StringAttribute{
							Description: "Source protocol: 'TCP' or 'UDP'.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("TCP", "UDP"),
							},
						},
						"source_port": schema.Int64Attribute{
							Description: "Source port on the host.",
							Required:    true,
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"dest_proto": schema.StringAttribute{
							Description: "Destination protocol: 'TCP' or 'UDP'.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("TCP", "UDP"),
							},
						},
						"dest_port": schema.Int64Attribute{
							Description: "Destination port in the container.",
							Required:    true,
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
					},
				},
			},
		},
	}
}


func (r *VirtInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VirtInstanceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check version requirement
	version := r.client.Version()
	if !version.AtLeast(25, 0) {
		resp.Diagnostics.AddError(
			"Unsupported TrueNAS Version",
			fmt.Sprintf("Container resources require TrueNAS 25.0 or later. Detected version: %s", version.String()),
		)
		return
	}

	// Build create params
	params := r.buildCreateParams(ctx, &data)
	containerName := data.Name.ValueString()

	// Call virt.instance.create (job-based)
	_, err := r.client.CallAndWait(ctx, "virt.instance.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Container",
			fmt.Sprintf("Unable to create container %q: %s", containerName, err.Error()),
		)
		return
	}

	// Query the container to get current state
	container, err := r.getVirtInstance(ctx, containerName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Query Container After Create",
			fmt.Sprintf("Unable to query container %q after create: %s", containerName, err.Error()),
		)
		return
	}

	if container == nil {
		resp.Diagnostics.AddError(
			"Container Not Found After Create",
			fmt.Sprintf("Container %q was not found after create", containerName),
		)
		return
	}

	// Map response to model
	r.mapVirtInstanceToModel(container, &data)

	// Query devices to get server-assigned names (if user didn't specify names).
	// Filter to only include devices we created (by matching source/destination for disks,
	// network for NICs, ports for proxies).
	if len(data.Disks) > 0 || len(data.NICs) > 0 || len(data.Proxies) > 0 {
		devices, err := r.queryDevices(ctx, container.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Query Container Devices",
				fmt.Sprintf("Unable to query devices for container %q after create: %s", containerName, err.Error()),
			)
			return
		}
		r.matchCreatedDevices(devices, &data)
	}

	// Handle desired_state - if user wants STOPPED but container started as RUNNING
	desiredState := data.DesiredState.ValueString()
	if desiredState == "" {
		desiredState = VirtInstanceStateRunning
	}

	if container.Status != desiredState {
		timeout := time.Duration(data.StateTimeout.ValueInt64()) * time.Second
		if timeout == 0 {
			timeout = 90 * time.Second
		}

		if desiredState == VirtInstanceStateRunning {
			_, err = r.client.CallAndWait(ctx, "virt.instance.start", data.ID.ValueString())
		} else {
			// virt.instance.stop takes: id (string), stop_args (object with timeout/force)
			stopArgs := map[string]any{"timeout": data.ShutdownTimeout.ValueInt64()}
			_, err = r.client.CallAndWait(ctx, "virt.instance.stop", []any{data.ID.ValueString(), stopArgs})
		}
		if err != nil {
			action := "start"
			if desiredState == VirtInstanceStateStopped {
				action = "stop"
			}
			resp.Diagnostics.AddError(
				"Unable to Set Container State",
				fmt.Sprintf("Unable to %s container %q: %s", action, containerName, err.Error()),
			)
			return
		}

		// Wait for stable state
		queryFunc := func(ctx context.Context, n string) (string, error) {
			return r.getVirtInstanceState(ctx, n)
		}

		finalState, err := waitForStableState(ctx, containerName, timeout, queryFunc)
		if err != nil {
			resp.Diagnostics.AddError(
				"Timeout Waiting for Container State",
				err.Error(),
			)
			return
		}

		data.State = types.StringValue(finalState)
	}

	// Re-query container to get fresh addresses (networking may take time to come up)
	// Small delay to allow networking to initialize after container start
	time.Sleep(2 * time.Second)
	freshContainer, _ := r.getVirtInstance(ctx, containerName)
	if freshContainer != nil {
		data.Addresses = mapAliasesToAddresses(freshContainer.Aliases)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VirtInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VirtInstanceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check version requirement
	version := r.client.Version()
	if !version.AtLeast(25, 0) {
		resp.Diagnostics.AddError(
			"Unsupported TrueNAS Version",
			fmt.Sprintf("Container resources require TrueNAS 25.0 or later. Detected version: %s", version.String()),
		)
		return
	}

	// Preserve user-specified values from prior state
	priorDesiredState := data.DesiredState
	priorStateTimeout := data.StateTimeout

	containerName := data.Name.ValueString()

	container, err := r.getVirtInstance(ctx, containerName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Container",
			fmt.Sprintf("Unable to read container %q: %s", containerName, err.Error()),
		)
		return
	}

	if container == nil {
		// Container was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response to model
	r.mapVirtInstanceToModel(container, &data)

	// Query devices and filter to only managed ones (those with names in our state).
	// This allows drift detection for managed devices while ignoring system defaults.
	managedNames := getManagedDeviceNames(&data)
	if len(managedNames) > 0 {
		devices, err := r.queryDevices(ctx, container.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Query Container Devices",
				fmt.Sprintf("Unable to query devices for container %q: %s", containerName, err.Error()),
			)
			return
		}
		r.mapDevicesToModel(devices, &data, managedNames)
	}
	// If no managed devices in state, preserve empty slices (don't query API)

	// Restore user-specified values from prior state
	data.DesiredState = priorDesiredState
	data.StateTimeout = priorStateTimeout

	// Default desired_state if null/unknown (e.g., after import)
	if data.DesiredState.IsNull() || data.DesiredState.IsUnknown() {
		data.DesiredState = types.StringValue(container.Status)
	}

	// Default state_timeout if null/unknown
	if data.StateTimeout.IsNull() || data.StateTimeout.IsUnknown() {
		data.StateTimeout = types.Int64Value(90)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VirtInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VirtInstanceResourceModel
	var stateData VirtInstanceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check version requirement
	version := r.client.Version()
	if !version.AtLeast(25, 0) {
		resp.Diagnostics.AddError(
			"Unsupported TrueNAS Version",
			fmt.Sprintf("Container resources require TrueNAS 25.0 or later. Detected version: %s", version.String()),
		)
		return
	}

	containerName := data.Name.ValueString()
	containerID := data.ID.ValueString()

	// Build update params and check if anything changed
	updateParams := r.buildUpdateParams(&data, &stateData)
	if len(updateParams) > 0 {
		params := []any{containerID, updateParams}

		_, err := r.client.CallAndWait(ctx, "virt.instance.update", params)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Update Container",
				fmt.Sprintf("Unable to update container %q: %s", containerName, err.Error()),
			)
			return
		}
	}

	// Reconcile devices (add/delete as needed)
	if err := r.reconcileDevices(ctx, containerID, &data, &stateData); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Container Devices",
			fmt.Sprintf("Unable to update devices for container %q: %s", containerName, err.Error()),
		)
		return
	}

	// Query current state
	currentState, err := r.getVirtInstanceState(ctx, containerName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Query Container State",
			fmt.Sprintf("Unable to query container %q state: %s", containerName, err.Error()),
		)
		return
	}

	// Get timeout from plan
	timeout := time.Duration(data.StateTimeout.ValueInt64()) * time.Second
	if timeout == 0 {
		timeout = 90 * time.Second
	}

	// Wait for transitional states to complete
	if !isVirtInstanceStableState(currentState) {
		queryFunc := func(ctx context.Context, n string) (string, error) {
			return r.getVirtInstanceState(ctx, n)
		}

		stableState, err := waitForStableState(ctx, containerName, timeout, queryFunc)
		if err != nil {
			resp.Diagnostics.AddError(
				"Timeout Waiting for Container State",
				err.Error(),
			)
			return
		}
		currentState = stableState
	}

	// Reconcile desired_state
	desiredState := data.DesiredState.ValueString()
	if desiredState == "" {
		desiredState = VirtInstanceStateRunning
	}

	if currentState != desiredState {
		shutdownTimeout := data.ShutdownTimeout.ValueInt64()
		if shutdownTimeout == 0 {
			shutdownTimeout = 30 // default
		}
		err := r.reconcileDesiredState(ctx, containerName, containerID, currentState, desiredState, timeout, shutdownTimeout, resp)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Reconcile Container State",
				err.Error(),
			)
			return
		}
		// Query final state after reconciliation
		currentState, err = r.getVirtInstanceState(ctx, containerName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Query Container State After Reconciliation",
				fmt.Sprintf("Unable to query container %q state: %s", containerName, err.Error()),
			)
			return
		}
	}

	// Query full container info to update state
	container, err := r.getVirtInstance(ctx, containerName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Query Container After Update",
			fmt.Sprintf("Unable to query container %q: %s", containerName, err.Error()),
		)
		return
	}

	if container != nil {
		r.mapVirtInstanceToModel(container, &data)
		// Devices are preserved from plan - reconcileDevices already handled add/delete
	} else {
		data.State = types.StringValue(currentState)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VirtInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VirtInstanceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	containerName := data.Name.ValueString()
	containerID := data.ID.ValueString()

	// Check current state - if running, stop first
	currentState, err := r.getVirtInstanceState(ctx, containerName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Query Container State",
			fmt.Sprintf("Unable to query container %q state: %s", containerName, err.Error()),
		)
		return
	}

	if currentState == VirtInstanceStateRunning {
		// virt.instance.stop takes: id (string), stop_args (object with timeout/force)
		shutdownTimeout := data.ShutdownTimeout.ValueInt64()
		if shutdownTimeout == 0 {
			shutdownTimeout = 30 // default
		}
		stopArgs := map[string]any{"timeout": shutdownTimeout}
		_, err := r.client.CallAndWait(ctx, "virt.instance.stop", []any{containerID, stopArgs})
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Stop Container",
				fmt.Sprintf("Unable to stop container %q before delete: %s", containerName, err.Error()),
			)
			return
		}
	}

	// Delete the container
	_, err = r.client.CallAndWait(ctx, "virt.instance.delete", containerID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Container",
			fmt.Sprintf("Unable to delete container %q: %s", containerName, err.Error()),
		)
		return
	}
}

func (r *VirtInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by container name
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

// buildCreateParams builds the API params from the resource model for create.
// Uses virt.instance.create API for TrueNAS 25.0+.
func (r *VirtInstanceResource) buildCreateParams(ctx context.Context, data *VirtInstanceResourceModel) map[string]any {
	// Build image string in format "name/version" (e.g., "alpine/3.20")
	imageStr := fmt.Sprintf("%s/%s", data.ImageName.ValueString(), data.ImageVersion.ValueString())

	// virt.instance.create accepts: name, image, instance_type, source_type, storage_pool, environment, autostart, cpu, memory
	params := map[string]any{
		"name":          data.Name.ValueString(),
		"image":         imageStr,
		"instance_type": "CONTAINER",
		"storage_pool":  data.StoragePool.ValueString(),
	}

	if !data.Autostart.IsNull() && !data.Autostart.IsUnknown() {
		params["autostart"] = data.Autostart.ValueBool()
	}

	// Build devices array
	devices := r.buildDevices(data)
	if len(devices) > 0 {
		params["devices"] = devices
	}

	return params
}

// buildDevices builds the devices array from disk, nic, and proxy blocks.
func (r *VirtInstanceResource) buildDevices(data *VirtInstanceResourceModel) []map[string]any {
	var devices []map[string]any

	// Add disk devices
	for _, disk := range data.Disks {
		dev := map[string]any{
			"dev_type":    "DISK",
			"source":      disk.Source.ValueString(),
			"destination": disk.Destination.ValueString(),
		}
		if !disk.Name.IsNull() && disk.Name.ValueString() != "" {
			dev["name"] = disk.Name.ValueString()
		}
		if !disk.Readonly.IsNull() {
			dev["readonly"] = disk.Readonly.ValueBool()
		}
		devices = append(devices, dev)
	}

	// Add NIC devices
	for _, nic := range data.NICs {
		dev := map[string]any{
			"dev_type": "NIC",
		}
		if !nic.Name.IsNull() && nic.Name.ValueString() != "" {
			dev["name"] = nic.Name.ValueString()
		}
		if !nic.Network.IsNull() && nic.Network.ValueString() != "" {
			dev["network"] = nic.Network.ValueString()
		}
		if !nic.NICType.IsNull() && nic.NICType.ValueString() != "" {
			dev["nic_type"] = nic.NICType.ValueString()
		}
		if !nic.Parent.IsNull() && nic.Parent.ValueString() != "" {
			dev["parent"] = nic.Parent.ValueString()
		}
		devices = append(devices, dev)
	}

	// Add proxy devices
	for _, proxy := range data.Proxies {
		dev := map[string]any{
			"dev_type":     "PROXY",
			"source_proto": proxy.SourceProto.ValueString(),
			"source_port":  proxy.SourcePort.ValueInt64(),
			"dest_proto":   proxy.DestProto.ValueString(),
			"dest_port":    proxy.DestPort.ValueInt64(),
		}
		if !proxy.Name.IsNull() && proxy.Name.ValueString() != "" {
			dev["name"] = proxy.Name.ValueString()
		}
		devices = append(devices, dev)
	}

	return devices
}

// buildUpdateParams builds the API params from the resource model for update.
// Only includes fields that have changed.
func (r *VirtInstanceResource) buildUpdateParams(plan, state *VirtInstanceResourceModel) map[string]any {
	params := map[string]any{}

	if !plan.Autostart.Equal(state.Autostart) && !plan.Autostart.IsNull() {
		params["autostart"] = plan.Autostart.ValueBool()
	}

	return params
}

// getVirtInstance retrieves a container by name using virt.instance.get_instance.
// Returns nil, nil if the container does not exist.
func (r *VirtInstanceResource) getVirtInstance(ctx context.Context, name string) (*virtInstanceAPIResponse, error) {
	result, err := r.client.Call(ctx, "virt.instance.get_instance", name)
	if err != nil {
		// Check if it's a "not found" error
		if isNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	var container virtInstanceAPIResponse
	if err := json.Unmarshal(result, &container); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &container, nil
}

// isNotFoundError checks if an error indicates the resource was not found.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// TrueNAS API returns errors containing "does not exist" or "not found" for missing resources
	errLower := strings.ToLower(err.Error())
	return strings.Contains(errLower, "does not exist") ||
		strings.Contains(errLower, "not found") ||
		strings.Contains(errLower, "no such instance")
}

// getVirtInstanceState queries the current state of a container.
func (r *VirtInstanceResource) getVirtInstanceState(ctx context.Context, name string) (string, error) {
	container, err := r.getVirtInstance(ctx, name)
	if err != nil {
		return "", err
	}

	if container == nil {
		return "", fmt.Errorf("container %q not found", name)
	}

	return container.Status, nil
}

// mapVirtInstanceToModel maps an API response to the resource model.
// Maps virt.instance API response fields to terraform resource model.
// Preserves plan/state values for fields that don't change (RequiresReplace).
func (r *VirtInstanceResource) mapVirtInstanceToModel(container *virtInstanceAPIResponse, data *VirtInstanceResourceModel) {
	data.ID = types.StringValue(container.ID)
	data.Name = types.StringValue(container.Name)
	data.StoragePool = types.StringValue(container.StoragePool)
	// Preserve ImageName and ImageVersion from plan/state - API may return different casing
	// These fields have RequiresReplace so they don't change during resource lifecycle
	data.State = types.StringValue(container.Status)
	data.Autostart = types.BoolValue(container.Autostart)

	// Use container ID as UUID (virt.instance uses name as ID)
	data.UUID = types.StringValue(container.ID)

	// Map IP addresses from aliases
	data.Addresses = mapAliasesToAddresses(container.Aliases)
}

// addressAttrTypes returns the attribute types for AddressModel used in types.List.
func addressAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":    types.StringType,
		"address": types.StringType,
		"netmask": types.Int64Type,
	}
}

// mapAliasesToAddresses converts API aliases to types.List.
func mapAliasesToAddresses(aliases []virtInstanceAlias) types.List {
	if len(aliases) == 0 {
		return types.ListValueMust(types.ObjectType{AttrTypes: addressAttrTypes()}, []attr.Value{})
	}

	elements := make([]attr.Value, len(aliases))
	for i, alias := range aliases {
		elements[i] = types.ObjectValueMust(addressAttrTypes(), map[string]attr.Value{
			"type":    types.StringValue(alias.Type),
			"address": types.StringValue(alias.Address),
			"netmask": types.Int64PointerValue(alias.Netmask),
		})
	}
	return types.ListValueMust(types.ObjectType{AttrTypes: addressAttrTypes()}, elements)
}

// queryDevices queries the devices attached to a container.
func (r *VirtInstanceResource) queryDevices(ctx context.Context, containerID string) ([]deviceAPIResponse, error) {
	result, err := r.client.Call(ctx, "virt.instance.device_list", containerID)
	if err != nil {
		return nil, err
	}

	var devices []deviceAPIResponse
	if err := json.Unmarshal(result, &devices); err != nil {
		return nil, fmt.Errorf("failed to unmarshal devices: %w", err)
	}

	return devices, nil
}

// getManagedDeviceNames returns a set of device names that are managed by terraform.
func getManagedDeviceNames(data *VirtInstanceResourceModel) map[string]bool {
	names := make(map[string]bool)
	for _, d := range data.Disks {
		if !d.Name.IsNull() && d.Name.ValueString() != "" {
			names[d.Name.ValueString()] = true
		}
	}
	for _, n := range data.NICs {
		if !n.Name.IsNull() && n.Name.ValueString() != "" {
			names[n.Name.ValueString()] = true
		}
	}
	for _, p := range data.Proxies {
		if !p.Name.IsNull() && p.Name.ValueString() != "" {
			names[p.Name.ValueString()] = true
		}
	}
	return names
}

// mapDevicesToModel maps API device responses to the resource model.
// Only includes devices that are in the managedNames set (to exclude system defaults).
// If managedNames is nil, all devices are included.
func (r *VirtInstanceResource) mapDevicesToModel(devices []deviceAPIResponse, data *VirtInstanceResourceModel, managedNames map[string]bool) {
	var disks []DiskModel
	var nics []NICModel
	var proxies []ProxyModel

	for _, dev := range devices {
		// Skip devices that aren't in our managed set (if filtering is enabled)
		if managedNames != nil && dev.Name != nil {
			if !managedNames[*dev.Name] {
				continue
			}
		}
		// Skip devices without names when filtering - we can only track named devices
		if managedNames != nil && dev.Name == nil {
			continue
		}

		switch dev.DevType {
		case "DISK":
			disk := DiskModel{
				Readonly: types.BoolValue(dev.Readonly),
			}
			if dev.Name != nil {
				disk.Name = types.StringValue(*dev.Name)
			} else {
				disk.Name = types.StringNull()
			}
			if dev.Source != nil {
				disk.Source = types.StringValue(*dev.Source)
			} else {
				disk.Source = types.StringNull()
			}
			if dev.Destination != nil {
				disk.Destination = types.StringValue(*dev.Destination)
			} else {
				disk.Destination = types.StringNull()
			}
			disks = append(disks, disk)

		case "NIC":
			nic := NICModel{}
			if dev.Name != nil {
				nic.Name = types.StringValue(*dev.Name)
			} else {
				nic.Name = types.StringNull()
			}
			if dev.Network != nil {
				nic.Network = types.StringValue(*dev.Network)
			} else {
				nic.Network = types.StringNull()
			}
			if dev.NICType != nil {
				nic.NICType = types.StringValue(*dev.NICType)
			} else {
				nic.NICType = types.StringNull()
			}
			if dev.Parent != nil {
				nic.Parent = types.StringValue(*dev.Parent)
			} else {
				nic.Parent = types.StringNull()
			}
			nics = append(nics, nic)

		case "PROXY":
			proxy := ProxyModel{}
			if dev.Name != nil {
				proxy.Name = types.StringValue(*dev.Name)
			} else {
				proxy.Name = types.StringNull()
			}
			if dev.SourceProto != nil {
				proxy.SourceProto = types.StringValue(*dev.SourceProto)
			}
			if dev.SourcePort != nil {
				proxy.SourcePort = types.Int64Value(*dev.SourcePort)
			}
			if dev.DestProto != nil {
				proxy.DestProto = types.StringValue(*dev.DestProto)
			}
			if dev.DestPort != nil {
				proxy.DestPort = types.Int64Value(*dev.DestPort)
			}
			proxies = append(proxies, proxy)
		}
	}

	data.Disks = disks
	data.NICs = nics
	data.Proxies = proxies
}

// matchCreatedDevices matches API devices to plan devices and fills in server-assigned names.
// This is called after Create to populate names for devices where user didn't specify a name.
func (r *VirtInstanceResource) matchCreatedDevices(apiDevices []deviceAPIResponse, data *VirtInstanceResourceModel) {
	// Match disks by source+destination
	for i := range data.Disks {
		planDisk := &data.Disks[i]
		// If name was specified, keep it. Otherwise find the matching device.
		if !planDisk.Name.IsNull() && planDisk.Name.ValueString() != "" {
			continue
		}
		for _, apiDev := range apiDevices {
			if apiDev.DevType != "DISK" || apiDev.Name == nil {
				continue
			}
			if apiDev.Source != nil && apiDev.Destination != nil {
				if *apiDev.Source == planDisk.Source.ValueString() &&
					*apiDev.Destination == planDisk.Destination.ValueString() {
					planDisk.Name = types.StringValue(*apiDev.Name)
					break
				}
			}
		}
	}

	// Match NICs by network (or parent for MACVLAN)
	for i := range data.NICs {
		planNIC := &data.NICs[i]
		if !planNIC.Name.IsNull() && planNIC.Name.ValueString() != "" {
			continue
		}
		for _, apiDev := range apiDevices {
			if apiDev.DevType != "NIC" || apiDev.Name == nil {
				continue
			}
			// Match by network if specified
			if !planNIC.Network.IsNull() && planNIC.Network.ValueString() != "" {
				if apiDev.Network != nil && *apiDev.Network == planNIC.Network.ValueString() {
					planNIC.Name = types.StringValue(*apiDev.Name)
					break
				}
			}
			// Match by parent if specified
			if !planNIC.Parent.IsNull() && planNIC.Parent.ValueString() != "" {
				if apiDev.Parent != nil && *apiDev.Parent == planNIC.Parent.ValueString() {
					planNIC.Name = types.StringValue(*apiDev.Name)
					break
				}
			}
		}
	}

	// Match proxies by source_proto+source_port+dest_proto+dest_port
	for i := range data.Proxies {
		planProxy := &data.Proxies[i]
		if !planProxy.Name.IsNull() && planProxy.Name.ValueString() != "" {
			continue
		}
		for _, apiDev := range apiDevices {
			if apiDev.DevType != "PROXY" || apiDev.Name == nil {
				continue
			}
			if apiDev.SourceProto != nil && apiDev.SourcePort != nil &&
				apiDev.DestProto != nil && apiDev.DestPort != nil {
				if *apiDev.SourceProto == planProxy.SourceProto.ValueString() &&
					*apiDev.SourcePort == planProxy.SourcePort.ValueInt64() &&
					*apiDev.DestProto == planProxy.DestProto.ValueString() &&
					*apiDev.DestPort == planProxy.DestPort.ValueInt64() {
					planProxy.Name = types.StringValue(*apiDev.Name)
					break
				}
			}
		}
	}
}

// reconcileDevices adds/removes devices to match the desired state.
func (r *VirtInstanceResource) reconcileDevices(ctx context.Context, containerID string, plan, state *VirtInstanceResourceModel) error {
	// Get current device names from state
	stateDeviceNames := make(map[string]bool)
	for _, d := range state.Disks {
		if !d.Name.IsNull() {
			stateDeviceNames[d.Name.ValueString()] = true
		}
	}
	for _, n := range state.NICs {
		if !n.Name.IsNull() {
			stateDeviceNames[n.Name.ValueString()] = true
		}
	}
	for _, p := range state.Proxies {
		if !p.Name.IsNull() {
			stateDeviceNames[p.Name.ValueString()] = true
		}
	}

	// Get desired device names from plan
	planDeviceNames := make(map[string]bool)
	for _, d := range plan.Disks {
		if !d.Name.IsNull() {
			planDeviceNames[d.Name.ValueString()] = true
		}
	}
	for _, n := range plan.NICs {
		if !n.Name.IsNull() {
			planDeviceNames[n.Name.ValueString()] = true
		}
	}
	for _, p := range plan.Proxies {
		if !p.Name.IsNull() {
			planDeviceNames[p.Name.ValueString()] = true
		}
	}

	// Delete devices that are in state but not in plan
	for name := range stateDeviceNames {
		if !planDeviceNames[name] {
			_, err := r.client.CallAndWait(ctx, "virt.instance.device_delete", []any{containerID, name})
			if err != nil {
				return fmt.Errorf("failed to delete device %q: %w", name, err)
			}
		}
	}

	// Add devices that are in plan but not in state
	for _, disk := range plan.Disks {
		name := disk.Name.ValueString()
		if name != "" && !stateDeviceNames[name] {
			dev := map[string]any{
				"dev_type":    "DISK",
				"name":        name,
				"source":      disk.Source.ValueString(),
				"destination": disk.Destination.ValueString(),
			}
			if !disk.Readonly.IsNull() {
				dev["readonly"] = disk.Readonly.ValueBool()
			}
			_, err := r.client.CallAndWait(ctx, "virt.instance.device_add", []any{containerID, dev})
			if err != nil {
				return fmt.Errorf("failed to add disk device %q: %w", name, err)
			}
		}
	}

	for _, nic := range plan.NICs {
		name := nic.Name.ValueString()
		if name != "" && !stateDeviceNames[name] {
			dev := map[string]any{
				"dev_type": "NIC",
				"name":     name,
			}
			if !nic.Network.IsNull() && nic.Network.ValueString() != "" {
				dev["network"] = nic.Network.ValueString()
			}
			if !nic.NICType.IsNull() && nic.NICType.ValueString() != "" {
				dev["nic_type"] = nic.NICType.ValueString()
			}
			if !nic.Parent.IsNull() && nic.Parent.ValueString() != "" {
				dev["parent"] = nic.Parent.ValueString()
			}
			_, err := r.client.CallAndWait(ctx, "virt.instance.device_add", []any{containerID, dev})
			if err != nil {
				return fmt.Errorf("failed to add NIC device %q: %w", name, err)
			}
		}
	}

	for _, proxy := range plan.Proxies {
		name := proxy.Name.ValueString()
		if name != "" && !stateDeviceNames[name] {
			dev := map[string]any{
				"dev_type":     "PROXY",
				"name":         name,
				"source_proto": proxy.SourceProto.ValueString(),
				"source_port":  proxy.SourcePort.ValueInt64(),
				"dest_proto":   proxy.DestProto.ValueString(),
				"dest_port":    proxy.DestPort.ValueInt64(),
			}
			_, err := r.client.CallAndWait(ctx, "virt.instance.device_add", []any{containerID, dev})
			if err != nil {
				return fmt.Errorf("failed to add proxy device %q: %w", name, err)
			}
		}
	}

	return nil
}

// reconcileDesiredState ensures the container is in the desired state.
func (r *VirtInstanceResource) reconcileDesiredState(
	ctx context.Context,
	name string,
	id string,
	currentState string,
	desiredState string,
	timeout time.Duration,
	shutdownTimeout int64,
	resp *resource.UpdateResponse,
) error {
	// Check if reconciliation is needed
	if currentState == desiredState {
		return nil
	}

	// Call the appropriate API
	var err error
	if desiredState == VirtInstanceStateRunning {
		_, err = r.client.CallAndWait(ctx, "virt.instance.start", id)
	} else {
		// virt.instance.stop takes: id (string), stop_args (object with timeout/force)
		stopArgs := map[string]any{"timeout": shutdownTimeout}
		_, err = r.client.CallAndWait(ctx, "virt.instance.stop", []any{id, stopArgs})
	}
	if err != nil {
		return fmt.Errorf("failed to %s container %q: %w", desiredState, name, err)
	}

	// Wait for stable state
	queryFunc := func(ctx context.Context, n string) (string, error) {
		return r.getVirtInstanceState(ctx, n)
	}

	finalState, err := waitForStableState(ctx, name, timeout, queryFunc)
	if err != nil {
		return err
	}

	// Verify we reached the desired state
	if finalState != desiredState {
		return fmt.Errorf("container %q reached state %s instead of desired %s", name, finalState, desiredState)
	}

	return nil
}

// isVirtInstanceStableState returns true if the state is stable (not transitional).
func isVirtInstanceStableState(state string) bool {
	switch state {
	case VirtInstanceStateRunning, VirtInstanceStateStopped:
		return true
	default:
		return false
	}
}
