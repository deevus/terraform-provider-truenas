# SSH API

SSH server configuration.

## SSH Configuration

### ssh.config
Get SSH configuration.
```bash
midclt call ssh.config
```

Returns:
- `id` - Config ID
- `bindiface` - Bind interfaces
- `tcpport` - SSH port
- `rootlogin` - Root login allowed
- `passwordauth` - Password auth allowed
- `kerberosauth` - Kerberos auth allowed
- `tcpfwd` - TCP forwarding allowed
- `compression` - Compression enabled
- `sftp_log_level` - SFTP log level
- `sftp_log_facility` - SFTP log facility
- `weak_ciphers` - Weak ciphers allowed
- `options` - Additional SSH options

### ssh.update
Update SSH configuration.
```bash
midclt call ssh.update '{
  "tcpport": 22,
  "rootlogin": false,
  "passwordauth": true,
  "tcpfwd": false,
  "compression": false
}'
```

Secure configuration:
```bash
midclt call ssh.update '{
  "tcpport": 22,
  "rootlogin": false,
  "passwordauth": false,
  "kerberosauth": false,
  "tcpfwd": false,
  "compression": false,
  "weak_ciphers": []
}'
```

With bind interfaces:
```bash
midclt call ssh.update '{
  "bindiface": ["eno1", "eno2"],
  "tcpport": 22
}'
```

With additional options:
```bash
midclt call ssh.update '{
  "options": "MaxAuthTries 3\nClientAliveInterval 300\nClientAliveCountMax 2"
}'
```

### ssh.bindiface_choices
Get available bind interfaces.
```bash
midclt call ssh.bindiface_choices
```

## Service Control

Start/stop SSH service:
```bash
midclt call service.update "ssh" '{"enable": true}'
midclt call service.control "ssh" "START"
midclt call service.control "ssh" "STOP"
midclt call service.control "ssh" "RESTART"
```

## SFTP Log Levels

| Level | Description |
|-------|-------------|
| `QUIET` | No logging |
| `FATAL` | Fatal errors only |
| `ERROR` | Errors |
| `INFO` | Informational |
| `VERBOSE` | Verbose |
| `DEBUG` | Debug level 1 |
| `DEBUG2` | Debug level 2 |
| `DEBUG3` | Debug level 3 |

## SFTP Log Facilities

| Facility | Description |
|----------|-------------|
| `DAEMON` | Daemon facility |
| `USER` | User facility |
| `AUTH` | Auth facility |
| `LOCAL0-7` | Local facilities |

## Security Recommendations

1. **Disable root login** - Use sudo instead
2. **Disable password auth** - Use SSH keys only
3. **Disable TCP forwarding** - Unless needed
4. **Change default port** - Security through obscurity
5. **Use strong ciphers** - Avoid weak ciphers
6. **Limit bind interfaces** - Only listen on needed interfaces
