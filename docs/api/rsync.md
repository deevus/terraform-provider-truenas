# Rsync API

Rsync task management for file synchronization.

## Rsync Tasks

### rsynctask.query
Query rsync tasks.
```bash
midclt call rsynctask.query
midclt call rsynctask.query '[[["enabled", "=", true]]]'
```

Returns:
- `id` - Task ID
- `path` - Local path
- `remotehost` - Remote host
- `remoteport` - Remote SSH port
- `remotemodule` - Remote rsync module
- `remotepath` - Remote path
- `direction` - PUSH or PULL
- `desc` - Description
- `user` - Run as user
- `mode` - MODULE or SSH
- `extra` - Extra rsync options
- `archive` - Archive mode
- `compress` - Compression
- `times` - Preserve times
- `preserveattr` - Preserve attributes
- `preserveperm` - Preserve permissions
- `recursive` - Recursive
- `delete` - Delete extraneous files
- `delayupdates` - Delay updates
- `quiet` - Quiet mode
- `enabled` - Task enabled
- `schedule` - Cron schedule
- `ssh_credentials` - SSH credential ID
- `validate_rpath` - Validate remote path
- `job` - Last job status

### rsynctask.create
Create an rsync task.

Push to remote (SSH mode):
```bash
midclt call rsynctask.create '{
  "path": "/mnt/tank/data",
  "remotehost": "backup.example.com",
  "direction": "PUSH",
  "mode": "SSH",
  "remotepath": "/backup/data",
  "user": "root",
  "ssh_credentials": <ssh_cred_id>,
  "archive": true,
  "compress": true,
  "recursive": true,
  "delete": true,
  "desc": "Daily backup to remote",
  "schedule": {
    "minute": "0",
    "hour": "2",
    "dom": "*",
    "month": "*",
    "dow": "*"
  },
  "enabled": true
}'
```

Pull from remote:
```bash
midclt call rsynctask.create '{
  "path": "/mnt/tank/incoming",
  "remotehost": "source.example.com",
  "direction": "PULL",
  "mode": "SSH",
  "remotepath": "/data/export",
  "user": "root",
  "ssh_credentials": <ssh_cred_id>,
  "archive": true,
  "compress": true,
  "recursive": true,
  "delete": false,
  "desc": "Pull data from source",
  "enabled": true
}'
```

Module mode (rsync daemon):
```bash
midclt call rsynctask.create '{
  "path": "/mnt/tank/data",
  "remotehost": "rsync.example.com",
  "remotemodule": "backup",
  "direction": "PUSH",
  "mode": "MODULE",
  "user": "rsync",
  "archive": true,
  "compress": true,
  "desc": "Rsync daemon backup",
  "enabled": true
}'
```

With extra options:
```bash
midclt call rsynctask.create '{
  "path": "/mnt/tank/data",
  "remotehost": "backup.example.com",
  "remotepath": "/backup",
  "direction": "PUSH",
  "mode": "SSH",
  "ssh_credentials": <ssh_cred_id>,
  "archive": true,
  "extra": "--exclude=*.tmp --exclude=.cache --bwlimit=10000",
  "enabled": true
}'
```

### rsynctask.update
Update an rsync task.
```bash
midclt call rsynctask.update <task_id> '{"enabled": false}'
```

### rsynctask.delete
Delete an rsync task.
```bash
midclt call rsynctask.delete <task_id>
```

### rsynctask.run
Run an rsync task immediately.
```bash
midclt call rsynctask.run <task_id>
```

## Mode Options

| Mode | Description |
|------|-------------|
| `SSH` | Rsync over SSH |
| `MODULE` | Rsync daemon module |

## Direction Options

| Direction | Description |
|-----------|-------------|
| `PUSH` | Local to remote |
| `PULL` | Remote to local |

## Common Rsync Options

| Option | Description |
|--------|-------------|
| `archive` | Archive mode (-a) |
| `compress` | Compress transfer (-z) |
| `times` | Preserve timestamps (-t) |
| `preserveattr` | Preserve extended attrs (-X) |
| `preserveperm` | Preserve permissions (-p) |
| `recursive` | Recursive (-r) |
| `delete` | Delete extraneous files (--delete) |
| `delayupdates` | Delay updates (--delay-updates) |
| `quiet` | Quiet mode (-q) |

## Extra Options Examples

```bash
# Bandwidth limit (KB/s)
"--bwlimit=10000"

# Exclude patterns
"--exclude=*.tmp --exclude=.git --exclude=node_modules"

# Include/exclude file
"--exclude-from=/etc/rsync-excludes.txt"

# Partial transfers
"--partial --partial-dir=.rsync-partial"

# Dry run (test)
"--dry-run"

# Verbose output
"-v --progress"
```
