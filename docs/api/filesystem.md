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

### filesystem.mkdir
Create a directory at the specified path.

```bash
# Create directory with default permissions (755)
midclt call filesystem.mkdir '{"path": "/mnt/tank/data/newdir"}'

# Create directory with specific permissions
midclt call filesystem.mkdir '{
  "path": "/mnt/tank/data/newdir",
  "options": {
    "mode": "700"
  }
}'
```

**Request Schema:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| path | string | Yes | - | Path for the new directory (min length: 1) |
| options.mode | string | No | `"755"` | Unix permission mode |
| options.raise_chmod_error | boolean | No | `true` | Whether to raise an error if chmod fails after mkdir |

**Notes:**
- Does not create parent directories (no `-p` equivalent)
- The `raise_chmod_error` option controls whether a failure to set permissions after creation raises an error

### filesystem.get
Job to get (read) file contents from the TrueNAS filesystem. Typically used indirectly via `core.download`.

```bash
# Direct call (returns job ID)
midclt call filesystem.get "/mnt/tank/data/file.txt"

# Via core.download (preferred for large files)
midclt call core.download '["filesystem.get", ["/etc/exports"], "exports.txt"]'
```

**Request Schema:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| path | string | Yes | Absolute path to file (min length: 1) |

**Notes:**
- This is a **job-based** method; returns a job ID
- WebUI uses it via `core.download` for certificate and CSR file exports
- For large files, prefer `core.download` which handles streaming

### filesystem.put
Job to write file contents to the TrueNAS filesystem.

```bash
# Write file content
midclt call filesystem.put "/mnt/tank/data/file.txt"

# With options
midclt call filesystem.put "/mnt/tank/data/file.txt" '{"append": true, "mode": 644}'
```

**Request Schema:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| path | string | Yes | - | Absolute path for the file (min length: 1) |
| options.append | boolean | No | `false` | Append to file instead of overwriting |
| options.mode | integer/null | No | `null` | Unix permission mode for the file |

**Notes:**
- This is a **job-based** method; returns a job ID
- WebUI uses it for ISO image uploads (VM wizard) and general file uploads
- File content is sent as the job's input payload (not as a parameter)

### filesystem.file_tail_follow

> **Note:** This is a WebSocket subscription method, not a standard API call.
> It does not appear in `core.get_methods` output.

Follow a file like `tail -f` (receives events as file grows).
```bash
midclt call filesystem.file_tail_follow "/var/log/messages" 100
```

## Ownership & Permissions

### filesystem.chown
Change owner or group of a file at the given path.

```bash
# Change owner by UID
midclt call filesystem.chown '{"path": "/mnt/tank/data", "uid": 1000}'

# Change owner by username
midclt call filesystem.chown '{"path": "/mnt/tank/data", "user": "admin"}'

# Change group
midclt call filesystem.chown '{"path": "/mnt/tank/data", "gid": 1000}'

# Change owner and group recursively
midclt call filesystem.chown '{
  "path": "/mnt/tank/data",
  "uid": 1000,
  "gid": 1000,
  "options": {
    "recursive": true,
    "traverse": false
  }
}'
```

**Request Schema:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| path | string | Yes | - | Target file/directory path (min length: 1) |
| uid | integer/null | No | `null` | Owner user ID (-1 to 2147483647) |
| user | string/null | No | `null` | Owner username (alternative to uid) |
| gid | integer/null | No | `null` | Owner group ID (-1 to 2147483647) |
| group | string/null | No | `null` | Owner group name (alternative to gid) |
| options.recursive | boolean | No | `false` | Apply ownership change to all children |
| options.traverse | boolean | No | `false` | If set, do not limit to single dataset/filesystem |

**Notes:**
- Can specify owner by either `uid` or `user` (not both needed)
- Can specify group by either `gid` or `group` (not both needed)
- Similar to `filesystem.setperm` but focused on ownership only (no mode change)
- When `recursive` is true, this becomes a job-based operation

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
| user | string | Owner username (alternative to uid) |
| gid | integer | Owner group ID |
| group | string | Owner group name (alternative to gid) |
| options.stripacl | boolean | Remove ACL entries |
| options.recursive | boolean | Apply to all children |
| options.traverse | boolean | Cross filesystem boundaries |

**Note:** When `options.recursive` is true, this becomes a job-based operation. Use `midclt -j` or poll `core.get_jobs` for completion.

### filesystem.can_access_as_user
Check if a specific user can access a path with given permissions.

```bash
# Check if user can read a path
midclt call filesystem.can_access_as_user "admin" "/mnt/tank/data" '{"read": true}'

# Check read+write+execute
midclt call filesystem.can_access_as_user "www" "/mnt/tank/data/uploads" '{
  "read": true,
  "write": true,
  "execute": false
}'
```

**Request Schema:**

| Parameter | Position | Type | Required | Default | Description |
|-----------|----------|------|----------|---------|-------------|
| username | 0 | string | No | - | Username to check access for |
| path | 1 | string | No | - | Path to check access on |
| permissions | 2 | object | No | `{}` | Permission checks to perform |
| permissions.read | - | boolean/null | No | `null` | Check read access |
| permissions.write | - | boolean/null | No | `null` | Check write access |
| permissions.execute | - | boolean/null | No | `null` | Check execute access |

**Notes:**
- Uses positional parameters (not a single object)
- Returns boolean indicating whether access is permitted
- Useful for validating permissions before performing operations as a specific user

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

## ZFS File Attributes

### filesystem.get_zfs_attributes
Get the current ZFS-specific file attributes/flags for a file at the given path.

```bash
midclt call filesystem.get_zfs_attributes "/mnt/tank/data/important.db"
```

