package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"al.essio.dev/pkg/shellescape"
	"github.com/deevus/terraform-provider-truenas/internal/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/crypto/ssh"
)

// ansiRegex matches ANSI escape sequences.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07`)

// SSHConfig holds configuration for SSH connection to TrueNAS.
type SSHConfig struct {
	Host               string
	Port               int
	User               string
	PrivateKey         string
	HostKeyFingerprint string
	MaxSessions        int // Maximum concurrent SSH sessions (0 = default of 5)
}

// Validate validates the SSHConfig and sets defaults.
func (c *SSHConfig) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.PrivateKey == "" {
		return errors.New("private_key is required")
	}
	if c.HostKeyFingerprint == "" {
		return errors.New("host_key_fingerprint is required")
	}

	// Set defaults
	if c.Port == 0 {
		c.Port = 22
	}
	if c.User == "" {
		c.User = "root"
	}

	return nil
}

// parsePrivateKey parses an OpenSSH private key.
func parsePrivateKey(key string) (ssh.Signer, error) {
	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {
		return nil, err
	}
	return signer, nil
}

// verifyHostKey creates a HostKeyCallback that validates against the configured fingerprint.
func verifyHostKey(expectedFingerprint string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fingerprint := ssh.FingerprintSHA256(key)
		if fingerprint != expectedFingerprint {
			return NewHostKeyError(hostname, expectedFingerprint, fingerprint)
		}
		return nil
	}
}

// sshDialer abstracts SSH connection for testing.
type sshDialer interface {
	Dial(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error)
}

// defaultDialer uses the real ssh.Dial function.
type defaultDialer struct{}

func (d *defaultDialer) Dial(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	return ssh.Dial(network, addr, config)
}

// sshSession abstracts an SSH session for testing.
type sshSession interface {
	Run(cmd string) error
	Output(cmd string) ([]byte, error)
	CombinedOutput(cmd string) ([]byte, error)
	Close() error
}

// sshClientWrapper abstracts SSH client operations for testing.
type sshClientWrapper interface {
	NewSession() (sshSession, error)
	Close() error
}

// realSSHClient wraps *ssh.Client for the interface.
type realSSHClient struct {
	client *ssh.Client
}

func (r *realSSHClient) NewSession() (sshSession, error) {
	return r.client.NewSession()
}

func (r *realSSHClient) Close() error {
	return r.client.Close()
}

// SSHClient implements Client interface using SSH/midclt.
type SSHClient struct {
	config        *SSHConfig
	client        *ssh.Client
	clientWrapper sshClientWrapper
	dialer        sshDialer
	mu            sync.Mutex
	sessionSem    chan struct{} // limits concurrent SSH sessions
	logger        Logger

	// Version (set during Connect)
	version   api.Version
	connected bool
}

// Compile-time check that SSHClient implements Client.
var _ Client = (*SSHClient)(nil)

// SSHOption configures an SSHClient.
type SSHOption func(*SSHClient)

// WithLogger sets the logger for the SSH client.
// If not provided, a NopLogger is used.
func WithLogger(l Logger) SSHOption {
	return func(c *SSHClient) {
		c.logger = l
	}
}

