# Create a host path directory for app storage
resource "truenas_host_path" "myapp_config" {
  path = "/mnt/tank/apps/myapp/config"
  mode = "755"
  uid  = 1000
  gid  = 1000
}

# Create another host path for data
resource "truenas_host_path" "myapp_data" {
  path = "/mnt/tank/apps/myapp/data"
  mode = "750"
  uid  = 1000
  gid  = 1000
}
