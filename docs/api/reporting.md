# Reporting API

System metrics and reporting.

## Real-time Data

### reporting.realtime
Get real-time system statistics.
```bash
midclt call reporting.realtime
```

Returns:
- `cpu` - CPU usage
- `disks` - Disk I/O stats
- `interfaces` - Network stats
- `memory` - Memory usage
- `virtual_memory` - Virtual memory
- `zfs` - ZFS ARC stats

## Netdata Graphs

### reporting.netdata_graphs
Get available Netdata graphs.
```bash
midclt call reporting.netdata_graphs
```

### reporting.netdata_get_data
Get data for specific graphs.
```bash
midclt call reporting.netdata_get_data '[
  {"name": "cpu", "identifier": null}
]' '{"start": "now-1h", "end": "now"}'
```

Common graph identifiers:
- `cpu` - CPU usage
- `memory` - Memory usage
- `disk` - Disk I/O
- `network` - Network traffic
- `zfs` - ZFS statistics

With time range:
```bash
midclt call reporting.netdata_get_data '[
  {"name": "cpu", "identifier": null},
  {"name": "memory", "identifier": null}
]' '{
  "start": "2024-01-01T00:00:00",
  "end": "2024-01-01T23:59:59",
  "aggregate": true
}'
```

## Reporting Exporters

### reporting.exporters.query
Query reporting exporters.
```bash
midclt call reporting.exporters.query
```

### reporting.exporters.exporter_schemas
Get available exporter types.
```bash
midclt call reporting.exporters.exporter_schemas
```

### reporting.exporters.create
Create a reporting exporter.

Graphite:
```bash
midclt call reporting.exporters.create '{
  "name": "Graphite Exporter",
  "type": "GRAPHITE",
  "enabled": true,
  "attributes": {
    "destination_ip": "graphite.example.com",
    "destination_port": 2003,
    "prefix": "truenas"
  }
}'
```

InfluxDB:
```bash
midclt call reporting.exporters.create '{
  "name": "InfluxDB Exporter",
  "type": "INFLUXDB",
  "enabled": true,
  "attributes": {
    "host": "influxdb.example.com",
    "port": 8086,
    "database": "truenas",
    "username": "admin",
    "password": "password"
  }
}'
```

### reporting.exporters.update
Update a reporting exporter.
```bash
midclt call reporting.exporters.update <exporter_id> '{"enabled": false}'
```

### reporting.exporters.delete
Delete a reporting exporter.
```bash
midclt call reporting.exporters.delete <exporter_id>
```

## Exporter Types

| Type | Description |
|------|-------------|
| `GRAPHITE` | Graphite carbon |
| `INFLUXDB` | InfluxDB |

## Graph Categories

| Category | Description |
|----------|-------------|
| cpu | CPU utilization |
| disk | Disk I/O |
| memory | Memory usage |
| network | Network traffic |
| nfs | NFS statistics |
| partition | Partition usage |
| system | System load |
| target | iSCSI target stats |
| ups | UPS statistics |
| zfs | ZFS ARC/L2ARC stats |

## WebUI Dashboard

### webui.main.dashboard.sys_info
Get system info for dashboard.
```bash
midclt call webui.main.dashboard.sys_info
```
