package client

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deevus/terraform-provider-truenas/internal/api"
	"golang.org/x/crypto/ssh"
)

// testHostKeyFingerprint is a valid SHA256 fingerprint format for testing.
const testHostKeyFingerprint = "SHA256:uL6HlVEzPRLQnJwO6l7F8VXNgJyiPXxrMIBqfq/8VKE"

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
	runFunc            func(cmd string) error
	outputFunc         func(cmd string) ([]byte, error)
	combinedOutputFunc func(cmd string) ([]byte, error)
	closeFunc          func() error
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

func (m *mockSession) CombinedOutput(cmd string) ([]byte, error) {
	if m.combinedOutputFunc != nil {
		return m.combinedOutputFunc(cmd)
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

func TestVerifyHostKey_Match(t *testing.T) {
	// Generate a test key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}
	pubKey := signer.PublicKey()
	expectedFingerprint := ssh.FingerprintSHA256(pubKey)

	callback := verifyHostKey(expectedFingerprint)
	err = callback("test-host", nil, pubKey)
	if err != nil {
		t.Errorf("expected no error for matching fingerprint, got: %v", err)
	}
}

func TestVerifyHostKey_Mismatch(t *testing.T) {
	// Generate a test key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}
	pubKey := signer.PublicKey()
	actualFingerprint := ssh.FingerprintSHA256(pubKey)

	// Use a different expected fingerprint
	wrongFingerprint := "SHA256:wrongWrongWrongWrongWrongWrongWrongWrongWro"

	callback := verifyHostKey(wrongFingerprint)
	err = callback("test-host", nil, pubKey)
	if err == nil {
		t.Fatal("expected error for mismatched fingerprint")
	}

	// Verify error contains both expected and actual fingerprints
	trueNASErr, ok := err.(*TrueNASError)
	if !ok {
		t.Fatalf("expected TrueNASError, got %T", err)
	}
	if trueNASErr.Code != "EHOSTKEY" {
		t.Errorf("expected code EHOSTKEY, got %q", trueNASErr.Code)
	}
	if trueNASErr.Message == "" {
		t.Error("expected error message to be non-empty")
	}
	// Check that the error message contains both fingerprints
	if !containsAll(trueNASErr.Message, wrongFingerprint, actualFingerprint) {
		t.Errorf("expected error message to contain both fingerprints, got: %s", trueNASErr.Message)
	}
}

// containsAll checks if s contains all the given substrings.
func containsAll(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

// contains checks if s contains sub.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && len(sub) > 0 && findSubstring(s, sub)))
}

// findSubstring returns true if sub is found in s.
func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

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

func TestSSHConfig_Validate_MissingFingerprint(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: "",
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for missing host key fingerprint")
	}

	if err.Error() != "host_key_fingerprint is required" {
		t.Errorf("expected 'host_key_fingerprint is required', got %q", err.Error())
	}
}

func TestSSHConfig_Validate_Defaults(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
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
		Host:               "truenas.local",
		Port:               2222,
		User:               "admin",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
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
		Host:               "",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	_, err := NewSSHClient(config)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

func TestNewSSHClient_ValidConfig(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
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
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
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
		Host:               "truenas.local",
		PrivateKey:         "invalid key",
		HostKeyFingerprint: testHostKeyFingerprint,
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
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
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
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
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

func TestSSHClient_Version_PanicsBeforeConnect(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, err := NewSSHClient(config)
	if err != nil {
		t.Fatalf("NewSSHClient() error = %v", err)
	}

	// Should panic because Connect() was not called
	defer func() {
		if r := recover(); r == nil {
			t.Error("Version() should have panicked when called before Connect()")
		}
	}()
	client.Version()
}

func TestSSHClient_Call_ConnectFails(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
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
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	mockSess := &mockSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			expected := `sudo midclt call system.info`
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
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	mockSess := &mockSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			// shellescape.Quote wraps the JSON in single quotes
			expected := `sudo midclt call app.query '[{"name":"test"}]'`
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
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
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
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	mockSess := &mockSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
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
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
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

func TestSSHClient_Call_ShellEscaping(t *testing.T) {
	// This test verifies that shell metacharacters are properly escaped
	// to prevent command injection attacks
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	// Test with shell metacharacters that could be used for injection
	testCases := []struct {
		name           string
		params         any
		expectedSuffix string
	}{
		{
			name:           "single quotes in params",
			params:         map[string]string{"name": "test'; rm -rf /"},
			expectedSuffix: `'{"name":"test'\'''; rm -rf /"}'`, // shellescape escapes single quotes
		},
		{
			name:           "backticks in params",
			params:         map[string]string{"cmd": "`whoami`"},
			expectedSuffix: `'{"cmd":"\` + "`" + `whoami\` + "`" + `"}'`,
		},
		{
			name:           "dollar sign in params",
			params:         map[string]string{"var": "$(id)"},
			expectedSuffix: `'{"var":"$(id)"}'`,
		},
		{
			name:           "semicolon in params",
			params:         map[string]string{"cmd": "; ls -la"},
			expectedSuffix: `'{"cmd":"; ls -la"}'`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedCmd string
			mockSess := &mockSession{
				combinedOutputFunc: func(cmd string) ([]byte, error) {
					capturedCmd = cmd
					return []byte(`{}`), nil
				},
			}

			mockClient := &mockSSHClient{
				newSessionFunc: func() (sshSession, error) {
					return mockSess, nil
				},
			}

			client.clientWrapper = mockClient

			_, err := client.Call(context.Background(), "test.method", tc.params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedPrefix := "sudo midclt call test.method "
			if len(capturedCmd) <= len(expectedPrefix) {
				t.Fatalf("command too short: %q", capturedCmd)
			}

			// Verify the command starts with the expected prefix
			if capturedCmd[:len(expectedPrefix)] != expectedPrefix {
				t.Errorf("expected prefix %q, got %q", expectedPrefix, capturedCmd[:len(expectedPrefix)])
			}

			// Verify the params are properly escaped (wrapped in single quotes)
			suffix := capturedCmd[len(expectedPrefix):]
			if suffix[0] != '\'' {
				t.Errorf("expected params to start with single quote, got %q", suffix)
			}
			if suffix[len(suffix)-1] != '\'' {
				t.Errorf("expected params to end with single quote, got %q", suffix)
			}
		})
	}
}

func TestSSHClient_CallAndWait_V25_UsesJobFlag(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	// Pre-set version to 25.x to use -j flag behavior
	client.version = api.Version{Major: 25, Minor: 4, Raw: "TrueNAS-25.04"}
	client.connected = true

	mockSess := &mockSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			// CallAndWait uses -j flag for job waiting on 25.x
			expected := `sudo midclt call -j app.create '{"name":"test"}'`
			if cmd != expected {
				t.Errorf("expected command %q, got %q", expected, cmd)
			}
			// Response contains progress output - caller should query state separately
			return []byte(`Progress: 100%\n{"name": "test", "state": "RUNNING"}`), nil
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

	// CallAndWait returns nil on success - callers should query state separately
	if result != nil {
		t.Errorf("expected nil result, got %s", result)
	}
}

func TestSSHClient_CallAndWait_V24_UsesPolling(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	// Pre-set version to 24.x to use polling behavior
	client.version = api.Version{Major: 24, Minor: 10, Raw: "TrueNAS-24.10"}
	client.connected = true

	callCount := 0
	mockSess := &mockSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			callCount++
			switch callCount {
			case 1:
				// First call: the actual method (without -j), returns job ID
				expected := `sudo midclt call app.create '{"name":"test"}'`
				if cmd != expected {
					t.Errorf("expected command %q, got %q", expected, cmd)
				}
				return []byte("12345"), nil // Job ID
			case 2:
				// Second call: poll core.get_jobs, job still running
				if !strings.Contains(cmd, "core.get_jobs") {
					t.Errorf("expected core.get_jobs poll, got %q", cmd)
				}
				return []byte(`{"id":12345,"state":"RUNNING","result":null,"error":null}`), nil
			case 3:
				// Third call: poll core.get_jobs, job completed
				return []byte(`{"id":12345,"state":"SUCCESS","result":null,"error":null}`), nil
			default:
				t.Errorf("unexpected call count: %d", callCount)
				return nil, fmt.Errorf("unexpected call")
			}
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

	if result != nil {
		t.Errorf("expected nil result, got %s", result)
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls (method + 2 polls), got %d", callCount)
	}
}

func TestSSHClient_CallAndWait_V24_JobFailure(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	// Pre-set version to 24.x
	client.version = api.Version{Major: 24, Minor: 10, Raw: "TrueNAS-24.10"}
	client.connected = true

	callCount := 0
	mockSess := &mockSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			callCount++
			switch callCount {
			case 1:
				// Method call returns job ID
				return []byte("12345"), nil
			case 2:
				// Poll returns failed job
				return []byte(`{"id":12345,"state":"FAILED","result":null,"error":"Something went wrong","exception":"Error: Something went wrong"}`), nil
			default:
				return nil, fmt.Errorf("unexpected call")
			}
		},
	}

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			return mockSess, nil
		},
	}

	client.clientWrapper = mockClient

	_, err := client.CallAndWait(context.Background(), "app.create", map[string]string{"name": "test"})
	if err == nil {
		t.Fatal("expected error for failed job, got nil")
	}

	if !strings.Contains(err.Error(), "Something went wrong") {
		t.Errorf("expected error to contain 'Something went wrong', got %q", err.Error())
	}
}

