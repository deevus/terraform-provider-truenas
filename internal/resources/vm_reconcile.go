package resources

import (
	"context"
	"fmt"

	truenas "github.com/deevus/truenas-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// reconcileDevices compares plan vs state devices and creates/updates/deletes as needed.
func (r *VMResource) reconcileDevices(ctx context.Context, vmID int64, plan, state *VMResourceModel) error {
	// Build maps of state device IDs to detect what exists
	stateDeviceIDs := make(map[int64]bool)
	collectDeviceIDs(stateDeviceIDs, state)

	planDeviceIDs := make(map[int64]bool)
	collectDeviceIDs(planDeviceIDs, plan)

	// Delete devices in state but not in plan
	if err := r.deleteRemovedDevices(ctx, stateDeviceIDs, planDeviceIDs); err != nil {
		return err
	}

	// Create/update devices
	if err := r.reconcileDiskDevices(ctx, vmID, plan.Disks, state.Disks); err != nil {
		return err
	}
	if err := r.reconcileRawDevices(ctx, vmID, plan.Raws, state.Raws); err != nil {
		return err
	}
	if err := r.reconcileCDROMDevices(ctx, vmID, plan.CDROMs, state.CDROMs); err != nil {
		return err
	}
	if err := r.reconcileNICDevices(ctx, vmID, plan.NICs, state.NICs); err != nil {
		return err
	}
	if err := r.reconcileDisplayDevices(ctx, vmID, plan.Displays, state.Displays); err != nil {
		return err
	}
	if err := r.reconcilePCIDevices(ctx, vmID, plan.PCIs, state.PCIs); err != nil {
		return err
	}
	if err := r.reconcileUSBDevices(ctx, vmID, plan.USBs, state.USBs); err != nil {
		return err
	}

	return nil
}

func collectDeviceIDs(ids map[int64]bool, data *VMResourceModel) {
	for _, d := range data.Disks {
		if !d.DeviceID.IsNull() && !d.DeviceID.IsUnknown() {
			ids[d.DeviceID.ValueInt64()] = true
		}
	}
	for _, d := range data.Raws {
		if !d.DeviceID.IsNull() && !d.DeviceID.IsUnknown() {
			ids[d.DeviceID.ValueInt64()] = true
		}
	}
	for _, d := range data.CDROMs {
		if !d.DeviceID.IsNull() && !d.DeviceID.IsUnknown() {
			ids[d.DeviceID.ValueInt64()] = true
		}
	}
	for _, d := range data.NICs {
		if !d.DeviceID.IsNull() && !d.DeviceID.IsUnknown() {
			ids[d.DeviceID.ValueInt64()] = true
		}
	}
	for _, d := range data.Displays {
		if !d.DeviceID.IsNull() && !d.DeviceID.IsUnknown() {
			ids[d.DeviceID.ValueInt64()] = true
		}
	}
	for _, d := range data.PCIs {
		if !d.DeviceID.IsNull() && !d.DeviceID.IsUnknown() {
			ids[d.DeviceID.ValueInt64()] = true
		}
	}
	for _, d := range data.USBs {
		if !d.DeviceID.IsNull() && !d.DeviceID.IsUnknown() {
			ids[d.DeviceID.ValueInt64()] = true
		}
	}
}

func (r *VMResource) deleteRemovedDevices(ctx context.Context, stateIDs, planIDs map[int64]bool) error {
	for id := range stateIDs {
		if !planIDs[id] {
			err := r.services.VM.DeleteDevice(ctx, id)
			if err != nil {
				return fmt.Errorf("failed to delete device %d: %w", id, err)
			}
		}
	}
	return nil
}

func (r *VMResource) reconcileDiskDevices(ctx context.Context, vmID int64, plan, state []VMDiskModel) error {
	stateByID := make(map[int64]VMDiskModel)
	for _, s := range state {
		if !s.DeviceID.IsNull() && !s.DeviceID.IsUnknown() {
			stateByID[s.DeviceID.ValueInt64()] = s
		}
	}

	for i, p := range plan {
		if p.DeviceID.IsNull() || p.DeviceID.IsUnknown() {
			// New device - create
			dev, err := r.services.VM.CreateDevice(ctx, buildDiskDeviceOpts(&p, vmID))
			if err != nil {
				return fmt.Errorf("failed to create disk device: %w", err)
			}
			plan[i].DeviceID = types.Int64Value(dev.ID)
		} else if s, ok := stateByID[p.DeviceID.ValueInt64()]; ok {
			// Existing device - update if changed
			if !diskEqual(p, s) {
				_, err := r.services.VM.UpdateDevice(ctx, p.DeviceID.ValueInt64(), buildDiskDeviceOpts(&p, vmID))
				if err != nil {
					return fmt.Errorf("failed to update disk device %d: %w", p.DeviceID.ValueInt64(), err)
				}
			}
		}
	}
	return nil
}

func diskEqual(a, b VMDiskModel) bool {
	return a.Path.Equal(b.Path) && a.Type.Equal(b.Type) &&
		a.LogicalSectorSize.Equal(b.LogicalSectorSize) &&
		a.PhysicalSectorSize.Equal(b.PhysicalSectorSize) &&
		a.IOType.Equal(b.IOType) && a.Serial.Equal(b.Serial)
}

func (r *VMResource) reconcileRawDevices(ctx context.Context, vmID int64, plan, state []VMRawModel) error {
	stateByID := make(map[int64]VMRawModel)
	for _, s := range state {
		if !s.DeviceID.IsNull() && !s.DeviceID.IsUnknown() {
			stateByID[s.DeviceID.ValueInt64()] = s
		}
	}
	for i, p := range plan {
		if p.DeviceID.IsNull() || p.DeviceID.IsUnknown() {
			dev, err := r.services.VM.CreateDevice(ctx, buildRawDeviceOpts(&p, vmID))
			if err != nil {
				return fmt.Errorf("failed to create raw device: %w", err)
			}
			plan[i].DeviceID = types.Int64Value(dev.ID)
		} else if s, ok := stateByID[p.DeviceID.ValueInt64()]; ok {
			if !rawEqual(p, s) {
				_, err := r.services.VM.UpdateDevice(ctx, p.DeviceID.ValueInt64(), buildRawDeviceOpts(&p, vmID))
				if err != nil {
					return fmt.Errorf("failed to update raw device: %w", err)
				}
			}
		}
	}
	return nil
}

func rawEqual(a, b VMRawModel) bool {
	return a.Path.Equal(b.Path) && a.Type.Equal(b.Type) && a.Boot.Equal(b.Boot) && a.Size.Equal(b.Size)
}

func (r *VMResource) reconcileCDROMDevices(ctx context.Context, vmID int64, plan, state []VMCDROMModel) error {
	stateByID := make(map[int64]VMCDROMModel)
	for _, s := range state {
		if !s.DeviceID.IsNull() && !s.DeviceID.IsUnknown() {
			stateByID[s.DeviceID.ValueInt64()] = s
		}
	}
	for i, p := range plan {
		if p.DeviceID.IsNull() || p.DeviceID.IsUnknown() {
			dev, err := r.services.VM.CreateDevice(ctx, buildCDROMDeviceOpts(&p, vmID))
			if err != nil {
				return fmt.Errorf("failed to create cdrom device: %w", err)
			}
			plan[i].DeviceID = types.Int64Value(dev.ID)
		} else if s, ok := stateByID[p.DeviceID.ValueInt64()]; ok {
			if !p.Path.Equal(s.Path) {
				_, err := r.services.VM.UpdateDevice(ctx, p.DeviceID.ValueInt64(), buildCDROMDeviceOpts(&p, vmID))
				if err != nil {
					return fmt.Errorf("failed to update cdrom device: %w", err)
				}
			}
		}
	}
	return nil
}

func (r *VMResource) reconcileNICDevices(ctx context.Context, vmID int64, plan, state []VMNICModel) error {
	stateByID := make(map[int64]VMNICModel)
	for _, s := range state {
		if !s.DeviceID.IsNull() && !s.DeviceID.IsUnknown() {
			stateByID[s.DeviceID.ValueInt64()] = s
		}
	}
	for i, p := range plan {
		if p.DeviceID.IsNull() || p.DeviceID.IsUnknown() {
			dev, err := r.services.VM.CreateDevice(ctx, buildNICDeviceOpts(&p, vmID))
			if err != nil {
				return fmt.Errorf("failed to create nic device: %w", err)
			}
			plan[i].DeviceID = types.Int64Value(dev.ID)
		} else if s, ok := stateByID[p.DeviceID.ValueInt64()]; ok {
			if !nicEqual(p, s) {
				_, err := r.services.VM.UpdateDevice(ctx, p.DeviceID.ValueInt64(), buildNICDeviceOpts(&p, vmID))
				if err != nil {
					return fmt.Errorf("failed to update nic device: %w", err)
				}
			}
		}
	}
	return nil
}

func nicEqual(a, b VMNICModel) bool {
	return a.Type.Equal(b.Type) && a.NICAttach.Equal(b.NICAttach) && a.MAC.Equal(b.MAC) && a.TrustGuestRXFilters.Equal(b.TrustGuestRXFilters)
}

func (r *VMResource) reconcileDisplayDevices(ctx context.Context, vmID int64, plan, state []VMDisplayModel) error {
	stateByID := make(map[int64]VMDisplayModel)
	for _, s := range state {
		if !s.DeviceID.IsNull() && !s.DeviceID.IsUnknown() {
			stateByID[s.DeviceID.ValueInt64()] = s
		}
	}
	for i, p := range plan {
		if p.DeviceID.IsNull() || p.DeviceID.IsUnknown() {
			dev, err := r.services.VM.CreateDevice(ctx, buildDisplayDeviceOpts(&p, vmID))
			if err != nil {
				return fmt.Errorf("failed to create display device: %w", err)
			}
			plan[i].DeviceID = types.Int64Value(dev.ID)
		} else if s, ok := stateByID[p.DeviceID.ValueInt64()]; ok {
			if !displayEqual(p, s) {
				_, err := r.services.VM.UpdateDevice(ctx, p.DeviceID.ValueInt64(), buildDisplayDeviceOpts(&p, vmID))
				if err != nil {
					return fmt.Errorf("failed to update display device: %w", err)
				}
			}
		}
	}
	return nil
}

func displayEqual(a, b VMDisplayModel) bool {
	return a.Type.Equal(b.Type) && a.Resolution.Equal(b.Resolution) && a.Bind.Equal(b.Bind) &&
		a.Web.Equal(b.Web) && a.Wait.Equal(b.Wait) && a.Port.Equal(b.Port) && a.WebPort.Equal(b.WebPort)
}

func (r *VMResource) reconcilePCIDevices(ctx context.Context, vmID int64, plan, state []VMPCIModel) error {
	stateByID := make(map[int64]VMPCIModel)
	for _, s := range state {
		if !s.DeviceID.IsNull() && !s.DeviceID.IsUnknown() {
			stateByID[s.DeviceID.ValueInt64()] = s
		}
	}
	for i, p := range plan {
		if p.DeviceID.IsNull() || p.DeviceID.IsUnknown() {
			dev, err := r.services.VM.CreateDevice(ctx, buildPCIDeviceOpts(&p, vmID))
			if err != nil {
				return fmt.Errorf("failed to create pci device: %w", err)
			}
			plan[i].DeviceID = types.Int64Value(dev.ID)
		} else if s, ok := stateByID[p.DeviceID.ValueInt64()]; ok {
			if !p.PPTDev.Equal(s.PPTDev) {
				_, err := r.services.VM.UpdateDevice(ctx, p.DeviceID.ValueInt64(), buildPCIDeviceOpts(&p, vmID))
				if err != nil {
					return fmt.Errorf("failed to update pci device: %w", err)
				}
			}
		}
	}
	return nil
}

func (r *VMResource) reconcileUSBDevices(ctx context.Context, vmID int64, plan, state []VMUSBModel) error {
	stateByID := make(map[int64]VMUSBModel)
	for _, s := range state {
		if !s.DeviceID.IsNull() && !s.DeviceID.IsUnknown() {
			stateByID[s.DeviceID.ValueInt64()] = s
		}
	}
	for i, p := range plan {
		if p.DeviceID.IsNull() || p.DeviceID.IsUnknown() {
			dev, err := r.services.VM.CreateDevice(ctx, buildUSBDeviceOpts(&p, vmID))
			if err != nil {
				return fmt.Errorf("failed to create usb device: %w", err)
			}
			plan[i].DeviceID = types.Int64Value(dev.ID)
		} else if s, ok := stateByID[p.DeviceID.ValueInt64()]; ok {
			if !usbEqual(p, s) {
				_, err := r.services.VM.UpdateDevice(ctx, p.DeviceID.ValueInt64(), buildUSBDeviceOpts(&p, vmID))
				if err != nil {
					return fmt.Errorf("failed to update usb device: %w", err)
				}
			}
		}
	}
	return nil
}

func usbEqual(a, b VMUSBModel) bool {
	return a.ControllerType.Equal(b.ControllerType) && a.Device.Equal(b.Device)
}

// reconcileState starts or stops the VM to match the desired state.
func (r *VMResource) reconcileState(ctx context.Context, vmID int64, currentState, desiredState string) error {
	if currentState == desiredState {
		return nil
	}

	if desiredState == VMStateRunning {
		return r.services.VM.StartVM(ctx, vmID)
	}

	// vm.stop is a job
	return r.services.VM.StopVM(ctx, vmID, truenas.StopVMOpts{Force: false, ForceAfterTimeout: true})
}
