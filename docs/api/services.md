# Services API

System service management.

## Service Operations

### service.query
Query all services.
```bash
midclt call service.query
midclt call service.query '[[["service", "=", "ssh"]]]'
midclt call service.query '[[["state", "=", "RUNNING"]]]'
```

Returns:
- `id` - Service ID
- `service` - Service name
- `enable` - Start on boot
- `state` - Current state (RUNNING, STOPPED)
- `pids` - Process IDs

### service.control
Control a service (start, stop, restart, reload).
```bash
midclt call service.control "ssh" "START"
midclt call service.control "ssh" "STOP"
midclt call service.control "ssh" "RESTART"
midclt call service.control "ssh" "RELOAD"
```

With options:
```bash
midclt call service.control "cifs" "START" '{"ha_propagate": true}'
```

### service.update
Update service auto-start setting.
```bash
midclt call service.update "ssh" '{"enable": true}'
midclt call service.update "ssh" '{"enable": false}'
```

## Available Services

| Service Name | Description |
|-------------|-------------|
| `ssh` | SSH server |
| `cifs` | SMB/CIFS server |
| `nfs` | NFS server |
| `ftp` | FTP server |
| `webdav` | WebDAV server |
| `iscsitarget` | iSCSI target |
| `snmp` | SNMP daemon |
| `ups` | UPS monitoring |
| `smartd` | S.M.A.R.T. monitoring |
| `docker` | Docker service |
| `kuberouter` | Kubernetes router |
| `kubernetes` | Kubernetes service |
| `openvpn_client` | OpenVPN client |
| `openvpn_server` | OpenVPN server |
| `s3` | S3 compatible server |
| `netdata` | System monitoring |
| `truecommand` | TrueCommand connection |
| `glusterd` | GlusterFS |
| `lldp` | Link Layer Discovery Protocol |
| `rsync` | Rsync daemon |
| `scst` | SCST iSCSI target |
| `libvirtd` | Libvirt hypervisor |
| `wireguard` | WireGuard VPN |

## Common Patterns

### Start a service
```bash
midclt call service.update "ssh" '{"enable": true}'
midclt call service.control "ssh" "START"
```

### Stop and disable a service
```bash
midclt call service.control "ftp" "STOP"
midclt call service.update "ftp" '{"enable": false}'
```

### Restart a service
```bash
midclt call service.control "cifs" "RESTART"
```

### Check if a service is running
```bash
midclt call service.query '[[["service", "=", "ssh"]]]'
```

### Get all running services
```bash
midclt call service.query '[[["state", "=", "RUNNING"]]]'
```