func TestSSHClient_CallAndWait_V24_SynchronousMethod(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	// Pre-set version to 24.x
	client.version = api.Version{Major: 24, Minor: 10, Raw: "TrueNAS-24.10"}
	client.connected = true

	mockSess := &mockSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			// Method returns a non-integer result (not a job ID)
			return []byte(`{"foo":"bar"}`), nil
		},
	}

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			return mockSess, nil
		},
	}

	client.clientWrapper = mockClient

	result, err := client.CallAndWait(context.Background(), "some.sync.method", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// For synchronous methods, returns the result directly
	if string(result) != `{"foo":"bar"}` {
		t.Errorf("expected result %q, got %q", `{"foo":"bar"}`, string(result))
	}
}

func TestSSHClient_Close_NotConnected(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
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
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
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
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
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

func TestSerializeParams_Nil(t *testing.T) {
	result, err := serializeParams(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestSerializeParams_SingleValue(t *testing.T) {
	result, err := serializeParams(map[string]string{"name": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := ` '{"name":"test"}'`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSerializeParams_Slice(t *testing.T) {
	result, err := serializeParams([]any{"myapp", map[string]any{"key": "value"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := ` '"myapp"' '{"key":"value"}'`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSerializeParams_SliceWithError(t *testing.T) {
	// A channel cannot be marshaled to JSON
	_, err := serializeParams([]any{make(chan int)})
	if err == nil {
		t.Fatal("expected error for unmarshallable type")
	}
}

func TestSSHClient_Call_PositionalArgs(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	mockSess := &mockSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			// []any slice should result in separate positional arguments
			expected := `sudo midclt call app.update '"myapp"' '{"custom_compose_config_string":"services:\n  web:\n    image: nginx"}'`
			if cmd != expected {
				t.Errorf("expected command %q, got %q", expected, cmd)
			}
			return []byte(`{"name": "myapp"}`), nil
		},
	}

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			return mockSess, nil
		},
	}

	client.clientWrapper = mockClient

	params := []any{"myapp", map[string]any{"custom_compose_config_string": "services:\n  web:\n    image: nginx"}}
	_, err := client.Call(context.Background(), "app.update", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSSHClient_CallAndWait_PositionalArgs(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	// Pre-set version to 25.x to use -j flag behavior
	client.version = api.Version{Major: 25, Minor: 4, Raw: "TrueNAS-25.04"}
	client.connected = true

	mockSess := &mockSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			// []any slice should result in separate positional arguments with -j flag
			expected := `sudo midclt call -j app.update '"myapp"' '{"key":"value"}'`
			if cmd != expected {
				t.Errorf("expected command %q, got %q", expected, cmd)
			}
			return []byte(`{"name": "myapp"}`), nil
		},
	}

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			return mockSess, nil
		},
	}

	client.clientWrapper = mockClient

	params := []any{"myapp", map[string]any{"key": "value"}}
	_, err := client.CallAndWait(context.Background(), "app.update", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSSHConfig_Validate_MaxSessions(t *testing.T) {
	tests := []struct {
		name        string
		maxSessions int
	}{
		{"zero is preserved", 0},       // Validate does not modify; NewSSHClient applies default
		{"negative is preserved", -5},  // Validate does not modify; NewSSHClient applies default
		{"positive is preserved", 25},  // positive values preserved as-is
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SSHConfig{
				Host:               "truenas.local",
				PrivateKey:         testPrivateKey,
				HostKeyFingerprint: testHostKeyFingerprint,
				MaxSessions:        tt.maxSessions,
			}
			err := config.Validate()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// MaxSessions should be unchanged by Validate (defaulting happens in NewSSHClient)
			if config.MaxSessions != tt.maxSessions {
				t.Errorf("MaxSessions = %d, want %d", config.MaxSessions, tt.maxSessions)
			}
		})
	}
}

func TestNewSSHClient_SessionSemaphore(t *testing.T) {
	tests := []struct {
		name             string
		maxSessions      int
		wantSemaphoreCap int
	}{
		{"default when zero", 0, 5},
		{"default when negative", -5, 5},
		{"custom value", 25, 25},
		{"small value", 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SSHConfig{
				Host:               "truenas.local",
				PrivateKey:         testPrivateKey,
				HostKeyFingerprint: testHostKeyFingerprint,
				MaxSessions:        tt.maxSessions,
			}
			client, err := NewSSHClient(config)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cap(client.sessionSem) != tt.wantSemaphoreCap {
				t.Errorf("sessionSem capacity = %d, want %d", cap(client.sessionSem), tt.wantSemaphoreCap)
			}
		})
	}
}

func TestSSHClient_AcquireSession(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
		MaxSessions:        2,
	}
	client, err := NewSSHClient(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Initially empty
	if len(client.sessionSem) != 0 {
		t.Errorf("sessionSem should be empty, got %d", len(client.sessionSem))
	}

	// Acquire first slot
	release1 := client.acquireSession()
	if len(client.sessionSem) != 1 {
		t.Errorf("sessionSem should have 1 item, got %d", len(client.sessionSem))
	}

	// Acquire second slot
	release2 := client.acquireSession()
	if len(client.sessionSem) != 2 {
		t.Errorf("sessionSem should have 2 items, got %d", len(client.sessionSem))
	}

	// Release first slot
	release1()
	if len(client.sessionSem) != 1 {
		t.Errorf("sessionSem should have 1 item after release, got %d", len(client.sessionSem))
	}

	// Release second slot
	release2()
	if len(client.sessionSem) != 0 {
		t.Errorf("sessionSem should be empty after both releases, got %d", len(client.sessionSem))
	}
}

func TestSSHClient_Call_RespectsSemaphore(t *testing.T) {
	var activeCount int32
	var maxActive int32

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			current := atomic.AddInt32(&activeCount, 1)
			// Track max concurrent
			for {
				old := atomic.LoadInt32(&maxActive)
				if current <= old || atomic.CompareAndSwapInt32(&maxActive, old, current) {
					break
				}
			}
			return &mockSession{
				combinedOutputFunc: func(cmd string) ([]byte, error) {
					time.Sleep(50 * time.Millisecond) // Simulate work
					return []byte(`{"result": "ok"}`), nil
				},
				closeFunc: func() error {
					atomic.AddInt32(&activeCount, -1)
					return nil
				},
			}, nil
		},
	}

	client := &SSHClient{
		config:        &SSHConfig{Host: "test", PrivateKey: testPrivateKey, HostKeyFingerprint: testHostKeyFingerprint},
		clientWrapper: mockClient,
		sessionSem:    make(chan struct{}, 2), // Limit to 2 concurrent
	}

	// Launch 5 concurrent calls
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = client.Call(context.Background(), "test.method", nil)
		}()
	}
	wg.Wait()

	// Max concurrent should never exceed semaphore limit
	if atomic.LoadInt32(&maxActive) > 2 {
		t.Errorf("max concurrent sessions = %d, want <= 2", maxActive)
	}
}

