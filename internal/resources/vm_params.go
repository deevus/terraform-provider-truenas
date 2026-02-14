package resources


func (r *VMResource) buildCreateParams(data *VMResourceModel) map[string]any {
	params := map[string]any{
		"name":   data.Name.ValueString(),
		"memory": data.Memory.ValueInt64(),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		params["description"] = data.Description.ValueString()
	}
	if !data.VCPUs.IsNull() && !data.VCPUs.IsUnknown() {
		params["vcpus"] = data.VCPUs.ValueInt64()
	}
	if !data.Cores.IsNull() && !data.Cores.IsUnknown() {
		params["cores"] = data.Cores.ValueInt64()
	}
	if !data.Threads.IsNull() && !data.Threads.IsUnknown() {
		params["threads"] = data.Threads.ValueInt64()
	}
	if !data.MinMemory.IsNull() && !data.MinMemory.IsUnknown() {
		params["min_memory"] = data.MinMemory.ValueInt64()
	}
	if !data.Autostart.IsNull() && !data.Autostart.IsUnknown() {
		params["autostart"] = data.Autostart.ValueBool()
	}
	if !data.Time.IsNull() && !data.Time.IsUnknown() {
		params["time"] = data.Time.ValueString()
	}
	if !data.Bootloader.IsNull() && !data.Bootloader.IsUnknown() {
		params["bootloader"] = data.Bootloader.ValueString()
	}
	if !data.BootloaderOVMF.IsNull() && !data.BootloaderOVMF.IsUnknown() {
		params["bootloader_ovmf"] = data.BootloaderOVMF.ValueString()
	}
	if !data.CPUMode.IsNull() && !data.CPUMode.IsUnknown() {
		params["cpu_mode"] = data.CPUMode.ValueString()
	}
	if !data.CPUModel.IsNull() && !data.CPUModel.IsUnknown() {
		params["cpu_model"] = data.CPUModel.ValueString()
	}
	if !data.ShutdownTimeout.IsNull() && !data.ShutdownTimeout.IsUnknown() {
		params["shutdown_timeout"] = data.ShutdownTimeout.ValueInt64()
	}

	return params
}

func (r *VMResource) buildUpdateParams(plan, state *VMResourceModel) map[string]any {
	params := map[string]any{}

	if !plan.Name.Equal(state.Name) {
		params["name"] = plan.Name.ValueString()
	}
	if !plan.Description.Equal(state.Description) {
		params["description"] = plan.Description.ValueString()
	}
	if !plan.VCPUs.Equal(state.VCPUs) {
		params["vcpus"] = plan.VCPUs.ValueInt64()
	}
	if !plan.Cores.Equal(state.Cores) {
		params["cores"] = plan.Cores.ValueInt64()
	}
	if !plan.Threads.Equal(state.Threads) {
		params["threads"] = plan.Threads.ValueInt64()
	}
	if !plan.Memory.Equal(state.Memory) {
		params["memory"] = plan.Memory.ValueInt64()
	}
	if !plan.MinMemory.Equal(state.MinMemory) {
		if plan.MinMemory.IsNull() {
			params["min_memory"] = nil
		} else {
			params["min_memory"] = plan.MinMemory.ValueInt64()
		}
	}
	if !plan.Autostart.Equal(state.Autostart) {
		params["autostart"] = plan.Autostart.ValueBool()
	}
	if !plan.Time.Equal(state.Time) {
		params["time"] = plan.Time.ValueString()
	}
	if !plan.Bootloader.Equal(state.Bootloader) {
		params["bootloader"] = plan.Bootloader.ValueString()
	}
	if !plan.BootloaderOVMF.Equal(state.BootloaderOVMF) {
		params["bootloader_ovmf"] = plan.BootloaderOVMF.ValueString()
	}
	if !plan.CPUMode.Equal(state.CPUMode) {
		params["cpu_mode"] = plan.CPUMode.ValueString()
	}
	if !plan.CPUModel.Equal(state.CPUModel) {
		if plan.CPUModel.IsNull() {
			params["cpu_model"] = nil
		} else {
			params["cpu_model"] = plan.CPUModel.ValueString()
		}
	}
	if !plan.ShutdownTimeout.Equal(state.ShutdownTimeout) {
		params["shutdown_timeout"] = plan.ShutdownTimeout.ValueInt64()
	}

	return params
}

// -- Device param builders --

func buildDiskDeviceParams(disk *VMDiskModel, vmID int64) map[string]any {
	attrs := map[string]any{"dtype": "DISK"}
	if !disk.Path.IsNull() {
		attrs["path"] = disk.Path.ValueString()
	}
	if !disk.Type.IsNull() && !disk.Type.IsUnknown() {
		attrs["type"] = disk.Type.ValueString()
	}
	if !disk.LogicalSectorSize.IsNull() {
		attrs["logical_sectorsize"] = disk.LogicalSectorSize.ValueInt64()
	}
	if !disk.PhysicalSectorSize.IsNull() {
		attrs["physical_sectorsize"] = disk.PhysicalSectorSize.ValueInt64()
	}
	if !disk.IOType.IsNull() && !disk.IOType.IsUnknown() {
		attrs["iotype"] = disk.IOType.ValueString()
	}
	if !disk.Serial.IsNull() {
		attrs["serial"] = disk.Serial.ValueString()
	}

	params := map[string]any{"vm": vmID, "attributes": attrs}
	if !disk.Order.IsNull() && !disk.Order.IsUnknown() {
		params["order"] = disk.Order.ValueInt64()
	}
	return params
}

