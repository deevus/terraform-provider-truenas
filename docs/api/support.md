# Support API

Support ticket and debug operations.

## Support Configuration

### support.config
Get support configuration.
```bash
midclt call support.config
```

### support.update
Update support configuration.
```bash
midclt call support.update '{
  "enabled": true,
  "name": "Admin",
  "title": "System Administrator",
  "email": "admin@example.com",
  "phone": "+1-555-0100",
  "secondary_name": "Backup Admin",
  "secondary_title": "Backup Administrator",
  "secondary_email": "backup@example.com",
  "secondary_phone": "+1-555-0101"
}'
```

### support.is_available
Check if support is available.
```bash
midclt call support.is_available
```

### support.is_available_and_enabled
Check if support is available and enabled.
```bash
midclt call support.is_available_and_enabled
```

## Support Tickets

### support.new_ticket
Create a new support ticket.
```bash
midclt call support.new_ticket '{
  "title": "Issue with pool performance",
  "body": "Detailed description of the issue...",
  "category": "Performance",
  "criticality": "inquiry",
  "environment": "Production",
  "phone": "+1-555-0100",
  "email": "admin@example.com",
  "name": "Admin",
  "attach_debug": true
}'
```

### support.similar_issues
Search for similar known issues.
```bash
midclt call support.similar_issues '"pool degraded performance"'
```

### support.attach_ticket_max_size
Get maximum attachment size for tickets.
```bash
midclt call support.attach_ticket_max_size
```

## Debug Operations

### system.debug
Generate system debug file.
```bash
midclt call system.debug
```

This creates a comprehensive debug archive containing:
- System configuration
- Log files
- Hardware information
- ZFS pool status
- Network configuration
- Service status

## TrueNAS Operations

### truenas.is_ix_hardware
Check if running on iXsystems hardware.
```bash
midclt call truenas.is_ix_hardware
```

### truenas.is_production
Check if system is in production mode.
```bash
midclt call truenas.is_production
```

### truenas.set_production
Set production mode.
```bash
midclt call truenas.set_production true
```

### truenas.get_eula
Get End User License Agreement.
```bash
midclt call truenas.get_eula
```

### truenas.is_eula_accepted
Check if EULA is accepted.
```bash
midclt call truenas.is_eula_accepted
```

### truenas.accept_eula
Accept the EULA.
```bash
midclt call truenas.accept_eula
```

### truenas.managed_by_truecommand
Check if managed by TrueCommand.
```bash
midclt call truenas.managed_by_truecommand
```

## TrueCommand (TN Connect)

### tn_connect.config
Get TrueCommand connection configuration.
```bash
midclt call tn_connect.config
```

### tn_connect.update
Update TrueCommand connection.
```bash
midclt call tn_connect.update '{
  "enabled": true,
  "api_key": "your-api-key"
}'
```

### tn_connect.generate_claim_token
Generate claim token for TrueCommand.
```bash
midclt call tn_connect.generate_claim_token
```

### tn_connect.get_registration_uri
Get TrueCommand registration URI.
```bash
midclt call tn_connect.get_registration_uri
```

### tn_connect.ips_with_hostnames
Get IPs with hostnames for TrueCommand.
```bash
midclt call tn_connect.ips_with_hostnames
```

## Ticket Criticality Levels

| Level | Description |
|-------|-------------|
| `inquiry` | General question |
| `suggestion` | Feature request |
| `bug` | Bug report |

## Support Workflow

1. **Check availability**:
```bash
midclt call support.is_available_and_enabled
```

2. **Search for similar issues**:
```bash
midclt call support.similar_issues '"your issue description"'
```

3. **Generate debug if needed**:
```bash
midclt call system.debug
```

4. **Create ticket**:
```bash
midclt call support.new_ticket '{
  "title": "...",
  "body": "...",
  "attach_debug": true
}'
```
