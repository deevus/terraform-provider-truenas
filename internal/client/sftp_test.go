package client

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/api"
)

// mockSSHSession is a test double for sshSession interface
type mockSSHSession struct {
	runFunc            func(cmd string) error
	outputFunc         func(cmd string) ([]byte, error)
	combinedOutputFunc func(cmd string) ([]byte, error)
	closeFunc          func() error
}

func (m *mockSSHSession) Run(cmd string) error {
	if m.runFunc != nil {
		return m.runFunc(cmd)
	}
	return nil
}

func (m *mockSSHSession) Output(cmd string) ([]byte, error) {
	if m.outputFunc != nil {
		return m.outputFunc(cmd)
	}
	return nil, nil
}

func (m *mockSSHSession) CombinedOutput(cmd string) ([]byte, error) {
	if m.combinedOutputFunc != nil {
		return m.combinedOutputFunc(cmd)
	}
	return nil, nil
}

func (m *mockSSHSession) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// mockSSHClientWrapper is a test double for sshClientWrapper interface
type mockSSHClientWrapper struct {
	newSessionFunc func() (sshSession, error)
	closeFunc      func() error
}

func (m *mockSSHClientWrapper) NewSession() (sshSession, error) {
	if m.newSessionFunc != nil {
		return m.newSessionFunc()
	}
	return nil, nil
}