func buildRawDeviceParams(raw *VMRawModel, vmID int64) map[string]any {
	attrs := map[string]any{"dtype": "RAW"}
	if !raw.Path.IsNull() {
		attrs["path"] = raw.Path.ValueString()
	}
	if !raw.Type.IsNull() && !raw.Type.IsUnknown() {
		attrs["type"] = raw.Type.ValueString()
	}
	if !raw.Boot.IsNull() && !raw.Boot.IsUnknown() {
		attrs["boot"] = raw.Boot.ValueBool()
	}
	if !raw.Size.IsNull() {
		attrs["size"] = raw.Size.ValueInt64()
	}
	if !raw.LogicalSectorSize.IsNull() {
		attrs["logical_sectorsize"] = raw.LogicalSectorSize.ValueInt64()
	}
	if !raw.PhysicalSectorSize.IsNull() {
		attrs["physical_sectorsize"] = raw.PhysicalSectorSize.ValueInt64()
	}
	if !raw.IOType.IsNull() && !raw.IOType.IsUnknown() {
		attrs["iotype"] = raw.IOType.ValueString()
	}
	if !raw.Serial.IsNull() {
		attrs["serial"] = raw.Serial.ValueString()
	}

	params := map[string]any{"vm": vmID, "attributes": attrs}
	if !raw.Order.IsNull() && !raw.Order.IsUnknown() {
		params["order"] = raw.Order.ValueInt64()
	}
	return params
}

func buildCDROMDeviceParams(cdrom *VMCDROMModel, vmID int64) map[string]any {
	attrs := map[string]any{"dtype": "CDROM"}
	if !cdrom.Path.IsNull() {
		attrs["path"] = cdrom.Path.ValueString()
	}

	params := map[string]any{"vm": vmID, "attributes": attrs}
	if !cdrom.Order.IsNull() && !cdrom.Order.IsUnknown() {
		params["order"] = cdrom.Order.ValueInt64()
	}
	return params
}

func buildNICDeviceParams(nic *VMNICModel, vmID int64) map[string]any {
	attrs := map[string]any{"dtype": "NIC"}
	if !nic.Type.IsNull() && !nic.Type.IsUnknown() {
		attrs["type"] = nic.Type.ValueString()
	}
	if !nic.NICAttach.IsNull() {
		attrs["nic_attach"] = nic.NICAttach.ValueString()
	}
	if !nic.MAC.IsNull() && !nic.MAC.IsUnknown() {
		attrs["mac"] = nic.MAC.ValueString()
	}
	if !nic.TrustGuestRXFilters.IsNull() && !nic.TrustGuestRXFilters.IsUnknown() {
		attrs["trust_guest_rx_filters"] = nic.TrustGuestRXFilters.ValueBool()
	}

	params := map[string]any{"vm": vmID, "attributes": attrs}
	if !nic.Order.IsNull() && !nic.Order.IsUnknown() {
		params["order"] = nic.Order.ValueInt64()
	}
	return params
}

func buildDisplayDeviceParams(display *VMDisplayModel, vmID int64) map[string]any {
	attrs := map[string]any{"dtype": "DISPLAY"}
	if !display.Type.IsNull() && !display.Type.IsUnknown() {
		attrs["type"] = display.Type.ValueString()
	}
	if !display.Resolution.IsNull() && !display.Resolution.IsUnknown() {
		attrs["resolution"] = display.Resolution.ValueString()
	}
	if !display.Port.IsNull() && !display.Port.IsUnknown() {
		attrs["port"] = display.Port.ValueInt64()
	}
	if !display.WebPort.IsNull() && !display.WebPort.IsUnknown() {
		attrs["web_port"] = display.WebPort.ValueInt64()
	}
	if !display.Bind.IsNull() && !display.Bind.IsUnknown() {
		attrs["bind"] = display.Bind.ValueString()
	}
	if !display.Wait.IsNull() && !display.Wait.IsUnknown() {
		attrs["wait"] = display.Wait.ValueBool()
	}
	if !display.Password.IsNull() && !display.Password.IsUnknown() {
		attrs["password"] = display.Password.ValueString()
	}
	if !display.Web.IsNull() && !display.Web.IsUnknown() {
		attrs["web"] = display.Web.ValueBool()
	}

	params := map[string]any{"vm": vmID, "attributes": attrs}
	if !display.Order.IsNull() && !display.Order.IsUnknown() {
		params["order"] = display.Order.ValueInt64()
	}
	return params
}

func buildPCIDeviceParams(pci *VMPCIModel, vmID int64) map[string]any {
	attrs := map[string]any{"dtype": "PCI"}
	if !pci.PPTDev.IsNull() {
		attrs["pptdev"] = pci.PPTDev.ValueString()
	}

	params := map[string]any{"vm": vmID, "attributes": attrs}
	if !pci.Order.IsNull() && !pci.Order.IsUnknown() {
		params["order"] = pci.Order.ValueInt64()
	}
	return params
}

func buildUSBDeviceParams(usb *VMUSBModel, vmID int64) map[string]any {
	attrs := map[string]any{"dtype": "USB"}
	if !usb.ControllerType.IsNull() && !usb.ControllerType.IsUnknown() {
		attrs["controller_type"] = usb.ControllerType.ValueString()
	}
	if !usb.Device.IsNull() {
		attrs["device"] = usb.Device.ValueString()
	}

	params := map[string]any{"vm": vmID, "attributes": attrs}
	if !usb.Order.IsNull() && !usb.Order.IsUnknown() {
		params["order"] = usb.Order.ValueInt64()
	}
	return params
}

