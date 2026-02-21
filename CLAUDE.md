# Terraform Provider TrueNAS

This repository contains a Terraform provider for TrueNAS SCALE and Community editions. It uses the `midclt` command to communicate with the TrueNAS API.

## Development workflow

1. Development tasks are conducted using `mise`. Run `mise tasks` to see what tasks are available.
2. To explore the TrueNAS API:
   - `mr api-docs api_methods` - Browse available API methods
   - `mr api-docs api_methods_{namespace}` - Browse methods in a namespace (e.g., `api_methods_cloudsync`)
   - `mr api-docs {doc}` - View formatted documentation (uses lynx, add `-r` for raw RST)
   - `mr midclt-method {method}` - Get JSON schema for a specific method (better for implementation)

### Design and implementation plans

When asked to write an implementation plan, the context should include the current code coverage from `mise run coverage`. In the verification tasks, verify that the code coverage has either improved or maintained with the baseline. 

### Finishing a development branch

When finishing a development branch:

1. Make sure coverage is equal to or better than the baseline.
2. Clean up the docs/plans/ folder and commit.

## Ethos

- Always write idiomatic terraform.
- Strive for 100% code coverage where possible.

## Git Rules

- Never use `git -C` flag. Always `cd` to the working directory first or work from the current directory.

## Worktrees

- Feature development uses git worktrees in `.worktrees/` (already in .gitignore).
- Copy local Claude settings to new worktrees

### Using `tea` from a worktree

`tea` cannot auto-detect the repository when running from a worktree. Specify all parameters explicitly:

```bash
# Find your login name (use the NAME column)
tea login list

# Create PR with explicit parameters
tea pr create \
  --login <login-name> \
  --repo sh/terraform-provider-truenas \
  --head <branch-name> \
  --base main \
  --title "..." \
  --description "..."
```
