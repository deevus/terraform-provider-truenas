# Sharing API

File sharing configuration for SMB, NFS, and WebDAV shares.

## SMB Shares

### sharing.smb.query
Query SMB shares.
```bash
midclt call sharing.smb.query
midclt call sharing.smb.query '[[["name", "=", "data"]]]'
midclt call sharing.smb.query '[[["enabled", "=", true]]]'
```

Returns:
- `id` - Share ID
- `name` - Share name
- `path` - Filesystem path
- `path_suffix` - Path suffix for home shares
- `home` - Home share flag
- `comment` - Share description
- `enabled` - Whether share is enabled
- `ro` - Read-only flag
- `browsable` - Visible in network browse
- `guestok` - Allow guest access
- `hostsallow` - Allowed hosts
- `hostsdeny` - Denied hosts
- `aapl_name_mangling` - Apple name mangling
- `abe` - Access-based enumeration
- `acl` - ACL enabled
- `durablehandle` - Durable handles enabled
- `streams` - NTFS streams support
- `timemachine` - Time Machine backup target
- `timemachine_quota` - Time Machine quota (bytes)
- `vuid` - Shadow copy ID
- `shadowcopy` - Shadow copy enabled
- `fsrvp` - File Server Remote VSS Protocol
- `recyclebin` - Recycle bin enabled
- `audit` - Audit logging
- `purpose` - Share purpose preset
- `locked` - Configuration locked

### sharing.smb.create
Create an SMB share.
```bash
midclt call sharing.smb.create '{
  "name": "data",
  "path": "/mnt/tank/data",
  "comment": "Data share",
  "enabled": true,
  "browsable": true,
  "guestok": false,
  "ro": false
}'
```

Time Machine share:
```bash
midclt call sharing.smb.create '{
  "name": "timemachine",
  "path": "/mnt/tank/timemachine",
  "purpose": "TIMEMACHINE",
  "timemachine": true,
  "timemachine_quota": 1099511627776
}'
```

Home share:
```bash
midclt call sharing.smb.create '{
  "name": "homes",
  "path": "/mnt/tank/homes",
  "home": true,
  "enabled": true
}'
```

### sharing.smb.update
Update an SMB share.
```bash
midclt call sharing.smb.update <share_id> '{
  "comment": "Updated description",
  "browsable": false
}'
```

### sharing.smb.delete
Delete an SMB share.
```bash
midclt call sharing.smb.delete <share_id>
```

### sharing.smb.getacl
Get share-level ACL.
```bash
midclt call sharing.smb.getacl <share_id>
```

### sharing.smb.setacl
Set share-level ACL.
```bash
midclt call sharing.smb.setacl '{
  "share_name": "data",
  "share_acl": [
    {"ae_who_sid": "S-1-1-0", "ae_perm": "FULL", "ae_type": "ALLOWED"}
  ]
}'
```

### sharing.smb.share_precheck
Validate share configuration before creation.
```bash
midclt call sharing.smb.share_precheck '{
  "name": "newshare",
  "path": "/mnt/tank/newshare"
}'
```

## NFS Shares

### sharing.nfs.query
Query NFS exports.
```bash
midclt call sharing.nfs.query
midclt call sharing.nfs.query '[[["enabled", "=", true]]]'
```

Returns:
- `id` - Export ID
- `path` - Exported path
- `aliases` - Path aliases
- `comment` - Export description
- `enabled` - Whether export is enabled
- `networks` - Allowed networks (CIDR)
- `hosts` - Allowed hosts
- `ro` - Read-only flag
- `maproot_user` - Map root to user
- `maproot_group` - Map root to group
- `mapall_user` - Map all to user
- `mapall_group` - Map all to group
- `security` - Security flavors (SYS, KRB5, etc.)

### sharing.nfs.create
Create an NFS export.
```bash
midclt call sharing.nfs.create '{
  "path": "/mnt/tank/nfsdata",
  "comment": "NFS export",
  "enabled": true,
  "networks": ["192.168.1.0/24"],
  "hosts": [],
  "ro": false,
  "maproot_user": "root",
  "maproot_group": "wheel"
}'
```

Secure export with Kerberos:
```bash
midclt call sharing.nfs.create '{
  "path": "/mnt/tank/secure",
  "networks": ["192.168.1.0/24"],
  "security": ["KRB5", "KRB5I", "KRB5P"]
}'
```

### sharing.nfs.update
Update an NFS export.
```bash
midclt call sharing.nfs.update <export_id> '{
  "networks": ["192.168.1.0/24", "10.0.0.0/8"],
  "ro": true
}'
```

### sharing.nfs.delete
Delete an NFS export.
```bash
midclt call sharing.nfs.delete <export_id>
```

## WebDAV Shares

### sharing.webshare.query
Query WebDAV shares.
```bash
midclt call sharing.webshare.query
```

### sharing.webshare.create
Create a WebDAV share.
```bash
midclt call sharing.webshare.create '{
  "name": "webdav",
  "path": "/mnt/tank/webdav",
  "enabled": true,
  "ro": false
}'
```

### sharing.webshare.update
Update a WebDAV share.
```bash
midclt call sharing.webshare.update <share_id> '{"ro": true}'
```

### sharing.webshare.delete
Delete a WebDAV share.
```bash
midclt call sharing.webshare.delete <share_id>
```

## SMB Global Configuration

### smb.config
Get SMB service configuration.
```bash
midclt call smb.config
```

### smb.update
Update SMB service configuration.
```bash
midclt call smb.update '{
  "netbiosname": "TRUENAS",
  "workgroup": "WORKGROUP",
  "description": "TrueNAS Server",
  "enable_smb1": false,
  "guest": "nobody"
}'
```

### smb.status
Get SMB service status.
```bash
midclt call smb.status
```

### smb.bindip_choices
Get available bind IP addresses.
```bash
midclt call smb.bindip_choices
```

### smb.unixcharset_choices
Get available Unix character sets.
```bash
midclt call smb.unixcharset_choices
```

## NFS Global Configuration

### nfs.config
Get NFS service configuration.
```bash
midclt call nfs.config
```

### nfs.update
Update NFS service configuration.
```bash
midclt call nfs.update '{
  "servers": 4,
  "udp": false,
  "v4": true,
  "v4_v3owner": false,
  "v4_krb": false
}'
```

### nfs.bindip_choices
Get available bind IP addresses.
```bash
midclt call nfs.bindip_choices
```

### nfs.add_principal
Add Kerberos principal for NFSv4.
```bash
midclt call nfs.add_principal '{"principal": "nfs/truenas.example.com@EXAMPLE.COM"}'
```

## WebDAV Global Configuration

### webshare.config
Get WebDAV service configuration.
```bash
midclt call webshare.config
```

### webshare.update
Update WebDAV service configuration.
```bash
midclt call webshare.update '{
  "protocol": "HTTPS",
  "tcpport": 8080,
  "tcpportssl": 8443
}'
```
