# Network API

Network configuration including interfaces, routes, and general network settings.

## Network Interfaces

### interface.query
Query network interfaces.
```bash
midclt call interface.query
midclt call interface.query '[[["name", "=", "eno1"]]]'
midclt call interface.query '[[["type", "=", "BRIDGE"]]]'
```

Returns:
- `id` - Interface name
- `name` - Interface name
- `fake` - Virtual interface flag
- `type` - PHYSICAL, BRIDGE, LINK_AGGREGATION, VLAN
- `state` - Interface state object
- `aliases` - IP address aliases
- `ipv4_dhcp` - DHCPv4 enabled
- `ipv6_auto` - IPv6 auto-config enabled
- `description` - Interface description
- `mtu` - MTU size
- `options` - Interface options
- `disable_offload_capabilities` - Disable NIC offloading

State object includes:
- `name` - Interface name
- `orig_name` - Original interface name
- `description` - Description
- `mtu` - MTU
- `cloned` - Cloned interface
- `flags` - Interface flags
- `nd6_flags` - IPv6 ND flags
- `capabilities` - Interface capabilities
- `link_state` - Link state (LINK_STATE_UP, etc.)
- `media_type` - Media type
- `media_subtype` - Media subtype
- `active_media_type` - Active media type
- `active_media_subtype` - Active media subtype
- `supported_media` - Supported media types
- `media_options` - Media options
- `link_address` - MAC address
- `rx_queues` - RX queues
- `tx_queues` - TX queues
- `aliases` - Configured aliases

### interface.create
Create a network interface.

Bridge interface:
```bash
midclt call interface.create '{
  "type": "BRIDGE",
  "name": "br0",
  "bridge_members": ["eno1", "eno2"],
  "aliases": [{"address": "192.168.1.10", "netmask": 24}]
}'
```

VLAN interface:
```bash
midclt call interface.create '{
  "type": "VLAN",
  "name": "vlan100",
  "vlan_parent_interface": "eno1",
  "vlan_tag": 100,
  "aliases": [{"address": "10.0.100.10", "netmask": 24}]
}'
```

Link aggregation (LAGG):
```bash
midclt call interface.create '{
  "type": "LINK_AGGREGATION",
  "name": "bond0",
  "lag_protocol": "LACP",
  "lag_ports": ["eno1", "eno2"],
  "aliases": [{"address": "192.168.1.10", "netmask": 24}]
}'
```

### interface.update
Update a network interface.
```bash
midclt call interface.update "eno1" '{
  "aliases": [{"address": "192.168.1.20", "netmask": 24}],
  "mtu": 9000
}'
```

Enable DHCP:
```bash
midclt call interface.update "eno1" '{
  "ipv4_dhcp": true,
  "aliases": []
}'
```

### interface.delete
Delete a network interface.
```bash
midclt call interface.delete "br0"
```

### interface.commit
Commit pending network changes.
```bash
midclt call interface.commit '{"rollback": true, "checkin_timeout": 60}'
```

### interface.rollback
Rollback uncommitted network changes.
```bash
midclt call interface.rollback
```

### interface.checkin
Check in after network changes (prevents automatic rollback).
```bash
midclt call interface.checkin
```

### interface.checkin_waiting
Check if waiting for checkin.
```bash
midclt call interface.checkin_waiting
```

### interface.cancel_rollback
Cancel pending rollback.
```bash
midclt call interface.cancel_rollback
```

### interface.has_pending_changes
Check for pending network changes.
```bash
midclt call interface.has_pending_changes
```

### interface.save_network_config
Save network configuration to file.
```bash
midclt call interface.save_network_config
```

### interface.services_restarted_on_sync
Get services that will restart on network sync.
```bash
midclt call interface.services_restarted_on_sync
```

### interface.network_config_to_be_removed
Get network config to be removed.
```bash
midclt call interface.network_config_to_be_removed
```

### interface.websocket_local_ip
Get local IP for websocket connection.
```bash
midclt call interface.websocket_local_ip
```

### Choice Methods

```bash
midclt call interface.bridge_members_choices
midclt call interface.lag_ports_choices
midclt call interface.lag_supported_protocols
midclt call interface.lacpdu_rate_choices
midclt call interface.xmit_hash_policy_choices
midclt call interface.vlan_parent_interface_choices
```

## Static Routes

### staticroute.query
Query static routes.
```bash
midclt call staticroute.query
```

### staticroute.create
Create a static route.
```bash
midclt call staticroute.create '{
  "destination": "10.0.0.0/8",
  "gateway": "192.168.1.1",
  "description": "Route to internal network"
}'
```

### staticroute.update
Update a static route.
```bash
midclt call staticroute.update <route_id> '{"gateway": "192.168.1.254"}'
```

### staticroute.delete
Delete a static route.
```bash
midclt call staticroute.delete <route_id>
```

## Network Configuration

### network.configuration.config
Get network configuration.
```bash
midclt call network.configuration.config
```

Returns:
- `id` - Config ID
- `hostname` - System hostname
- `domain` - Domain name
- `ipv4gateway` - Default IPv4 gateway
- `ipv6gateway` - Default IPv6 gateway
- `nameserver1` - Primary DNS
- `nameserver2` - Secondary DNS
- `nameserver3` - Tertiary DNS
- `httpproxy` - HTTP proxy
- `netwait_enabled` - Wait for network on boot
- `netwait_ip` - IPs to wait for
- `hosts` - Custom hosts entries
- `domains` - Search domains
- `service_announcement` - Service announcement settings
- `activity` - Activity choices

### network.configuration.update
Update network configuration.
```bash
midclt call network.configuration.update '{
  "hostname": "truenas",
  "domain": "local",
  "ipv4gateway": "192.168.1.1",
  "nameserver1": "8.8.8.8",
  "nameserver2": "8.8.4.4"
}'
```

### network.configuration.activity_choices
Get available activity choices.
```bash
midclt call network.configuration.activity_choices
```

### network.general.summary
Get network summary.
```bash
midclt call network.general.summary
```

## IPMI

### ipmi.is_loaded
Check if IPMI is available.
```bash
midclt call ipmi.is_loaded
```

### ipmi.lan.query
Query IPMI LAN configuration.
```bash
midclt call ipmi.lan.query
```

### ipmi.lan.update
Update IPMI LAN configuration.
```bash
midclt call ipmi.lan.update <channel_id> '{
  "ipaddress": "192.168.1.100",
  "netmask": "255.255.255.0",
  "gateway": "192.168.1.1",
  "dhcp": false,
  "vlan_id": null,
  "password": "newpassword"
}'
```

### ipmi.chassis.identify
Identify chassis (blink light).
```bash
midclt call ipmi.chassis.identify "ON"
midclt call ipmi.chassis.identify "OFF"
```

### ipmi.chassis.info
Get chassis information.
```bash
midclt call ipmi.chassis.info
```

### ipmi.sel.elist
Get IPMI System Event Log.
```bash
midclt call ipmi.sel.elist
```

### ipmi.sel.clear
Clear IPMI System Event Log.
```bash
midclt call ipmi.sel.clear
```

## Interface Types

| Type | Description |
|------|-------------|
| `PHYSICAL` | Physical network interface |
| `BRIDGE` | Network bridge |
| `LINK_AGGREGATION` | Bonded interface (LAGG) |
| `VLAN` | VLAN interface |

## LAGG Protocols

| Protocol | Description |
|----------|-------------|
| `LACP` | Link Aggregation Control Protocol |
| `FAILOVER` | Failover mode |
| `LOADBALANCE` | Load balancing |
| `ROUNDROBIN` | Round-robin |
| `NONE` | No protocol |