func TestSSHClient_CallAndWait_RespectsSemaphore(t *testing.T) {
	var activeCount int32
	var maxActive int32

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			current := atomic.AddInt32(&activeCount, 1)
			for {
				old := atomic.LoadInt32(&maxActive)
				if current <= old || atomic.CompareAndSwapInt32(&maxActive, old, current) {
					break
				}
			}
			return &mockSession{
				combinedOutputFunc: func(cmd string) ([]byte, error) {
					time.Sleep(50 * time.Millisecond)
					return []byte(`{}`), nil
				},
				closeFunc: func() error {
					atomic.AddInt32(&activeCount, -1)
					return nil
				},
			}, nil
		},
	}

	client := &SSHClient{
		config:        &SSHConfig{Host: "test", PrivateKey: testPrivateKey, HostKeyFingerprint: testHostKeyFingerprint},
		clientWrapper: mockClient,
		sessionSem:    make(chan struct{}, 2),
		version:       api.Version{Major: 25, Minor: 4, Raw: "TrueNAS-25.04"},
	}
	client.connected = true // Mark version as cached

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = client.CallAndWait(context.Background(), "test.method", nil)
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&maxActive) > 2 {
		t.Errorf("max concurrent sessions = %d, want <= 2", maxActive)
	}
}

func TestSSHClient_ReadFile_RespectsSemaphore(t *testing.T) {
	var activeCount int32
	var maxActive int32

	mockClient := &mockSSHClient{
		newSessionFunc: func() (sshSession, error) {
			current := atomic.AddInt32(&activeCount, 1)
			// Track max concurrent
			for {
				old := atomic.LoadInt32(&maxActive)
				if current <= old || atomic.CompareAndSwapInt32(&maxActive, old, current) {
					break
				}
			}
			return &mockSession{
				outputFunc: func(cmd string) ([]byte, error) {
					time.Sleep(50 * time.Millisecond) // Simulate work
					return []byte("file content"), nil
				},
				closeFunc: func() error {
					atomic.AddInt32(&activeCount, -1)
					return nil
				},
			}, nil
		},
	}

	client := &SSHClient{
		config:        &SSHConfig{Host: "test", PrivateKey: testPrivateKey, HostKeyFingerprint: testHostKeyFingerprint},
		clientWrapper: mockClient,
		sessionSem:    make(chan struct{}, 2), // Limit to 2 concurrent
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = client.ReadFile(context.Background(), "/test/file")
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&maxActive) > 2 {
		t.Errorf("max concurrent sessions = %d, want <= 2", maxActive)
	}
}