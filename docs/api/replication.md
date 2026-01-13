# Replication API

ZFS replication for disaster recovery and backup.

## Replication Tasks

### replication.query
Query replication tasks.
```bash
midclt call replication.query
```

Returns:
- `id` - Task ID
- `name` - Task name
- `source_datasets` - Source dataset paths
- `target_dataset` - Target dataset path
- `recursive` - Include child datasets
- `exclude` - Excluded datasets
- `properties` - Replicate properties
- `properties_exclude` - Excluded properties
- `properties_override` - Property overrides
- `replicate` - Replicate everything or custom
- `encryption` - Encryption settings
- `netcat_active_side` - Netcat configuration
- `netcat_active_side_listen_address` - Netcat listen address
- `netcat_active_side_port_min` - Netcat port range min
- `netcat_active_side_port_max` - Netcat port range max
- `netcat_passive_side_connect_address` - Netcat connect address
- `direction` - PUSH or PULL
- `transport` - SSH, SSH+NETCAT, LOCAL, LEGACY
- `ssh_credentials` - SSH credential ID
- `retention_policy` - Snapshot retention
- `lifetime_value` - Retention value
- `lifetime_unit` - Retention unit
- `compression` - Transfer compression
- `speed_limit` - Bandwidth limit
- `large_block` - Large blocks
- `embed` - Embedded data
- `compressed` - Compressed send
- `retries` - Retry count
- `schedule` - Cron schedule
- `restrict_schedule` - Restrict run times
- `only_matching_schedule` - Only matching schedule
- `allow_from_scratch` - Allow full initial sync
- `hold_pending_snapshots` - Hold pending snapshots
- `auto` - Auto-run on schedule
- `enabled` - Task enabled
- `state` - Current state
- `job` - Last job info

### replication.create
Create a replication task.

Local replication:
```bash
midclt call replication.create '{
  "name": "local-backup",
  "source_datasets": ["tank/data"],
  "target_dataset": "backup/data",
  "direction": "PUSH",
  "transport": "LOCAL",
  "recursive": true,
  "auto": true,
  "retention_policy": "SOURCE",
  "schedule": {"minute": "0", "hour": "*/4", "dom": "*", "month": "*", "dow": "*"},
  "enabled": true
}'
```

Remote SSH replication:
```bash
midclt call replication.create '{
  "name": "remote-backup",
  "source_datasets": ["tank/data"],
  "target_dataset": "tank/replicated",
  "direction": "PUSH",
  "transport": "SSH",
  "ssh_credentials": <ssh_cred_id>,
  "recursive": true,
  "auto": true,
  "retention_policy": "CUSTOM",
  "lifetime_value": 30,
  "lifetime_unit": "DAY",
  "compression": "LZ4",
  "speed_limit": null,
  "schedule": {"minute": "0", "hour": "2", "dom": "*", "month": "*", "dow": "*"},
  "enabled": true
}'
```

Pull replication:
```bash
midclt call replication.create '{
  "name": "pull-from-remote",
  "source_datasets": ["tank/source"],
  "target_dataset": "tank/pulled",
  "direction": "PULL",
  "transport": "SSH",
  "ssh_credentials": <ssh_cred_id>,
  "recursive": false,
  "retention_policy": "NONE",
  "enabled": true
}'
```

### replication.update
Update a replication task.
```bash
midclt call replication.update <task_id> '{"enabled": false}'
```

### replication.delete
Delete a replication task.
```bash
midclt call replication.delete <task_id>
```

### replication.run
Run a replication task immediately.
```bash
midclt call replication.run <task_id>
```

### replication.restore
Restore from replication target.
```bash
midclt call replication.restore '{
  "name": "restore-task",
  "target_dataset": "tank/restored"
}'
```

### replication.count_eligible_manual_snapshots
Count snapshots eligible for replication.
```bash
midclt call replication.count_eligible_manual_snapshots '["tank/data"]' '["auto-%Y-%m-%d_%H-%M"]' "SSH" <ssh_cred_id>
```

### replication.target_unmatched_snapshots
Find unmatched snapshots on target.
```bash
midclt call replication.target_unmatched_snapshots "PUSH" "tank/data" "tank/replicated" "SSH" <ssh_cred_id>
```

### replication.list_naming_schemas
List snapshot naming schemas.
```bash
midclt call replication.list_naming_schemas
```

## Replication Configuration

### replication.config.config
Get replication global configuration.
```bash
midclt call replication.config.config
```

### replication.config.update
Update replication global configuration.
```bash
midclt call replication.config.update '{"max_parallel_replication_tasks": 2}'
```

## Retention Policies

| Policy | Description |
|--------|-------------|
| `SOURCE` | Match source retention |
| `CUSTOM` | Custom lifetime |
| `NONE` | Keep all snapshots |

## Transport Types

| Transport | Description |
|-----------|-------------|
| `LOCAL` | Local replication |
| `SSH` | SSH transport |
| `SSH+NETCAT` | SSH with netcat for data |
| `LEGACY` | Legacy transport |

## Compression Options

| Compression | Description |
|-------------|-------------|
| `LZ4` | LZ4 compression (fast) |
| `PIGZ` | Parallel gzip |
| `PLZIP` | Parallel lzip |
| `DISABLED` | No compression |

## SSH Credentials

Replication uses keychain credentials for SSH authentication. See the [Keychain API](./keychain.md) for credential management.

## Complete Replication Setup Example

```bash
# 1. Create SSH keypair
midclt call keychaincredential.generate_ssh_key_pair

# 2. Create SSH credential
midclt call keychaincredential.create '{
  "name": "backup-server",
  "type": "SSH_CREDENTIALS",
  "attributes": {
    "host": "backup.example.com",
    "port": 22,
    "username": "root",
    "private_key": <private_key_id>,
    "remote_host_key": "ssh-rsa AAAA..."
  }
}'

# 3. Create snapshot task
midclt call pool.snapshottask.create '{
  "dataset": "tank/data",
  "recursive": true,
  "lifetime_value": 1,
  "lifetime_unit": "WEEK",
  "naming_schema": "auto-%Y-%m-%d_%H-%M",
  "schedule": {"minute": "0", "hour": "*", "dom": "*", "month": "*", "dow": "*"}
}'

# 4. Create replication task
midclt call replication.create '{
  "name": "offsite-backup",
  "source_datasets": ["tank/data"],
  "target_dataset": "backup/data",
  "direction": "PUSH",
  "transport": "SSH",
  "ssh_credentials": <ssh_cred_id>,
  "recursive": true,
  "auto": true,
  "retention_policy": "CUSTOM",
  "lifetime_value": 30,
  "lifetime_unit": "DAY",
  "schedule": {"minute": "30", "hour": "*/4", "dom": "*", "month": "*", "dow": "*"},
  "enabled": true
}'
```
