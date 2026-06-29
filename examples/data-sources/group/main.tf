# Look up the wheel group
data "truenas_group" "wheel" {
  name = "wheel"
}

output "wheel_gid" {
  value = data.truenas_group.wheel.gid
}

output "wheel_members" {
  value = data.truenas_group.wheel.users
}
