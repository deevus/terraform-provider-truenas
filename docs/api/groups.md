# Group Management API

Group account management operations.

## Group Operations

### group.query
Query groups.
```bash
midclt call group.query
midclt call group.query '[[["name", "=", "wheel"]]]'
midclt call group.query '[[["gid", ">", 1000]]]'
midclt call group.query '[]' '{"extra": {"search_dscache": true}}'
```

Returns:
- `id` - Internal ID
- `gid` - Unix group ID
- `name` - Group name
- `group` - Group name (alias)
- `builtin` - Whether group is a system group
- `sudo_commands` - Allowed sudo commands
- `sudo_commands_nopasswd` - Passwordless sudo commands
- `smb` - SMB access enabled
- `users` - Member user IDs
- `local` - Whether group is local (not from directory service)

### group.create
Create a new group.
```bash
midclt call group.create '{
  "name": "newgroup"
}'
```

With specific GID:
```bash
midclt call group.create '{
  "name": "newgroup",
  "gid": 2000,
  "smb": true
}'
```

With sudo permissions:
```bash
midclt call group.create '{
  "name": "admins",
  "sudo_commands": ["ALL"],
  "sudo_commands_nopasswd": []
}'
```

### group.update
Update a group.
```bash
midclt call group.update <group_id> '{
  "name": "newname"
}'
```

Update sudo permissions:
```bash
midclt call group.update <group_id> '{
  "sudo_commands": ["/usr/bin/systemctl restart *"],
  "sudo_commands_nopasswd": ["/usr/bin/systemctl status *"]
}'
```

### group.delete
Delete a group.
```bash
midclt call group.delete <group_id>
midclt call group.delete <group_id> '{"delete_users": false}'
```

### group.get_group_obj
Get group object by GID or name.
```bash
midclt call group.get_group_obj '{"gid": 1000}'
midclt call group.get_group_obj '{"groupname": "wheel"}'
```

### group.get_next_gid
Get next available GID.
```bash
midclt call group.get_next_gid
```

## Group Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Group name (required) |
| `gid` | integer | Unix group ID (auto-assigned if not provided) |
| `smb` | boolean | Enable SMB access for group |
| `sudo_commands` | array | Allowed sudo commands |
| `sudo_commands_nopasswd` | array | Passwordless sudo commands |
| `users` | array | Member user IDs |

## Common Group Patterns

### Create admin group with sudo access
```bash
midclt call group.create '{
  "name": "admins",
  "sudo_commands": ["ALL"],
  "smb": false
}'
```

### Create SMB-only group
```bash
midclt call group.create '{
  "name": "smbusers",
  "smb": true,
  "sudo_commands": []
}'
```

### Add user to group
To add a user to a group, update the user's `groups` array:
```bash
midclt call user.update <user_id> '{"groups": [<group_id1>, <group_id2>]}'
```
