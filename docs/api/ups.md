# UPS API

Uninterruptible Power Supply monitoring and management.

## UPS Configuration

### ups.config
Get UPS configuration.
```bash
midclt call ups.config
```

Returns:
- `id` - Config ID
- `mode` - MASTER or SLAVE
- `identifier` - UPS identifier
- `remotehost` - Remote NUT host (slave mode)
- `remoteport` - Remote NUT port
- `driver` - UPS driver
- `port` - UPS port/device
- `options` - Driver options
- `optionsupsd` - upsd options
- `description` - Description
- `shutdown` - Shutdown mode
- `shutdowntimer` - Shutdown timer (seconds)
- `shutdowncmd` - Shutdown command
- `nocommwarntime` - No comm warning time
- `monpwd` - Monitor password
- `monuser` - Monitor user
- `powerdown` - Power down UPS
- `hostsync` - Host sync time

### ups.update

Update UPS configuration. This is a singleton config and does not require an ID parameter.

Master mode (direct USB connection):
```bash
midclt call ups.update '{
  "mode": "MASTER",
  "identifier": "ups",
  "driver": "usbhid-ups",
  "port": "auto",
  "shutdown": "BATT",
  "shutdowntimer": 30,
  "monuser": "upsmon",
  "monpwd": "secret"
}'
```

Master mode (serial connection):
```bash
midclt call ups.update '{
  "mode": "MASTER",
  "identifier": "ups",
  "driver": "apcsmart",
  "port": "/dev/ttyS0",
  "shutdown": "LOWBATT"
}'
```

Slave mode (remote NUT server):
```bash
midclt call ups.update '{
  "mode": "SLAVE",
  "identifier": "ups",
  "remotehost": "192.168.1.10",
  "remoteport": 3493,
  "shutdown": "BATT",
  "shutdowntimer": 30
}'
```

With custom shutdown command:
```bash
midclt call ups.update '{
  "shutdown": "BATT",
  "shutdowntimer": 60,
  "shutdowncmd": "/sbin/shutdown -h +1"
}'
```

### ups.driver_choices
Get available UPS drivers.
```bash
midclt call ups.driver_choices
```

### ups.port_choices
Get available UPS ports.
```bash
midclt call ups.port_choices
```

## Service Control

```bash
midclt call service.update "ups" '{"enable": true}'
midclt call service.control "ups" "START"
```

## Modes

| Mode | Description |
|------|-------------|
| `MASTER` | Direct UPS connection |
| `SLAVE` | Remote NUT server |

## Shutdown Modes

| Mode | Description |
|------|-------------|
| `LOWBATT` | Shutdown on low battery |
| `BATT` | Shutdown on battery after timer |

## Common Drivers

| Driver | Description |
|--------|-------------|
| `usbhid-ups` | USB HID UPS (most common) |
| `apcsmart` | APC Smart-UPS serial |
| `blazer_usb` | Megatec/Q1 USB |
| `blazer_ser` | Megatec/Q1 serial |
| `snmp-ups` | SNMP network UPS |
| `nutdrv_qx` | Generic Q* protocol |
| `tripplite_usb` | Tripp Lite USB |
| `cyberpower` | CyberPower |

## Driver Options

Common driver options (in `options` field):
```bash
midclt call ups.update '{
  "options": "pollinterval = 5\noffdelay = 30\nondelay = 0"
}'
```

| Option | Description |
|--------|-------------|
| `pollinterval` | Polling interval (seconds) |
| `offdelay` | Delay before UPS powers off |
| `ondelay` | Delay before UPS powers on |
| `lowbatt` | Low battery threshold |
| `battery_voltage_high` | Full battery voltage |
| `battery_voltage_low` | Empty battery voltage |

## Complete UPS Setup Example

```bash
# 1. Configure UPS
midclt call ups.update '{
  "mode": "MASTER",
  "identifier": "myups",
  "driver": "usbhid-ups",
  "port": "auto",
  "shutdown": "BATT",
  "shutdowntimer": 120,
  "monuser": "admin",
  "monpwd": "secret",
  "powerdown": true
}'

# 2. Enable and start service
midclt call service.update "ups" '{"enable": true}'
midclt call service.control "ups" "START"

# 3. Optionally configure alert for UPS events
midclt call alertservice.create '{
  "name": "UPS Email",
  "type": "Mail",
  "attributes": {"email": "admin@example.com"},
  "level": "WARNING",
  "enabled": true
}'
```
