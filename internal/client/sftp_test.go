package client

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"time"
)

// mockSFTPClient is a test double for sftpClient interface
type mockSFTPClient struct {
	createFunc    func(path string) (sftpFile, error)
	mkdirAllFunc  func(path string) error
	statFunc      func(path string) (fs.FileInfo, error)
	removeFunc    func(path string) error
	removeDirFunc func(path string) error
	removeAllFunc func(path string) error
	openFunc      func(path string) (sftpFile, error)
	chmodFunc     func(path string, mode fs.FileMode) error
	chownFunc     func(path string, uid, gid int) error
	readDirFunc   func(path string) ([]fs.FileInfo, error)
	closeFunc     func() error
}

func (m *mockSFTPClient) Create(path string) (sftpFile, error) {
	if m.createFunc != nil {
		return m.createFunc(path)
	}
	return nil, nil
}

func (m *mockSFTPClient) MkdirAll(path string) error {
	if m.mkdirAllFunc != nil {
		return m.mkdirAllFunc(path)
	}
	return nil
}

func (m *mockSFTPClient) Stat(path string) (fs.FileInfo, error) {
	if m.statFunc != nil {
		return m.statFunc(path)
	}
	return nil, nil
}

func (m *mockSFTPClient) Remove(path string) error {
	if m.removeFunc != nil {
		return m.removeFunc(path)
	}
	return nil
}

func (m *mockSFTPClient) RemoveDirectory(path string) error {
	if m.removeDirFunc != nil {
		return m.removeDirFunc(path)
	}
	return nil
}

func (m *mockSFTPClient) RemoveAll(path string) error {
	if m.removeAllFunc != nil {
		return m.removeAllFunc(path)
	}
	return nil
}

func (m *mockSFTPClient) Open(path string) (sftpFile, error) {
	if m.openFunc != nil {
		return m.openFunc(path)
	}
	return nil, nil
}

func (m *mockSFTPClient) Chmod(path string, mode fs.FileMode) error {
	if m.chmodFunc != nil {
		return m.chmodFunc(path, mode)
	}
	return nil
}

func (m *mockSFTPClient) Chown(path string, uid, gid int) error {
	if m.chownFunc != nil {
		return m.chownFunc(path, uid, gid)
	}
	return nil
}

func (m *mockSFTPClient) ReadDir(path string) ([]fs.FileInfo, error) {
	if m.readDirFunc != nil {
		return m.readDirFunc(path)
	}
	return nil, nil
}

func (m *mockSFTPClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// mockSFTPFile is a test double for sftpFile interface
type mockSFTPFile struct {
	writeFunc func(p []byte) (int, error)
	readFunc  func(p []byte) (int, error)
	closeFunc func() error
}

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

func (m *mockSFTPFile) Write(p []byte) (int, error) {
	if m.writeFunc != nil {
		return m.writeFunc(p)
	}
	return len(p), nil
}

func (m *mockSFTPFile) Read(p []byte) (int, error) {
	if m.readFunc != nil {
		return m.readFunc(p)
	}
	return 0, nil
}

func (m *mockSFTPFile) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestSSHClient_WriteFile_Success(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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

	err := client.WriteFile(context.Background(), "/mnt/storage/test.txt", []byte("hello world"), 0644, 1000, 1000)
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
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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

	err := client.WriteFile(context.Background(), "/mnt/storage/test.txt", []byte("hello"), 0644, -1, -1)
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

// mockFileInfo implements fs.FileInfo for testing
type mockFileInfo struct {
	size int64
}

func (m *mockFileInfo) Name() string       { return "test" }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() fs.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() any           { return nil }

func TestSSHClient_ReadFile_Success(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	content := []byte("file content here")
	readDone := false
	mockFile := &mockSFTPFile{
		readFunc: func(p []byte) (int, error) {
			if readDone {
				return 0, io.EOF
			}
			n := copy(p, content)
			readDone = true
			return n, nil
		},
	}

	mockSFTP := &mockSFTPClient{
		openFunc: func(path string) (sftpFile, error) {
			if path != "/mnt/storage/test.txt" {
				t.Errorf("expected path '/mnt/storage/test.txt', got %q", path)
			}
			return mockFile, nil
		},
	}

	client.sftpClient = mockSFTP

	result, err := client.ReadFile(context.Background(), "/mnt/storage/test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != "file content here" {
		t.Errorf("expected 'file content here', got %q", string(result))
	}
}

func TestSSHClient_ReadFile_NotFound(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockSFTP := &mockSFTPClient{
		openFunc: func(path string) (sftpFile, error) {
			return nil, errors.New("file not found")
		},
	}

	client.sftpClient = mockSFTP

	_, err := client.ReadFile(context.Background(), "/mnt/storage/missing.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSSHClient_DeleteFile_Success(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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

func TestSSHClient_ReadFile_PartialReads(t *testing.T) {
	// Test that ReadFile handles partial reads correctly (large files)
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	// Simulate a large file where Read returns partial data
	fullContent := []byte("This is a large file that requires multiple reads to complete")
	readOffset := 0

	mockFile := &mockSFTPFile{
		readFunc: func(p []byte) (int, error) {
			// Simulate partial read - only return 10 bytes at a time
			remaining := len(fullContent) - readOffset
			if remaining == 0 {
				return 0, io.EOF
			}
			chunkSize := 10
			if chunkSize > remaining {
				chunkSize = remaining
			}
			if chunkSize > len(p) {
				chunkSize = len(p)
			}
			copy(p, fullContent[readOffset:readOffset+chunkSize])
			readOffset += chunkSize
			return chunkSize, nil
		},
	}

	mockSFTP := &mockSFTPClient{
		openFunc: func(path string) (sftpFile, error) {
			return mockFile, nil
		},
		statFunc: func(path string) (fs.FileInfo, error) {
			return &mockFileInfo{size: int64(len(fullContent))}, nil
		},
	}

	client.sftpClient = mockSFTP

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
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
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
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

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
