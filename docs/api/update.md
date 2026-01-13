# Update API

System update operations.

## Update Operations

### update.status
Get current update status.
```bash
midclt call update.status
```

Returns:
- `status` - AVAILABLE, UNAVAILABLE, REBOOT_REQUIRED
- `available` - Update available flag
- `changelog` - Change log
- `version` - Available version
- `filename` - Update filename

### update.config
Get update configuration.
```bash
midclt call update.config
```

Returns:
- `id` - Config ID
- `auto_check` - Auto-check enabled
- `train` - Update train

### update.update
Update the system.
```bash
midclt call update.update
midclt call update.update '{"reboot": true}'
```

With train change:
```bash
midclt call update.update '{"train": "TrueNAS-SCALE-Dragonfish", "reboot": false}'
```

### update.file
Update from file.
```bash
midclt call update.file "/path/to/update.file"
midclt call update.file "/path/to/update.file" '{"reboot": true}'
```

### update.run
Same as update.update, runs the update.
```bash
midclt call update.run
```

### update.profile_choices
Get available update profiles/trains.
```bash
midclt call update.profile_choices
```

## Update Trains

| Train | Description |
|-------|-------------|
| `TrueNAS-SCALE-Dragonfish` | Current stable |
| `TrueNAS-SCALE-ElectricEel` | Next stable |
| `TrueNAS-SCALE-Nightlies` | Development builds |

## Update Process

1. Check for updates:
```bash
midclt call update.status
```

2. Download and apply update:
```bash
midclt call update.update '{"reboot": false}'
```

3. Reboot to complete:
```bash
midclt call system.reboot
```

Or update and reboot in one step:
```bash
midclt call update.update '{"reboot": true}'
```

## Offline Update

For air-gapped systems:

1. Download update file from TrueNAS website
2. Transfer to TrueNAS system
3. Apply from file:
```bash
midclt call update.file "/mnt/tank/updates/TrueNAS-SCALE-24.04.2.2.update" '{"reboot": true}'
```
