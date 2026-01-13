# Tunables API

System tunables (sysctl) configuration.

## Tunable Operations

### tunable.query
Query tunables.
```bash
midclt call tunable.query
midclt call tunable.query '[[["type", "=", "SYSCTL"]]]'
```

Returns:
- `id` - Tunable ID
- `type` - SYSCTL, LOADER, RC
- `var` - Variable name
- `value` - Variable value
- `comment` - Description
- `enabled` - Tunable enabled

### tunable.create
Create a tunable.

Sysctl tunable:
```bash
midclt call tunable.create '{
  "type": "SYSCTL",
  "var": "net.ipv4.tcp_tw_reuse",
  "value": "1",
  "comment": "Enable TCP TIME-WAIT reuse",
  "enabled": true
}'
```

ZFS tunable:
```bash
midclt call tunable.create '{
  "type": "SYSCTL",
  "var": "vfs.zfs.arc_max",
  "value": "17179869184",
  "comment": "Limit ARC to 16GB",
  "enabled": true
}'
```

Loader tunable:
```bash
midclt call tunable.create '{
  "type": "LOADER",
  "var": "kern.geom.label.disk_ident.enable",
  "value": "0",
  "comment": "Disable disk ident labels",
  "enabled": true
}'
```

### tunable.update
Update a tunable.
```bash
midclt call tunable.update <tunable_id> '{
  "value": "2",
  "enabled": true
}'
```

### tunable.delete
Delete a tunable.
```bash
midclt call tunable.delete <tunable_id>
```

### tunable.tunable_type_choices
Get tunable type choices.
```bash
midclt call tunable.tunable_type_choices
```

## Tunable Types

| Type | Description | Applies |
|------|-------------|---------|
| `SYSCTL` | Runtime kernel parameter | Immediately |
| `LOADER` | Boot-time loader variable | After reboot |
| `RC` | RC configuration variable | After reboot |

## Common ZFS Tunables

| Variable | Description | Default |
|----------|-------------|---------|
| `vfs.zfs.arc_max` | Maximum ARC size | 50% RAM |
| `vfs.zfs.arc_min` | Minimum ARC size | 16MB |
| `vfs.zfs.arc_meta_limit` | ARC metadata limit | 25% of arc_max |
| `vfs.zfs.prefetch_disable` | Disable prefetch | 0 |
| `vfs.zfs.vdev.scrub_min_active` | Min active scrub I/Os | 1 |
| `vfs.zfs.vdev.scrub_max_active` | Max active scrub I/Os | 2 |
| `vfs.zfs.resilver_min_time_ms` | Min resilver time slice | 3000 |
| `vfs.zfs.txg.timeout` | TXG timeout (seconds) | 5 |

## Common Network Tunables

| Variable | Description | Default |
|----------|-------------|---------|
| `net.core.rmem_max` | Max receive buffer | 212992 |
| `net.core.wmem_max` | Max send buffer | 212992 |
| `net.ipv4.tcp_rmem` | TCP receive buffer | auto |
| `net.ipv4.tcp_wmem` | TCP send buffer | auto |
| `net.ipv4.tcp_timestamps` | TCP timestamps | 1 |
| `net.ipv4.tcp_sack` | TCP SACK | 1 |

## Common VM Tunables

| Variable | Description | Default |
|----------|-------------|---------|
| `vm.swappiness` | Swappiness | 60 |
| `vm.dirty_ratio` | Dirty page ratio | 20 |
| `vm.dirty_background_ratio` | Background dirty ratio | 10 |
| `vm.overcommit_memory` | Overcommit behavior | 0 |

## Example: Optimize for iSCSI

```bash
# Increase dirty page ratio for better write performance
midclt call tunable.create '{"type": "SYSCTL", "var": "vm.dirty_ratio", "value": "40", "enabled": true}'
midclt call tunable.create '{"type": "SYSCTL", "var": "vm.dirty_background_ratio", "value": "10", "enabled": true}'
```

## Example: Limit ARC for VMs

```bash
# Limit ARC to 16GB to leave memory for VMs
midclt call tunable.create '{"type": "SYSCTL", "var": "vfs.zfs.arc_max", "value": "17179869184", "enabled": true}'
```
