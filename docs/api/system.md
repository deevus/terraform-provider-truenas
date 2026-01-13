# System API

System configuration, information, and management operations.

## System Information

### system.info
Get system information.
```bash
midclt call system.info
```

Returns:
- `version` - TrueNAS version
- `buildtime` - Build timestamp
- `hostname` - System hostname
- `physmem` - Physical memory (bytes)
- `model` - CPU model
- `cores` - CPU cores
- `physical_cores` - Physical CPU cores
- `loadavg` - Load averages
- `uptime` - System uptime
- `uptime_seconds` - Uptime in seconds
- `system_serial` - Hardware serial
- `system_product` - Hardware product
- `system_manufacturer` - Hardware manufacturer
- `license` - License information
- `boottime` - Boot timestamp
- `datetime` - Current datetime
- `birthday` - First boot date
- `timezone` - System timezone
- `system_product_version` - Product version

### system.host_id
Get system host ID.
```bash
midclt call system.host_id
```

### system.product_type
Get product type (SCALE, SCALE_ENTERPRISE, etc.).
```bash
midclt call system.product_type
```

## General Configuration

### system.general.config
Get general system configuration.
```bash
midclt call system.general.config
```

Returns:
- `id` - Config ID
- `ui_address` - Web UI bind addresses
- `ui_v6address` - IPv6 bind addresses
- `ui_port` - HTTP port
- `ui_httpsport` - HTTPS port
- `ui_httpsredirect` - Redirect HTTP to HTTPS
- `ui_httpsprotocols` - Allowed HTTPS protocols
- `ui_certificate` - UI certificate ID
- `ui_x_frame_options` - X-Frame-Options header
- `crash_reporting` - Crash reporting enabled
- `usage_collection` - Usage stats enabled
- `language` - UI language
- `kbdmap` - Keyboard map
- `timezone` - System timezone
- `wizardshown` - Setup wizard completed

### system.general.update
Update general configuration.
```bash
midclt call system.general.update '{
  "timezone": "America/Los_Angeles",
  "language": "en",
  "ui_httpsredirect": true
}'
```

### system.general.timezone_choices
Get available timezones.
```bash
midclt call system.general.timezone_choices
```

### system.general.kbdmap_choices
Get available keyboard maps.
```bash
midclt call system.general.kbdmap_choices
```

### system.general.ui_address_choices
Get available UI bind addresses.
```bash
midclt call system.general.ui_address_choices
```

### system.general.ui_certificate_choices
Get available UI certificates.
```bash
midclt call system.general.ui_certificate_choices
```

### system.general.ui_httpsprotocols_choices
Get available HTTPS protocols.
```bash
midclt call system.general.ui_httpsprotocols_choices
```

### system.general.ui_restart
Restart web UI.
```bash
midclt call system.general.ui_restart
```

## Advanced Configuration

### system.advanced.config
Get advanced system configuration.
```bash
midclt call system.advanced.config
```

Returns:
- `id` - Config ID
- `consolemenu` - Console menu enabled
- `serialconsole` - Serial console enabled
- `serialport` - Serial port device
- `serialspeed` - Serial port speed
- `swapondrive` - Swap size on drives
- `autotune` - Auto-tuning enabled
- `debugkernel` - Debug kernel enabled
- `uploadcrash` - Crash upload enabled
- `motd` - Message of the day
- `boot_scrub` - Boot pool scrub interval
- `fqdn_syslog` - Use FQDN in syslog
- `sysloglevel` - Syslog level
- `syslogserver` - Remote syslog server
- `syslog_transport` - Syslog transport (UDP/TCP/TLS)
- `syslog_tls_certificate` - Syslog TLS certificate
- `syslog_tls_certificate_authority` - Syslog TLS CA
- `isolated_gpu_pci_ids` - GPUs for VM passthrough
- `kernel_extra_options` - Extra kernel options
- `sed_user` - SED user
- `login_banner` - Login banner text

### system.advanced.update
Update advanced configuration.
```bash
midclt call system.advanced.update '{
  "motd": "Welcome to TrueNAS",
  "serialconsole": true,
  "serialport": "ttyS0",
  "serialspeed": "115200"
}'
```

### system.advanced.login_banner
Get login banner.
```bash
midclt call system.advanced.login_banner
```

### system.advanced.serial_port_choices
Get available serial ports.
```bash
midclt call system.advanced.serial_port_choices
```

### system.advanced.syslog_certificate_choices
Get available syslog certificates.
```bash
midclt call system.advanced.syslog_certificate_choices
```

### system.advanced.syslog_certificate_authority_choices
Get available syslog CAs.
```bash
midclt call system.advanced.syslog_certificate_authority_choices
```

### system.advanced.get_gpu_pci_choices
Get available GPUs for passthrough.
```bash
midclt call system.advanced.get_gpu_pci_choices
```

### system.advanced.update_gpu_pci_ids
Update isolated GPUs for VM passthrough.
```bash
midclt call system.advanced.update_gpu_pci_ids '["0000:01:00.0"]'
```

### system.advanced.sed_global_password
Get SED global password status.
```bash
midclt call system.advanced.sed_global_password
```

### system.advanced.sed_global_password_is_set
Check if SED password is set.
```bash
midclt call system.advanced.sed_global_password_is_set
```

## Security Configuration

### system.security.config
Get security configuration.
```bash
midclt call system.security.config
```

### system.security.update
Update security configuration.
```bash
midclt call system.security.update '{
  "enable_fips": true
}'
```

### system.security.info.fips_available
Check if FIPS mode is available.
```bash
midclt call system.security.info.fips_available
```

## NTP Servers

### system.ntpserver.query
Query NTP servers.
```bash
midclt call system.ntpserver.query
```

### system.ntpserver.create
Add an NTP server.
```bash
midclt call system.ntpserver.create '{
  "address": "pool.ntp.org",
  "burst": true,
  "iburst": true,
  "prefer": true
}'
```

### system.ntpserver.update
Update an NTP server.
```bash
midclt call system.ntpserver.update <server_id> '{"prefer": false}'
```

### system.ntpserver.delete
Delete an NTP server.
```bash
midclt call system.ntpserver.delete <server_id>
```

## System Operations

### system.reboot
Reboot the system.
```bash
midclt call system.reboot
midclt call system.reboot '{"delay": 60}'
```

### system.reboot.info
Get pending reboot information.
```bash
midclt call system.reboot.info
```

### system.shutdown
Shutdown the system.
```bash
midclt call system.shutdown
midclt call system.shutdown '{"delay": 60}'
```

### system.debug
Generate debug file.
```bash
midclt call system.debug
```

### system.license_update
Update system license.
```bash
midclt call system.license_update '"<license_key>"'
```

## System Dataset

### systemdataset.config
Get system dataset configuration.
```bash
midclt call systemdataset.config
```

### systemdataset.pool_choices
Get available pools for system dataset.
```bash
midclt call systemdataset.pool_choices
```

### systemdataset.update
Update system dataset location.
```bash
midclt call systemdataset.update '{"pool": "tank"}'
```

## Configuration Management

### config.reset
Reset configuration to defaults.
```bash
midclt call config.reset '{"reboot": true}'
```
