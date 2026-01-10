terraform {
  required_providers {
    truenas = {
      source  = "local/truenas"
      version = "~> 0.1"
    }
  }
}

provider "truenas" {
  host        = "192.168.1.100"
  auth_method = "ssh"

  ssh {
    port        = 22
    user        = "root"
    private_key = file("~/.ssh/truenas_ed25519")
  }
}
