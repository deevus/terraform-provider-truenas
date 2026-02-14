package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// getVM retrieves a VM by ID using vm.get_instance.
func (r *VMResource) getVM(ctx context.Context, id int64) (*vmAPIResponse, error) {
	result, err := r.client.Call(ctx, "vm.get_instance", id)
	if err != nil {
		return nil, err
	}

	var vm vmAPIResponse
	if err := json.Unmarshal(result, &vm); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &vm, nil
}

// mapVMToModel maps an API response to the resource model.
func (r *VMResource) mapVMToModel(vm *vmAPIResponse, data *VMResourceModel) {
	data.ID = types.StringValue(strconv.FormatInt(vm.ID, 10))
	data.Name = types.StringValue(vm.Name)
	data.Description = types.StringValue(vm.Description)
	data.VCPUs = types.Int64Value(vm.VCPUs)
	data.Cores = types.Int64Value(vm.Cores)
	data.Threads = types.Int64Value(vm.Threads)
	data.Memory = types.Int64Value(vm.Memory)
	if vm.MinMemory != nil {
		data.MinMemory = types.Int64Value(*vm.MinMemory)
	} else {
		data.MinMemory = types.Int64Null()
	}
	data.Autostart = types.BoolValue(vm.Autostart)
	data.Time = types.StringValue(vm.Time)
	data.Bootloader = types.StringValue(vm.Bootloader)
	data.BootloaderOVMF = types.StringValue(vm.BootloaderOVMF)
	data.CPUMode = types.StringValue(vm.CPUMode)
	if vm.CPUModel != nil {
		data.CPUModel = types.StringValue(*vm.CPUModel)
	} else {
		data.CPUModel = types.StringNull()
	}
	data.ShutdownTimeout = types.Int64Value(vm.ShutdownTimeout)
	data.CommandLineArgs = types.StringValue(vm.CommandLineArgs)
	data.State = types.StringValue(vm.Status.State)
	data.DisplayAvailable = types.BoolValue(vm.DisplayAvailable)
}

// queryVMDevices queries devices for a VM using vm.device.query.
func (r *VMResource) queryVMDevices(ctx context.Context, vmID int64) ([]vmDeviceAPIResponse, error) {
	filter := []any{[]any{[]any{"vm", "=", vmID}}}
	result, err := r.client.Call(ctx, "vm.device.query", filter)
	if err != nil {
		return nil, err
	}

	var devices []vmDeviceAPIResponse
	if err := json.Unmarshal(result, &devices); err != nil {
		return nil, fmt.Errorf("parse devices: %w", err)
	}

	return devices, nil
}

// mapDevicesToModel maps API device responses to the resource model.
func (r *VMResource) mapDevicesToModel(devices []vmDeviceAPIResponse, data *VMResourceModel) {
	data.Disks = nil
	data.Raws = nil
	data.CDROMs = nil
	data.NICs = nil
	data.Displays = nil
	data.PCIs = nil
	data.USBs = nil

	for _, dev := range devices {
		dtype, _ := dev.Attributes["dtype"].(string)
		switch dtype {
		case "DISK":
			data.Disks = append(data.Disks, mapDiskDevice(dev))
		case "RAW":
			data.Raws = append(data.Raws, mapRawDevice(dev))
		case "CDROM":
			data.CDROMs = append(data.CDROMs, mapCDROMDevice(dev))
		case "NIC":
			data.NICs = append(data.NICs, mapNICDevice(dev))
		case "DISPLAY":
			data.Displays = append(data.Displays, mapDisplayDevice(dev))
		case "PCI":
			data.PCIs = append(data.PCIs, mapPCIDevice(dev))
		case "USB":
			data.USBs = append(data.USBs, mapUSBDevice(dev))
		}
	}
}

// preserveRawExists copies the exists attribute from prior RAW devices to mapped ones.
// exists is a create-time API flag not returned in query responses, so it must be
// preserved from the plan/state to avoid inconsistent results after apply.
func preserveRawExists(mapped, prior []VMRawModel) {
	// Build lookup by device ID
	priorByID := make(map[int64]VMRawModel)
	for _, p := range prior {
		if !p.DeviceID.IsNull() && !p.DeviceID.IsUnknown() {
			priorByID[p.DeviceID.ValueInt64()] = p
		}
	}

	for i := range mapped {
		if !mapped[i].DeviceID.IsNull() && !mapped[i].DeviceID.IsUnknown() {
			if p, ok := priorByID[mapped[i].DeviceID.ValueInt64()]; ok {
				mapped[i].Exists = p.Exists
				continue
			}
		}
		// Fallback: match by index for newly created devices
		if i < len(prior) {
			mapped[i].Exists = prior[i].Exists
		}
	}
}

