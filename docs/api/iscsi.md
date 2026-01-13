# iSCSI API

iSCSI target configuration for block-level storage sharing.

## Global Configuration

### iscsi.global.config
Get iSCSI global configuration.
```bash
midclt call iscsi.global.config
```

Returns:
- `basename` - Base name for targets
- `isns_servers` - iSNS servers
- `pool_avail_threshold` - Pool availability threshold
- `alua` - ALUA enabled

### iscsi.global.update
Update iSCSI global configuration.
```bash
midclt call iscsi.global.update '{
  "basename": "iqn.2005-10.org.freenas.ctl",
  "isns_servers": [],
  "pool_avail_threshold": 80,
  "alua": false
}'
```

### iscsi.global.sessions
List active iSCSI sessions.
```bash
midclt call iscsi.global.sessions
```

## Portals

### iscsi.portal.query
Query iSCSI portals.
```bash
midclt call iscsi.portal.query
```

Returns:
- `id` - Portal ID
- `tag` - Portal group tag
- `comment` - Description
- `discovery_authmethod` - Auth method (NONE, CHAP, MUTUAL_CHAP)
- `discovery_authgroup` - Auth group ID
- `listen` - Listen addresses

### iscsi.portal.create
Create an iSCSI portal.
```bash
midclt call iscsi.portal.create '{
  "comment": "Main portal",
  "discovery_authmethod": "NONE",
  "listen": [
    {"ip": "0.0.0.0", "port": 3260}
  ]
}'
```

With CHAP authentication:
```bash
midclt call iscsi.portal.create '{
  "comment": "Secured portal",
  "discovery_authmethod": "CHAP",
  "discovery_authgroup": <auth_id>,
  "listen": [
    {"ip": "192.168.1.10", "port": 3260}
  ]
}'
```

### iscsi.portal.update
Update an iSCSI portal.
```bash
midclt call iscsi.portal.update <portal_id> '{
  "listen": [
    {"ip": "192.168.1.10", "port": 3260},
    {"ip": "192.168.1.11", "port": 3260}
  ]
}'
```

### iscsi.portal.delete
Delete an iSCSI portal.
```bash
midclt call iscsi.portal.delete <portal_id>
```

### iscsi.portal.listen_ip_choices
Get available IP addresses for portals.
```bash
midclt call iscsi.portal.listen_ip_choices
```

## Initiators

### iscsi.initiator.query
Query authorized initiators.
```bash
midclt call iscsi.initiator.query
```

Returns:
- `id` - Initiator ID
- `comment` - Description
- `initiators` - Allowed initiator IQNs
- `auth_network` - Allowed networks

### iscsi.initiator.create
Create an initiator group.
```bash
midclt call iscsi.initiator.create '{
  "comment": "VMware hosts",
  "initiators": [
    "iqn.1998-01.com.vmware:esxi-host-1",
    "iqn.1998-01.com.vmware:esxi-host-2"
  ],
  "auth_network": ["192.168.1.0/24"]
}'
```

Allow all initiators:
```bash
midclt call iscsi.initiator.create '{
  "comment": "Allow all",
  "initiators": [],
  "auth_network": []
}'
```

### iscsi.initiator.update
Update an initiator group.
```bash
midclt call iscsi.initiator.update <initiator_id> '{
  "initiators": ["iqn.1998-01.com.vmware:esxi-host-3"]
}'
```

### iscsi.initiator.delete
Delete an initiator group.
```bash
midclt call iscsi.initiator.delete <initiator_id>
```

## Authentication

### iscsi.auth.query
Query CHAP authentication credentials.
```bash
midclt call iscsi.auth.query
```

Returns:
- `id` - Auth ID
- `tag` - Auth group tag
- `user` - CHAP username
- `peeruser` - Mutual CHAP username

### iscsi.auth.create
Create CHAP credentials.
```bash
midclt call iscsi.auth.create '{
  "tag": 1,
  "user": "chapuser",
  "secret": "secretpassword123"
}'
```

With mutual CHAP:
```bash
midclt call iscsi.auth.create '{
  "tag": 1,
  "user": "chapuser",
  "secret": "secretpassword123",
  "peeruser": "peerchapuser",
  "peersecret": "peersecretpassword123"
}'
```

### iscsi.auth.update
Update CHAP credentials.
```bash
midclt call iscsi.auth.update <auth_id> '{"secret": "newsecret123"}'
```

### iscsi.auth.delete
Delete CHAP credentials.
```bash
midclt call iscsi.auth.delete <auth_id>
```

## Targets

### iscsi.target.query
Query iSCSI targets.
```bash
midclt call iscsi.target.query
```

