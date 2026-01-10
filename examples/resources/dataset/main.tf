# Create a new ZFS dataset for application data
resource "truenas_dataset" "app_data" {
  pool        = "tank"
  path        = "apps/myapp"
  compression = "lz4"
  quota       = "100G"
  atime       = "off"
}

# Create a nested dataset using parent reference
resource "truenas_dataset" "app_logs" {
  parent = truenas_dataset.app_data.id
  name   = "logs"
}