func (m *mockSSHClientWrapper) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestSSHClient_WriteFile_Success(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	var capturedCmd string
	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			capturedCmd = cmd
			return []byte("true"), nil
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	params := WriteFileParams{
		Content: []byte("hello world"),
		Mode:    0644,
		UID:     IntPtr(1000),
		GID:     IntPtr(1000),
	}
	err := client.WriteFile(context.Background(), "/mnt/storage/test.txt", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify filesystem.file_receive was called
	if !strings.Contains(capturedCmd, "filesystem.file_receive") {
		t.Errorf("expected command to call filesystem.file_receive, got %q", capturedCmd)
	}

	// Verify path is in the command
	if !strings.Contains(capturedCmd, "/mnt/storage/test.txt") {
		t.Error("expected command to contain file path")
	}

	// Verify base64-encoded content is in the command (hello world -> aGVsbG8gd29ybGQ=)
	if !strings.Contains(capturedCmd, "aGVsbG8gd29ybGQ=") {
		t.Error("expected command to contain base64-encoded content")
	}
}

func TestSSHClient_WriteFile_Error(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			return []byte("[EPERM] permission denied"), errors.New("exit status 1")
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	params := DefaultWriteFileParams([]byte("hello"))
	err := client.WriteFile(context.Background(), "/mnt/storage/test.txt", params)
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

func TestSSHClient_ReadFile_Success(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	content := []byte("file content here")
	var capturedCmd string
	mockSession := &mockSSHSession{
		outputFunc: func(cmd string) ([]byte, error) {
			capturedCmd = cmd
			return content, nil
		},
	}

	mockSSHClient := &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	client.clientWrapper = mockSSHClient

	result, err := client.ReadFile(context.Background(), "/mnt/storage/test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != "file content here" {
		t.Errorf("expected 'file content here', got %q", string(result))
	}

	// Verify the command uses sudo cat with the path
	expectedCmd := "sudo cat /mnt/storage/test.txt"
	if capturedCmd != expectedCmd {
		t.Errorf("expected command %q, got %q", expectedCmd, capturedCmd)
	}
}

func TestSSHClient_ReadFile_NotFound(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	mockSession := &mockSSHSession{
		outputFunc: func(cmd string) ([]byte, error) {
			return nil, errors.New("cat: /mnt/storage/missing.txt: No such file or directory")
		},
	}

	mockSSHClient := &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	client.clientWrapper = mockSSHClient

	_, err := client.ReadFile(context.Background(), "/mnt/storage/missing.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSSHClient_DeleteFile_Success(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	var capturedCmd string
	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			capturedCmd = cmd
			return nil, nil
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	err := client.DeleteFile(context.Background(), "/mnt/storage/test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify sudo rm was called with the correct path
	if !strings.Contains(capturedCmd, "sudo") || !strings.Contains(capturedCmd, "rm") {
		t.Errorf("expected sudo rm command, got %q", capturedCmd)
	}
	if !strings.Contains(capturedCmd, "/mnt/storage/test.txt") {
		t.Errorf("expected path '/mnt/storage/test.txt' in command, got %q", capturedCmd)
	}
}

func TestSSHClient_DeleteFile_NotFound(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			return []byte("rm: cannot remove '/mnt/storage/missing.txt': No such file or directory"), errors.New("exit status 1")
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	err := client.DeleteFile(context.Background(), "/mnt/storage/missing.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSSHClient_FileExists_True(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			// Return valid JSON response from filesystem.stat
			return []byte(`{"type": "FILE", "mode": 33188}`), nil
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	exists, err := client.FileExists(context.Background(), "/mnt/storage/test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !exists {
		t.Error("expected file to exist")
	}
}

func TestSSHClient_FileExists_False(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			return []byte("[ENOENT] Path /mnt/storage/missing.txt not found"), errors.New("exit status 1")
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	exists, err := client.FileExists(context.Background(), "/mnt/storage/missing.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exists {
		t.Error("expected file to not exist")
	}
}

func TestSSHClient_MkdirAll_Success(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	var capturedCmd string
	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			capturedCmd = cmd
			return []byte("true"), nil
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	err := client.MkdirAll(context.Background(), "/mnt/storage/apps/myapp/config", 0755)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify filesystem.mkdir was called
	if !strings.Contains(capturedCmd, "filesystem.mkdir") {
		t.Errorf("expected filesystem.mkdir in command, got %q", capturedCmd)
	}
	if !strings.Contains(capturedCmd, "/mnt/storage/apps/myapp/config") {
		t.Errorf("expected path in command, got %q", capturedCmd)
	}
}

func TestSSHClient_MkdirAll_PermissionDenied(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			return []byte("[EPERM] Permission denied"), errors.New("exit status 1")
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	err := client.MkdirAll(context.Background(), "/mnt/storage/protected", 0755)
	if err == nil {
		t.Fatal("expected error for permission denied")
	}
}

func TestSSHClient_MkdirAll_IncludesMode(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	var capturedCmd string
	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			capturedCmd = cmd
			return []byte("true"), nil
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	err := client.MkdirAll(context.Background(), "/mnt/storage/apps/config", 0750)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify mode is included in the command (0750 in octal string format)
	if !strings.Contains(capturedCmd, "0750") {
		t.Errorf("expected mode '0750' in command, got %q", capturedCmd)
	}
}

func TestSSHClient_ReadFile_LargeFile(t *testing.T) {
	// Test that ReadFile handles large file content correctly
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	// Simulate a large file content
	fullContent := []byte("This is a large file that requires multiple reads to complete")

	mockSession := &mockSSHSession{
		outputFunc: func(cmd string) ([]byte, error) {
			return fullContent, nil
		},
	}

	mockSSHClient := &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	client.clientWrapper = mockSSHClient

	result, err := client.ReadFile(context.Background(), "/mnt/storage/large.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != string(fullContent) {
		t.Errorf("expected full content %q, got %q", string(fullContent), string(result))
	}
}

func TestSSHClient_RemoveDir_Success(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	var capturedCmd string
	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			capturedCmd = cmd
			return nil, nil
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	err := client.RemoveDir(context.Background(), "/mnt/storage/apps/myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify sudo rmdir was called
	if !strings.Contains(capturedCmd, "sudo") || !strings.Contains(capturedCmd, "rmdir") {
		t.Errorf("expected sudo rmdir command, got %q", capturedCmd)
	}
	if !strings.Contains(capturedCmd, "/mnt/storage/apps/myapp") {
		t.Errorf("expected path in command, got %q", capturedCmd)
	}
}

func TestSSHClient_RemoveDir_NotEmpty(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			return []byte("rmdir: failed to remove '/mnt/storage/apps/myapp': Directory not empty"), errors.New("exit status 1")
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	err := client.RemoveDir(context.Background(), "/mnt/storage/apps/myapp")
	if err == nil {
		t.Fatal("expected error for non-empty directory")
	}
}

func TestSSHClient_RemoveDir_NotFound(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			return []byte("rmdir: failed to remove '/mnt/storage/missing': No such file or directory"), errors.New("exit status 1")
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	err := client.RemoveDir(context.Background(), "/mnt/storage/missing")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestSSHClient_RemoveAll_Success(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	var capturedCmd string
	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			capturedCmd = cmd
			return nil, nil
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	err := client.RemoveAll(context.Background(), "/mnt/storage/apps/myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify sudo rm -rf was called
	if !strings.Contains(capturedCmd, "sudo") || !strings.Contains(capturedCmd, "rm") || !strings.Contains(capturedCmd, "-rf") {
		t.Errorf("expected sudo rm -rf command, got %q", capturedCmd)
	}
	if !strings.Contains(capturedCmd, "/mnt/storage/apps/myapp") {
		t.Errorf("expected path in command, got %q", capturedCmd)
	}
}

func TestSSHClient_RemoveAll_PermissionDenied(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			return []byte("rm: cannot remove '/mnt/storage/protected': Permission denied"), errors.New("exit status 1")
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	err := client.RemoveAll(context.Background(), "/mnt/storage/protected")
	if err == nil {
		t.Fatal("expected error for permission denied")
	}
}

func TestSSHClient_Chown_Success(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	// Pre-set version to 25.x for -j flag behavior
	client.version = api.Version{Major: 25, Minor: 4, Raw: "TrueNAS-25.04"}
	client.connected = true

	var capturedCmd string
	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			capturedCmd = cmd
			return nil, nil
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	err := client.Chown(context.Background(), "/mnt/storage/apps/config.yaml", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify filesystem.chown was called with the -j flag
	if !strings.Contains(capturedCmd, "filesystem.chown") {
		t.Errorf("expected filesystem.chown in command, got %q", capturedCmd)
	}
	if !strings.Contains(capturedCmd, "-j") {
		t.Errorf("expected -j flag in command, got %q", capturedCmd)
	}
	if !strings.Contains(capturedCmd, "/mnt/storage/apps/config.yaml") {
		t.Errorf("expected path in command, got %q", capturedCmd)
	}
}

func TestSSHClient_Chown_PermissionDenied(t *testing.T) {
	config := &SSHConfig{
		Host:               "truenas.local",
		PrivateKey:         testPrivateKey,
		HostKeyFingerprint: testHostKeyFingerprint,
	}

	client, _ := NewSSHClient(config)

	// Pre-set version to 25.x for -j flag behavior
	client.version = api.Version{Major: 25, Minor: 4, Raw: "TrueNAS-25.04"}
	client.connected = true

	mockSession := &mockSSHSession{
		combinedOutputFunc: func(cmd string) ([]byte, error) {
			return []byte("[EPERM] Operation not permitted"), errors.New("exit status 1")
		},
	}

	client.clientWrapper = &mockSSHClientWrapper{
		newSessionFunc: func() (sshSession, error) {
			return mockSession, nil
		},
	}

	err := client.Chown(context.Background(), "/mnt/storage/protected.txt", 1000, 1000)
	if err == nil {
		t.Fatal("expected error for permission denied")
	}
}
