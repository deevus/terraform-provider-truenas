package client

import (
	"context"
	"encoding/json"
	"io/fs"
	"testing"
)

func TestMockClient_Call(t *testing.T) {
	expected := json.RawMessage(`{"name": "test"}`)

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method != "app.query" {
				t.Errorf("expected method app.query, got %s", method)
			}
			return expected, nil
		},
	}

	result, err := mock.Call(context.Background(), "app.query", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != string(expected) {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestMockClient_CallAndWait(t *testing.T) {
	expected := json.RawMessage(`{"status": "complete"}`)

	mock := &MockClient{
		CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return expected, nil
		},
	}

	result, err := mock.CallAndWait(context.Background(), "app.create", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != string(expected) {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestMockClient_Call_NilFunc(t *testing.T) {
	mock := &MockClient{}

	result, err := mock.Call(context.Background(), "app.query", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil result, got %s", result)
	}
}

func TestMockClient_CallAndWait_NilFunc(t *testing.T) {
	mock := &MockClient{}

	result, err := mock.CallAndWait(context.Background(), "app.create", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil result, got %s", result)
	}
}

func TestMockClient_Close(t *testing.T) {
	closeCalled := false
	mock := &MockClient{
		CloseFunc: func() error {
			closeCalled = true
			return nil
		},
	}

	err := mock.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !closeCalled {
		t.Error("expected CloseFunc to be called")
	}
}

func TestMockClient_Close_NilFunc(t *testing.T) {
	mock := &MockClient{}

	err := mock.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMockClient_WriteFile_NilFunc(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()

	params := DefaultWriteFileParams([]byte("content"))
	err := mock.WriteFile(ctx, "/test", params)
	if err != nil {
		t.Errorf("expected nil error for nil WriteFileFunc, got %v", err)
	}
}

func TestMockClient_WriteFile_WithFunc(t *testing.T) {
	called := false
	mock := &MockClient{
		WriteFileFunc: func(ctx context.Context, path string, params WriteFileParams) error {
			called = true
			return nil
		},
	}
	ctx := context.Background()

	params := DefaultWriteFileParams([]byte("content"))
	err := mock.WriteFile(ctx, "/test", params)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected WriteFileFunc to be called")
	}
}

func TestMockClient_ReadFile_NilFunc(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()

	result, err := mock.ReadFile(ctx, "/test")
	if err != nil {
		t.Errorf("expected nil error for nil ReadFileFunc, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for nil ReadFileFunc, got %v", result)
	}
}

func TestMockClient_ReadFile_WithFunc(t *testing.T) {
	expected := []byte("file content")
	mock := &MockClient{
		ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
			return expected, nil
		},
	}
	ctx := context.Background()

	result, err := mock.ReadFile(ctx, "/test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(result) != string(expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestMockClient_DeleteFile_NilFunc(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()

	err := mock.DeleteFile(ctx, "/test")
	if err != nil {
		t.Errorf("expected nil error for nil DeleteFileFunc, got %v", err)
	}
}

func TestMockClient_DeleteFile_WithFunc(t *testing.T) {
	called := false
	mock := &MockClient{
		DeleteFileFunc: func(ctx context.Context, path string) error {
			called = true
			return nil
		},
	}
	ctx := context.Background()

	err := mock.DeleteFile(ctx, "/test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected DeleteFileFunc to be called")
	}
}

func TestMockClient_FileExists_NilFunc(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()

	exists, err := mock.FileExists(ctx, "/test")
	if err != nil {
		t.Errorf("expected nil error for nil FileExistsFunc, got %v", err)
	}
	if exists {
		t.Error("expected false for nil FileExistsFunc")
	}
}

func TestMockClient_FileExists_WithFunc(t *testing.T) {
	mock := &MockClient{
		FileExistsFunc: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
	}
	ctx := context.Background()

	exists, err := mock.FileExists(ctx, "/test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected true from FileExistsFunc")
	}
}

func TestMockClient_MkdirAll_NilFunc(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()

	err := mock.MkdirAll(ctx, "/test", 0755)
	if err != nil {
		t.Errorf("expected nil error for nil MkdirAllFunc, got %v", err)
	}
}

func TestMockClient_MkdirAll_WithFunc(t *testing.T) {
	called := false
	mock := &MockClient{
		MkdirAllFunc: func(ctx context.Context, path string, mode fs.FileMode) error {
			called = true
			return nil
		},
	}
	ctx := context.Background()

	err := mock.MkdirAll(ctx, "/test", 0755)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected MkdirAllFunc to be called")
	}
}

func TestMockClient_Connect_NilFunc(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()

	err := mock.Connect(ctx)
	if err != nil {
		t.Errorf("expected nil error for nil ConnectFunc, got %v", err)
	}
}

func TestMockClient_Connect_WithFunc(t *testing.T) {
	called := false
	mock := &MockClient{
		ConnectFunc: func(ctx context.Context) error {
			called = true
			return nil
		},
	}
	ctx := context.Background()

	err := mock.Connect(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected ConnectFunc to be called")
	}
}

func TestMockClient_RemoveDir_NilFunc(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()

	err := mock.RemoveDir(ctx, "/test")
	if err != nil {
		t.Errorf("expected nil error for nil RemoveDirFunc, got %v", err)
	}
}

func TestMockClient_RemoveDir_WithFunc(t *testing.T) {
	called := false
	mock := &MockClient{
		RemoveDirFunc: func(ctx context.Context, path string) error {
			called = true
			return nil
		},
	}
	ctx := context.Background()

	err := mock.RemoveDir(ctx, "/test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected RemoveDirFunc to be called")
	}
}

func TestMockClient_RemoveAll_NilFunc(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()

	err := mock.RemoveAll(ctx, "/test")
	if err != nil {
		t.Errorf("expected nil error for nil RemoveAllFunc, got %v", err)
	}
}

func TestMockClient_RemoveAll_WithFunc(t *testing.T) {
	called := false
	mock := &MockClient{
		RemoveAllFunc: func(ctx context.Context, path string) error {
			called = true
			return nil
		},
	}
	ctx := context.Background()

	err := mock.RemoveAll(ctx, "/test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected RemoveAllFunc to be called")
	}
}

func TestMockClient_Chown_NilFunc(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()

	err := mock.Chown(ctx, "/test", 1000, 1000)
	if err != nil {
		t.Errorf("expected nil error for nil ChownFunc, got %v", err)
	}
}

func TestMockClient_Chown_WithFunc(t *testing.T) {
	called := false
	mock := &MockClient{
		ChownFunc: func(ctx context.Context, path string, uid, gid int) error {
			called = true
			return nil
		},
	}
	ctx := context.Background()

	err := mock.Chown(ctx, "/test", 1000, 1000)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected ChownFunc to be called")
	}
}

func TestMockClient_ChmodRecursive_NilFunc(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()

	err := mock.ChmodRecursive(ctx, "/test", 0755)
	if err != nil {
		t.Errorf("expected nil error for nil ChmodRecursiveFunc, got %v", err)
	}
}

func TestMockClient_ChmodRecursive_WithFunc(t *testing.T) {
	called := false
	mock := &MockClient{
		ChmodRecursiveFunc: func(ctx context.Context, path string, mode fs.FileMode) error {
			called = true
			return nil
		},
	}
	ctx := context.Background()

	err := mock.ChmodRecursive(ctx, "/test", 0755)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected ChmodRecursiveFunc to be called")
	}
}
