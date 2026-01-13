# Virtual Machines API

Virtual machine management operations.

## VM Operations

### vm.query
Query virtual machines.
```bash
midclt call vm.query
midclt call vm.query '[[["name", "=", "testvm"]]]'
midclt call vm.query '[[["status.state", "=", "RUNNING"]]]'
```

Returns:
- `id` - VM ID
- `name` - VM name
- `description` - VM description
- `vcpus` - Virtual CPUs
- `memory` - Memory in MB
- `min_memory` - Minimum memory (balloon)
- `autostart` - Start on boot
- `time` - Clock type (LOCAL, UTC)
- `bootloader` - UEFI or UEFI_CSM
- `bootloader_ovmf` - OVMF firmware type
- `cores` - CPU cores per socket
- `threads` - Threads per core
- `hyperv_enlightenments` - Hyper-V enlightenments
- `shutdown_timeout` - Shutdown timeout (seconds)
- `cpu_mode` - CPU mode (CUSTOM, HOST-MODEL, etc.)
- `cpu_model` - CPU model name
- `cpuset` - CPU pinning
- `nodeset` - NUMA node pinning
- `pin_vcpus` - Pin vCPUs to physical CPUs
- `hide_from_msr` - Hide hypervisor from MSR
- `suspend_on_snapshot` - Suspend during snapshot
- `ensure_display_device` - Ensure display device exists
- `arch_type` - Architecture (i686, x86_64)
- `machine_type` - Machine type (pc, q35)
- `uuid` - VM UUID
- `command_line_args` - Extra QEMU arguments
- `status` - Current status object
- `devices` - Attached devices

### vm.create
Create a virtual machine.
```bash
midclt call vm.create '{
  "name": "testvm",
  "description": "Test virtual machine",
  "vcpus": 2,
  "cores": 1,
  "threads": 1,
  "memory": 2048,
  "bootloader": "UEFI",
  "autostart": false,
  "time": "UTC",
  "shutdown_timeout": 90
}'
```

### vm.update
Update a virtual machine.
```bash
midclt call vm.update <vm_id> '{
  "vcpus": 4,
  "memory": 4096,
  "description": "Updated description"
}'
```

### vm.delete
Delete a virtual machine.
```bash
midclt call vm.delete <vm_id>
midclt call vm.delete <vm_id> '{"zvols": true, "force": false}'
```

### vm.start
Start a virtual machine.
```bash
midclt call vm.start <vm_id>
midclt call vm.start <vm_id> '{"overcommit": false}'
```

### vm.stop
Stop a virtual machine (graceful shutdown).
```bash
midclt call vm.stop <vm_id>
midclt call vm.stop <vm_id> '{"force": false, "force_after_timeout": true}'
```

### vm.poweroff
Force power off a virtual machine.
```bash
midclt call vm.poweroff <vm_id>
```

### vm.resume
Resume a suspended virtual machine.
```bash
midclt call vm.resume <vm_id>
```

### vm.clone
Clone a virtual machine.
```bash
midclt call vm.clone <vm_id> "cloned-vm"
```

### vm.get_available_memory
Get available memory for VMs.
```bash
midclt call vm.get_available_memory
```

### vm.get_display_devices
Get display devices for a VM.
```bash
midclt call vm.get_display_devices <vm_id>
```

### vm.get_display_web_uri
Get web display URI (VNC/SPICE).
```bash
midclt call vm.get_display_web_uri <vm_id>
midclt call vm.get_display_web_uri <vm_id> "192.168.1.10"
```

### vm.log_file_download
Get VM log file path for download.
```bash
midclt call vm.log_file_download <vm_id>
```

### vm.random_mac
Generate a random MAC address.
```bash
midclt call vm.random_mac
```

### vm.port_wizard
Get suggested port for VNC/SPICE.
```bash
midclt call vm.port_wizard
```

### vm.maximum_supported_vcpus
Get maximum supported vCPUs.
```bash
midclt call vm.maximum_supported_vcpus
```

### vm.virtualization_details
Get virtualization details.
```bash
midclt call vm.virtualization_details
```

### Choice Methods
```bash
midclt call vm.bootloader_options
midclt call vm.cpu_model_choices
midclt call vm.resolution_choices
```

