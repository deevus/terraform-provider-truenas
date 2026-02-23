package resources

import (
	truenas "github.com/deevus/truenas-go"
)

func (r *VMResource) buildCreateOpts(data *VMResourceModel) truenas.CreateVMOpts {
	opts := truenas.CreateVMOpts{
		Name:            data.Name.ValueString(),
		Description:     data.Description.ValueString(),
		VCPUs:           data.VCPUs.ValueInt64(),
		Cores:           data.Cores.ValueInt64(),
		Threads:         data.Threads.ValueInt64(),
		Memory:          data.Memory.ValueInt64(),
		Autostart:       data.Autostart.ValueBool(),
		Time:            data.Time.ValueString(),
		Bootloader:      data.Bootloader.ValueString(),
		BootloaderOVMF:  data.BootloaderOVMF.ValueString(),
		CPUMode:         data.CPUMode.ValueString(),
		ShutdownTimeout: data.ShutdownTimeout.ValueInt64(),
		CommandLineArgs: data.CommandLineArgs.ValueString(),
	}
	if !data.MinMemory.IsNull() && !data.MinMemory.IsUnknown() {
		v := data.MinMemory.ValueInt64()
		opts.MinMemory = &v
	}
	if !data.CPUModel.IsNull() && !data.CPUModel.IsUnknown() {
		opts.CPUModel = data.CPUModel.ValueString()
	}
	return opts
}

func (r *VMResource) buildUpdateOpts(plan, state *VMResourceModel) (*truenas.UpdateVMOpts, bool) {
	opts := r.buildCreateOpts(plan)
	changed := !plan.Name.Equal(state.Name) ||
		!plan.Description.Equal(state.Description) ||
		!plan.VCPUs.Equal(state.VCPUs) ||
		!plan.Cores.Equal(state.Cores) ||
		!plan.Threads.Equal(state.Threads) ||
		!plan.Memory.Equal(state.Memory) ||
		!plan.MinMemory.Equal(state.MinMemory) ||
		!plan.Autostart.Equal(state.Autostart) ||
		!plan.Time.Equal(state.Time) ||
		!plan.Bootloader.Equal(state.Bootloader) ||
		!plan.BootloaderOVMF.Equal(state.BootloaderOVMF) ||
		!plan.CPUMode.Equal(state.CPUMode) ||
		!plan.CPUModel.Equal(state.CPUModel) ||
		!plan.ShutdownTimeout.Equal(state.ShutdownTimeout) ||
		!plan.CommandLineArgs.Equal(state.CommandLineArgs)
	if !changed {
		return nil, false
	}
	return &opts, true
}

// -- Device opts builders --

func buildDiskDeviceOpts(disk *VMDiskModel, vmID int64) truenas.CreateVMDeviceOpts {
	d := &truenas.DiskDevice{}
	if !disk.Path.IsNull() {
		d.Path = disk.Path.ValueString()
	}
	if !disk.Type.IsNull() && !disk.Type.IsUnknown() {
		d.Type = disk.Type.ValueString()
	}
	if !disk.IOType.IsNull() && !disk.IOType.IsUnknown() {
		d.IOType = disk.IOType.ValueString()
	}
	if !disk.Serial.IsNull() && !disk.Serial.IsUnknown() {
		d.Serial = disk.Serial.ValueString()
	}
	if !disk.LogicalSectorSize.IsNull() && !disk.LogicalSectorSize.IsUnknown() {
		v := disk.LogicalSectorSize.ValueInt64()
		d.Logical_Sector_Size = &v
	}
	if !disk.PhysicalSectorSize.IsNull() && !disk.PhysicalSectorSize.IsUnknown() {
		v := disk.PhysicalSectorSize.ValueInt64()
		d.PhysicalSectorSize = &v
	}

	opts := truenas.CreateVMDeviceOpts{
		VM:         vmID,
		DeviceType: truenas.DeviceTypeDisk,
		Disk:       d,
	}
	if !disk.Order.IsNull() && !disk.Order.IsUnknown() {
		v := disk.Order.ValueInt64()
		opts.Order = &v
	}
	return opts
}

func buildRawDeviceOpts(raw *VMRawModel, vmID int64) truenas.CreateVMDeviceOpts {
	r := &truenas.RawDevice{}
	if !raw.Path.IsNull() {
		r.Path = raw.Path.ValueString()
	}
	if !raw.Type.IsNull() && !raw.Type.IsUnknown() {
		r.Type = raw.Type.ValueString()
	}
	if !raw.Boot.IsNull() && !raw.Boot.IsUnknown() {
		r.Boot = raw.Boot.ValueBool()
	}
	if !raw.Exists.IsNull() && !raw.Exists.IsUnknown() {
		r.Exists = raw.Exists.ValueBool()
	}
	if !raw.IOType.IsNull() && !raw.IOType.IsUnknown() {
		r.IOType = raw.IOType.ValueString()
	}
	if !raw.Serial.IsNull() && !raw.Serial.IsUnknown() {
		r.Serial = raw.Serial.ValueString()
	}
	if !raw.Size.IsNull() && !raw.Size.IsUnknown() {
		v := raw.Size.ValueInt64()
		r.Size = &v
	}
	if !raw.LogicalSectorSize.IsNull() && !raw.LogicalSectorSize.IsUnknown() {
		v := raw.LogicalSectorSize.ValueInt64()
		r.Logical_Sector_Size = &v
	}
	if !raw.PhysicalSectorSize.IsNull() && !raw.PhysicalSectorSize.IsUnknown() {
		v := raw.PhysicalSectorSize.ValueInt64()
		r.PhysicalSectorSize = &v
	}

	opts := truenas.CreateVMDeviceOpts{
		VM:         vmID,
		DeviceType: truenas.DeviceTypeRaw,
		Raw:        r,
	}
	if !raw.Order.IsNull() && !raw.Order.IsUnknown() {
		v := raw.Order.ValueInt64()
		opts.Order = &v
	}
	return opts
}

