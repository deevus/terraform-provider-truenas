# Create a group for developers
resource "truenas_group" "developers" {
  name = "developers"
  smb  = false
}

# Create a group with a specific GID and sudo access
resource "truenas_group" "admins" {
  name = "admins"
  gid  = 5000
  smb  = false

  sudo_commands_nopasswd = ["ALL"]
}