// NewSSHClient creates a new SSH client for TrueNAS.
func NewSSHClient(config *SSHConfig, opts ...SSHOption) (*SSHClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	maxSessions := config.MaxSessions
	if maxSessions <= 0 {
		maxSessions = 5 // default
	}

	c := &SSHClient{
		config:     config,
		dialer:     &defaultDialer{},
		sessionSem: make(chan struct{}, maxSessions),
		logger:     NopLogger{},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// acquireSession blocks until a session slot is available and returns a release function.
func (c *SSHClient) acquireSession() func() {
	c.sessionSem <- struct{}{}
	return func() {
		<-c.sessionSem
	}
}

// connect establishes the SSH connection if not already connected.
func (c *SSHClient) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Already connected
	if c.client != nil {
		return nil
	}

	signer, err := parsePrivateKey(c.config.PrivateKey)
	if err != nil {
		return err
	}

	sshConfig := &ssh.ClientConfig{
		User: c.config.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: verifyHostKey(c.config.HostKeyFingerprint),
	}

	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	client, err := c.dialer.Dial("tcp", addr, sshConfig)
	if err != nil {
		return NewConnectionError(c.config.Host, c.config.Port, err)
	}

	c.client = client
	c.clientWrapper = &realSSHClient{client: client}
	return nil
}

// serializeParams converts params to shell-escaped command arguments.
// If params is a []any slice, each element becomes a separate argument.
// This is needed for TrueNAS CRUD update methods which expect (id, data) as separate args.
func serializeParams(params any) (string, error) {
	if params == nil {
		return "", nil
	}

	// Handle []any slices specially - each element becomes a separate argument
	if slice, ok := params.([]any); ok {
		var args string
		for _, arg := range slice {
			argJSON, err := json.Marshal(arg)
			if err != nil {
				return "", err
			}
			args += " " + shellescape.Quote(string(argJSON))
		}
		return args, nil
	}

	// Default: serialize as single argument
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	return " " + shellescape.Quote(string(paramsJSON)), nil
}

// Call executes a midclt command and returns the parsed JSON response.
// Note: The ctx parameter is accepted for interface compatibility and future
// use (e.g., command cancellation, timeouts) but is not currently used.
func (c *SSHClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Acquire session slot (blocks if at limit)
	release := c.acquireSession()
	defer release()

	// Ensure we're connected (only if not already mocked)
	if c.clientWrapper == nil {
		if err := c.connect(); err != nil {
			return nil, err
		}
	}

	// Build command (use sudo for non-root users with sudo access)
	cmd := fmt.Sprintf("sudo midclt call %s", method)
	paramsStr, err := serializeParams(params)
	if err != nil {
		return nil, err
	}
	cmd += paramsStr

	// Log request
	tflog.Debug(ctx, "API request", map[string]any{
		"method": method,
		"params": paramsStr,
	})

	// Create session
	session, err := c.clientWrapper.NewSession()
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.Close() }()

	// Execute command - use CombinedOutput to capture stderr for error messages
	output, err := session.CombinedOutput(cmd)

	// Log response
	tflog.Debug(ctx, "API response", map[string]any{
		"method": method,
		"output": string(output),
		"error":  err,
	})

	if err != nil {
		// Include the output (which contains stderr) in the error message
		if len(output) > 0 {
			return nil, fmt.Errorf("%w: %s", err, string(output))
		}
		return nil, err
	}

	return json.RawMessage(output), nil
}

// CallAndWait executes a command and waits for job completion.
// On TrueNAS 25.x+, uses midclt's -j flag which relies on core.subscribe.
// On TrueNAS 24.x, polls core.get_jobs since core.subscribe doesn't exist.
// Note: The response is not parsed as it contains unparseable progress output.
// Callers should query the resource state separately after this returns.
func (c *SSHClient) CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Use cached version to determine job waiting strategy
	version := c.Version()

	// TrueNAS 25.x+ supports core.subscribe, so midclt -j works
	// TrueNAS 24.x doesn't have core.subscribe, so we poll core.get_jobs
	if version.AtLeast(25, 0) {
		return c.callAndWaitWithFlag(ctx, method, params)
	}
	return c.callAndWaitWithPolling(ctx, method, params)
}

// callAndWaitWithFlag uses midclt's -j flag for job waiting (TrueNAS 25.x+).
func (c *SSHClient) callAndWaitWithFlag(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Acquire session slot (blocks if at limit)
	release := c.acquireSession()
	defer release()

	// Ensure we're connected (only if not already mocked)
	if c.clientWrapper == nil {
		if err := c.connect(); err != nil {
			return nil, err
		}
	}

	// Build command with -j flag for job waiting
	cmd := fmt.Sprintf("sudo midclt call -j %s", method)
	paramsStr, err := serializeParams(params)
	if err != nil {
		return nil, err
	}
	cmd += paramsStr

	// Log request
	tflog.Debug(ctx, "API request (job)", map[string]any{
		"method": method,
		"params": paramsStr,
	})

	// Create session
	session, err := c.clientWrapper.NewSession()
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.Close() }()

	// Execute command - use CombinedOutput to capture stderr for error messages
	output, err := session.CombinedOutput(cmd)

	// Log response
	tflog.Debug(ctx, "API response (job)", map[string]any{
		"method": method,
		"output": string(output),
		"error":  err,
	})
	if err != nil {
		// Strip ANSI codes for cleaner error messages
		cleaned := ansiRegex.ReplaceAll(output, nil)
		if len(cleaned) > 0 {
			// Parse into structured error
			tnErr := ParseTrueNASError(string(cleaned))
			// Fetch app lifecycle log if applicable (using cat)
			EnrichAppLifecycleError(ctx, tnErr, func(ctx context.Context, path string) (string, error) {
				output, err := c.runSudoOutput(ctx, "cat", path)
				return string(output), err
			})
			return nil, tnErr
		}
		return nil, err
	}

	// Success - return nil as callers should query state separately
	return nil, nil
}

