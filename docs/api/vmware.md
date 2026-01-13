# VMware API

VMware vCenter/ESXi integration for snapshot coordination.

## VMware Credentials

### vmware.query
Query VMware credentials.
```bash
midclt call vmware.query
```

Returns:
- `id` - Credential ID
- `hostname` - vCenter/ESXi hostname
- `username` - Username
- `datastore` - Associated datastores

### vmware.create
Create VMware credentials.
```bash
midclt call vmware.create '{
  "hostname": "vcenter.example.com",
  "username": "administrator@vsphere.local",
  "password": "password",
  "filesystem": "tank/vmware"
}'
```

For ESXi host:
```bash
midclt call vmware.create '{
  "hostname": "esxi01.example.com",
  "username": "root",
  "password": "password",
  "filesystem": "tank/vms"
}'
```

### vmware.update
Update VMware credentials.
```bash
midclt call vmware.update <cred_id> '{
  "password": "newpassword"
}'
```

### vmware.delete
Delete VMware credentials.
```bash
midclt call vmware.delete <cred_id>
```

## VMware Integration

### vmware.dataset_has_vms
Check if dataset has VMs.
```bash
midclt call vmware.dataset_has_vms '"tank/vmware"'
```

### vmware.match_datastores_with_datasets
Match VMware datastores with ZFS datasets.
```bash
midclt call vmware.match_datastores_with_datasets <cred_id>
```

## VMware Snapshot Coordination

When configured, TrueNAS can coordinate with VMware to:

1. **Quiesce VMs** before taking ZFS snapshots
2. **Create VMware snapshots** for crash-consistent backups
3. **Remove VMware snapshots** after ZFS snapshot completes

This ensures consistent backups of VMs stored on NFS datastores.

## Setup Workflow

1. **Create NFS share for VMware**:
```bash
midclt call sharing.nfs.create '{
  "path": "/mnt/tank/vmware",
  "networks": ["192.168.1.0/24"],
  "maproot_user": "root",
  "maproot_group": "wheel"
}'
```

2. **Add datastore in vCenter** pointing to NFS share

3. **Add VMware credentials**:
```bash
midclt call vmware.create '{
  "hostname": "vcenter.example.com",
  "username": "administrator@vsphere.local",
  "password": "password",
  "filesystem": "tank/vmware"
}'
```

4. **Create snapshot task** (VMware snapshots will be coordinated automatically):
```bash
midclt call pool.snapshottask.create '{
  "dataset": "tank/vmware",
  "recursive": true,
  "lifetime_value": 1,
  "lifetime_unit": "WEEK",
  "naming_schema": "auto-%Y-%m-%d_%H-%M",
  "schedule": {"minute": "0", "hour": "0", "dom": "*", "month": "*", "dow": "*"},
  "vmware_sync": true
}'
```

## Requirements

- VMware vCenter 6.5+ or ESXi 6.5+
- VMs on NFS datastore backed by TrueNAS
- VMware Tools installed in guest VMs (for quiescing)
- Network connectivity between TrueNAS and vCenter/ESXi
