# Static Routes API

Static network route configuration.

## Route Operations

### staticroute.query
Query static routes.
```bash
midclt call staticroute.query
```

Returns:
- `id` - Route ID
- `destination` - Destination network (CIDR)
- `gateway` - Gateway IP address
- `description` - Route description

### staticroute.create
Create a static route.
```bash
midclt call staticroute.create '{
  "destination": "10.0.0.0/8",
  "gateway": "192.168.1.1",
  "description": "Route to internal network"
}'
```

Multiple routes:
```bash
# Route to datacenter
midclt call staticroute.create '{
  "destination": "10.10.0.0/16",
  "gateway": "192.168.1.254",
  "description": "Datacenter network"
}'

# Route to VPN network
midclt call staticroute.create '{
  "destination": "172.16.0.0/12",
  "gateway": "192.168.1.253",
  "description": "VPN network"
}'
```

### staticroute.update
Update a static route.
```bash
midclt call staticroute.update <route_id> '{
  "gateway": "192.168.1.254",
  "description": "Updated route"
}'
```

### staticroute.delete
Delete a static route.
```bash
midclt call staticroute.delete <route_id>
```

## Common Use Cases

### Route to secondary subnet
```bash
midclt call staticroute.create '{
  "destination": "192.168.2.0/24",
  "gateway": "192.168.1.1",
  "description": "Secondary LAN"
}'
```

### Route to VPN
```bash
midclt call staticroute.create '{
  "destination": "10.8.0.0/24",
  "gateway": "192.168.1.10",
  "description": "OpenVPN network"
}'
```

### Default route (use network config instead)
For default gateway, use `network.configuration.update` instead:
```bash
midclt call network.configuration.update '{
  "ipv4gateway": "192.168.1.1"
}'
```

## Route Format

| Field | Format | Example |
|-------|--------|---------|
| destination | CIDR notation | `10.0.0.0/8` |
| gateway | IPv4 address | `192.168.1.1` |
| description | Text | `Route to datacenter` |