// callAndWaitWithPolling polls core.get_jobs for job completion (TrueNAS 24.x).
// This is needed because TrueNAS 24.x doesn't have core.subscribe.
func (c *SSHClient) callAndWaitWithPolling(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Call the method without -j to get the job ID
	result, err := c.Call(ctx, method, params)
	if err != nil {
		return nil, err
	}

	// Parse the job ID from the result
	var jobID int64
	if err := json.Unmarshal(result, &jobID); err != nil {
		// If it's not a job ID, the method completed synchronously
		return result, nil
	}

	// Poll core.get_jobs until the job completes
	return c.pollJobCompletion(ctx, jobID)
}

// jobStatus represents a job from core.get_jobs.
type jobStatus struct {
	ID        int64           `json:"id"`
	State     string          `json:"state"`
	Result    json.RawMessage `json:"result"`
	Error     *string         `json:"error"`
	Exception *string         `json:"exception"`
	ExcInfo   *struct {
		Type   string `json:"type"`
		Errno  *int   `json:"errno"`
		Repr   string `json:"repr"`
		Extra  any    `json:"extra"`
	} `json:"exc_info"`
}

// pollJobCompletion polls core.get_jobs until the job reaches a terminal state.
func (c *SSHClient) pollJobCompletion(ctx context.Context, jobID int64) (json.RawMessage, error) {
	// Build the query filter for this job ID
	filter := []any{[]any{"id", "=", jobID}}
	options := map[string]any{"get": true}
	params := []any{filter, options}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result, err := c.Call(ctx, "core.get_jobs", params)
		if err != nil {
			return nil, fmt.Errorf("failed to poll job %d: %w", jobID, err)
		}

		var job jobStatus
		if err := json.Unmarshal(result, &job); err != nil {
			return nil, fmt.Errorf("failed to parse job status: %w", err)
		}

		switch job.State {
		case "SUCCESS":
			// Job completed successfully
			return nil, nil
		case "FAILED", "ABORTED":
			// Job failed - construct error from job info
			errMsg := "job failed"
			if job.Error != nil && *job.Error != "" {
				errMsg = *job.Error
			} else if job.Exception != nil && *job.Exception != "" {
				errMsg = *job.Exception
			}
			tnErr := ParseTrueNASError(errMsg)
			// Fetch app lifecycle log if applicable
			EnrichAppLifecycleError(ctx, tnErr, func(ctx context.Context, path string) (string, error) {
				output, err := c.runSudoOutput(ctx, "cat", path)
				return string(output), err
			})
			return nil, tnErr
		case "RUNNING", "WAITING":
			// Job still in progress, wait before polling again
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(500 * time.Millisecond):
				continue
			}
		default:
			// Unknown state, treat as still running
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(500 * time.Millisecond):
				continue
			}
		}
	}
}

// Close closes the SSH connection.
func (c *SSHClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.clientWrapper == nil {
		return nil
	}

	err := c.clientWrapper.Close()
	c.client = nil
	c.clientWrapper = nil
	return err
}

// Connect establishes the SSH connection and detects TrueNAS version.
// Must be called before using the client.
func (c *SSHClient) Connect(ctx context.Context) error {
	if err := c.connect(); err != nil {
		return err
	}

	result, err := c.Call(ctx, "system.version", nil)
	if err != nil {
		return fmt.Errorf("failed to detect TrueNAS version: %w", err)
	}

	// system.version returns a raw string (not JSON), so convert directly
	// and trim any whitespace/newlines
	raw := strings.TrimSpace(string(result))

	c.version, err = api.ParseVersion(raw)
	if err != nil {
		return err
	}

	c.connected = true
	return nil
}

// Version returns the cached TrueNAS version.
// Panics if called before Connect() - fail fast on programmer error.
func (c *SSHClient) Version() api.Version {
	if !c.connected {
		panic("client.Version() called before Connect()")
	}
	return c.version
}

// WriteFile writes content to a file on the remote system using the TrueNAS
// filesystem.file_receive API. This runs with root privileges via middleware,
// allowing writes to paths that would otherwise require elevated permissions.
func (c *SSHClient) WriteFile(ctx context.Context, path string, fileParams WriteFileParams) error {
	b64Content := base64.StdEncoding.EncodeToString(fileParams.Content)

	// Convert nil UID/GID to -1 for the TrueNAS API (unchanged)
	uid := -1
	if fileParams.UID != nil {
		uid = *fileParams.UID
	}
	gid := -1
	if fileParams.GID != nil {
		gid = *fileParams.GID
	}

	params := []any{
		path,
		b64Content,
		map[string]any{
			"mode": int(fileParams.Mode),
			"uid":  uid,
			"gid":  gid,
		},
	}
	_, err := c.Call(ctx, "filesystem.file_receive", params)
	if err != nil {
		return fmt.Errorf("failed to write file %q: %w", path, err)
	}
	return nil
}

