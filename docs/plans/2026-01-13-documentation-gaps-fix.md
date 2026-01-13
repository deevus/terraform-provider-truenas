# Plan: Fix Documentation Gaps

## Problem

Documentation is out of sync with implementation after recent changes:
1. Host key verification added `host_key_fingerprint` (required) - not documented
2. File operations now use `sudo rm`/`rmdir` - docs only mention `midclt`
3. Several resource attributes undocumented

## Files to Update

### 1. `docs/index.md` - Provider Documentation

**Changes needed:**

#### A. Update Example Usage (lines 33-42)
Add `host_key_fingerprint` to SSH block:
```terraform
ssh {
  port                 = 22
  user                 = "root"
  private_key          = file("~/.ssh/truenas_ed25519")
  host_key_fingerprint = "SHA256:..."  # Get with: ssh-keygen -lf /etc/ssh/ssh_host_rsa_key.pub
}
```

#### B. Update Second Example (lines 99-107)
Same change - add `host_key_fingerprint`.

#### C. Update Schema Section (lines 57-68)
Add to "Required" under nested ssh schema:
```
- `host_key_fingerprint` (String) SHA256 fingerprint of the server's SSH host key for MITM protection.
```

#### D. Update Sudo Commands (line 90)
Change from:
```
Allowed sudo commands with no password: /usr/bin/midclt
```
To:
```
Allowed sudo commands with no password: /usr/bin/midclt, /bin/rm, /bin/rmdir
```

#### E. Add Host Key Section
After "Requirements" section, add guidance on obtaining fingerprint:
```markdown
## Getting the Host Key Fingerprint

To obtain your TrueNAS server's SSH host key fingerprint:

```bash
ssh-keygen -lf /etc/ssh/ssh_host_rsa_key.pub
```

This outputs something like:
```
3072 SHA256:xyzABC123... root@truenas (RSA)
```

Copy the `SHA256:...` portion into your provider configuration.
```

---

### 2. `README.md` - Project Readme

**Changes needed:**

#### A. Update Provider Example
Add `host_key_fingerprint` to the example SSH block (around line 21-29).

---

### 3. `docs/resources/file.md` - File Resource

**Changes needed:**

#### A. Add `force_destroy` to Optional Schema
```markdown
- `force_destroy` (Boolean) Change file ownership to root before deletion to handle permission issues from app containers.
```

---

### 4. `docs/resources/host_path.md` - Host Path Resource

**Changes needed:**

#### A. Add Deprecation Notice at Top
```markdown
~> **Deprecated:** Use `truenas_dataset` with nested datasets instead. `host_path` relies on SFTP which may not work with non-root SSH users. Datasets are created via the TrueNAS API and provide better ZFS integration.
```

#### B. Add `force_destroy` to Optional Schema
```markdown
- `force_destroy` (Boolean) Force deletion of non-empty directories (recursive delete).
```

---

### 5. `docs/resources/dataset.md` - Dataset Resource

**Changes needed:**

#### A. Add Missing Attributes to Optional Schema
```markdown
- `mode` (String) Unix mode for dataset mountpoint (e.g., "0755").
- `uid` (Number) Owner user ID for dataset mountpoint.
- `gid` (Number) Owner group ID for dataset mountpoint.
- `force_destroy` (Boolean) When destroying, also delete all child datasets. Defaults to false.
```

#### B. Add `full_path` to Computed/Read-Only
```markdown
- `full_path` (String) Full filesystem path to the dataset mountpoint.
```

#### C. Add Deprecation Note for `mount_path`
```markdown
- `mount_path` (String, Deprecated) Use `full_path` instead.
```

---

## Task Summary

| File | Priority | Changes |
|------|----------|---------|
| `docs/index.md` | Critical | Examples, schema, sudo commands, fingerprint guide |
| `README.md` | Critical | Example with fingerprint |
| `docs/resources/file.md` | Moderate | `force_destroy` attribute |
| `docs/resources/host_path.md` | Moderate | Deprecation notice, `force_destroy` |
| `docs/resources/dataset.md` | Moderate | Multiple missing attributes |

## Verification

1. All provider examples include `host_key_fingerprint`
2. Sudo requirements list all three commands
3. `terraform providers schema -json` output matches documented schema
4. No broken markdown links
5. Examples are syntactically valid Terraform

## Notes

- Documentation uses tfplugindocs-generated schema sections - may need regeneration
- Consider running `go generate` if schema docs are auto-generated
