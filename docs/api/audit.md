# Audit API

System audit logging.

## Audit Configuration

### audit.config
Get audit configuration.
```bash
midclt call audit.config
```

Returns:
- `id` - Config ID
- `retention` - Log retention (days)
- `reservation` - Reserved space (GiB)
- `quota` - Maximum space (GiB)
- `quota_fill_warning` - Warning threshold (%)
- `quota_fill_critical` - Critical threshold (%)

### audit.update
Update audit configuration.
```bash
midclt call audit.update '{
  "retention": 30,
  "reservation": 1,
  "quota": 5,
  "quota_fill_warning": 75,
  "quota_fill_critical": 95
}'
```

## Audit Queries

### audit.query
Query audit logs.
```bash
midclt call audit.query
```

With filters:
```bash
midclt call audit.query '[["service", "=", "SMB"]]'
midclt call audit.query '[["username", "=", "admin"]]'
midclt call audit.query '[["event", "=", "AUTHENTICATION"]]'
```

Time range:
```bash
midclt call audit.query '[]' '{
  "start": "2024-01-01T00:00:00",
  "end": "2024-01-31T23:59:59"
}'
```

Returns:
- `audit_id` - Audit entry ID
- `timestamp` - Event timestamp
- `message_timestamp` - Message timestamp
- `address` - Source address
- `username` - Username
- `session` - Session ID
- `service` - Service name (SMB, NFS, SSH, etc.)
- `service_data` - Service-specific data
- `event` - Event type
- `event_data` - Event-specific data
- `success` - Success flag

### audit.export
Export audit logs.
```bash
midclt call audit.export '{"format": "CSV"}'
midclt call audit.export '{"format": "JSON"}'
```

With filters:
```bash
midclt call audit.export '{
  "format": "CSV",
  "query_filters": [["service", "=", "SMB"]],
  "start": "2024-01-01T00:00:00",
  "end": "2024-01-31T23:59:59"
}'
```

## Audit Event Types

| Event | Description |
|-------|-------------|
| `AUTHENTICATION` | Login attempts |
| `AUTHORIZATION` | Permission checks |
| `CREATE` | Resource creation |
| `MODIFY` | Resource modification |
| `DELETE` | Resource deletion |
| `READ` | Resource access |
| `CONNECT` | Connection events |
| `DISCONNECT` | Disconnection events |

## Audited Services

| Service | Description |
|---------|-------------|
| `SMB` | SMB/CIFS file sharing |
| `NFS` | NFS file sharing |
| `SSH` | SSH access |
| `MIDDLEWARE` | API calls |
| `SUDO` | Privilege escalation |

## Audit Log Analysis

### Find failed logins
```bash
midclt call audit.query '[[
  ["event", "=", "AUTHENTICATION"],
  ["success", "=", false]
]]'
```

### Find file access by user
```bash
midclt call audit.query '[[
  ["username", "=", "someuser"],
  ["service", "=", "SMB"]
]]'
```

### Export monthly report
```bash
midclt call audit.export '{
  "format": "CSV",
  "start": "2024-01-01T00:00:00",
  "end": "2024-01-31T23:59:59"
}'
```
