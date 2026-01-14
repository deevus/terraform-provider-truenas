# ReadFile: Replace SFTP with sudo cat

## Problem

`ReadFile()` uses SFTP which operates as the SSH user. When files are owned by other users (e.g., app containers), reading fails with permission denied:

```
Error: Unable to read file "/mnt/storage/apps/thelounge/tailscale/ts-serve.json":
permission denied
```

## Solution

Replace SFTP-based file reading with `sudo cat` using a new `runSudoOutput` helper.

## Design Decisions

1. **New `runSudoOutput` method** - Separate from `runSudo` to keep clear separation between fire-and-forget commands and commands needing output
2. **Simple error wrapping** - No parsing of cat error messages; Terraform resource already checks `FileExists` first
3. **Direct `cat` usage** - No base64 encoding; config files are text and SSH handles binary stdout fine
4. **Leave SFTP in place** - Still used by `ReadDir` and `host_path.go`; cleanup is a separate concern

## Implementation

### 1. Add `runSudoOutput` to `internal/client/ssh.go`

```go
// runSudoOutput executes a command with sudo via SSH and returns stdout.
func (c *SSHClient) runSudoOutput(ctx context.Context, args ...string) ([]byte, error) {
    release := c.acquireSession()
    defer release()

    if c.clientWrapper == nil {
        if err := c.connect(); err != nil {
            return nil, err
        }
    }

    var escaped []string
    for _, arg := range args {
        escaped = append(escaped, shellescape.Quote(arg))
    }
    cmd := "sudo " + strings.Join(escaped, " ")

    session, err := c.clientWrapper.NewSession()
    if err != nil {
        return nil, err
    }
    defer session.Close()

    output, err := session.Output(cmd)
    if err != nil {
        return nil, err
    }
    return output, nil
}
```

### 2. Update `ReadFile` in `internal/client/ssh.go`

```go
// ReadFile reads the content of a file from the remote system.
func (c *SSHClient) ReadFile(ctx context.Context, path string) ([]byte, error) {
    output, err := c.runSudoOutput(ctx, "cat", path)
    if err != nil {
        return nil, fmt.Errorf("failed to read file %q: %w", path, err)
    }
    return output, nil
}
```

### 3. Update tests in `internal/client/sftp_test.go`

- Update `TestSSHClient_ReadFile_Success` to mock SSH session instead of SFTP
- Update `TestSSHClient_ReadFile_NotFound` to test error wrapping
- Remove `TestSSHClient_ReadFile_PartialReads` (no longer relevant)

## Files to Modify

1. `internal/client/ssh.go` - add `runSudoOutput`, update `ReadFile`
2. `internal/client/sftp_test.go` - update `ReadFile` tests

## Verification

1. Run `mise run test` to verify all tests pass
2. Test against real TrueNAS with non-root SSH user
3. Verify drift detection works for files owned by app containers
