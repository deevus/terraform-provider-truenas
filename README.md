# Terraform Provider for TrueNAS

A Terraform provider for managing TrueNAS SCALE and Community editions.

## Installation

```hcl
terraform {
  required_providers {
    truenas = {
      source  = "deevus/truenas"
      version = "~> 0.1"
    }
  }
}
```

## Usage

```hcl
provider "truenas" {
  host        = "192.168.1.100"
  auth_method = "ssh"

  ssh {
    user                 = "terraform"
    private_key          = file("~/.ssh/terraform_ed25519")
    host_key_fingerprint = "SHA256:..."  # ssh-keyscan <host> | ssh-keygen -lf -
  }
}

# Create a dataset
resource "truenas_dataset" "example" {
  pool = "tank"
  name = "example"
}
```

## Features

- **Data Sources**: Query pools and datasets
- **Resources**: Manage datasets, host paths, files, and applications

## Documentation

Full documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/deevus/truenas/latest/docs).

## Requirements

- TrueNAS SCALE or TrueNAS Community
- SSH access with a user configured for `midclt`, `rm`, and `rmdir` (see [User Setup](https://registry.terraform.io/providers/deevus/truenas/latest/docs#truenas-user-setup))

## License

MIT License - see [LICENSE](LICENSE) for details.
