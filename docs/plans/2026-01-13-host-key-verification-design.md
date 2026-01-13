# Host Key Verification Design

## Problem

The SSH client uses `ssh.InsecureIgnoreHostKey()` which accepts any host key without verification. This makes connections vulnerable to man-in-the-middle attacks.

## Solution

Add required `host_key_fingerprint` configuration option. Users provide the SHA256 fingerprint of their TrueNAS server's SSH host key.

## Configuration

```hcl
provider "truenas" {
  host        = "truenas.local"
  auth_method = "ssh"

  ssh {
    private_key          = file("~/.ssh/id_ed25519")
    host_key_fingerprint = "SHA256:xyzABC123..."  # Required
  }
}
```

**Get the fingerprint on your TrueNAS server:**
```bash
ssh-keygen -lvf /etc/ssh/ssh_host_rsa_key.pub
```

## Implementation

### Schema Changes (provider.go)

Add `host_key_fingerprint` as required string in SSH block:

```go
"host_key_fingerprint": schema.StringAttribute{
    Description: "SHA256 fingerprint of the TrueNAS server's SSH host key. " +
        "Get it with: ssh-keygen -lvf /etc/ssh/ssh_host_rsa_key.pub",
    Required:  true,
    Sensitive: false,
},
```

### SSHConfig Changes (ssh.go)

Add field to struct:

```go
type SSHConfig struct {
    Host               string
    Port               int
    User               string
    PrivateKey         string
    HostKeyFingerprint string  // New
}
```

Update `Validate()` to require fingerprint:

```go
if c.HostKeyFingerprint == "" {
    return errors.New("host_key_fingerprint is required")
}
```

### Verification Function (ssh.go)

```go
// verifyHostKey creates a HostKeyCallback that validates against the configured fingerprint.
func verifyHostKey(expectedFingerprint string) ssh.HostKeyCallback {
    return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
        fingerprint := ssh.FingerprintSHA256(key)
        if fingerprint != expectedFingerprint {
            return fmt.Errorf(
                "host key verification failed for %s: expected %s, got %s",
                hostname, expectedFingerprint, fingerprint,
            )
        }
        return nil
    }
}
```

### Update connect() Function

Replace:
```go
HostKeyCallback: ssh.InsecureIgnoreHostKey(),
```

With:
```go
HostKeyCallback: verifyHostKey(c.config.HostKeyFingerprint),
```

## Error Handling

### New Error Constructor (errors.go)

```go
func NewHostKeyError(host string, expected, actual string) *TrueNASError {
    return &TrueNASError{
        Code:    "EHOSTKEY",
        Message: fmt.Sprintf("host key verification failed for %s: expected %s, got %s", host, expected, actual),
        Suggestion: "Verify the fingerprint on your TrueNAS server: ssh-keygen -lvf /etc/ssh/ssh_host_rsa_key.pub",
    }
}
```

### Missing Fingerprint Error

```
Error: Missing required argument

The argument "ssh.host_key_fingerprint" is required to prevent man-in-the-middle attacks.

Get your server's fingerprint:
  ssh-keygen -lvf /etc/ssh/ssh_host_rsa_key.pub

Then add to your provider configuration:
  ssh {
    host_key_fingerprint = "SHA256:..."
  }
```

### Verification Failure Error

```
Error: Host key verification failed

host key verification failed for truenas.local:22: expected SHA256:abc123..., got SHA256:xyz789...

This could indicate:
- The server's SSH key has changed (rotation, reinstall)
- A man-in-the-middle attack
- Incorrect fingerprint in configuration

Verify the fingerprint on your TrueNAS server:
  ssh-keygen -lvf /etc/ssh/ssh_host_rsa_key.pub
```

## Testing

### Unit Tests (ssh_test.go)

1. `TestVerifyHostKey_Match` - Valid fingerprint passes
2. `TestVerifyHostKey_Mismatch` - Wrong fingerprint returns error with both expected and actual values
3. `TestSSHConfig_Validate_MissingFingerprint` - Validation fails without fingerprint

### Test Approach

- Generate test key pair in test setup
- Use `ssh.FingerprintSHA256()` to compute expected fingerprint
- Mock SSH dialer to present test public key

## Breaking Change

This is a breaking change. Existing configurations without `host_key_fingerprint` will fail with a clear error message explaining how to get and add the fingerprint.

## Files to Modify

1. `internal/provider/provider.go` - Add schema attribute
2. `internal/client/ssh.go` - Add SSHConfig field, verifyHostKey function, update connect()
3. `internal/client/errors.go` - Add NewHostKeyError constructor
4. `internal/client/ssh_test.go` - Add verification tests
5. `internal/provider/provider_test.go` - Update tests with fingerprint
