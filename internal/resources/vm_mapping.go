package resources

import (
	"strconv"

	truenas "github.com/deevus/truenas-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mapVMToModel maps a truenas.VM to the resource model.
func (r *VMResource) mapVMToModel(vm *truenas.VM, data *VMResourceModel) {
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
	if vm.CPUModel != "" {
		data.CPUModel = types.StringValue(vm.CPUModel)
	} else {
		data.CPUModel = types.StringNull()
	}
	data.ShutdownTimeout = types.Int64Value(vm.ShutdownTimeout)
	data.CommandLineArgs = types.StringValue(vm.CommandLineArgs)
	data.State = types.StringValue(vm.State)
	data.DisplayAvailable = types.BoolValue(false)
}

// mapDevicesToModel maps truenas.VMDevice slices to the resource model.
func (r *VMResource) mapDevicesToModel(devices []truenas.VMDevice, data *VMResourceModel) {
	data.Disks = nil
	data.Raws = nil
	data.CDROMs = nil
	data.NICs = nil
	data.Displays = nil
	data.PCIs = nil
	data.USBs = nil

	for _, dev := range devices {
		switch dev.DeviceType {
		case truenas.DeviceTypeDisk:
			data.Disks = append(data.Disks, mapDiskDevice(dev))
		case truenas.DeviceTypeRaw:
			data.Raws = append(data.Raws, mapRawDevice(dev))
		case truenas.DeviceTypeCDROM:
			data.CDROMs = append(data.CDROMs, mapCDROMDevice(dev))
		case truenas.DeviceTypeNIC:
			data.NICs = append(data.NICs, mapNICDevice(dev))
		case truenas.DeviceTypeDisplay:
			data.Displays = append(data.Displays, mapDisplayDevice(dev))
		case truenas.DeviceTypePCI:
			data.PCIs = append(data.PCIs, mapPCIDevice(dev))
		case truenas.DeviceTypeUSB:
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

func mapDiskDevice(dev truenas.VMDevice) VMDiskModel {
	m := VMDiskModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	if dev.Disk != nil {
		m.Path = nonEmptyStringValue(dev.Disk.Path)
		m.Type = nonEmptyStringValue(dev.Disk.Type)
		m.IOType = nonEmptyStringValue(dev.Disk.IOType)
		m.Serial = nonEmptyStringValue(dev.Disk.Serial)
		m.LogicalSectorSize = nilableInt64Value(dev.Disk.Logical_Sector_Size)
		m.PhysicalSectorSize = nilableInt64Value(dev.Disk.PhysicalSectorSize)
	}
	return m
}

func mapRawDevice(dev truenas.VMDevice) VMRawModel {
	m := VMRawModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	if dev.Raw != nil {
		m.Path = nonEmptyStringValue(dev.Raw.Path)
		m.Type = nonEmptyStringValue(dev.Raw.Type)
		m.IOType = nonEmptyStringValue(dev.Raw.IOType)
		m.Serial = nonEmptyStringValue(dev.Raw.Serial)
		m.Boot = types.BoolValue(dev.Raw.Boot)
		// exists is a create-time flag, not stored state — preserve plan/state value
		m.Size = nilableInt64Value(dev.Raw.Size)
		m.LogicalSectorSize = nilableInt64Value(dev.Raw.Logical_Sector_Size)
		m.PhysicalSectorSize = nilableInt64Value(dev.Raw.PhysicalSectorSize)
	}
	return m
}

func mapCDROMDevice(dev truenas.VMDevice) VMCDROMModel {
	m := VMCDROMModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	if dev.CDROM != nil {
		m.Path = nonEmptyStringValue(dev.CDROM.Path)
	}
	return m
}

func mapNICDevice(dev truenas.VMDevice) VMNICModel {
	m := VMNICModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	if dev.NIC != nil {
		m.Type = nonEmptyStringValue(dev.NIC.Type)
		m.NICAttach = nonEmptyStringValue(dev.NIC.NICAttach)
		m.MAC = nonEmptyStringValue(dev.NIC.MAC)
		m.TrustGuestRXFilters = types.BoolValue(dev.NIC.TrustGuestRxFilters)
	}
	return m
}

func mapDisplayDevice(dev truenas.VMDevice) VMDisplayModel {
	m := VMDisplayModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	if dev.Display != nil {
		m.Type = nonEmptyStringValue(dev.Display.Type)
		m.Resolution = nonEmptyStringValue(dev.Display.Resolution)
		m.Port = zeroableInt64Value(dev.Display.Port)
		m.WebPort = zeroableInt64Value(dev.Display.WebPort)
		m.Bind = nonEmptyStringValue(dev.Display.Bind)
		m.Wait = types.BoolValue(dev.Display.Wait)
		m.Password = nonEmptyStringValue(dev.Display.Password)
		m.Web = types.BoolValue(dev.Display.Web)
	}
	return m
}

func mapPCIDevice(dev truenas.VMDevice) VMPCIModel {
	m := VMPCIModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	if dev.PCI != nil {
		m.PPTDev = nonEmptyStringValue(dev.PCI.PPTDev)
	}
	return m
}

func mapUSBDevice(dev truenas.VMDevice) VMUSBModel {
	m := VMUSBModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	if dev.USB != nil {
		m.ControllerType = nonEmptyStringValue(dev.USB.ControllerType)
		m.Device = nonEmptyStringValue(dev.USB.Device)
	}
	return m
}

// nonEmptyStringValue returns a types.String from a value, or null if empty.
func nonEmptyStringValue(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// nilableInt64Value converts an *int64 to types.Int64, returning null if nil.
func nilableInt64Value(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}

// zeroableInt64Value converts an int64 to types.Int64, returning null if zero.
func zeroableInt64Value(v int64) types.Int64 {
	if v == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(v)
}
