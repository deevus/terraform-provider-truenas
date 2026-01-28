# Terraform Provider TrueNAS

This repository contains a Terraform provider for TrueNAS SCALE and Community editions. It uses the `midclt` command to communicate with the TrueNAS API.

## Development workflow

1. Development tasks are conducted using `mise`. Run `mise tasks` to see what tasks are available.

### Design and implementation plans

When asked to write an implementation plan, the context should include the current code coverage from `mise run coverage`. In the verification tasks, verify that the code coverage has either improved or maintained with the baseline. 

### Finishing a development branch

When finishing a development branch:

1. Make sure coverage is equal to or better than the baseline.
2. Clean up the docs/plans/ folder and commit.

## Ethos

- Always write idiomatic terraform.
- Strive for 100% code coverage where possible.
