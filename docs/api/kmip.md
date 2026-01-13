# KMIP API

Key Management Interoperability Protocol for enterprise key management.

## KMIP Configuration

### kmip.config
Get KMIP configuration.
```bash
midclt call kmip.config
```

Returns:
- `id` - Config ID
- `server` - KMIP server address
- `port` - KMIP port
- `certificate` - Client certificate ID
- `certificate_authority` - CA certificate ID
- `manage_sed_disks` - Manage SED disk keys
- `manage_zfs_keys` - Manage ZFS encryption keys
- `enabled` - KMIP enabled
- `change_server` - Server change pending

### kmip.update
Update KMIP configuration.
```bash
midclt call kmip.update '{
  "server": "kmip.example.com",
  "port": 5696,
  "certificate": <client_cert_id>,
  "certificate_authority": <ca_cert_id>,
  "manage_sed_disks": true,
  "manage_zfs_keys": true,
  "enabled": true
}'
```

## KMIP Operations

### kmip.sync_keys
Synchronize keys with KMIP server.
```bash
midclt call kmip.sync_keys
```

### kmip.kmip_sync_pending
Check if key sync is pending.
```bash
midclt call kmip.kmip_sync_pending
```

### kmip.clear_sync_pending_keys
Clear pending sync keys.
```bash
midclt call kmip.clear_sync_pending_keys
```

## KMIP Setup

1. **Obtain certificates** from your KMIP server administrator:
   - Client certificate (for TrueNAS to authenticate)
   - CA certificate (to verify KMIP server)

2. **Import certificates**:
```bash
# Import CA certificate
midclt call certificate.create '{
  "name": "kmip-ca",
  "create_type": "CERTIFICATE_CREATE_IMPORTED",
  "certificate": "-----BEGIN CERTIFICATE-----\n..."
}'

# Import client certificate
midclt call certificate.create '{
  "name": "kmip-client",
  "create_type": "CERTIFICATE_CREATE_IMPORTED",
  "certificate": "-----BEGIN CERTIFICATE-----\n...",
  "privatekey": "-----BEGIN PRIVATE KEY-----\n..."
}'
```

3. **Configure KMIP**:
```bash
midclt call kmip.update '{
  "server": "kmip.example.com",
  "port": 5696,
  "certificate": <client_cert_id>,
  "certificate_authority": <ca_cert_id>,
  "manage_zfs_keys": true,
  "enabled": true
}'
```

4. **Sync keys**:
```bash
midclt call kmip.sync_keys
```

## Key Types Managed

| Type | Description |
|------|-------------|
| ZFS encryption keys | Keys for encrypted datasets |
| SED disk keys | Self-encrypting drive authentication keys |

## Enterprise Integration

KMIP is used in enterprise environments for:
- Centralized key management
- Compliance requirements
- Key lifecycle management
- Disaster recovery key escrow

Compatible KMIP servers:
- HashiCorp Vault (Enterprise)
- Thales CipherTrust
- IBM Security Key Lifecycle Manager
- Dell EMC CloudLink
- Gemalto SafeNet KeySecure
