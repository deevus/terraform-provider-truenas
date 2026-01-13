# Disks & Devices API

Disk and hardware device management.

## Disk Operations

### disk.query
Query disk information.
```bash
midclt call disk.query
midclt call disk.query '[[["name", "=", "sda"]]]'
midclt call disk.query '[[["pool", "=", "tank"]]]'
```

Returns:
- `identifier` - Disk identifier
- `name` - Device name (sda, nvme0n1)
- `subsystem` - Subsystem type
- `number` - Disk number
- `serial` - Serial number
- `lunid` - LUN ID
- `size` - Size in bytes
- `description` - Description
- `transfermode` - Transfer mode
- `hddstandby` - HDD standby timeout
- `advpowermgmt` - Advanced power management
- `togglesmart` - SMART enabled
- `smartoptions` - SMART options
- `expiretime` - Expire time
- `critical` - Critical temperature
- `difference` - Temperature difference alert
- `informational` - Informational temperature
- `model` - Model name
- `rotationrate` - Rotation rate (RPM)
- `type` - HDD, SSD
- `zfs_guid` - ZFS GUID
- `bus` - Bus type
- `devpath` - Device path
- `enclosure` - Enclosure info
- `pool` - Pool name if in pool
- `passwd` - SED password set
- `kmip_uid` - KMIP UID
- `supports_smart` - SMART support

### disk.details
Get detailed disk information.
```bash
midclt call disk.details
```

### disk.update
Update disk settings.
```bash
midclt call disk.update "sda" '{
  "hddstandby": "ALWAYS ON",
  "advpowermgmt": "DISABLED",
  "togglesmart": true,
  "critical": 60,
  "difference": 5,
  "informational": 45
}'
```

### disk.wipe
Wipe a disk.
```bash
midclt call disk.wipe "sda" "QUICK"
midclt call disk.wipe "sda" "FULL"
midclt call disk.wipe "sda" "FULL_RANDOM"
```

### disk.temperatures
Get disk temperatures.
```bash
midclt call disk.temperatures
midclt call disk.temperatures '["sda", "sdb"]'
```

### disk.temperature_agg
Get aggregated temperature data.
```bash
midclt call disk.temperature_agg '["sda"]' '{"start": "2024-01-01", "end": "2024-01-31"}'
```

### disk.temperature_alerts
Get temperature alert thresholds.
```bash
midclt call disk.temperature_alerts '["sda"]'
```

## SED (Self-Encrypting Drive) Operations

### disk.unlock_sed
Unlock a SED.
```bash
midclt call disk.unlock_sed "sda" "password"
```

### disk.reset_sed
Reset SED password.
```bash
midclt call disk.reset_sed "sda" "currentpassword" "newpassword"
```

## Device Information

### device.get_info
Get device information.
```bash
midclt call device.get_info "DISK"
midclt call device.get_info "SERIAL"
midclt call device.get_info "GPU"
```

Device types:
- `DISK` - Storage devices
- `SERIAL` - Serial ports
- `GPU` - Graphics cards

## Enclosure Management

### enclosure.label.set
Set enclosure slot label.
```bash
midclt call enclosure.label.set "slot_1" "Disk Bay 1"
```

### webui.enclosure.dashboard
Get enclosure dashboard data.
```bash
midclt call webui.enclosure.dashboard
```

## RDMA

### rdma.capable_protocols
Check RDMA-capable protocols.
```bash
midclt call rdma.capable_protocols
```

## HDD Standby Options

| Value | Description |
|-------|-------------|
| `ALWAYS ON` | Never spin down |
| `5` | 5 minutes |
| `10` | 10 minutes |
| `20` | 20 minutes |
| `30` | 30 minutes |
| `60` | 60 minutes |
| `120` | 2 hours |
| `180` | 3 hours |
| `240` | 4 hours |
| `300` | 5 hours |
| `330` | 5.5 hours |

## Advanced Power Management

| Value | Description |
|-------|-------------|
| `DISABLED` | Disabled |
| `1` | Minimum power with standby |
| `64` | Low power |
| `127` | Medium power |
| `128` | Medium power without standby |
| `192` | High performance |
| `254` | Maximum performance |

## Disk Wipe Methods

| Method | Description |
|--------|-------------|
| `QUICK` | Quick wipe (first/last sectors) |
| `FULL` | Full zero wipe |
| `FULL_RANDOM` | Full random wipe |
