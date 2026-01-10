# Retrieve information about an existing ZFS pool
data "truenas_pool" "main" {
  name = "tank"
}

output "pool_path" {
  value = data.truenas_pool.main.path
}

output "pool_status" {
  value = data.truenas_pool.main.status
}

output "pool_available_bytes" {
  value = data.truenas_pool.main.available_bytes
}
