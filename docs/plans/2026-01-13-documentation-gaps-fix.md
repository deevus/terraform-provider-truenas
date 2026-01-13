# Plan: Fix Documentation Gaps

**Status:** Completed

## Problem

Documentation was out of sync with implementation after recent changes:
1. Host key verification added `host_key_fingerprint` (required) - not documented
2. File operations now use `sudo rm`/`rmdir` - docs only mentioned `midclt`
3. Several resource attributes undocumented

## Files Updated

| File | Changes |
|------|---------|
| `docs/index.md` | Examples, schema, sudo commands, fingerprint guide |
| `README.md` | Example with fingerprint |
| `internal/provider/provider.go` | Fix fingerprint command in schema description |
| `internal/client/errors.go` | Fix fingerprint command in error suggestion |
| `docs/resources/file.md` | `force_destroy` attribute |
| `docs/resources/host_path.md` | Deprecation notice, `force_destroy` |
| `docs/resources/dataset.md` | `mode`, `uid`, `gid`, `full_path`, `force_destroy`, deprecations |

## Key Changes

### SSH Host Key Fingerprint

Added to all provider examples:
```terraform
ssh {
  host_key_fingerprint = "SHA256:..."  # ssh-keyscan <host> | ssh-keygen -lf -
}
```

Added documentation section explaining how to obtain fingerprint:
```bash
ssh-keyscan -p 22 truenas.local 2>/dev/null | ssh-keygen -lf -
```

### Sudo Commands

Updated from `/usr/bin/midclt` to:
```
/usr/bin/midclt, /bin/rm, /bin/rmdir
```

### Resource Attributes

- `truenas_file`: Added `force_destroy`
- `truenas_host_path`: Added deprecation notice, `force_destroy`
- `truenas_dataset`: Added `mode`, `uid`, `gid`, `full_path`, `force_destroy`, deprecation notes for `name` and `mount_path`
