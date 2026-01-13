# NVMe-oF API

NVMe over Fabrics target configuration.

## Global Configuration

### nvmet.global.config
Get NVMe-oF global configuration.
```bash
midclt call nvmet.global.config
```

### nvmet.global.update
Update NVMe-oF global configuration.
```bash
midclt call nvmet.global.update '{
  "enabled": true
}'
```

## Subsystems

### nvmet.subsys.query
Query NVMe subsystems.
```bash
midclt call nvmet.subsys.query
```

### nvmet.subsys.create
Create an NVMe subsystem.
```bash
midclt call nvmet.subsys.create '{
  "name": "nqn.2024-01.com.example:nvme:subsys1",
  "serial": "TRUENAS001"
}'
```

### nvmet.subsys.update
Update an NVMe subsystem.
```bash
midclt call nvmet.subsys.update <subsys_id> '{"serial": "TRUENAS002"}'
```

### nvmet.subsys.delete
Delete an NVMe subsystem.
```bash
midclt call nvmet.subsys.delete <subsys_id>
```

## Namespaces (LUNs)

### nvmet.namespace.query
Query NVMe namespaces.
```bash
midclt call nvmet.namespace.query
```

### nvmet.namespace.create
Create an NVMe namespace.
```bash
midclt call nvmet.namespace.create '{
  "subsys": <subsys_id>,
  "device": "zvol/tank/nvme/ns1",
  "device_nguid": null,
  "device_uuid": null,
  "enabled": true
}'
```

### nvmet.namespace.update
Update an NVMe namespace.
```bash
midclt call nvmet.namespace.update <namespace_id> '{"enabled": false}'
```

### nvmet.namespace.delete
Delete an NVMe namespace.
```bash
midclt call nvmet.namespace.delete <namespace_id>
```

## Ports

### nvmet.port.query
Query NVMe ports.
```bash
midclt call nvmet.port.query
```

### nvmet.port.create
Create an NVMe port.

RDMA transport:
```bash
midclt call nvmet.port.create '{
  "addr_trtype": "RDMA",
  "addr_traddr": "192.168.1.10",
  "addr_trsvcid": "4420",
  "addr_adrfam": "IPV4"
}'
```

TCP transport:
```bash
midclt call nvmet.port.create '{
  "addr_trtype": "TCP",
  "addr_traddr": "192.168.1.10",
  "addr_trsvcid": "4420",
  "addr_adrfam": "IPV4"
}'
```

### nvmet.port.update
Update an NVMe port.
```bash
midclt call nvmet.port.update <port_id> '{"addr_trsvcid": "4421"}'
```

### nvmet.port.delete
Delete an NVMe port.
```bash
midclt call nvmet.port.delete <port_id>
```

### nvmet.port.transport_address_choices
Get available transport addresses.
```bash
midclt call nvmet.port.transport_address_choices
```

## Port-Subsystem Associations

### nvmet.port_subsys.query
Query port-subsystem associations.
```bash
midclt call nvmet.port_subsys.query
```

### nvmet.port_subsys.create
Associate a subsystem with a port.
```bash
midclt call nvmet.port_subsys.create '{
  "port": <port_id>,
  "subsys": <subsys_id>
}'
```

### nvmet.port_subsys.delete
Remove port-subsystem association.
```bash
midclt call nvmet.port_subsys.delete <assoc_id>
```

## Hosts

### nvmet.host.query
Query allowed hosts.
```bash
midclt call nvmet.host.query
```

### nvmet.host.create
Create an allowed host.
```bash
midclt call nvmet.host.create '{
  "hostnqn": "nqn.2024-01.com.example:host:server1"
}'
```

With DH-CHAP authentication:
```bash
midclt call nvmet.host.create '{
  "hostnqn": "nqn.2024-01.com.example:host:server1",
  "dhchap_key": "DHHC-1:00:...",
  "dhchap_dhgroup": "NULL",
  "dhchap_hash": "SHA256"
}'
```

### nvmet.host.update
Update a host.
```bash
midclt call nvmet.host.update <host_id> '{"dhchap_key": "DHHC-1:01:..."}'
```

### nvmet.host.delete
Delete a host.
```bash
midclt call nvmet.host.delete <host_id>
```

### nvmet.host.generate_key
Generate a DH-CHAP key.
```bash
midclt call nvmet.host.generate_key
```

### nvmet.host.dhchap_dhgroup_choices
Get DH-CHAP DH group choices.
```bash
midclt call nvmet.host.dhchap_dhgroup_choices
```

### nvmet.host.dhchap_hash_choices
Get DH-CHAP hash choices.
```bash
midclt call nvmet.host.dhchap_hash_choices
```

## Host-Subsystem Associations

### nvmet.host_subsys.query
Query host-subsystem associations.
```bash
midclt call nvmet.host_subsys.query
```

### nvmet.host_subsys.create
Allow a host to access a subsystem.
```bash
midclt call nvmet.host_subsys.create '{
  "host": <host_id>,
  "subsys": <subsys_id>
}'
```

### nvmet.host_subsys.delete
Remove host-subsystem access.
```bash
midclt call nvmet.host_subsys.delete <assoc_id>
```

## Transport Types

| Type | Description |
|------|-------------|
| `TCP` | NVMe over TCP |
| `RDMA` | NVMe over RDMA |
| `FC` | NVMe over Fibre Channel |

## Complete NVMe-oF Setup

```bash
# 1. Create zvol for namespace
midclt call pool.dataset.create '{"name": "tank/nvme/ns1", "type": "VOLUME", "volsize": 107374182400}'

# 2. Create subsystem
midclt call nvmet.subsys.create '{"name": "nqn.2024-01.com.example:nvme:subsys1"}'

# 3. Create namespace
midclt call nvmet.namespace.create '{"subsys": 1, "device": "zvol/tank/nvme/ns1", "enabled": true}'

# 4. Create port
midclt call nvmet.port.create '{"addr_trtype": "TCP", "addr_traddr": "0.0.0.0", "addr_trsvcid": "4420", "addr_adrfam": "IPV4"}'

# 5. Associate subsystem with port
midclt call nvmet.port_subsys.create '{"port": 1, "subsys": 1}'

# 6. Optionally create host for access control
midclt call nvmet.host.create '{"hostnqn": "nqn.2024-01.com.example:host:client1"}'
midclt call nvmet.host_subsys.create '{"host": 1, "subsys": 1}'

# 7. Enable NVMe-oF
midclt call nvmet.global.update '{"enabled": true}'
```
