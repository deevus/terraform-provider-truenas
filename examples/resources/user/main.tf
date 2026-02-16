# Create a basic user with an auto-created primary group
resource "truenas_user" "jdoe" {
  username     = "jdoe"
  full_name    = "John Doe"
  email        = "jdoe@example.com"
  password     = var.user_password
  group_create = true
  shell        = "/usr/bin/bash"
  smb          = false
}

# Create a user assigned to an existing group
resource "truenas_user" "deploy" {
  username          = "deploy"
  full_name         = "Deploy User"
  password_disabled = true
  group_id          = truenas_group.developers.gid
  sshpubkey         = file("~/.ssh/deploy.pub")
  shell             = "/usr/bin/bash"
  smb               = false
}
