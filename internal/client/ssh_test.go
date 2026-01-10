package client

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/ssh"
)

// mockDialer is a test double for sshDialer.
type mockDialer struct {
	dialFunc func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error)
}

func (m *mockDialer) Dial(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	if m.dialFunc != nil {
		return m.dialFunc(network, addr, config)
	}
	return nil, nil
}

// mockSession is a test double for sshSession.
type mockSession struct {
	runFunc    func(cmd string) error
	outputFunc func(cmd string) ([]byte, error)
	closeFunc  func() error
}

func (m *mockSession) Run(cmd string) error {
	if m.runFunc != nil {
		return m.runFunc(cmd)
	}
	return nil
}

func (m *mockSession) Output(cmd string) ([]byte, error) {
	if m.outputFunc != nil {
		return m.outputFunc(cmd)
	}
	return nil, nil
}

func (m *mockSession) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// mockSSHClient is a test double for sshClientWrapper.
type mockSSHClient struct {
	newSessionFunc func() (sshSession, error)
	closeFunc      func() error
}

func (m *mockSSHClient) NewSession() (sshSession, error) {
	if m.newSessionFunc != nil {
		return m.newSessionFunc()
	}
	return nil, nil
}

func (m *mockSSHClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// Test ED25519 key for testing
const testPrivateKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACCtws8zNrmNWDx+nxb26zA2iTVTn4TZQyK1yANm0XiawgAAAJjjXr/4416/
+AAAAAtzc2gtZWQyNTUxOQAAACCtws8zNrmNWDx+nxb26zA2iTVTn4TZQyK1yANm0Xiawg
AAAEARU6QyekrrGEM7eyo5JKVU08PPAbbO19sp/dB3xMSpaq3CzzM2uY1YPH6fFvbrMDaJ
NVOfhNlDIrXIA2bReJrCAAAAEnRlc3RAZXhhbXBsZS5sb2NhbAECAw==
-----END OPENSSH PRIVATE KEY-----`

func TestSSHConfig_Validate_MissingHost(t *testing.T) {
	config := &SSHConfig{
		Host:       "",
		PrivateKey: testPrivateKey,
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for missing host")
	}

	if err.Error() != "host is required" {
		t.Errorf("expected 'host is required', got %q", err.Error())
	}
}

func TestSSHConfig_Validate_MissingPrivateKey(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: "",
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for missing private key")
	}

	if err.Error() != "private_key is required" {
		t.Errorf("expected 'private_key is required', got %q", err.Error())
	}
}

func TestSSHConfig_Validate_Defaults(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Port != 22 {
		t.Errorf("expected default port 22, got %d", config.Port)
	}

	if config.User != "root" {
		t.Errorf("expected default user 'root', got %q", config.User)
	}
}

func TestSSHConfig_Validate_CustomValues(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		Port:       2222,
		User:       "admin",
		PrivateKey: testPrivateKey,
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Port != 2222 {
		t.Errorf("expected port 2222, got %d", config.Port)
	}

	if config.User != "admin" {
		t.Errorf("expected user 'admin', got %q", config.User)
	}
}

func TestParsePrivateKey_InvalidKey(t *testing.T) {
	_, err := parsePrivateKey("not a valid key")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestParsePrivateKey_ValidKey(t *testing.T) {
	signer, err := parsePrivateKey(testPrivateKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if signer == nil {
		t.Error("expected non-nil signer")
	}
}

func TestNewSSHClient_InvalidConfig(t *testing.T) {
	config := &SSHConfig{
		Host:       "",
		PrivateKey: testPrivateKey,
	}

	_, err := NewSSHClient(config)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

func TestNewSSHClient_ValidConfig(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, err := NewSSHClient(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client == nil {
		t.Error("expected non-nil client")
	}

	// Verify defaults were applied
	if config.Port != 22 {
		t.Errorf("expected port 22, got %d", config.Port)
	}
	if config.User != "root" {
		t.Errorf("expected user 'root', got %q", config.User)
	}
}

func TestSSHClient_Connect_AlreadyConnected(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)
	// Simulate already connected by setting a non-nil client
	client.client = &ssh.Client{}

	err := client.connect()
	if err != nil {
		t.Fatalf("unexpected error when already connected: %v", err)
	}
}

func TestSSHClient_Connect_InvalidKey(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: "invalid key",
	}
	// Bypass validation for this test
	config.Port = 22
	config.User = "root"

	client := &SSHClient{
		config: config,
		dialer: &mockDialer{},
	}

	err := client.connect()
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestSSHClient_Connect_DialError(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)
	client.dialer = &mockDialer{
		dialFunc: func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
			return nil, errors.New("connection refused")
		},
	}

	err := client.connect()
	if err == nil {
		t.Fatal("expected error for dial failure")
	}

	// Check that it's a TrueNASError with ECONNREFUSED
	trueNASErr, ok := err.(*TrueNASError)
	if !ok {
		t.Fatalf("expected TrueNASError, got %T", err)
	}
	if trueNASErr.Code != "ECONNREFUSED" {
		t.Errorf("expected code ECONNREFUSED, got %q", trueNASErr.Code)
	}
}

func TestSSHClient_Connect_Success(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)
	mockClient := &ssh.Client{}
	client.dialer = &mockDialer{
		dialFunc: func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
			// Verify correct address format
			if addr != "truenas.local:22" {
				t.Errorf("expected addr truenas.local:22, got %s", addr)
			}
			if network != "tcp" {
				t.Errorf("expected network tcp, got %s", network)
			}
			return mockClient, nil
		},
	}

	err := client.connect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.client != mockClient {
		t.Error("expected client to be set")
	}
}

func TestSSHClient_Call_ConnectFails(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)
	client.dialer = &mockDialer{
		dialFunc: func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
			return nil, errors.New("connection refused")
		},
	}

	_, err := client.Call(context.Background(), "system.info", nil)
	if err == nil {
		t.Fatal("expected error when connect fails")
	}
}

func TestSSHClient_Call_WithoutParams(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockSess := &mockSession{
		outputFunc: func(cmd string) ([]byte, error) {
			expected := `midclt call system.info`
			if cmd != expected {
				t.Errorf("expected command %q, got %q", expected, cmd)
			}
			return []byte(`{"version": "24.04"}`), nil
		},
	}

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			return mockSess, nil
		},
	}

	client.clientWrapper = mockClient

	result, err := client.Call(context.Background(), "system.info", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != `{"version": "24.04"}` {
		t.Errorf("expected result {\"version\": \"24.04\"}, got %s", result)
	}
}

func TestSSHClient_Call_WithParams(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockSess := &mockSession{
		outputFunc: func(cmd string) ([]byte, error) {
			expected := `midclt call app.query '[{"name":"test"}]'`
			if cmd != expected {
				t.Errorf("expected command %q, got %q", expected, cmd)
			}
			return []byte(`[{"id": 1}]`), nil
		},
	}

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			return mockSess, nil
		},
	}

	client.clientWrapper = mockClient

	result, err := client.Call(context.Background(), "app.query", []map[string]string{{"name": "test"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != `[{"id": 1}]` {
		t.Errorf("expected result [{\"id\": 1}], got %s", result)
	}
}

func TestSSHClient_Call_SessionError(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			return nil, errors.New("session error")
		},
	}

	client.clientWrapper = mockClient

	_, err := client.Call(context.Background(), "system.info", nil)
	if err == nil {
		t.Fatal("expected error when session fails")
	}
}

func TestSSHClient_Call_OutputError(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockSess := &mockSession{
		outputFunc: func(cmd string) ([]byte, error) {
			return nil, errors.New("command failed")
		},
	}

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			return mockSess, nil
		},
	}

	client.clientWrapper = mockClient

	_, err := client.Call(context.Background(), "system.info", nil)
	if err == nil {
		t.Fatal("expected error when output fails")
	}
}

func TestSSHClient_Call_ParamsMarshalError(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)
	client.clientWrapper = &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			return &mockSession{}, nil
		},
	}

	// Channels cannot be marshaled to JSON
	_, err := client.Call(context.Background(), "system.info", make(chan int))
	if err == nil {
		t.Fatal("expected error for unmarshalable params")
	}
}

func TestSSHClient_CallAndWait(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockSess := &mockSession{
		outputFunc: func(cmd string) ([]byte, error) {
			expected := `midclt call app.create '{"name":"test"}'`
			if cmd != expected {
				t.Errorf("expected command %q, got %q", expected, cmd)
			}
			return []byte(`{"id": 123}`), nil
		},
	}

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			return mockSess, nil
		},
	}

	client.clientWrapper = mockClient

	result, err := client.CallAndWait(context.Background(), "app.create", map[string]string{"name": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != `{"id": 123}` {
		t.Errorf("expected result {\"id\": 123}, got %s", result)
	}
}

func TestSSHClient_Close_NotConnected(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	// Close when not connected should be a no-op
	err := client.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSSHClient_Close_Connected(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	closeCalled := false
	mockClient := &mockSSHClient{
		closeFunc: func() error {
			closeCalled = true
			return nil
		},
	}

	client.clientWrapper = mockClient

	err := client.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !closeCalled {
		t.Error("expected close to be called")
	}

	// Verify state is cleared
	if client.clientWrapper != nil {
		t.Error("expected clientWrapper to be nil after close")
	}
}

func TestSSHClient_Close_Error(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockClient := &mockSSHClient{
		closeFunc: func() error {
			return errors.New("close error")
		},
	}

	client.clientWrapper = mockClient

	err := client.Close()
	if err == nil {
		t.Fatal("expected error")
	}
}