func buildCDROMDeviceOpts(cdrom *VMCDROMModel, vmID int64) truenas.CreateVMDeviceOpts {
	c := &truenas.CDROMDevice{}
	if !cdrom.Path.IsNull() {
		c.Path = cdrom.Path.ValueString()
	}

	opts := truenas.CreateVMDeviceOpts{
		VM:         vmID,
		DeviceType: truenas.DeviceTypeCDROM,
		CDROM:      c,
	}
	if !cdrom.Order.IsNull() && !cdrom.Order.IsUnknown() {
		v := cdrom.Order.ValueInt64()
		opts.Order = &v
	}
	return opts
}

func buildNICDeviceOpts(nic *VMNICModel, vmID int64) truenas.CreateVMDeviceOpts {
	n := &truenas.NICDevice{}
	if !nic.Type.IsNull() && !nic.Type.IsUnknown() {
		n.Type = nic.Type.ValueString()
	}
	if !nic.NICAttach.IsNull() {
		n.NICAttach = nic.NICAttach.ValueString()
	}
	if !nic.MAC.IsNull() && !nic.MAC.IsUnknown() {
		n.MAC = nic.MAC.ValueString()
	}
	if !nic.TrustGuestRXFilters.IsNull() && !nic.TrustGuestRXFilters.IsUnknown() {
		n.TrustGuestRxFilters = nic.TrustGuestRXFilters.ValueBool()
	}

	opts := truenas.CreateVMDeviceOpts{
		VM:         vmID,
		DeviceType: truenas.DeviceTypeNIC,
		NIC:        n,
	}
	if !nic.Order.IsNull() && !nic.Order.IsUnknown() {
		v := nic.Order.ValueInt64()
		opts.Order = &v
	}
	return opts
}

func buildDisplayDeviceOpts(display *VMDisplayModel, vmID int64) truenas.CreateVMDeviceOpts {
	d := &truenas.DisplayDevice{}
	if !display.Type.IsNull() && !display.Type.IsUnknown() {
		d.Type = display.Type.ValueString()
	}
	if !display.Resolution.IsNull() && !display.Resolution.IsUnknown() {
		d.Resolution = display.Resolution.ValueString()
	}
	if !display.Port.IsNull() && !display.Port.IsUnknown() {
		d.Port = display.Port.ValueInt64()
	}
	if !display.WebPort.IsNull() && !display.WebPort.IsUnknown() {
		d.WebPort = display.WebPort.ValueInt64()
	}
	if !display.Bind.IsNull() && !display.Bind.IsUnknown() {
		d.Bind = display.Bind.ValueString()
	}
	if !display.Wait.IsNull() && !display.Wait.IsUnknown() {
		d.Wait = display.Wait.ValueBool()
	}
	if !display.Password.IsNull() && !display.Password.IsUnknown() {
		d.Password = display.Password.ValueString()
	}
	if !display.Web.IsNull() && !display.Web.IsUnknown() {
		d.Web = display.Web.ValueBool()
	}

	opts := truenas.CreateVMDeviceOpts{
		VM:         vmID,
		DeviceType: truenas.DeviceTypeDisplay,
		Display:    d,
	}
	if !display.Order.IsNull() && !display.Order.IsUnknown() {
		v := display.Order.ValueInt64()
		opts.Order = &v
	}
	return opts
}

func buildPCIDeviceOpts(pci *VMPCIModel, vmID int64) truenas.CreateVMDeviceOpts {
	p := &truenas.PCIDevice{}
	if !pci.PPTDev.IsNull() {
		p.PPTDev = pci.PPTDev.ValueString()
	}

	opts := truenas.CreateVMDeviceOpts{
		VM:         vmID,
		DeviceType: truenas.DeviceTypePCI,
		PCI:        p,
	}
	if !pci.Order.IsNull() && !pci.Order.IsUnknown() {
		v := pci.Order.ValueInt64()
		opts.Order = &v
	}
	return opts
}

func buildUSBDeviceOpts(usb *VMUSBModel, vmID int64) truenas.CreateVMDeviceOpts {
	u := &truenas.USBDevice{}
	if !usb.ControllerType.IsNull() && !usb.ControllerType.IsUnknown() {
		u.ControllerType = usb.ControllerType.ValueString()
	}
	if !usb.Device.IsNull() {
		u.Device = usb.Device.ValueString()
	}

	opts := truenas.CreateVMDeviceOpts{
		VM:         vmID,
		DeviceType: truenas.DeviceTypeUSB,
		USB:        u,
	}
	if !usb.Order.IsNull() && !usb.Order.IsUnknown() {
		v := usb.Order.ValueInt64()
		opts.Order = &v
	}
	return opts
}
