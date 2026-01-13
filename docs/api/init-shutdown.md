# Init/Shutdown Scripts API

Scripts that run at system startup or shutdown.

## Script Operations

### initshutdownscript.query
Query init/shutdown scripts.
```bash
midclt call initshutdownscript.query
```

Returns:
- `id` - Script ID
- `type` - COMMAND or SCRIPT
- `command` - Command to run (if type=COMMAND)
- `script` - Script content (if type=SCRIPT)
- `when` - PREINIT, POSTINIT, SHUTDOWN
- `enabled` - Script enabled
- `timeout` - Execution timeout
- `comment` - Description

### initshutdownscript.create
Create an init/shutdown script.

Command at startup:
```bash
midclt call initshutdownscript.create '{
  "type": "COMMAND",
  "command": "/usr/local/bin/startup-task.sh",
  "when": "POSTINIT",
  "enabled": true,
  "timeout": 60,
  "comment": "Run startup task"
}'
```

Script content at shutdown:
```bash
midclt call initshutdownscript.create '{
  "type": "SCRIPT",
  "script": "#!/bin/bash\necho \"Shutting down\" >> /var/log/shutdown.log",
  "when": "SHUTDOWN",
  "enabled": true,
  "timeout": 30,
  "comment": "Log shutdown"
}'
```

Early initialization script:
```bash
midclt call initshutdownscript.create '{
  "type": "COMMAND",
  "command": "/usr/local/bin/early-init.sh",
  "when": "PREINIT",
  "enabled": true,
  "timeout": 120,
  "comment": "Early initialization"
}'
```

### initshutdownscript.update
Update an init/shutdown script.
```bash
midclt call initshutdownscript.update <script_id> '{
  "enabled": false
}'
```

### initshutdownscript.delete
Delete an init/shutdown script.
```bash
midclt call initshutdownscript.delete <script_id>
```

## Script Types

| Type | Description |
|------|-------------|
| `COMMAND` | Execute a command/path |
| `SCRIPT` | Execute inline script content |

## Execution Timing

| When | Description |
|------|-------------|
| `PREINIT` | Before services start |
| `POSTINIT` | After services start |
| `SHUTDOWN` | During system shutdown |

## Best Practices

1. **Use absolute paths** for commands
2. **Set appropriate timeouts** to prevent boot delays
3. **Test scripts manually** before enabling
4. **Keep PREINIT scripts fast** - they delay service startup
5. **Log output** for debugging:
   ```bash
   /path/to/script.sh >> /var/log/init-script.log 2>&1
   ```
