# Keychain Credentials API

SSH keys and credential management for replication and remote access.

## Credential Operations

### keychaincredential.query
Query keychain credentials.
```bash
midclt call keychaincredential.query
midclt call keychaincredential.query '[[["type", "=", "SSH_CREDENTIALS"]]]'
```

Returns:
- `id` - Credential ID
- `name` - Credential name
- `type` - Credential type
- `attributes` - Type-specific attributes

### keychaincredential.create
Create a keychain credential.

SSH key pair:
```bash
midclt call keychaincredential.create '{
  "name": "replication-key",
  "type": "SSH_KEY_PAIR",
  "attributes": {
    "private_key": "-----BEGIN OPENSSH PRIVATE KEY-----\n...",
    "public_key": "ssh-rsa AAAA..."
  }
}'
```

SSH credentials (for remote connection):
```bash
midclt call keychaincredential.create '{
  "name": "backup-server",
  "type": "SSH_CREDENTIALS",
  "attributes": {
    "host": "backup.example.com",
    "port": 22,
    "username": "root",
    "private_key": <key_pair_id>,
    "remote_host_key": "ssh-rsa AAAA...",
    "cipher": "STANDARD",
    "connect_timeout": 10
  }
}'
```

### keychaincredential.update
Update a keychain credential.
```bash
midclt call keychaincredential.update <cred_id> '{
  "name": "new-name"
}'
```

### keychaincredential.delete
Delete a keychain credential.
```bash
midclt call keychaincredential.delete <cred_id>
```

### keychaincredential.used_by
Check what uses this credential.
```bash
midclt call keychaincredential.used_by <cred_id>
```

## SSH Key Generation

### keychaincredential.generate_ssh_key_pair
Generate a new SSH key pair.
```bash
midclt call keychaincredential.generate_ssh_key_pair
```

Returns:
- `private_key` - Private key PEM
- `public_key` - Public key

With options:
```bash
midclt call keychaincredential.generate_ssh_key_pair '{"key_type": "RSA", "key_bits": 4096}'
midclt call keychaincredential.generate_ssh_key_pair '{"key_type": "ECDSA", "key_bits": 521}'
midclt call keychaincredential.generate_ssh_key_pair '{"key_type": "ED25519"}'
```

### keychaincredential.remote_ssh_host_key_scan
Scan remote host for SSH host key.
```bash
midclt call keychaincredential.remote_ssh_host_key_scan '{"host": "backup.example.com", "port": 22}'
```

### keychaincredential.setup_ssh_connection
Set up SSH connection (semi-automatic).
```bash
midclt call keychaincredential.setup_ssh_connection '{
  "private_key": <key_pair_id>,
  "connection_name": "backup-connection",
  "setup_type": "MANUAL",
  "manual_setup": {
    "host": "backup.example.com",
    "port": 22,
    "username": "root"
  }
}'
```

Or with semi-automatic setup (requires root password):
```bash
midclt call keychaincredential.setup_ssh_connection '{
  "private_key": <key_pair_id>,
  "connection_name": "backup-connection",
  "setup_type": "SEMI-AUTOMATIC",
  "semi_automatic_setup": {
    "url": "https://backup.example.com",
    "admin_username": "admin",
    "password": "admin_password",
    "username": "root"
  }
}'
```

## Credential Types

| Type | Description |
|------|-------------|
| `SSH_KEY_PAIR` | SSH public/private key pair |
| `SSH_CREDENTIALS` | SSH connection credentials |

## SSH Key Types

| Type | Description | Recommended Bits |
|------|-------------|------------------|
| `RSA` | RSA key | 4096 |
| `ECDSA` | ECDSA key | 256, 384, 521 |
| `ED25519` | Ed25519 key | N/A (fixed) |

## Cipher Options

| Cipher | Description |
|--------|-------------|
| `STANDARD` | Standard ciphers |
| `FAST` | Fast ciphers (less secure) |
| `DISABLED` | No encryption (not recommended) |

## Complete SSH Setup Example

```bash
# 1. Generate SSH key pair
KEYS=$(midclt call keychaincredential.generate_ssh_key_pair '{"key_type": "ED25519"}')

# 2. Create key pair credential
midclt call keychaincredential.create '{
  "name": "replication-keypair",
  "type": "SSH_KEY_PAIR",
  "attributes": {
    "private_key": "<private_key_from_step_1>",
    "public_key": "<public_key_from_step_1>"
  }
}'

# 3. Scan remote host key
midclt call keychaincredential.remote_ssh_host_key_scan '{"host": "remote-truenas.example.com", "port": 22}'

# 4. Create SSH credentials
midclt call keychaincredential.create '{
  "name": "remote-truenas",
  "type": "SSH_CREDENTIALS",
  "attributes": {
    "host": "remote-truenas.example.com",
    "port": 22,
    "username": "root",
    "private_key": <key_pair_id>,
    "remote_host_key": "<host_key_from_step_3>",
    "cipher": "STANDARD"
  }
}'

# 5. Use credentials in replication task (see replication.md for full parameters)
midclt call replication.create '{"name": "offsite-replication", "ssh_credentials": 1, ...}'
```
