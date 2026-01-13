# Cron Jobs API

Scheduled task management.

## Cron Job Operations

### cronjob.query
Query cron jobs.
```bash
midclt call cronjob.query
midclt call cronjob.query '[[["enabled", "=", true]]]'
```

Returns:
- `id` - Job ID
- `user` - User to run as
- `command` - Command to execute
- `description` - Job description
- `enabled` - Job enabled
- `stdout` - Capture stdout
- `stderr` - Capture stderr
- `schedule` - Cron schedule object

Schedule object:
- `minute` - Minute (0-59, *, */n)
- `hour` - Hour (0-23, *, */n)
- `dom` - Day of month (1-31, *, */n)
- `month` - Month (1-12, *, */n)
- `dow` - Day of week (0-6 or mon-sun, *, */n)

### cronjob.create
Create a cron job.
```bash
midclt call cronjob.create '{
  "user": "root",
  "command": "/usr/local/bin/backup.sh",
  "description": "Daily backup script",
  "enabled": true,
  "stdout": true,
  "stderr": true,
  "schedule": {
    "minute": "0",
    "hour": "2",
    "dom": "*",
    "month": "*",
    "dow": "*"
  }
}'
```

Run every 15 minutes:
```bash
midclt call cronjob.create '{
  "user": "root",
  "command": "/usr/local/bin/check-health.sh",
  "description": "Health check",
  "enabled": true,
  "schedule": {
    "minute": "*/15",
    "hour": "*",
    "dom": "*",
    "month": "*",
    "dow": "*"
  }
}'
```

Weekdays only:
```bash
midclt call cronjob.create '{
  "user": "root",
  "command": "/usr/local/bin/workday-task.sh",
  "description": "Workday task",
  "enabled": true,
  "schedule": {
    "minute": "0",
    "hour": "9",
    "dom": "*",
    "month": "*",
    "dow": "1-5"
  }
}'
```

### cronjob.update
Update a cron job.
```bash
midclt call cronjob.update <job_id> '{
  "enabled": false
}'
```

Change schedule:
```bash
midclt call cronjob.update <job_id> '{
  "schedule": {
    "minute": "30",
    "hour": "3",
    "dom": "*",
    "month": "*",
    "dow": "*"
  }
}'
```

### cronjob.delete
Delete a cron job.
```bash
midclt call cronjob.delete <job_id>
```

### cronjob.run
Run a cron job immediately.
```bash
midclt call cronjob.run <job_id>
```

## Schedule Syntax

| Field | Values | Special Characters |
|-------|--------|-------------------|
| minute | 0-59 | *, */n, n-m |
| hour | 0-23 | *, */n, n-m |
| dom | 1-31 | *, */n, n-m |
| month | 1-12 | *, */n, n-m |
| dow | 0-6 (0=Sunday) | *, */n, n-m, mon-sun |

## Common Schedules

| Description | minute | hour | dom | month | dow |
|-------------|--------|------|-----|-------|-----|
| Every minute | * | * | * | * | * |
| Every hour | 0 | * | * | * | * |
| Daily at midnight | 0 | 0 | * | * | * |
| Daily at 2am | 0 | 2 | * | * | * |
| Weekly (Sunday midnight) | 0 | 0 | * | * | 0 |
| Monthly (1st at midnight) | 0 | 0 | 1 | * | * |
| Every 15 minutes | */15 | * | * | * | * |
| Every 6 hours | 0 | */6 | * | * | * |
| Weekdays at 9am | 0 | 9 | * | * | 1-5 |
| First Monday of month | 0 | 0 | 1-7 | * | 1 |

## Init/Shutdown Scripts

For scripts that run at boot or shutdown, see [Init/Shutdown Scripts](./init-shutdown.md).
