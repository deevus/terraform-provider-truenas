# Alerts API

Alert management and notification services.

## Alerts

### alert.list
List current alerts.
```bash
midclt call alert.list
```

Returns array of:
- `uuid` - Alert UUID
- `source` - Alert source
- `klass` - Alert class
- `args` - Alert arguments
- `node` - Node name
- `key` - Alert key
- `datetime` - Alert timestamp
- `last_occurrence` - Last occurrence time
- `dismissed` - Dismissed flag
- `mail` - Email sent flag
- `text` - Alert message
- `id` - Alert ID
- `level` - CRITICAL, WARNING, INFO, etc.
- `formatted` - Formatted message
- `one_shot` - One-shot alert

### alert.dismiss
Dismiss an alert.
```bash
midclt call alert.dismiss "<alert_uuid>"
```

### alert.restore
Restore a dismissed alert.
```bash
midclt call alert.restore "<alert_uuid>"
```

### alert.list_categories
List alert categories.
```bash
midclt call alert.list_categories
```

### alert.list_policies
List alert policies.
```bash
midclt call alert.list_policies
```

## Alert Classes

### alertclasses.config
Get alert class configuration.
```bash
midclt call alertclasses.config
```

### alertclasses.update
Update alert class configuration.
```bash
midclt call alertclasses.update '{
  "classes": {
    "UPSOnBattery": {"level": "WARNING", "policy": "IMMEDIATELY"}
  }
}'
```

## Alert Services

### alertservice.query
Query alert notification services.
```bash
midclt call alertservice.query
```

Returns:
- `id` - Service ID
- `name` - Service name
- `type` - Service type
- `attributes` - Service-specific attributes
- `level` - Minimum alert level
- `enabled` - Service enabled

### alertservice.create
Create an alert service.

Email:
```bash
midclt call alertservice.create '{
  "name": "Email Alerts",
  "type": "Mail",
  "attributes": {
    "email": "admin@example.com"
  },
  "level": "WARNING",
  "enabled": true
}'
```

Slack:
```bash
midclt call alertservice.create '{
  "name": "Slack Alerts",
  "type": "Slack",
  "attributes": {
    "url": "https://hooks.slack.com/services/XXX/YYY/ZZZ"
  },
  "level": "WARNING",
  "enabled": true
}'
```

PagerDuty:
```bash
midclt call alertservice.create '{
  "name": "PagerDuty",
  "type": "PagerDuty",
  "attributes": {
    "service_key": "your-service-key",
    "client_name": "TrueNAS"
  },
  "level": "CRITICAL",
  "enabled": true
}'
```

OpsGenie:
```bash
midclt call alertservice.create '{
  "name": "OpsGenie",
  "type": "OpsGenie",
  "attributes": {
    "api_key": "your-api-key",
    "api_url": "https://api.opsgenie.com"
  },
  "level": "CRITICAL",
  "enabled": true
}'
```

Mattermost:
```bash
midclt call alertservice.create '{
  "name": "Mattermost",
  "type": "Mattermost",
  "attributes": {
    "url": "https://mattermost.example.com/hooks/xxx",
    "username": "TrueNAS",
    "channel": "alerts",
    "icon_url": ""
  },
  "level": "WARNING",
  "enabled": true
}'
```

Telegram:
```bash
midclt call alertservice.create '{
  "name": "Telegram",
  "type": "Telegram",
  "attributes": {
    "bot_token": "your-bot-token",
    "chat_ids": ["123456789"]
  },
  "level": "WARNING",
  "enabled": true
}'
```

SNMP Trap:
```bash
midclt call alertservice.create '{
  "name": "SNMP Traps",
  "type": "SNMPTrap",
  "attributes": {
    "host": "snmp-server.example.com",
    "port": 162,
    "v3": false,
    "community": "public"
  },
  "level": "WARNING",
  "enabled": true
}'
```

VictorOps:
```bash
midclt call alertservice.create '{
  "name": "VictorOps",
  "type": "VictorOps",
  "attributes": {
    "api_key": "your-api-key",
    "routing_key": "your-routing-key"
  },
  "level": "CRITICAL",
  "enabled": true
}'
```

AWS SNS:
```bash
midclt call alertservice.create '{
  "name": "AWS SNS",
  "type": "AWSSNS",
  "attributes": {
    "region": "us-east-1",
    "topic_arn": "arn:aws:sns:us-east-1:123456789:alerts",
    "aws_access_key_id": "AKIAXXXXXXXX",
    "aws_secret_access_key": "secretkey"
  },
  "level": "WARNING",
  "enabled": true
}'
```

### alertservice.update
Update an alert service.
```bash
midclt call alertservice.update <service_id> '{"level": "CRITICAL"}'
```

### alertservice.delete
Delete an alert service.
```bash
midclt call alertservice.delete <service_id>
```

### alertservice.test
Test an alert service.
```bash
midclt call alertservice.test '{
  "name": "Test Email",
  "type": "Mail",
  "attributes": {"email": "admin@example.com"},
  "level": "INFO",
  "enabled": true
}'
```

## Alert Levels

| Level | Description |
|-------|-------------|
| `INFO` | Informational |
| `NOTICE` | Notice |
| `WARNING` | Warning |
| `ERROR` | Error |
| `CRITICAL` | Critical |
| `ALERT` | Alert |
| `EMERGENCY` | Emergency |

## Alert Policies

| Policy | Description |
|--------|-------------|
| `IMMEDIATELY` | Send immediately |
| `HOURLY` | Batch hourly |
| `DAILY` | Batch daily |
| `NEVER` | Never send |

## Alert Service Types

| Type | Description |
|------|-------------|
| `Mail` | Email |
| `Slack` | Slack webhook |
| `Mattermost` | Mattermost webhook |
| `PagerDuty` | PagerDuty |
| `OpsGenie` | OpsGenie |
| `Telegram` | Telegram bot |
| `VictorOps` | VictorOps (Splunk On-Call) |
| `AWSSNS` | AWS Simple Notification Service |
| `SNMPTrap` | SNMP trap |
| `InfluxDB` | InfluxDB |
