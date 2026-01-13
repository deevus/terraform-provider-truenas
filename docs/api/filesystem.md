# Filesystem API

Filesystem operations including permissions and ACLs.

## File Operations

### filesystem.stat
Get file/directory information.
```bash
midclt call filesystem.stat "/mnt/tank/data"
```

<!-- Source: internal/resources/host_path.go:39-43 -->
**Response Schema:**

| Field | Type | Description |
|-------|------|-------------|
| mode | integer | Unix permission mode (includes file type bits) |
| uid | integer | Owner user ID |
| gid | integer | Owner group ID |
| size | integer | File size in bytes |
| atime | float | Access time (Unix timestamp) |
| mtime | float | Modification time (Unix timestamp) |
| ctime | float | Change time (Unix timestamp) |
| dev | integer | Device ID |
| inode | integer | Inode number |
| nlink | integer | Number of hard links |
| user | string | Owner username |
| group | string | Owner group name |
| acl | boolean | ACL enabled flag |
| is_mountpoint | boolean | Whether path is a mountpoint |
| is_ctldir | boolean | Whether path is a .zfs control directory |

**Note:** `mode` is numeric and includes file type bits. Use `mode & 0777` for permissions only.

<details>
<summary>Example Response</summary>

```json
{
  "mode": 16877,
  "uid": 1000,
  "gid": 1000,
  "size": 4096,
  "atime": 1234567890.0,
  "mtime": 1234567890.0,
  "ctime": 1234567890.0,
  "dev": 65024,
  "inode": 123456,
  "nlink": 2,
  "user": "admin",
  "group": "admin",
  "acl": false,
  "is_mountpoint": false,
  "is_ctldir": false
}
```
</details>

### filesystem.statfs
Get filesystem statistics.
```bash
midclt call filesystem.statfs "/mnt/tank/data"
```

Returns:
- `flags` - Mount flags
- `fstype` - Filesystem type
- `source` - Source device/dataset
- `dest` - Mount destination
- `blocksize` - Block size
- `total_blocks` - Total blocks
- `free_blocks` - Free blocks
- `avail_blocks` - Available blocks
- `total_bytes` - Total size in bytes
- `free_bytes` - Free space in bytes
- `avail_bytes` - Available space in bytes

### filesystem.listdir
List directory contents.
```bash
midclt call filesystem.listdir "/mnt/tank/data"
midclt call filesystem.listdir "/mnt/tank/data" '[["name", "^", "test"]]'
midclt call filesystem.listdir "/mnt/tank/data" '[]' '{"limit": 100, "offset": 0}'
```

Returns array of entries with:
- `name` - Entry name
- `path` - Full path
- `realpath` - Resolved symlink path
- `type` - FILE, DIRECTORY, SYMLINK, OTHER
- `size` - Size in bytes
- `mode` - Unix permissions
- `acl` - ACL enabled
- `uid` - Owner user ID
- `gid` - Owner group ID
- `is_mountpoint` - Mountpoint flag
- `is_ctldir` - Control directory flag

### filesystem.file_tail_follow
Follow a file like `tail -f`.
```bash
midclt call filesystem.file_tail_follow "/var/log/messages" 100
```

## Permissions

### filesystem.setperm
Set Unix permissions on a file/directory.
```bash
midclt call filesystem.setperm '{
  "path": "/mnt/tank/data",
  "mode": "755",
  "uid": 1000,
  "gid": 1000
}'
```

With recursive option:
```bash
midclt call filesystem.setperm '{
  "path": "/mnt/tank/data",
  "mode": "755",
  "uid": 1000,
  "gid": 1000,
  "options": {
    "recursive": true,
    "traverse": false
  }
}'
```

<!-- Source: internal/resources/host_path.go:275-285 -->
**Request Schema:**

| Field | Type | Description |
|-------|------|-------------|
| path | string | Target file/directory path |
| mode | string | Unix permission mode (e.g., "755") |
| uid | integer | Owner user ID |
| gid | integer | Owner group ID |
| options.stripacl | boolean | Remove ACL entries |
| options.recursive | boolean | Apply to all children |
| options.traverse | boolean | Cross filesystem boundaries |

**Note:** When `options.recursive` is true, this becomes a job-based operation. Use `midclt -j` or poll `core.get_jobs` for completion.

## ACL Operations

### filesystem.getacl
Get ACL for a path.
```bash
midclt call filesystem.getacl "/mnt/tank/data"
midclt call filesystem.getacl "/mnt/tank/data" true  # Simplified format
```

Returns:
- `path` - File path
- `trivial` - Whether ACL is trivial (basic Unix perms only)
- `acltype` - NFS4 or POSIX1E
- `uid` - Owner user ID
- `gid` - Owner group ID
- `acl` - Array of ACL entries

