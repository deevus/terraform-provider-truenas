# Fibre Channel API

Fibre Channel target and port management.

## Fibre Channel Capabilities

### fc.capable
Check if system supports Fibre Channel.
```bash
midclt call fc.capable
```

## FC Host Ports

### fc.fc_host.query
Query Fibre Channel host ports.
```bash
midclt call fc.fc_host.query
```

Returns:
- `id` - Port ID
- `name` - Port name
- `port_type` - Port type
- `port_state` - Port state
- `physical_port_state` - Physical state
- `speed` - Current speed
- `supported_speeds` - Supported speeds
- `max_npiv_vports` - Max NPIV virtual ports
- `npiv_vports_inuse` - NPIV ports in use
- `node_name` - WWN node name
- `port_name` - WWN port name

### fc.fc_host.update
Update FC host port settings.
```bash
midclt call fc.fc_host.update <port_id> '{"npiv_enabled": true}'
```

## FC Ports (iSCSI/FC Targets)

### fcport.query
Query FC target ports.
```bash
midclt call fcport.query
```

### fcport.create
Create an FC target port.
```bash
midclt call fcport.create '{
  "port": "<wwpn>",
  "target": <target_id>
}'
```

### fcport.delete
Delete an FC target port.
```bash
midclt call fcport.delete <fcport_id>
```

### fcport.port_choices
Get available FC ports.
```bash
midclt call fcport.port_choices
```

### fcport.status
Get FC port status.
```bash
midclt call fcport.status
```

## FC Target Configuration

Fibre Channel targets use the same target/extent model as iSCSI:

1. **Create extent** (LUN):
```bash
midclt call iscsi.extent.create '{
  "name": "fc-extent",
  "type": "DISK",
  "disk": "zvol/tank/fc/lun0"
}'
```

2. **Create target** with FC mode:
```bash
midclt call iscsi.target.create '{
  "name": "fc-target",
  "mode": "FC"
}'
```

3. **Associate extent with target**:
```bash
midclt call iscsi.targetextent.create '{
  "target": <target_id>,
  "extent": <extent_id>,
  "lunid": 0
}'
```

4. **Create FC port mapping**:
```bash
midclt call fcport.create '{
  "port": "<wwpn>",
  "target": <target_id>
}'
```

## Target Modes

| Mode | Description |
|------|-------------|
| `ISCSI` | iSCSI only |
| `FC` | Fibre Channel only |
| `BOTH` | iSCSI and Fibre Channel |

## FC Speed Options

| Speed | Description |
|-------|-------------|
| 1 Gbps | 1 Gigabit |
| 2 Gbps | 2 Gigabit |
| 4 Gbps | 4 Gigabit |
| 8 Gbps | 8 Gigabit |
| 16 Gbps | 16 Gigabit |
| 32 Gbps | 32 Gigabit |

## NPIV (N_Port ID Virtualization)

NPIV allows multiple virtual ports per physical port:
```bash
# Enable NPIV on a port
midclt call fc.fc_host.update <port_id> '{"npiv_enabled": true}'
```

This is useful for:
- Multi-tenancy
- Virtual machine FC access
- Testing and development
