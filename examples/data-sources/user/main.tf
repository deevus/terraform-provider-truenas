# Look up the root user
data "truenas_user" "root" {
  username = "root"
}

output "root_uid" {
  value = data.truenas_user.root.uid
}

output "root_home" {
  value = data.truenas_user.root.home
}

output "root_group_id" {
  value = data.truenas_user.root.group_id
}
