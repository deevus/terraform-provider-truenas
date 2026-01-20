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

	"al.essio.dev/pkg/shellescape"
	"github.com/deevus/terraform-provider-truenas/internal/api"
	"github.com/pkg/sftp"
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

// sftpClient abstracts SFTP operations for testing.
type sftpClient interface {
	Create(path string) (sftpFile, error)
	MkdirAll(path string) error
	Stat(path string) (fs.FileInfo, error)
	Remove(path string) error
	RemoveDirectory(path string) error
	RemoveAll(path string) error
	Open(path string) (sftpFile, error)
	Chmod(path string, mode fs.FileMode) error
	Chown(path string, uid, gid int) error
	ReadDir(path string) ([]fs.FileInfo, error)
	Close() error
}

// sftpFile abstracts SFTP file operations for testing.
type sftpFile interface {
	Write(p []byte) (int, error)
	Read(p []byte) (int, error)
	Close() error
}

// realSFTPClient wraps *sftp.Client for the interface.
type realSFTPClient struct {
	client *sftp.Client
}

func (r *realSFTPClient) Create(path string) (sftpFile, error) {
	return r.client.Create(path)
}

func (r *realSFTPClient) MkdirAll(path string) error {
	return r.client.MkdirAll(path)
}

func (r *realSFTPClient) Stat(path string) (fs.FileInfo, error) {
	return r.client.Stat(path)
}

func (r *realSFTPClient) Remove(path string) error {
	return r.client.Remove(path)
}

func (r *realSFTPClient) RemoveDirectory(path string) error {
	return r.client.RemoveDirectory(path)
}

func (r *realSFTPClient) RemoveAll(path string) error {
	return r.client.RemoveAll(path)
}

func (r *realSFTPClient) Open(path string) (sftpFile, error) {
	return r.client.Open(path)
}

func (r *realSFTPClient) Chmod(path string, mode fs.FileMode) error {
	return r.client.Chmod(path, mode)
}

func (r *realSFTPClient) Chown(path string, uid, gid int) error {
	return r.client.Chown(path, uid, gid)
}

func (r *realSFTPClient) ReadDir(path string) ([]fs.FileInfo, error) {
	return r.client.ReadDir(path)
}

func (r *realSFTPClient) Close() error {
	return r.client.Close()
}

// SSHClient implements Client interface using SSH/midclt.
type SSHClient struct {
	config        *SSHConfig
	client        *ssh.Client
	clientWrapper sshClientWrapper
	sftpClient    sftpClient
	dialer        sshDialer
	mu            sync.Mutex
	sessionSem    chan struct{} // limits concurrent SSH sessions

	// Version caching
	version     api.Version
	versionErr  error
	versionOnce sync.Once
}

// Compile-time check that SSHClient implements Client.
var _ Client = (*SSHClient)(nil)

// NewSSHClient creates a new SSH client for TrueNAS.
func NewSSHClient(config *SSHConfig) (*SSHClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	maxSessions := config.MaxSessions
	if maxSessions <= 0 {
		maxSessions = 5 // default
	}

	return &SSHClient{
		config:     config,
		dialer:     &defaultDialer{},
		sessionSem: make(chan struct{}, maxSessions),
	}, nil
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

	// Create session
	session, err := c.clientWrapper.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// Execute command - use CombinedOutput to capture stderr for error messages
	output, err := session.CombinedOutput(cmd)
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
// Uses midclt's -j flag to wait for long-running jobs.
// Note: The response is not parsed as it contains unparseable progress output.
// Callers should query the resource state separately after this returns.
func (c *SSHClient) CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error) {
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

	// Create session
	session, err := c.clientWrapper.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// Execute command - use CombinedOutput to capture stderr for error messages
	output, err := session.CombinedOutput(cmd)
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

// Close closes the SSH connection.
func (c *SSHClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sftpClient != nil {
		c.sftpClient.Close()
		c.sftpClient = nil
	}

	if c.clientWrapper == nil {
		return nil
	}

	err := c.clientWrapper.Close()
	c.client = nil
	c.clientWrapper = nil
	return err
}

// GetVersion returns the TrueNAS version. Probes once and caches the result.
func (c *SSHClient) GetVersion(ctx context.Context) (api.Version, error) {
	c.versionOnce.Do(func() {
		result, err := c.Call(ctx, "system.version", nil)
		if err != nil {
			c.versionErr = fmt.Errorf("failed to detect TrueNAS version: %w", err)
			return
		}

		// system.version returns a raw string (not JSON), so convert directly
		// and trim any whitespace/newlines
		raw := strings.TrimSpace(string(result))

		c.version, c.versionErr = api.ParseVersion(raw)
	})
	return c.version, c.versionErr
}

// connectSFTP establishes the SFTP connection if not already connected.
func (c *SSHClient) connectSFTP() error {
	// First ensure SSH is connected (this has its own locking)
	if err := c.connect(); err != nil {
		return err
	}

	// Now lock to check/set SFTP client
	c.mu.Lock()
	defer c.mu.Unlock()

	// Already have SFTP client (double-check after acquiring lock)
	if c.sftpClient != nil {
		return nil
	}

	// Create SFTP client
	sftpConn, err := sftp.NewClient(c.client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}

	c.sftpClient = &realSFTPClient{client: sftpConn}
	return nil
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
	defer session.Close()

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
	defer session.Close()

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
