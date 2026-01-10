package client

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"
)

// mockSFTPClient is a test double for sftpClient interface
type mockSFTPClient struct {
	createFunc   func(path string) (sftpFile, error)
	mkdirAllFunc func(path string) error
	statFunc     func(path string) (fs.FileInfo, error)
	removeFunc   func(path string) error
	openFunc     func(path string) (sftpFile, error)
	chmodFunc    func(path string, mode fs.FileMode) error
	closeFunc    func() error
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

	var writtenContent []byte
	var writtenPath string

	mockFile := &mockSFTPFile{
		writeFunc: func(p []byte) (int, error) {
			writtenContent = p
			return len(p), nil
		},
	}

	mockSFTP := &mockSFTPClient{
		createFunc: func(path string) (sftpFile, error) {
			writtenPath = path
			return mockFile, nil
		},
		chmodFunc: func(path string, mode fs.FileMode) error {
			return nil
		},
	}

	client.sftpClient = mockSFTP

	err := client.WriteFile(context.Background(), "/mnt/storage/test.txt", []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if writtenPath != "/mnt/storage/test.txt" {
		t.Errorf("expected path '/mnt/storage/test.txt', got %q", writtenPath)
	}

	if string(writtenContent) != "hello world" {
		t.Errorf("expected content 'hello world', got %q", string(writtenContent))
	}
}

func TestSSHClient_WriteFile_CreateError(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockSFTP := &mockSFTPClient{
		createFunc: func(path string) (sftpFile, error) {
			return nil, errors.New("permission denied")
		},
	}

	client.sftpClient = mockSFTP

	err := client.WriteFile(context.Background(), "/mnt/storage/test.txt", []byte("hello"), 0644)
	if err == nil {
		t.Fatal("expected error for create failure")
	}
}

func TestSSHClient_WriteFile_WriteError(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockFile := &mockSFTPFile{
		writeFunc: func(p []byte) (int, error) {
			return 0, errors.New("disk full")
		},
	}

	mockSFTP := &mockSFTPClient{
		createFunc: func(path string) (sftpFile, error) {
			return mockFile, nil
		},
	}

	client.sftpClient = mockSFTP

	err := client.WriteFile(context.Background(), "/mnt/storage/test.txt", []byte("hello"), 0644)
	if err == nil {
		t.Fatal("expected error for write failure")
	}
}

func TestSSHClient_WriteFile_ChmodError(t *testing.T) {
	config := &SSHConfig{
		Host:       "truenas.local",
		PrivateKey: testPrivateKey,
	}

	client, _ := NewSSHClient(config)

	mockFile := &mockSFTPFile{
		writeFunc: func(p []byte) (int, error) {
			return len(p), nil
		},
	}

	mockSFTP := &mockSFTPClient{
		createFunc: func(path string) (sftpFile, error) {
			return mockFile, nil
		},
		chmodFunc: func(path string, mode fs.FileMode) error {
			return errors.New("operation not permitted")
		},
	}

	client.sftpClient = mockSFTP

	err := client.WriteFile(context.Background(), "/mnt/storage/test.txt", []byte("hello"), 0644)
	if err == nil {
		t.Fatal("expected error for chmod failure")
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
	mockFile := &mockSFTPFile{
		readFunc: func(p []byte) (int, error) {
			copy(p, content)
			return len(content), nil
		},
	}

	mockSFTP := &mockSFTPClient{
		openFunc: func(path string) (sftpFile, error) {
			if path != "/mnt/storage/test.txt" {
				t.Errorf("expected path '/mnt/storage/test.txt', got %q", path)
			}
			return mockFile, nil
		},
		statFunc: func(path string) (fs.FileInfo, error) {
			return &mockFileInfo{size: int64(len(content))}, nil
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
