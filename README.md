# Terraform Provider for TrueNAS

[![Terraform Registry](https://img.shields.io/badge/terraform-registry-blueviolet?logo=terraform)](https://registry.terraform.io/providers/deevus/truenas/latest)
[![Terraform Provider Downloads](https://img.shields.io/terraform/provider/downloads/deevus/truenas)](https://registry.terraform.io/providers/deevus/truenas/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![GitHub release](https://img.shields.io/github/v/release/deevus/terraform-provider-truenas)](https://github.com/deevus/terraform-provider-truenas/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/deevus/terraform-provider-truenas)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/deevus/terraform-provider-truenas)](https://goreportcard.com/report/github.com/deevus/terraform-provider-truenas)
[![Commercial Support](https://img.shields.io/badge/support-available-brightgreen)](#support)

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

## Support

Need help deploying terraform-provider-truenas at your org? I offer implementation support, custom development, and training through my consultancy: [simonhartcher.com](https://simonhartcher.com). Email in bio.

## License

MIT License - see [LICENSE](LICENSE) for details.
