# Extract client/ Package to truenas-go Module

**Issue:** #44
**Date:** 2026-02-21

## Summary

Extract `internal/client/` into the `truenas-go` library as a `client/` subpackage (`github.com/deevus/truenas-go/client`). Then update the terraform provider to consume from the library, deleting both `internal/client/` and `internal/api/`.

## Design Decisions

- **client/ subpackage** — not root package. Keeps data types (root) separate from transport layer (client/).
- **Rewrite imports during extraction** — no temporary `replace` directives in truenas-go. Client files import `github.com/deevus/truenas-go` (root) for api types.
- **Tag v0.1.0** — provider depends on the tag directly, no `replace` directive needed.
- **Delete both internal/client/ and internal/api/** — fold the api import migration into this issue since it's a mechanical find-and-replace.

## Scope

### Files extracted to truenas-go/client/

- `client.go` — Client interface + MockClient
- `ssh.go` — SSHClient implementation
- `websocket.go` — WebSocketClient (channel-based async)
- `ratelimit.go` — RateLimitedClient wrapper
- `jobs.go` — Job/JobPoller for long-running operations
- `errors.go` — TrueNASError parsing and enrichment
- `jsonrpc.go` — JSON-RPC request/response types
- `retry.go` — Exponential backoff, retry classifiers
- `midclt.go` — SSH midclt command wrapper
- `logger.go` — Logger interface + NopLogger
- All corresponding `*_test.go` files

### Import changes in truenas-go

- `github.com/deevus/terraform-provider-truenas/internal/api` -> `github.com/deevus/truenas-go`
- `api.Version` -> `truenas.Version`, `api.ParseVersion` -> `truenas.ParseVersion`, etc.

### New dependencies in truenas-go go.mod

- `golang.org/x/crypto` (ssh)
- `github.com/gorilla/websocket`
- `golang.org/x/time/rate`
- `al.essio.dev/pkg/shellescape`
- `github.com/pkg/sftp`
- `github.com/kr/fs` (sftp transitive)

### Import changes in terraform-provider-truenas (~35 files)

- `internal/client` -> `github.com/deevus/truenas-go/client`
- `internal/api` -> `truenas "github.com/deevus/truenas-go"` (alias, then `api.` -> `truenas.`)

## Sequencing

### Phase 1: truenas-go repo

1. Copy `internal/client/` files into `client/` subpackage
2. Rewrite `internal/api` imports to `github.com/deevus/truenas-go`
3. Update `go.mod` with new dependencies
4. Run tests, ensure all pass
5. Tag `v0.1.0`, push

### Phase 2: terraform-provider-truenas repo

6. Delete `internal/client/` and `internal/api/`
7. Update imports across all ~35 files:
   - `internal/client` -> `github.com/deevus/truenas-go/client`
   - `internal/api` -> `truenas "github.com/deevus/truenas-go"`
8. Mechanical rename: `api.` -> `truenas.` in consuming code
9. `go get github.com/deevus/truenas-go@v0.1.0`
10. Run tests, ensure all pass

## Verification

- All existing tests pass in both repos
- No Terraform imports in truenas-go
- Coverage baseline maintained or improved:
  - client: 90.3%
  - api: 94.4%
  - datasources: 98.9%
  - provider: 89.5%
  - resources: 90.0%
  - types: 74.6%

## Not in Scope

- No behavioral changes to client code
- No renaming of types or methods (beyond import alias)
- No changes to resource/datasource logic
- Service layer creation (separate issues #45-#52)
