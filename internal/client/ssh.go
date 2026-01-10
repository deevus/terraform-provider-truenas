package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"sync"

	"al.essio.dev/pkg/shellescape"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// ansiRegex matches ANSI escape sequences.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07`)

// SSHConfig holds configuration for SSH connection to TrueNAS.
type SSHConfig struct {
	Host       string
	Port       int
	User       string
	PrivateKey string
}

// Validate validates the SSHConfig and sets defaults.
func (c *SSHConfig) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.PrivateKey == "" {
		return errors.New("private_key is required")
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
	Open(path string) (sftpFile, error)
	Chmod(path string, mode fs.FileMode) error
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

func (r *realSFTPClient) Open(path string) (sftpFile, error) {
	return r.client.Open(path)
}

func (r *realSFTPClient) Chmod(path string, mode fs.FileMode) error {
	return r.client.Chmod(path, mode)
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
}

// Compile-time check that SSHClient implements Client.
var _ Client = (*SSHClient)(nil)

// NewSSHClient creates a new SSH client for TrueNAS.
func NewSSHClient(config *SSHConfig) (*SSHClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &SSHClient{
		config: config,
		dialer: &defaultDialer{},
	}, nil
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
		// TODO: InsecureIgnoreHostKey is insecure and should be replaced with
		// proper host key verification in production. Consider adding known_hosts
		// file support or host key fingerprint configuration.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
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
func (c *SSHClient) CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error) {
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
			return nil, fmt.Errorf("%w: %s", err, string(cleaned))
		}
		return nil, err
	}

	// Strip ANSI escape codes from output (from progress bars)
	cleaned := ansiRegex.ReplaceAll(output, nil)

	// The JSON result is typically on the last non-empty line after progress output
	lines := bytes.Split(cleaned, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		line := bytes.TrimSpace(lines[i])
		if len(line) > 0 && (line[0] == '{' || line[0] == '[') {
			return json.RawMessage(line), nil
		}
	}

	// If no JSON found, return the cleaned output
	return json.RawMessage(bytes.TrimSpace(cleaned)), nil
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

// connectSFTP establishes the SFTP connection if not already connected.
func (c *SSHClient) connectSFTP() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Already have SFTP client
	if c.sftpClient != nil {
		return nil
	}

	// Ensure SSH is connected first (unlock mutex temporarily)
	c.mu.Unlock()
	if err := c.connect(); err != nil {
		c.mu.Lock()
		return err
	}
	c.mu.Lock()

	// Create SFTP client
	sftpConn, err := sftp.NewClient(c.client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}

	c.sftpClient = &realSFTPClient{client: sftpConn}
	return nil
}

// WriteFile writes content to a file on the remote system.
func (c *SSHClient) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	// Connect SFTP if needed (skip if already mocked)
	if c.sftpClient == nil {
		if err := c.connectSFTP(); err != nil {
			return err
		}
	}

	// Create the file
	file, err := c.sftpClient.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", path, err)
	}
	defer file.Close()

	// Write content
	_, err = file.Write(content)
	if err != nil {
		return fmt.Errorf("failed to write to file %q: %w", path, err)
	}

	// Set permissions
	if err := c.sftpClient.Chmod(path, mode); err != nil {
		return fmt.Errorf("failed to set permissions on %q: %w", path, err)
	}

	return nil
}

// ReadFile reads the content of a file from the remote system.
func (c *SSHClient) ReadFile(ctx context.Context, path string) ([]byte, error) {
	// Connect SFTP if needed (skip if already mocked)
	if c.sftpClient == nil {
		if err := c.connectSFTP(); err != nil {
			return nil, err
		}
	}

	// Open the file
	file, err := c.sftpClient.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer file.Close()

	// Read all content - handles partial reads correctly
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}

	return content, nil
}

// DeleteFile removes a file from the remote system.
func (c *SSHClient) DeleteFile(ctx context.Context, path string) error {
	// Connect SFTP if needed (skip if already mocked)
	if c.sftpClient == nil {
		if err := c.connectSFTP(); err != nil {
			return err
		}
	}

	if err := c.sftpClient.Remove(path); err != nil {
		return fmt.Errorf("failed to delete file %q: %w", path, err)
	}

	return nil
}

// FileExists checks if a file exists on the remote system.
func (c *SSHClient) FileExists(ctx context.Context, path string) (bool, error) {
	// Connect SFTP if needed (skip if already mocked)
	if c.sftpClient == nil {
		if err := c.connectSFTP(); err != nil {
			return false, err
		}
	}

	_, err := c.sftpClient.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat file %q: %w", path, err)
	}

	return true, nil
}

// MkdirAll creates a directory and all parent directories.
func (c *SSHClient) MkdirAll(ctx context.Context, path string, mode fs.FileMode) error {
	// Connect SFTP if needed (skip if already mocked)
	if c.sftpClient == nil {
		if err := c.connectSFTP(); err != nil {
			return err
		}
	}

	if err := c.sftpClient.MkdirAll(path); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", path, err)
	}

	// Apply the requested permissions
	if err := c.sftpClient.Chmod(path, mode); err != nil {
		return fmt.Errorf("failed to set permissions on directory %q: %w", path, err)
	}

	return nil
}
