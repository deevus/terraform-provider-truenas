# User Management API

User account management operations.

## User Operations

### user.query
Query user accounts.
```bash
midclt call user.query
midclt call user.query '[[["username", "=", "admin"]]]'
midclt call user.query '[[["uid", ">", 1000]]]'
midclt call user.query '[]' '{"extra": {"search_dscache": true}}'
```

Returns:
- `id` - Internal ID
- `uid` - Unix user ID
- `username` - Login name
- `group` - Primary group object
- `groups` - Additional group IDs
- `home` - Home directory
- `shell` - Login shell
- `full_name` - Display name (GECOS)
- `email` - Email address
- `password_disabled` - Whether password auth is disabled
- `locked` - Whether account is locked
- `smb` - SMB access enabled
- `ssh_password_enabled` - SSH password auth enabled
- `sshpubkey` - SSH public key
- `builtin` - Whether user is a system account
- `local` - Whether user is local (not from directory service)

### user.create
Create a new user.
```bash
midclt call user.create '{
  "username": "newuser",
  "full_name": "New User",
  "email": "newuser@example.com",
  "password": "secretpassword",
  "group_create": true,
  "home": "/mnt/tank/home/newuser",
  "home_create": true,
  "shell": "/usr/bin/bash",
  "smb": true,
  "ssh_password_enabled": false
}'
```

With existing primary group:
```bash
midclt call user.create '{
  "username": "newuser",
  "full_name": "New User",
  "password": "secretpassword",
  "group": 1000,
  "groups": [1001, 1002],
  "home": "/nonexistent",
  "shell": "/usr/sbin/nologin"
}'
```

### user.update
Update a user account.
```bash
midclt call user.update <user_id> '{
  "full_name": "Updated Name",
  "email": "newemail@example.com"
}'
```

Change password:
```bash
midclt call user.update <user_id> '{"password": "newpassword"}'
```

Add SSH key:
```bash
midclt call user.update <user_id> '{"sshpubkey": "ssh-rsa AAAA... user@host"}'
```

Lock/unlock account:
```bash
midclt call user.update <user_id> '{"locked": true}'
midclt call user.update <user_id> '{"locked": false}'
```

### user.delete
Delete a user account.
```bash
midclt call user.delete <user_id>
midclt call user.delete <user_id> '{"delete_group": true}'
```

### user.set_password
Set user password (alternative method).
```bash
midclt call user.set_password '{
  "username": "admin",
  "old_password": "oldpass",
  "new_password": "newpass"
}'
```

### user.get_user_obj
Get user object by UID or username.
```bash
midclt call user.get_user_obj '{"uid": 1000}'
midclt call user.get_user_obj '{"username": "admin"}'
midclt call user.get_user_obj '{"uid": 1000, "get_groups": true}'
```

### user.get_next_uid
Get next available UID.
```bash
midclt call user.get_next_uid
```

### user.shell_choices
Get available shell choices.
```bash
midclt call user.shell_choices
```

Returns map of shell paths to descriptions:
```json
{
  "/usr/bin/bash": "bash",
  "/usr/bin/zsh": "zsh",
  "/usr/bin/sh": "sh",
  "/usr/sbin/nologin": "nologin",
  "/bin/false": "false"
}
```

### user.has_local_administrator_set_up
Check if local administrator is configured.
```bash
midclt call user.has_local_administrator_set_up
```

### user.setup_local_administrator
Set up local administrator account.
```bash
midclt call user.setup_local_administrator '{
  "username": "admin",
  "password": "adminpassword",
  "ec2": {"instance_id": "..."}
}'
```

## User Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `username` | string | Login name (required) |
| `uid` | integer | Unix user ID (auto-assigned if not provided) |
| `group` | integer | Primary group ID |
| `group_create` | boolean | Create matching group |
| `groups` | array | Additional group IDs |
| `home` | string | Home directory path |
| `home_mode` | string | Home directory permissions (e.g., "700") |
| `home_create` | boolean | Create home directory |
| `shell` | string | Login shell path |
| `full_name` | string | Display name |
| `email` | string | Email address |
| `password` | string | Password (write-only) |
| `password_disabled` | boolean | Disable password auth |
| `locked` | boolean | Lock account |
| `smb` | boolean | Enable SMB access |
| `ssh_password_enabled` | boolean | Enable SSH password auth |
| `sshpubkey` | string | SSH public key |
| `sudo_commands` | array | Allowed sudo commands |
| `sudo_commands_nopasswd` | array | Passwordless sudo commands |
