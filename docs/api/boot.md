# Boot Environment API

Boot pool and boot environment management.

## Boot Pool Operations

### boot.get_state
Get boot pool state.
```bash
midclt call boot.get_state
```

Returns ZFS pool state information for the boot pool.

### boot.attach
Attach a disk to the boot pool (mirror).
```bash
midclt call boot.attach "sda" '{"expand": true}'
```

### boot.detach
Detach a disk from the boot pool.
```bash
midclt call boot.detach "sda"
```

### boot.replace
Replace a disk in the boot pool.
```bash
midclt call boot.replace "sda" "sdb"
```

### boot.set_scrub_interval
Set boot pool scrub interval.
```bash
midclt call boot.set_scrub_interval 7  # days
```

## Boot Environments

### boot.environment.query
Query boot environments.
```bash
midclt call boot.environment.query
```

Returns:
- `id` - Boot environment name
- `realname` - Real dataset name
- `name` - Display name
- `active` - Active boot environment
- `activated` - Will be active on next boot
- `can_activate` - Can be activated
- `created` - Creation timestamp
- `used` - Space used
- `rawspace` - Raw space
- `keep` - Prevent pruning

### boot.environment.activate
Activate a boot environment for next boot.
```bash
midclt call boot.environment.activate "24.04.2"
```

### boot.environment.clone
Clone a boot environment.
```bash
midclt call boot.environment.clone "24.04.2" '{"target": "24.04.2-backup"}'
```

### boot.environment.destroy
Delete a boot environment.
```bash
midclt call boot.environment.destroy "old-environment"
```

### boot.environment.keep
Set keep flag to prevent pruning.
```bash
midclt call boot.environment.keep "important-env" '{"keep": true}'
```

Remove keep flag:
```bash
midclt call boot.environment.keep "important-env" '{"keep": false}'
```

## Boot Environment Management

Boot environments allow you to:
- Roll back to previous system states
- Test updates before committing
- Keep known-good configurations

### Before Update
```bash
# Clone current environment
midclt call boot.environment.clone "current" '{"target": "pre-update-backup"}'

# Mark as keep
midclt call boot.environment.keep "pre-update-backup" '{"keep": true}'

# Proceed with update
midclt call update.update
```

### Roll Back
```bash
# If update fails, activate previous environment
midclt call boot.environment.activate "pre-update-backup"

# Reboot
midclt call system.reboot
```

### Clean Up
```bash
# List boot environments
midclt call boot.environment.query

# Delete old ones (that aren't marked keep)
midclt call boot.environment.destroy "old-env-name"
```

## Boot Pool Best Practices

1. **Mirror the boot pool** - Always use at least 2 disks
2. **Use SSDs** - Faster boot and resilience
3. **Regular scrubs** - Set scrub interval to 7 days
4. **Keep boot environments** - Mark important ones with keep flag
5. **Prune old environments** - Remove unused boot environments