NFS4 ACL entry structure:
- `tag` - owner@, group@, everyone@, USER, GROUP
- `id` - User/group ID (for USER/GROUP tags)
- `who` - User/group name
- `type` - ALLOW or DENY
- `perms` - Permission object (READ_DATA, WRITE_DATA, etc.)
- `flags` - Inheritance flags (FILE_INHERIT, DIRECTORY_INHERIT, etc.)

POSIX ACL entry structure:
- `tag` - USER_OBJ, GROUP_OBJ, OTHER, USER, GROUP, MASK
- `id` - User/group ID
- `who` - User/group name
- `perms` - Permission object (READ, WRITE, EXECUTE)
- `default` - Whether this is a default ACL entry

### filesystem.setacl
Set ACL on a path.

NFS4 ACL example:
```bash
midclt call filesystem.setacl '{
  "path": "/mnt/tank/data",
  "dacl": [
    {
      "tag": "owner@",
      "type": "ALLOW",
      "perms": {"BASIC": "FULL_CONTROL"},
      "flags": {"BASIC": "INHERIT"}
    },
    {
      "tag": "group@",
      "type": "ALLOW",
      "perms": {"BASIC": "MODIFY"},
      "flags": {"BASIC": "INHERIT"}
    },
    {
      "tag": "everyone@",
      "type": "ALLOW",
      "perms": {"BASIC": "READ"},
      "flags": {"BASIC": "INHERIT"}
    }
  ],
  "acltype": "NFS4",
  "options": {
    "recursive": true,
    "traverse": false
  }
}'
```

POSIX ACL example:
```bash
midclt call filesystem.setacl '{
  "path": "/mnt/tank/data",
  "dacl": [
    {"tag": "USER_OBJ", "perms": {"READ": true, "WRITE": true, "EXECUTE": true}},
    {"tag": "GROUP_OBJ", "perms": {"READ": true, "WRITE": false, "EXECUTE": true}},
    {"tag": "OTHER", "perms": {"READ": true, "WRITE": false, "EXECUTE": false}},
    {"tag": "USER", "id": 1000, "perms": {"READ": true, "WRITE": true, "EXECUTE": true}},
    {"tag": "MASK", "perms": {"READ": true, "WRITE": true, "EXECUTE": true}}
  ],
  "acltype": "POSIX1E"
}'
```

Options:
- `recursive` - Apply to all children
- `traverse` - Cross filesystem boundaries
- `canonicalize` - Reorder ACL entries

## ACL Templates

### filesystem.acltemplate.by_path
Get ACL templates available for a path.
```bash
midclt call filesystem.acltemplate.by_path "/mnt/tank/data"
midclt call filesystem.acltemplate.by_path "/mnt/tank/data" "NFS4"
```

### filesystem.acltemplate.create
Create a custom ACL template.
```bash
midclt call filesystem.acltemplate.create '{
  "name": "Custom Template",
  "acltype": "NFS4",
  "acl": [
    {"tag": "owner@", "type": "ALLOW", "perms": {"BASIC": "FULL_CONTROL"}, "flags": {"BASIC": "INHERIT"}}
  ]
}'
```

### filesystem.acltemplate.delete
Delete an ACL template.
```bash
midclt call filesystem.acltemplate.delete <template_id>
```

## NFS4 ACL Permissions

Basic permission sets:
- `FULL_CONTROL` - All permissions
- `MODIFY` - Read, write, execute, delete
- `READ` - Read and execute
- `TRAVERSE` - Execute only

Individual permissions:
- `READ_DATA` / `LIST_DIRECTORY`
- `WRITE_DATA` / `ADD_FILE`
- `APPEND_DATA` / `ADD_SUBDIRECTORY`
- `READ_NAMED_ATTRS`
- `WRITE_NAMED_ATTRS`
- `EXECUTE`
- `DELETE_CHILD`
- `READ_ATTRIBUTES`
- `WRITE_ATTRIBUTES`
- `DELETE`
- `READ_ACL`
- `WRITE_ACL`
- `WRITE_OWNER`
- `SYNCHRONIZE`

## NFS4 ACL Flags

Basic flag sets:
- `INHERIT` - FILE_INHERIT, DIRECTORY_INHERIT
- `NOINHERIT` - No inheritance flags

Individual flags:
- `FILE_INHERIT` - ACE inherited by files
- `DIRECTORY_INHERIT` - ACE inherited by directories
- `NO_PROPAGATE_INHERIT` - Don't propagate to subdirectories
- `INHERIT_ONLY` - ACE not applied to this object
- `INHERITED` - ACE was inherited

## POSIX ACL Permissions

- `READ` - Read permission
- `WRITE` - Write permission
- `EXECUTE` - Execute permission