**Request Schema:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| path | string | Yes | Absolute file path (min length: 1) |

**Response Schema:**

| Field | Type | Description |
|-------|------|-------------|
| readonly | boolean | READONLY MS-DOS attribute; file may not be written to |
| hidden | boolean | HIDDEN MS-DOS attribute; file hidden from SMB clients |
| system | boolean | SYSTEM MS-DOS attribute; presented to SMB clients |
| archive | boolean | ARCHIVE MS-DOS attribute; reset to true on modification |
| immutable | boolean | File may not be altered or deleted (STATX_ATTR_IMMUTABLE) |
| nounlink | boolean | File may be altered but not deleted |
| appendonly | boolean | File may only be opened with O_APPEND (STATX_ATTR_APPEND) |
| offline | boolean | OFFLINE MS-DOS attribute; presented to SMB clients |
| sparse | boolean | SPARSE MS-DOS attribute; presented to SMB clients |

### filesystem.set_zfs_attributes
Set special ZFS-related file flags on the specified path.

```bash
# Make file immutable
midclt call filesystem.set_zfs_attributes '{
  "path": "/mnt/tank/data/important.db",
  "zfs_file_attributes": {
    "immutable": true
  }
}'

# Set multiple attributes
midclt call filesystem.set_zfs_attributes '{
  "path": "/mnt/tank/data/config.json",
  "zfs_file_attributes": {
    "readonly": true,
    "nounlink": true
  }
}'

# Clear an attribute
midclt call filesystem.set_zfs_attributes '{
  "path": "/mnt/tank/data/important.db",
  "zfs_file_attributes": {
    "immutable": false
  }
}'
```

**Request Schema:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| path | string | Yes | - | Absolute file path (min length: 1) |
| zfs_file_attributes | object | Yes | - | Attributes to set (only specified attributes are changed) |
| zfs_file_attributes.readonly | boolean/null | No | `null` | READONLY MS-DOS attribute. File may not be written to (does not impact existing opens) |
| zfs_file_attributes.hidden | boolean/null | No | `null` | HIDDEN MS-DOS attribute. SMB HIDDEN flag, file hidden from SMB clients |
| zfs_file_attributes.system | boolean/null | No | `null` | SYSTEM MS-DOS attribute. Presented to SMB clients, no local filesystem impact |
| zfs_file_attributes.archive | boolean/null | No | `null` | ARCHIVE MS-DOS attribute. Reset to true whenever file is modified |
| zfs_file_attributes.immutable | boolean/null | No | `null` | File may not be altered or deleted. Shows as IMMUTABLE in `filesystem.stat` |
| zfs_file_attributes.nounlink | boolean/null | No | `null` | File may be altered but not deleted |
| zfs_file_attributes.appendonly | boolean/null | No | `null` | File may only be opened with O_APPEND flag. Shows as APPEND in `filesystem.stat` |
| zfs_file_attributes.offline | boolean/null | No | `null` | OFFLINE MS-DOS attribute. Presented to SMB clients, no local filesystem impact |
| zfs_file_attributes.sparse | boolean/null | No | `null` | SPARSE MS-DOS attribute. Presented to SMB clients, no local filesystem impact |

**Notes:**
- Only specified attributes are modified; `null` values leave the attribute unchanged
- `immutable`, `nounlink`, and `appendonly` have real filesystem effects
- `readonly`, `hidden`, `system`, `archive`, `offline`, `sparse` are MS-DOS/SMB attributes with no local filesystem impact (except `readonly` which prevents writes)

## ACL Templates

### filesystem.acltemplate.query
List and query all ACL templates.

```bash
# List all templates
midclt call filesystem.acltemplate.query

# Filter by name
midclt call filesystem.acltemplate.query '[["name", "=", "Custom Template"]]'

# With pagination
midclt call filesystem.acltemplate.query '[]' '{"limit": 10, "offset": 0}'
```

**Request Schema:**

| Parameter | Position | Type | Required | Default | Description |
|-----------|----------|------|----------|---------|-------------|
| filters | 0 | array | No | `[]` | Standard query filters |
| options | 1 | object | No | - | Standard query options |
| options.count | - | boolean | No | `false` | Return count instead of results |
| options.get | - | boolean | No | `false` | Return first matching result |
| options.limit | - | integer | No | `0` | Max results (0 = unlimited) |
| options.offset | - | integer | No | `0` | Starting offset for pagination |
| options.order_by | - | array | No | `[]` | Field names to sort by (prefix `-` for reverse) |
| options.select | - | array | No | `[]` | Fields to include in response |

### filesystem.acltemplate.get_instance
Get a specific ACL template by its ID.

```bash
midclt call filesystem.acltemplate.get_instance 1
```

**Request Schema:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | integer | Yes | Template ID |
| options | object | No | Standard query options (select, extend, etc.) |

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

### filesystem.acltemplate.update
Update an existing ACL template.

```bash
midclt call filesystem.acltemplate.update 1 '{
  "name": "Updated Template",
  "comment": "Modified for new requirements",
  "acl": [
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
    }
  ]
}'
```

**Request Schema:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | integer | Yes | Template ID to update |
| data.name | string | No | New template name |
| data.acltype | string | No | ACL type: `NFS4` or `POSIX1E` |
| data.acl | array | No | Updated ACL entries (NFS4 or POSIX1E format, same as `filesystem.setacl` dacl) |
| data.comment | string | No | Template description/comment |

**Notes:**
- Only specified fields are updated
- ACL entry format is identical to `filesystem.setacl` dacl entries (see ACL Operations section)
- Supports both NFS4 and POSIX1E ACL entry formats

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
