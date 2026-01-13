# Failover/HA API

High Availability configuration for TrueNAS Enterprise.

## Failover Status

### failover.status
Get failover status.
```bash
midclt call failover.status
```

Returns:
- `MASTER` - Active controller
- `BACKUP` - Standby controller
- `SINGLE` - Non-HA system
- `ERROR` - Error state

### failover.node
Get current node identifier.
```bash
midclt call failover.node
```

Returns `A` or `B`.

### failover.licensed
Check if HA is licensed.
```bash
midclt call failover.licensed
```

### failover.config
Get failover configuration.
```bash
midclt call failover.config
```

Returns:
- `id` - Config ID
- `disabled` - Failover disabled
- `timeout` - Failover timeout
- `master` - Preferred master

### failover.disabled.reasons
Get reasons failover is disabled.
```bash
midclt call failover.disabled.reasons
```

## Failover Configuration

### failover.update
Update failover configuration.
```bash
midclt call failover.update '{
  "disabled": false,
  "timeout": 0,
  "master": true
}'
```

Disable failover:
```bash
midclt call failover.update '{"disabled": true}'
```

## Failover Operations

### failover.become_passive
Make current controller passive (trigger failover).
```bash
midclt call failover.become_passive
```

### failover.sync_to_peer
Sync configuration to peer controller.
```bash
midclt call failover.sync_to_peer
```

### failover.sync_from_peer
Sync configuration from peer controller.
```bash
midclt call failover.sync_from_peer
```

## Reboot Operations

### failover.reboot.info
Get reboot information.
```bash
midclt call failover.reboot.info
```

### failover.reboot.other_node
Reboot the other node.
```bash
midclt call failover.reboot.other_node
```

## Upgrade Operations

### failover.upgrade
Upgrade HA cluster.
```bash
midclt call failover.upgrade
```

This performs a rolling upgrade:
1. Upgrades standby controller
2. Fails over to upgraded controller
3. Upgrades original master

## HA Status Indicators

| Status | Description |
|--------|-------------|
| `MASTER` | Active, serving data |
| `BACKUP` | Standby, ready to take over |
| `SINGLE` | Non-HA or HA not configured |
| `ELECTING` | Election in progress |
| `IMPORTING` | Importing pools |
| `ERROR` | Error state |

## HA Architecture

TrueNAS Enterprise HA consists of:
- Two controllers (A and B)
- Shared storage enclosure
- Heartbeat network
- Virtual IP for clients

Only one controller is active at a time. The standby controller monitors the active controller and takes over if it fails.

## Common HA Operations

### Check HA status
```bash
midclt call failover.licensed
midclt call failover.status
midclt call failover.node
```

### Planned failover
```bash
# On the master controller
midclt call failover.become_passive
```

### Sync configuration
```bash
# After making changes on master
midclt call failover.sync_to_peer
```

### Emergency disable
```bash
# Disable HA temporarily for maintenance
midclt call failover.update '{"disabled": true}'

# Re-enable after maintenance
midclt call failover.update '{"disabled": false}'
```