func mapDiskDevice(dev vmDeviceAPIResponse) VMDiskModel {
	m := VMDiskModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	m.Path = stringAttrFromMap(dev.Attributes, "path")
	m.Type = stringAttrFromMap(dev.Attributes, "type")
	m.IOType = stringAttrFromMap(dev.Attributes, "iotype")
	m.Serial = stringAttrFromMap(dev.Attributes, "serial")
	m.LogicalSectorSize = intAttrFromMap(dev.Attributes, "logical_sectorsize")
	m.PhysicalSectorSize = intAttrFromMap(dev.Attributes, "physical_sectorsize")
	return m
}

func mapRawDevice(dev vmDeviceAPIResponse) VMRawModel {
	m := VMRawModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	m.Path = stringAttrFromMap(dev.Attributes, "path")
	m.Type = stringAttrFromMap(dev.Attributes, "type")
	m.IOType = stringAttrFromMap(dev.Attributes, "iotype")
	m.Serial = stringAttrFromMap(dev.Attributes, "serial")
	m.Boot = boolAttrFromMap(dev.Attributes, "boot")
	// exists is a create-time flag, not stored state — preserve plan/state value
	m.Size = intAttrFromMap(dev.Attributes, "size")
	m.LogicalSectorSize = intAttrFromMap(dev.Attributes, "logical_sectorsize")
	m.PhysicalSectorSize = intAttrFromMap(dev.Attributes, "physical_sectorsize")
	return m
}

func mapCDROMDevice(dev vmDeviceAPIResponse) VMCDROMModel {
	return VMCDROMModel{
		DeviceID: types.Int64Value(dev.ID),
		Path:     stringAttrFromMap(dev.Attributes, "path"),
		Order:    types.Int64Value(dev.Order),
	}
}

func mapNICDevice(dev vmDeviceAPIResponse) VMNICModel {
	m := VMNICModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	m.Type = stringAttrFromMap(dev.Attributes, "type")
	m.NICAttach = stringAttrFromMap(dev.Attributes, "nic_attach")
	m.MAC = stringAttrFromMap(dev.Attributes, "mac")
	m.TrustGuestRXFilters = boolAttrFromMap(dev.Attributes, "trust_guest_rx_filters")
	return m
}

func mapDisplayDevice(dev vmDeviceAPIResponse) VMDisplayModel {
	m := VMDisplayModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	m.Type = stringAttrFromMap(dev.Attributes, "type")
	m.Resolution = stringAttrFromMap(dev.Attributes, "resolution")
	m.Port = intAttrFromMap(dev.Attributes, "port")
	m.WebPort = intAttrFromMap(dev.Attributes, "web_port")
	m.Bind = stringAttrFromMap(dev.Attributes, "bind")
	m.Wait = boolAttrFromMap(dev.Attributes, "wait")
	m.Password = stringAttrFromMap(dev.Attributes, "password")
	m.Web = boolAttrFromMap(dev.Attributes, "web")
	return m
}

func mapPCIDevice(dev vmDeviceAPIResponse) VMPCIModel {
	return VMPCIModel{
		DeviceID: types.Int64Value(dev.ID),
		PPTDev:   stringAttrFromMap(dev.Attributes, "pptdev"),
		Order:    types.Int64Value(dev.Order),
	}
}

func mapUSBDevice(dev vmDeviceAPIResponse) VMUSBModel {
	return VMUSBModel{
		DeviceID:       types.Int64Value(dev.ID),
		ControllerType: stringAttrFromMap(dev.Attributes, "controller_type"),
		Device:         stringAttrFromMap(dev.Attributes, "device"),
		Order:          types.Int64Value(dev.Order),
	}
}

// Attribute extraction helpers

func stringAttrFromMap(m map[string]any, key string) types.String {
	v, ok := m[key]
	if !ok || v == nil {
		return types.StringNull()
	}
	s, ok := v.(string)
	if !ok {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func intAttrFromMap(m map[string]any, key string) types.Int64 {
	v, ok := m[key]
	if !ok || v == nil {
		return types.Int64Null()
	}
	switch n := v.(type) {
	case float64:
		return types.Int64Value(int64(n))
	case int64:
		return types.Int64Value(n)
	case json.Number:
		i, _ := n.Int64()
		return types.Int64Value(i)
	}
	return types.Int64Null()
}

func boolAttrFromMap(m map[string]any, key string) types.Bool {
	v, ok := m[key]
	if !ok || v == nil {
		return types.BoolNull()
	}
	b, ok := v.(bool)
	if !ok {
		return types.BoolNull()
	}
	return types.BoolValue(b)
}
