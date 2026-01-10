package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/crypto/ssh"
)

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

// Call executes a midclt command and returns the parsed JSON response.
func (c *SSHClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Ensure we're connected (only if not already mocked)
	if c.clientWrapper == nil {
		if err := c.connect(); err != nil {
			return nil, err
		}
	}

	// Build command
	cmd := fmt.Sprintf("midclt call %s", method)
	if params != nil {
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		cmd = fmt.Sprintf("%s '%s'", cmd, string(paramsJSON))
	}

	// Create session
	session, err := c.clientWrapper.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// Execute command
	output, err := session.Output(cmd)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(output), nil
}

// CallAndWait executes a command and waits for job completion.
// For now, this simply delegates to Call. Future implementation
// may add job polling logic.
func (c *SSHClient) CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error) {
	return c.Call(ctx, method, params)
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
