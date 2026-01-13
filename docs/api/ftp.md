# FTP API

FTP server configuration.

## FTP Configuration

### ftp.config
Get FTP configuration.
```bash
midclt call ftp.config
```

Returns:
- `id` - Config ID
- `port` - FTP port
- `clients` - Maximum clients
- `ipconnections` - Connections per IP
- `loginattempt` - Login attempts
- `timeout` - Idle timeout
- `timeout_notransfer` - No transfer timeout
- `rootlogin` - Root login allowed
- `onlyanonymous` - Anonymous only
- `anonpath` - Anonymous path
- `onlylocal` - Local users only
- `banner` - Login banner
- `filemask` - File creation mask
- `dirmask` - Directory creation mask
- `fxp` - FXP transfers allowed
- `resume` - Resume transfers
- `defaultroot` - Chroot users
- `ident` - Ident protocol
- `reversedns` - Reverse DNS lookup
- `masqaddress` - Masquerade address
- `passiveportsmin` - Passive port min
- `passiveportsmax` - Passive port max
- `localuserbw` - Local user bandwidth
- `localuserdownbandwidth` - Local download bandwidth
- `anonuserbw` - Anonymous upload bandwidth
- `anonuserdownbandwidth` - Anonymous download bandwidth
- `tls` - TLS enabled
- `tls_policy` - TLS policy
- `tls_opt_allow_client_renegotiations` - Allow renegotiations
- `tls_opt_allow_dot_login` - Allow .login
- `tls_opt_allow_per_user` - Per-user TLS
- `tls_opt_common_name_required` - CN required
- `tls_opt_enable_diags` - TLS diagnostics
- `tls_opt_export_cert_data` - Export cert data
- `tls_opt_no_cert_request` - No cert request
- `tls_opt_no_empty_fragments` - No empty fragments
- `tls_opt_no_session_reuse_required` - No session reuse
- `tls_opt_stdenvvars` - Standard env vars
- `tls_opt_dns_name_required` - DNS name required
- `tls_opt_ip_address_required` - IP address required
- `ssltls_certificate` - TLS certificate ID
- `options` - Additional options

### ftp.update
Update FTP configuration.

Basic configuration:
```bash
midclt call ftp.update '{
  "port": 21,
  "clients": 32,
  "ipconnections": 0,
  "loginattempt": 3,
  "timeout": 600,
  "rootlogin": false,
  "defaultroot": true
}'
```

With TLS:
```bash
midclt call ftp.update '{
  "tls": true,
  "tls_policy": "on",
  "ssltls_certificate": 1,
  "passiveportsmin": 49152,
  "passiveportsmax": 65535
}'
```

Anonymous FTP:
```bash
midclt call ftp.update '{
  "onlyanonymous": true,
  "anonpath": "/mnt/tank/public",
  "anonuserbw": 1048576,
  "anonuserdownbandwidth": 0
}'
```

Local users only:
```bash
midclt call ftp.update '{
  "onlylocal": true,
  "defaultroot": true,
  "localuserbw": 0,
  "localuserdownbandwidth": 0
}'
```

With masquerade address (for NAT):
```bash
midclt call ftp.update '{
  "masqaddress": "public.ip.address",
  "passiveportsmin": 49152,
  "passiveportsmax": 65535
}'
```

## Service Control

```bash
midclt call service.update "ftp" '{"enable": true}'
midclt call service.control "ftp" "START"
```

## TLS Policies

| Policy | Description |
|--------|-------------|
| `on` | TLS required |
| `off` | TLS disabled |
| `data` | TLS required for data |
| `!data` | TLS required except data |
| `auth` | TLS required for auth |
| `ctrl` | TLS required for control |
| `ctrl+data` | TLS required for both |
| `ctrl+!data` | TLS for control, not data |
| `auth+data` | TLS for auth and data |
| `auth+!data` | TLS for auth, not data |

## Security Recommendations

1. **Enable TLS** - Always use FTPS in production
2. **Chroot users** - Enable `defaultroot`
3. **Disable root login** - Never allow root FTP
4. **Limit connections** - Set `clients` and `ipconnections`
5. **Set passive ports** - Configure for firewall
6. **Prefer SFTP** - FTP is inherently less secure
