# SSH Session Semaphore Design

**Date:** 2026-01-14
**Status:** Approved

## Problem

When Terraform runs multiple resources in parallel, each file operation (stat, mkdir, write, etc.) creates a new SSH session/channel. SSH servers have channel limits (typically ~100 per connection). When limits are hit, the server rejects new channels with `ssh: rejected: connect failed (open failed)`.

## Solution

Add a counting semaphore to `SSHClient` to limit concurrent SSH operations.

## Design Decisions

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| Mechanism | Buffered channel | Idiomatic Go, no external dependencies |
| Default limit | 10 concurrent sessions | Conservative; leaves headroom for server limits |
| Configuration | `max_ssh_sessions` provider attribute | Discoverable, documented in schema |
| Blocking behavior | Block and wait indefinitely | Matches Terraform's expectation that operations complete |
| Scope | Shared semaphore for SSH and SFTP | Both use same underlying connection/channels |

## Implementation

### SSHConfig Changes

```go
type SSHConfig struct {
    Host               string
    Port               int
    User               string
    PrivateKey         string
    HostKeyFingerprint string
    MaxSessions        int  // NEW: 0 means use default (10)
}
```

### SSHClient Changes

```go
type SSHClient struct {
    config        *SSHConfig
    client        *ssh.Client
    clientWrapper sshClientWrapper
    sftpClient    sftpClient
    dialer        sshDialer
    mu            sync.Mutex
    sessionSem    chan struct{}  // NEW: limits concurrent operations
}
```

### Initialization

```go
func NewSSHClient(config *SSHConfig) (*SSHClient, error) {
    if err := config.Validate(); err != nil {
        return nil, err
    }

    maxSessions := config.MaxSessions
    if maxSessions <= 0 {
        maxSessions = 10  // default
    }

    return &SSHClient{
        config:     config,
        dialer:     &defaultDialer{},
        sessionSem: make(chan struct{}, maxSessions),
    }, nil
}
```

### Acquire/Release Pattern

```go
// acquireSession blocks until a session slot is available.
func (c *SSHClient) acquireSession() func() {
    c.sessionSem <- struct{}{}
    return func() {
        <-c.sessionSem
    }
}
```

### Usage Pattern

```go
func (c *SSHClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
    release := c.acquireSession()
    defer release()

    // ... existing connection and session logic
}
```

## Methods Requiring Modification

Methods that create SSH sessions directly:

| Method | Reason |
|--------|--------|
| `Call()` | Creates session via `NewSession()` |
| `CallAndWait()` | Creates session via `NewSession()` |
| `runSudo()` | Creates session via `NewSession()` |
| `ReadFile()` | Uses SFTP (shared semaphore) |

Methods that don't need changes (they call methods that already acquire):

- `WriteFile()` - calls `Call()`
- `FileExists()` - calls `Call()`
- `MkdirAll()` - calls `Call()`
- `Chown()` - calls `CallAndWait()`
- `ChmodRecursive()` - calls `CallAndWait()`
- `DeleteFile()` - calls `runSudo()`
- `RemoveDir()` - calls `runSudo()`
- `RemoveAll()` - calls `runSudo()`

## Provider Schema

```go
"max_ssh_sessions": schema.Int64Attribute{
    Optional:    true,
    Description: "Maximum concurrent SSH sessions. Defaults to 10. Increase if you have many resources and a capable server, decrease if you see connection errors.",
},
```

## Testing

Semaphore behavior is directly testable via buffer length:

```go
client := &SSHClient{
    sessionSem:    make(chan struct{}, 2),
    clientWrapper: mockClient,
}

assert.Equal(t, 0, len(client.sessionSem))  // all slots free

release := client.acquireSession()
assert.Equal(t, 1, len(client.sessionSem))  // one slot taken

release()
assert.Equal(t, 0, len(client.sessionSem))  // released
```

Concurrent behavior can be tested by tracking max concurrent active operations.

## Files to Modify

1. `internal/client/ssh.go` - Add semaphore field, `acquireSession()`, modify 4 methods
2. `internal/provider/provider.go` - Add schema attribute, pass to config
3. `internal/client/ssh_test.go` - Add semaphore behavior tests

## Not Changing

- No external dependencies
- No changes to public interfaces
- No changes to resource implementations
