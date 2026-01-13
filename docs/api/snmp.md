# SNMP API

SNMP monitoring configuration.

## SNMP Configuration

### snmp.config
Get SNMP configuration.
```bash
midclt call snmp.config
```

Returns:
- `id` - Config ID
- `location` - System location
- `contact` - Contact information
- `traps` - Enable traps
- `v3` - SNMPv3 enabled
- `community` - SNMPv1/v2c community string
- `v3_username` - SNMPv3 username
- `v3_authtype` - Auth type (MD5, SHA)
- `v3_privproto` - Privacy protocol (AES, DES)
- `loglevel` - Log level
- `options` - Additional options
- `zilstat` - ZIL statistics

### snmp.update
Update SNMP configuration.

SNMPv2c configuration:
```bash
midclt call snmp.update '{
  "location": "Server Room A",
  "contact": "admin@example.com",
  "community": "public",
  "traps": false,
  "v3": false
}'
```

SNMPv3 configuration:
```bash
midclt call snmp.update '{
  "location": "Server Room A",
  "contact": "admin@example.com",
  "v3": true,
  "v3_username": "snmpuser",
  "v3_authtype": "SHA",
  "v3_password": "authpassword",
  "v3_privproto": "AES",
  "v3_privpassphrase": "privpassword"
}'
```

With ZIL statistics:
```bash
midclt call snmp.update '{
  "zilstat": true
}'
```

## Service Control

```bash
midclt call service.update "snmp" '{"enable": true}'
midclt call service.control "snmp" "START"
```

## Authentication Types

| Type | Description |
|------|-------------|
| `MD5` | MD5 authentication |
| `SHA` | SHA authentication |

## Privacy Protocols

| Protocol | Description |
|----------|-------------|
| `AES` | AES encryption |
| `DES` | DES encryption (less secure) |

## Log Levels

| Level | Description |
|-------|-------------|
| `0` | Emergency |
| `1` | Alert |
| `2` | Critical |
| `3` | Error |
| `4` | Warning |
| `5` | Notice |
| `6` | Info |
| `7` | Debug |

## Common OIDs

| OID | Description |
|-----|-------------|
| `.1.3.6.1.2.1.1.1` | System description |
| `.1.3.6.1.2.1.1.3` | System uptime |
| `.1.3.6.1.2.1.1.5` | System name |
| `.1.3.6.1.2.1.1.6` | System location |
| `.1.3.6.1.2.1.2.2` | Interface table |
| `.1.3.6.1.2.1.25.2` | Storage table |
| `.1.3.6.1.2.1.25.3.3` | Processor load |

## TrueNAS-Specific OIDs

TrueNAS exposes ZFS statistics through custom OIDs under:
- `.1.3.6.1.4.1.50536` - TrueNAS enterprise OID
