package client

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/api"
)

func TestNewRateLimitedClient(t *testing.T) {
	mock := &MockClient{}

	client := NewRateLimitedClient(mock, 18, 3, &SSHRetryClassifier{})

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.client != mock {
		t.Error("expected client to wrap mock")
	}
	if client.maxRetries != 3 {
		t.Errorf("expected maxRetries=3, got %d", client.maxRetries)
	}
}

func TestNewRateLimitedClient_Defaults(t *testing.T) {
	mock := &MockClient{}

	// Test with zero values - should use defaults
	client := NewRateLimitedClient(mock, 0, 0, nil)

	if client.limiter == nil {
		t.Error("expected limiter to be initialized with default")
	}
	if client.classifier == nil {
		t.Error("expected classifier to default to SSHRetryClassifier")
	}
}

func TestRateLimitedClient_Call_Success(t *testing.T) {
	expected := json.RawMessage(`{"result": "ok"}`)
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return expected, nil
		},
	}

	client := NewRateLimitedClient(mock, 60, 3, &SSHRetryClassifier{})
	result, err := client.Call(context.Background(), "test.method", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != string(expected) {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestRateLimitedClient_Call_RetryOnTransientError(t *testing.T) {
	callCount := 0
	expected := json.RawMessage(`{"result": "ok"}`)

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			if callCount < 3 {
				return nil, errors.New("Failed connection handshake")
			}
			return expected, nil
		},
	}

	// Use high rate limit to avoid rate limiting delays in test
	client := NewRateLimitedClient(mock, 6000, 3, &SSHRetryClassifier{})
	result, err := client.Call(context.Background(), "test.method", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
	if string(result) != string(expected) {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestRateLimitedClient_Call_NoRetryOnValidationError(t *testing.T) {
	callCount := 0

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			return nil, errors.New("[EINVAL] name: invalid")
		},
	}

	client := NewRateLimitedClient(mock, 6000, 3, &SSHRetryClassifier{})
	_, err := client.Call(context.Background(), "test.method", nil)

	if err == nil {
		t.Fatal("expected error")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retries), got %d", callCount)
	}
}

func TestRateLimitedClient_Call_ExhaustedRetries(t *testing.T) {
	callCount := 0

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			return nil, errors.New("connection refused")
		},
	}

	client := NewRateLimitedClient(mock, 6000, 2, &SSHRetryClassifier{})
	_, err := client.Call(context.Background(), "test.method", nil)

	if err == nil {
		t.Fatal("expected error after exhausted retries")
	}
	// Initial call + 2 retries = 3 total
	if callCount != 3 {
		t.Errorf("expected 3 calls (1 + 2 retries), got %d", callCount)
	}
	if !strings.Contains(err.Error(), "after 2 retries") {
		t.Errorf("expected retry exhaustion error, got: %v", err)
	}
}

func TestRateLimitedClient_Call_ContextCancellation(t *testing.T) {
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return nil, errors.New("connection refused")
		},
	}

	client := NewRateLimitedClient(mock, 6000, 3, &SSHRetryClassifier{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Call(ctx, "test.method", nil)

	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestRateLimitedClient_Call_RetriesDisabled(t *testing.T) {
	callCount := 0

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			return nil, errors.New("connection refused")
		},
	}

	// maxRetries = 0 means no retries
	client := NewRateLimitedClient(mock, 6000, 0, &SSHRetryClassifier{})
	_, err := client.Call(context.Background(), "test.method", nil)

	if err == nil {
		t.Fatal("expected error")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (retries disabled), got %d", callCount)
	}
}

