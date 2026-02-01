package client

import (
	"context"
	"encoding/json"
	"io/fs"

	"github.com/deevus/terraform-provider-truenas/internal/api"
)

// WriteFileParams contains parameters for writing a file.
type WriteFileParams struct {
	Content []byte      // Required - file data to write
	Mode    fs.FileMode // Default: 0644
	UID     *int        // nil = unchanged, pointer allows explicit 0 (root)
	GID     *int        // nil = unchanged, pointer allows explicit 0 (root)
}

// DefaultWriteFileParams returns params with sensible defaults.
// Mode defaults to 0644. UID/GID are nil (unchanged).
func DefaultWriteFileParams(content []byte) WriteFileParams {
	return WriteFileParams{
		Content: content,
		Mode:    0644,
		// UID and GID default to nil (zero value for *int)
	}
}

// IntPtr returns a pointer to an int. Helper for setting UID/GID.
func IntPtr(i int) *int { return &i }

// Client defines the interface for communicating with TrueNAS.
type Client interface {
	// Connect establishes connection and detects TrueNAS version.
	// Must be called before using the client.
	Connect(ctx context.Context) error

	// Version returns the cached TrueNAS version.
	// Panics if called before Connect() - fail fast on programmer error.
	Version() api.Version

	// Call executes a midclt command and returns the parsed JSON response.
	Call(ctx context.Context, method string, params any) (json.RawMessage, error)

	// CallAndWait executes a command and waits for job completion.
	CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error)

	// WriteFile writes content to a file on the remote system.
	WriteFile(ctx context.Context, path string, params WriteFileParams) error

	// ReadFile reads the content of a file from the remote system.
	ReadFile(ctx context.Context, path string) ([]byte, error)

	// DeleteFile removes a file from the remote system.
	DeleteFile(ctx context.Context, path string) error

	// RemoveDir removes an empty directory from the remote system.
	RemoveDir(ctx context.Context, path string) error

	// RemoveAll recursively removes a directory and all its contents.
	RemoveAll(ctx context.Context, path string) error

	// FileExists checks if a file exists on the remote system.
	FileExists(ctx context.Context, path string) (bool, error)

	// Chown changes the ownership of a file or directory.
	Chown(ctx context.Context, path string, uid, gid int) error

	// ChmodRecursive recursively changes permissions on a directory and all contents.
	ChmodRecursive(ctx context.Context, path string, mode fs.FileMode) error

	// MkdirAll creates a directory and all parent directories.
	MkdirAll(ctx context.Context, path string, mode fs.FileMode) error

	// Close closes the connection.
	Close() error
}

// MockClient is a test double for Client.
type MockClient struct {
	ConnectFunc        func(ctx context.Context) error
	VersionVal         api.Version
	CallFunc           func(ctx context.Context, method string, params any) (json.RawMessage, error)
	CallAndWaitFunc    func(ctx context.Context, method string, params any) (json.RawMessage, error)
	WriteFileFunc      func(ctx context.Context, path string, params WriteFileParams) error
	ReadFileFunc       func(ctx context.Context, path string) ([]byte, error)
	DeleteFileFunc     func(ctx context.Context, path string) error
	RemoveDirFunc      func(ctx context.Context, path string) error
	RemoveAllFunc      func(ctx context.Context, path string) error
	FileExistsFunc     func(ctx context.Context, path string) (bool, error)
	ChownFunc          func(ctx context.Context, path string, uid, gid int) error
	ChmodRecursiveFunc func(ctx context.Context, path string, mode fs.FileMode) error
	MkdirAllFunc       func(ctx context.Context, path string, mode fs.FileMode) error
	CloseFunc          func() error
}

func (m *MockClient) Connect(ctx context.Context) error {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx)
	}
	return nil
}

func (m *MockClient) Version() api.Version {
	return m.VersionVal
}

func (m *MockClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if m.CallFunc != nil {
		return m.CallFunc(ctx, method, params)
	}
	return nil, nil
}

func (m *MockClient) CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if m.CallAndWaitFunc != nil {
		return m.CallAndWaitFunc(ctx, method, params)
	}
	return nil, nil
}

func (m *MockClient) WriteFile(ctx context.Context, path string, params WriteFileParams) error {
	if m.WriteFileFunc != nil {
		return m.WriteFileFunc(ctx, path, params)
	}
	return nil
}

func (m *MockClient) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(ctx, path)
	}
	return nil, nil
}

func (m *MockClient) DeleteFile(ctx context.Context, path string) error {
	if m.DeleteFileFunc != nil {
		return m.DeleteFileFunc(ctx, path)
	}
	return nil
}

func (m *MockClient) RemoveDir(ctx context.Context, path string) error {
	if m.RemoveDirFunc != nil {
		return m.RemoveDirFunc(ctx, path)
	}
	return nil
}

func (m *MockClient) RemoveAll(ctx context.Context, path string) error {
	if m.RemoveAllFunc != nil {
		return m.RemoveAllFunc(ctx, path)
	}
	return nil
}

func (m *MockClient) FileExists(ctx context.Context, path string) (bool, error) {
	if m.FileExistsFunc != nil {
		return m.FileExistsFunc(ctx, path)
	}
	return false, nil
}

func (m *MockClient) Chown(ctx context.Context, path string, uid, gid int) error {
	if m.ChownFunc != nil {
		return m.ChownFunc(ctx, path, uid, gid)
	}
	return nil
}

func (m *MockClient) ChmodRecursive(ctx context.Context, path string, mode fs.FileMode) error {
	if m.ChmodRecursiveFunc != nil {
		return m.ChmodRecursiveFunc(ctx, path, mode)
	}
	return nil
}

func (m *MockClient) MkdirAll(ctx context.Context, path string, mode fs.FileMode) error {
	if m.MkdirAllFunc != nil {
		return m.MkdirAllFunc(ctx, path, mode)
	}
	return nil
}

func (m *MockClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