Returns:
- `id` - Target ID
- `name` - Target name (appended to basename)
- `alias` - Target alias
- `mode` - ISCSI, FC, BOTH
- `groups` - Portal/initiator group associations

### iscsi.target.create
Create an iSCSI target.
```bash
midclt call iscsi.target.create '{
  "name": "target0",
  "alias": "Data target",
  "mode": "ISCSI",
  "groups": [
    {
      "portal": <portal_id>,
      "initiator": <initiator_id>,
      "auth": null,
      "authmethod": "NONE"
    }
  ]
}'
```

With CHAP authentication:
```bash
midclt call iscsi.target.create '{
  "name": "secure-target",
  "mode": "ISCSI",
  "groups": [
    {
      "portal": <portal_id>,
      "initiator": <initiator_id>,
      "auth": <auth_id>,
      "authmethod": "CHAP"
    }
  ]
}'
```

### iscsi.target.update
Update an iSCSI target.
```bash
midclt call iscsi.target.update <target_id> '{
  "alias": "Updated alias"
}'
```

### iscsi.target.delete
Delete an iSCSI target.
```bash
midclt call iscsi.target.delete <target_id>
```

### iscsi.target.validate_name
Validate a target name.
```bash
midclt call iscsi.target.validate_name "target0"
```

## Extents

### iscsi.extent.query
Query iSCSI extents (LUNs).
```bash
midclt call iscsi.extent.query
```

Returns:
- `id` - Extent ID
- `name` - Extent name
- `type` - DISK or FILE
- `disk` - Disk/zvol path
- `path` - File path
- `filesize` - File size
- `blocksize` - Block size
- `pblocksize` - Physical block size
- `avail_threshold` - Availability threshold
- `comment` - Description
- `serial` - Serial number
- `naa` - NAA identifier
- `insecure_tpc` - Allow insecure TPC
- `xen` - Xen compatibility
- `rpm` - Reported RPM
- `ro` - Read-only
- `enabled` - Extent enabled

### iscsi.extent.create
Create an extent from zvol.
```bash
midclt call iscsi.extent.create '{
  "name": "extent0",
  "type": "DISK",
  "disk": "zvol/tank/iscsi/lun0",
  "blocksize": 512,
  "comment": "iSCSI LUN"
}'
```

Create a file-based extent:
```bash
midclt call iscsi.extent.create '{
  "name": "file-extent",
  "type": "FILE",
  "path": "/mnt/tank/iscsi/extent0.img",
  "filesize": 10737418240,
  "blocksize": 512
}'
```

### iscsi.extent.update
Update an extent.
```bash
midclt call iscsi.extent.update <extent_id> '{
  "comment": "Updated description",
  "ro": true
}'
```

### iscsi.extent.delete
Delete an extent.
```bash
midclt call iscsi.extent.delete <extent_id>
midclt call iscsi.extent.delete <extent_id> '{"remove": true, "force": false}'
```

### iscsi.extent.disk_choices
Get available disks/zvols for extents.
```bash
midclt call iscsi.extent.disk_choices
```

## Target-Extent Associations

### iscsi.targetextent.query
Query target-extent mappings.
```bash
midclt call iscsi.targetextent.query
```

Returns:
- `id` - Association ID
- `target` - Target ID
- `extent` - Extent ID
- `lunid` - LUN ID

### iscsi.targetextent.create
Associate an extent with a target.
```bash
midclt call iscsi.targetextent.create '{
  "target": <target_id>,
  "extent": <extent_id>,
  "lunid": 0
}'
```

### iscsi.targetextent.delete
Remove a target-extent association.
```bash
midclt call iscsi.targetextent.delete <association_id>
```

## Complete iSCSI Setup Example

```bash
# 1. Create a zvol for the LUN
midclt call pool.dataset.create '{"name": "tank/iscsi/lun0", "type": "VOLUME", "volsize": 107374182400}'

# 2. Create a portal
midclt call iscsi.portal.create '{"comment": "Default", "listen": [{"ip": "0.0.0.0", "port": 3260}]}'

# 3. Create an initiator group (allow all)
midclt call iscsi.initiator.create '{"comment": "Allow all", "initiators": [], "auth_network": []}'

# 4. Create the target
midclt call iscsi.target.create '{"name": "target0", "groups": [{"portal": 1, "initiator": 1, "authmethod": "NONE"}]}'

# 5. Create the extent
midclt call iscsi.extent.create '{"name": "extent0", "type": "DISK", "disk": "zvol/tank/iscsi/lun0"}'

# 6. Associate extent with target
midclt call iscsi.targetextent.create '{"target": 1, "extent": 1, "lunid": 0}'

# 7. Start iSCSI service
midclt call service.update "iscsitarget" '{"enable": true}'
midclt call service.control "iscsitarget" "START"
```