func TestRateLimitedClient_CallAndWait_Success(t *testing.T) {
	expected := json.RawMessage(`{"job": "complete"}`)
	mock := &MockClient{
		CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return expected, nil
		},
	}

	client := NewRateLimitedClient(mock, 60, 3, &SSHRetryClassifier{})
	result, err := client.CallAndWait(context.Background(), "test.method", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != string(expected) {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestRateLimitedClient_CallAndWait_RetryOnTransientError(t *testing.T) {
	callCount := 0
	expected := json.RawMessage(`{"job": "complete"}`)

	mock := &MockClient{
		CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			if callCount < 2 {
				return nil, errors.New("Unexpected closure of remote connection")
			}
			return expected, nil
		},
	}

	client := NewRateLimitedClient(mock, 6000, 3, &SSHRetryClassifier{})
	result, err := client.CallAndWait(context.Background(), "test.method", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
	if string(result) != string(expected) {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestRateLimitedClient_DelegatingMethods(t *testing.T) {
	ctx := context.Background()

	t.Run("Connect", func(t *testing.T) {
		called := false
		mock := &MockClient{
			ConnectFunc: func(ctx context.Context) error {
				called = true
				return nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		err := client.Connect(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected Connect to be called")
		}
	})

	t.Run("Version", func(t *testing.T) {
		expected := api.Version{Flavor: api.FlavorScale, Major: 24, Minor: 10}
		mock := &MockClient{
			VersionVal: expected,
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		result := client.Version()
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("WriteFile", func(t *testing.T) {
		called := false
		mock := &MockClient{
			WriteFileFunc: func(ctx context.Context, path string, params WriteFileParams) error {
				called = true
				return nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		err := client.WriteFile(ctx, "/test", DefaultWriteFileParams([]byte("content")))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected WriteFile to be called on underlying client")
		}
	})

	t.Run("ReadFile", func(t *testing.T) {
		expected := []byte("file content")
		mock := &MockClient{
			ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
				return expected, nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		result, err := client.ReadFile(ctx, "/test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result) != string(expected) {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("DeleteFile", func(t *testing.T) {
		called := false
		mock := &MockClient{
			DeleteFileFunc: func(ctx context.Context, path string) error {
				called = true
				return nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		err := client.DeleteFile(ctx, "/test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected DeleteFile to be called on underlying client")
		}
	})

	t.Run("RemoveDir", func(t *testing.T) {
		called := false
		mock := &MockClient{
			RemoveDirFunc: func(ctx context.Context, path string) error {
				called = true
				return nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		err := client.RemoveDir(ctx, "/test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected RemoveDir to be called on underlying client")
		}
	})

	t.Run("RemoveAll", func(t *testing.T) {
		called := false
		mock := &MockClient{
			RemoveAllFunc: func(ctx context.Context, path string) error {
				called = true
				return nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		err := client.RemoveAll(ctx, "/test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected RemoveAll to be called on underlying client")
		}
	})

	t.Run("FileExists", func(t *testing.T) {
		mock := &MockClient{
			FileExistsFunc: func(ctx context.Context, path string) (bool, error) {
				return true, nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		exists, err := client.FileExists(ctx, "/test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected FileExists to return true")
		}
	})

	t.Run("Chown", func(t *testing.T) {
		called := false
		mock := &MockClient{
			ChownFunc: func(ctx context.Context, path string, uid, gid int) error {
				called = true
				return nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		err := client.Chown(ctx, "/test", 1000, 1000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected Chown to be called on underlying client")
		}
	})

	t.Run("ChmodRecursive", func(t *testing.T) {
		called := false
		mock := &MockClient{
			ChmodRecursiveFunc: func(ctx context.Context, path string, mode fs.FileMode) error {
				called = true
				return nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		err := client.ChmodRecursive(ctx, "/test", 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected ChmodRecursive to be called on underlying client")
		}
	})

	t.Run("MkdirAll", func(t *testing.T) {
		called := false
		mock := &MockClient{
			MkdirAllFunc: func(ctx context.Context, path string, mode fs.FileMode) error {
				called = true
				return nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		err := client.MkdirAll(ctx, "/test/subdir", 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected MkdirAll to be called on underlying client")
		}
	})

	t.Run("Close", func(t *testing.T) {
		closed := false
		mock := &MockClient{
			CloseFunc: func() error {
				closed = true
				return nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		err := client.Close()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !closed {
			t.Error("expected Close to be called on underlying client")
		}
	})
}
