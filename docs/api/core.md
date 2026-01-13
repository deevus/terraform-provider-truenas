# Core API

Core system operations and job management.

## Job Management

### core.get_jobs
Query background jobs.
```bash
midclt call core.get_jobs
midclt call core.get_jobs '[[["state", "=", "RUNNING"]]]'
midclt call core.get_jobs '[[["method", "=", "pool.create"]]]'
```

Returns:
- `id` - Job ID
- `method` - API method
- `arguments` - Method arguments
- `transient` - Transient job flag
- `description` - Job description
- `abortable` - Can be aborted
- `logs_path` - Log file path
- `logs_excerpt` - Log excerpt
- `progress` - Progress object
- `result` - Job result
- `error` - Error message
- `exception` - Exception details
- `exc_info` - Exception info
- `state` - WAITING, RUNNING, SUCCESS, FAILED, ABORTED
- `time_started` - Start timestamp
- `time_finished` - Finish timestamp

### core.job_abort
Abort a running job.
```bash
midclt call core.job_abort <job_id>
```

### core.job_download_logs
Download job logs.
```bash
midclt call core.job_download_logs <job_id>
```

## Bulk Operations

### core.bulk
Execute multiple API calls in bulk.
```bash
midclt call core.bulk "user.create" '[
  [{"username": "user1", "full_name": "User 1", "password": "pass1"}],
  [{"username": "user2", "full_name": "User 2", "password": "pass2"}]
]'
```

Each call is executed independently, and results are returned as an array.

## File Downloads

### core.download
Generate download URL for a file.
```bash
midclt call core.download "pool.dataset.export_key" '["tank/encrypted"]' "key.txt"
```

Returns:
- `url` - Download URL
- `job_id` - Associated job ID

## Shell Operations

### core.resize_shell
Resize a shell session.
```bash
midclt call core.resize_shell "<shell_id>" '{"rows": 40, "cols": 120}'
```

## Event Subscriptions

### core.subscribe
Subscribe to events (WebSocket only).
```bash
# WebSocket message format:
{"msg": "sub", "id": "unique-id", "name": "core.get_jobs"}
```

Common subscription events:
- `core.get_jobs` - Job updates
- `disk.query` - Disk changes
- `pool.query` - Pool changes
- `interface.query` - Network changes
- `alert.list` - Alert updates
- `vm.query` - VM status changes
- `reporting.realtime` - Real-time reporting data

## Job States

| State | Description |
|-------|-------------|
| `WAITING` | Job is queued |
| `RUNNING` | Job is executing |
| `SUCCESS` | Job completed successfully |
| `FAILED` | Job failed with error |
| `ABORTED` | Job was aborted |

## Progress Object

The progress object in job responses contains:
- `percent` - Completion percentage (0-100)
- `description` - Current operation description
- `extra` - Extra progress data

## Common Job Methods

Jobs are created automatically for long-running operations:
- `pool.create` - Pool creation
- `pool.export` - Pool export
- `pool.import_pool` - Pool import
- `pool.scrub.run` - Pool scrub
- `replication.run` - Replication task
- `cloudsync.sync` - Cloud sync task
- `update.update` - System update
- `system.debug` - Debug file generation
- `vm.start` - VM startup
- `app.create` - Application installation
