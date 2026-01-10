# Retrieve information about an existing dataset
data "truenas_dataset" "apps" {
  pool = "tank"
  path = "apps"
}

output "dataset_mount_path" {
  value = data.truenas_dataset.apps.mount_path
}

output "dataset_compression" {
  value = data.truenas_dataset.apps.compression
}