// ReadFile reads the content of a file from the remote system.
func (c *SSHClient) ReadFile(ctx context.Context, path string) ([]byte, error) {
	output, err := c.runSudoOutput(ctx, "cat", path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}
	return output, nil
}

// runSudo executes a command with sudo via SSH.
func (c *SSHClient) runSudo(ctx context.Context, args ...string) error {
	// Acquire session slot (blocks if at limit)
	release := c.acquireSession()
	defer release()

	// Ensure we're connected (only if not already mocked)
	if c.clientWrapper == nil {
		if err := c.connect(); err != nil {
			return err
		}
	}

	// Build command with proper escaping
	var escaped []string
	for _, arg := range args {
		escaped = append(escaped, shellescape.Quote(arg))
	}
	cmd := "sudo " + strings.Join(escaped, " ")

	// Create session
	session, err := c.clientWrapper.NewSession()
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()

	// Execute command
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		if len(output) > 0 {
			return fmt.Errorf("%w: %s", err, string(output))
		}
		return err
	}
	return nil
}

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
	defer func() { _ = session.Close() }()

	output, err := session.Output(cmd)
	if err != nil {
		return nil, err
	}
	return output, nil
}

// DeleteFile removes a file from the remote system using sudo rm.
func (c *SSHClient) DeleteFile(ctx context.Context, path string) error {
	if err := c.runSudo(ctx, "rm", path); err != nil {
		return fmt.Errorf("failed to delete file %q: %w", path, err)
	}
	return nil
}

// RemoveDir removes an empty directory from the remote system using sudo rmdir.
func (c *SSHClient) RemoveDir(ctx context.Context, path string) error {
	if err := c.runSudo(ctx, "rmdir", path); err != nil {
		return fmt.Errorf("failed to remove directory %q: %w", path, err)
	}
	return nil
}

// FileExists checks if a file exists on the remote system using the TrueNAS
// filesystem.stat API. This runs with root privileges via middleware.
func (c *SSHClient) FileExists(ctx context.Context, path string) (bool, error) {
	_, err := c.Call(ctx, "filesystem.stat", path)
	if err != nil {
		// Parse the error to check for ENOENT (file not found)
		parsed := ParseTrueNASError(err.Error())
		if parsed.Code == "ENOENT" {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat file %q: %w", path, err)
	}
	return true, nil
}

// MkdirAll creates a directory and all parent directories using the TrueNAS
// filesystem.mkdir API. This runs with root privileges via middleware.
func (c *SSHClient) MkdirAll(ctx context.Context, path string, mode fs.FileMode) error {
	params := map[string]any{
		"path": path,
		"options": map[string]any{
			"mode": fmt.Sprintf("%04o", mode),
		},
	}
	_, err := c.Call(ctx, "filesystem.mkdir", params)
	if err != nil {
		return fmt.Errorf("failed to create directory %q: %w", path, err)
	}
	return nil
}

// RemoveAll recursively removes a directory and all its contents using sudo rm -rf.
func (c *SSHClient) RemoveAll(ctx context.Context, path string) error {
	if err := c.runSudo(ctx, "rm", "-rf", path); err != nil {
		return fmt.Errorf("failed to remove directory %q: %w", path, err)
	}
	return nil
}

// Chown changes the ownership of a file or directory using the TrueNAS
// filesystem.chown API. This runs with root privileges via middleware.
func (c *SSHClient) Chown(ctx context.Context, path string, uid, gid int) error {
	params := map[string]any{
		"path": path,
		"uid":  uid,
		"gid":  gid,
	}
	_, err := c.CallAndWait(ctx, "filesystem.chown", params)
	if err != nil {
		return fmt.Errorf("failed to change ownership of %q: %w", path, err)
	}
	return nil
}

// ChmodRecursive recursively changes permissions on a directory and all contents
// using the TrueNAS filesystem.setperm API. This runs with root privileges via middleware.
func (c *SSHClient) ChmodRecursive(ctx context.Context, path string, mode fs.FileMode) error {
	params := map[string]any{
		"path": path,
		"mode": fmt.Sprintf("%04o", mode),
		"options": map[string]any{
			"recursive": true,
		},
	}
	_, err := c.CallAndWait(ctx, "filesystem.setperm", params)
	if err != nil {
		return fmt.Errorf("failed to chmod %q: %w", path, err)
	}
	return nil
}