## VM Devices

### vm.device.query
Query VM devices.
```bash
midclt call vm.device.query
midclt call vm.device.query '[[["vm", "=", <vm_id>]]]'
```

### vm.device.create
Create a VM device.

Disk device (zvol):
```bash
midclt call vm.device.create '{
  "vm": <vm_id>,
  "dtype": "DISK",
  "attributes": {
    "path": "/dev/zvol/tank/vms/testvm-disk0",
    "type": "VIRTIO",
    "logical_sectorsize": null,
    "physical_sectorsize": null
  },
  "order": 1000
}'
```

NIC device:
```bash
midclt call vm.device.create '{
  "vm": <vm_id>,
  "dtype": "NIC",
  "attributes": {
    "type": "VIRTIO",
    "mac": "00:a0:98:xx:xx:xx",
    "nic_attach": "br0"
  },
  "order": 1001
}'
```

CD-ROM device:
```bash
midclt call vm.device.create '{
  "vm": <vm_id>,
  "dtype": "CDROM",
  "attributes": {
    "path": "/mnt/tank/iso/ubuntu.iso"
  },
  "order": 1002
}'
```

Display device (VNC):
```bash
midclt call vm.device.create '{
  "vm": <vm_id>,
  "dtype": "DISPLAY",
  "attributes": {
    "type": "VNC",
    "bind": "0.0.0.0",
    "port": 5900,
    "resolution": "1024x768",
    "web": true,
    "password": ""
  },
  "order": 1003
}'
```

RAW file device:
```bash
midclt call vm.device.create '{
  "vm": <vm_id>,
  "dtype": "RAW",
  "attributes": {
    "path": "/mnt/tank/vms/testvm-disk1.img",
    "type": "VIRTIO",
    "size": 10737418240,
    "boot": false
  },
  "order": 1004
}'
```

PCI passthrough:
```bash
midclt call vm.device.create '{
  "vm": <vm_id>,
  "dtype": "PCI",
  "attributes": {
    "pptdev": "0000:01:00.0"
  },
  "order": 1005
}'
```

USB passthrough:
```bash
midclt call vm.device.create '{
  "vm": <vm_id>,
  "dtype": "USB",
  "attributes": {
    "controller_type": "qemu-xhci",
    "device": "usb_8086_1234_..."
  },
  "order": 1006
}'
```

### vm.device.update
Update a VM device.
```bash
midclt call vm.device.update <device_id> '{
  "attributes": {
    "path": "/mnt/tank/iso/different.iso"
  }
}'
```

### vm.device.delete
Delete a VM device.
```bash
midclt call vm.device.delete <device_id>
midclt call vm.device.delete <device_id> '{"zvol": true, "raw_file": true, "force": false}'
```

### vm.device.convert
Convert RAW device to VIRTIO.
```bash
midclt call vm.device.convert <device_id>
```

### vm.device.virtual_size
Get virtual size of a device.
```bash
midclt call vm.device.virtual_size <device_id>
```

### Choice Methods
```bash
midclt call vm.device.bind_choices
midclt call vm.device.disk_choices
midclt call vm.device.nic_attach_choices
midclt call vm.device.passthrough_device_choices
midclt call vm.device.usb_passthrough_choices
midclt call vm.device.usb_controller_choices
```

## Device Types

| Type | Description |
|------|-------------|
| `DISK` | Block device (zvol) |
| `RAW` | Raw file disk |
| `CDROM` | CD-ROM/ISO image |
| `NIC` | Network interface |
| `DISPLAY` | VNC/SPICE display |
| `PCI` | PCI passthrough |
| `USB` | USB passthrough |

## Disk Types

| Type | Description |
|------|-------------|
| `VIRTIO` | VirtIO block device |
| `AHCI` | AHCI/SATA emulation |
| `SCSI` | SCSI emulation |
| `IDE` | IDE emulation |

## NIC Types

| Type | Description |
|------|-------------|
| `VIRTIO` | VirtIO network |
| `E1000` | Intel E1000 emulation |

## Display Types

| Type | Description |
|------|-------------|
| `VNC` | VNC display |
| `SPICE` | SPICE display |
