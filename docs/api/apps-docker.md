# Applications & Docker API

Application deployment and Docker container management.

## Applications

### app.query
Query installed applications.
```bash
midclt call app.query
midclt call app.query '[[["name", "=", "plex"]]]'
midclt call app.query '[[["state", "=", "RUNNING"]]]'
```

Returns:
- `id` - App ID
- `name` - App name
- `state` - RUNNING, STOPPED, DEPLOYING
- `upgrade_available` - Boolean
- `human_version` - Human-readable version
- `version` - Semantic version
- `metadata` - App metadata
- `notes` - Release notes

### app.available
List available applications from catalog.
```bash
midclt call app.available
midclt call app.available '{"category": "media"}'
```

### app.available_space
Check available space for apps.
```bash
midclt call app.available_space
```

### app.categories
Get available app categories.
```bash
midclt call app.categories
```

### app.similar
Find similar applications.
```bash
midclt call app.similar "plex"
```

### app.latest
Get latest apps from catalog.
```bash
midclt call app.latest
```

### app.create
Install an application.
```bash
midclt call app.create '{
  "app_name": "plex",
  "catalog_app": "plex",
  "train": "stable",
  "values": {
    "network": {"host_network": true},
    "resources": {"limits": {"cpu": "4", "memory": "4Gi"}}
  }
}'
```

### app.update
Update application configuration.
```bash
midclt call app.update "plex" '{"values": {"network": {"host_network": false}}}'
```

### app.upgrade
Upgrade an application.
```bash
midclt call app.upgrade "plex"
midclt call app.upgrade "plex" '{"app_version": "1.2.3"}'
```

### app.upgrade_summary
Get upgrade summary for an app.
```bash
midclt call app.upgrade_summary "plex"
```

### app.rollback
Rollback an application.
```bash
midclt call app.rollback "plex" "1.0.0"
```

### app.rollback_versions
Get available rollback versions.
```bash
midclt call app.rollback_versions "plex"
```

### app.start
Start an application.
```bash
midclt call app.start "plex"
```

### app.stop
Stop an application.
```bash
midclt call app.stop "plex"
```

### app.redeploy
Redeploy an application.
```bash
midclt call app.redeploy "plex"
```

### app.delete
Delete an application.
```bash
midclt call app.delete "plex"
```

### app.stats
Get application statistics.
```bash
midclt call app.stats "plex"
```

### app.convert_to_custom
Convert app to custom app.
```bash
midclt call app.convert_to_custom "plex"
```

## App IX Volumes

### app.ix_volume.exists
Check if IX volume exists.
```bash
midclt call app.ix_volume.exists "plex" "config"
```

## App Images

### app.image.query
Query application images.
```bash
midclt call app.image.query
```

### app.image.pull
Pull an image.
```bash
midclt call app.image.pull '{"image": "nginx:latest"}'
```

### app.image.delete
Delete an image.
```bash
midclt call app.image.delete "<image_id>"
```

### app.image.dockerhub_rate_limit
Check DockerHub rate limit.
```bash
midclt call app.image.dockerhub_rate_limit
```

## App Registry

### app.registry.query
Query configured registries.
```bash
midclt call app.registry.query
```

### app.registry.create
Add a registry.
```bash
midclt call app.registry.create '{
  "name": "ghcr",
  "uri": "ghcr.io",
  "username": "user",
  "password": "token"
}'
```

### app.registry.update
Update a registry.
```bash
midclt call app.registry.update <registry_id> '{"password": "newtoken"}'
```

### app.registry.delete
Delete a registry.
```bash
midclt call app.registry.delete <registry_id>
```

## Catalog

### catalog.config
Get catalog configuration.
```bash
midclt call catalog.config
```

### catalog.update
Update catalog configuration.
```bash
midclt call catalog.update '{"preferred_trains": ["stable", "enterprise"]}'
```

### catalog.sync
Sync catalog from remote.
```bash
midclt call catalog.sync
```

### catalog.trains
Get available catalog trains.
```bash
midclt call catalog.trains
```

### catalog.get_app_details
Get detailed app information.
```bash
midclt call catalog.get_app_details "plex" '{"train": "stable"}'
```

## Docker

### docker.config
Get Docker configuration.
```bash
midclt call docker.config
```

### docker.update
Update Docker configuration.
```bash
midclt call docker.update '{"pool": "tank", "nvidia": true}'
```

### docker.status
Get Docker service status.
```bash
midclt call docker.status
```

### docker.state
Get Docker state.
```bash
midclt call docker.state
```

## Containers (Legacy)

### container.query
Query containers.
```bash
midclt call container.query
```

### container.create
Create a container.
```bash
midclt call container.create '{
  "image": "nginx:latest",
  "name": "nginx-test"
}'
```

### container.start
Start a container.
```bash
midclt call container.start <container_id>
```

### container.stop
Stop a container.
```bash
midclt call container.stop <container_id>
```

### container.update
Update a container.
```bash
midclt call container.update <container_id> '{"restart_policy": "always"}'
```

### container.delete
Delete a container.
```bash
midclt call container.delete <container_id>
```

### container.get_instance
Get container instance details.
```bash
midclt call container.get_instance <container_id>
```

### container.metrics
Get container metrics.
```bash
midclt call container.metrics
```

### container.pool_choices
Get available pools for containers.
```bash
midclt call container.pool_choices
```

## Container Devices

### container.device.query
Query container devices.
```bash
midclt call container.device.query
```

### container.device.create
Create a container device.
```bash
midclt call container.device.create '{
  "container_id": <id>,
  "dtype": "GPU",
  "gpu": "0000:01:00.0"
}'
```

### container.device.update
Update a container device.
```bash
midclt call container.device.update <device_id> '{"gpu": "0000:02:00.0"}'
```

### container.device.delete
Delete a container device.
```bash
midclt call container.device.delete <device_id>
```

### Choice Methods
```bash
midclt call container.device.gpu_choices
midclt call container.device.nic_attach_choices
midclt call container.device.usb_choices
```

## LXC Configuration

### lxc.config
Get LXC configuration.
```bash
midclt call lxc.config
```

### lxc.update
Update LXC configuration.
```bash
midclt call lxc.update '{"pool": "tank"}'
```

### lxc.bridge_choices
Get LXC bridge choices.
```bash
midclt call lxc.bridge_choices
```
