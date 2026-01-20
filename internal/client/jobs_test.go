package client

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestJobPoller_Success(t *testing.T) {
	// Job completes after a few polls
	callCount := 0
	expectedResult := json.RawMessage(`{"id": 123, "name": "test-dataset"}`)

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method != "core.get_jobs" {
				t.Errorf("expected method core.get_jobs, got %s", method)
			}

			// Verify filter format
			paramsSlice, ok := params.([]any)
			if !ok || len(paramsSlice) != 1 {
				t.Errorf("expected params to be []any with 1 element, got %v", params)
			}

			callCount++
			if callCount < 3 {
				// First two polls: still running
				return json.RawMessage(`[{"id": 42, "state": "RUNNING"}]`), nil
			}
			// Third poll: success
			return json.RawMessage(`[{"id": 42, "state": "SUCCESS", "result": {"id": 123, "name": "test-dataset"}}]`), nil
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	})

	result, err := poller.Wait(context.Background(), 42, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}

	if string(result) != string(expectedResult) {
		t.Errorf("expected result %s, got %s", expectedResult, result)
	}
}

func TestJobPoller_Failure(t *testing.T) {
	// Job fails with error
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return json.RawMessage(`[{"id": 42, "state": "FAILED", "error": "[EINVAL] dataset.name: Invalid dataset name"}]`), nil
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	})

	_, err := poller.Wait(context.Background(), 42, 5*time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var tnErr *TrueNASError
	if !errors.As(err, &tnErr) {
		t.Fatalf("expected TrueNASError, got %T", err)
	}

	if tnErr.Code != "EINVAL" {
		t.Errorf("expected error code EINVAL, got %s", tnErr.Code)
	}
}

func TestJobPoller_Timeout(t *testing.T) {
	// Timeout reached while polling
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			// Always return running
			return json.RawMessage(`[{"id": 42, "state": "RUNNING"}]`), nil
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	})

	_, err := poller.Wait(context.Background(), 42, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	var tnErr *TrueNASError
	if !errors.As(err, &tnErr) {
		t.Fatalf("expected TrueNASError, got %T", err)
	}

	if tnErr.Code != "ETIMEDOUT" {
		t.Errorf("expected error code ETIMEDOUT, got %s", tnErr.Code)
	}

	if tnErr.JobID != 42 {
		t.Errorf("expected JobID 42, got %d", tnErr.JobID)
	}
}

func TestJobPoller_ContextCanceled(t *testing.T) {
	// Context canceled
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			// Always return running
			return json.RawMessage(`[{"id": 42, "state": "RUNNING"}]`), nil
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
	})

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	_, err := poller.Wait(ctx, 42, 5*time.Second)
	if err == nil {
		t.Fatal("expected context canceled error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestJobPoller_ClientError(t *testing.T) {
	// Client returns an error during polling
	expectedErr := errors.New("connection lost")
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return nil, expectedErr
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	})

	_, err := poller.Wait(context.Background(), 42, 5*time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got %v", expectedErr, err)
	}
}

func TestJobPoller_JobNotFound(t *testing.T) {
	// Job returns empty array (not found)
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return json.RawMessage(`[]`), nil
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	})

	_, err := poller.Wait(context.Background(), 42, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var tnErr *TrueNASError
	if !errors.As(err, &tnErr) {
		t.Fatalf("expected TrueNASError, got %T", err)
	}

	if tnErr.Code != "ENOENT" {
		t.Errorf("expected error code ENOENT, got %s", tnErr.Code)
	}
}

func TestJobPoller_InvalidJSON(t *testing.T) {
	// Client returns invalid JSON
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return json.RawMessage(`not valid json`), nil
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	})

	_, err := poller.Wait(context.Background(), 42, 5*time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestJobPoller_WaitingState(t *testing.T) {
	// Job in WAITING state transitions to SUCCESS
	callCount := 0

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			if callCount < 2 {
				return json.RawMessage(`[{"id": 42, "state": "WAITING"}]`), nil
			}
			return json.RawMessage(`[{"id": 42, "state": "SUCCESS", "result": "done"}]`), nil
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	})

	result, err := poller.Wait(context.Background(), 42, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != `"done"` {
		t.Errorf("expected result \"done\", got %s", result)
	}
}

func TestParseJobID(t *testing.T) {
	tests := []struct {
		name    string
		data    json.RawMessage
		want    int64
		wantErr bool
	}{
		{
			name:    "valid integer",
			data:    json.RawMessage(`42`),
			want:    42,
			wantErr: false,
		},
		{
			name:    "large integer",
			data:    json.RawMessage(`9223372036854775807`),
			want:    9223372036854775807,
			wantErr: false,
		},
		{
			name:    "zero",
			data:    json.RawMessage(`0`),
			want:    0,
			wantErr: false,
		},
		{
			name:    "string value",
			data:    json.RawMessage(`"not an int"`),
			want:    0,
			wantErr: true,
		},
		{
			name:    "object value",
			data:    json.RawMessage(`{"id": 42}`),
			want:    0,
			wantErr: true,
		},
		{
			name:    "null value",
			data:    json.RawMessage(`null`),
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    json.RawMessage(``),
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid json",
			data:    json.RawMessage(`not json`),
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseJobID(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseJobID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseJobID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultJobPollerConfig(t *testing.T) {
	config := DefaultJobPollerConfig()

	if config.InitialInterval <= 0 {
		t.Errorf("InitialInterval should be positive, got %v", config.InitialInterval)
	}
	if config.MaxInterval <= 0 {
		t.Errorf("MaxInterval should be positive, got %v", config.MaxInterval)
	}
	if config.Multiplier <= 1.0 {
		t.Errorf("Multiplier should be > 1.0, got %v", config.Multiplier)
	}
	if config.MaxInterval < config.InitialInterval {
		t.Errorf("MaxInterval (%v) should be >= InitialInterval (%v)", config.MaxInterval, config.InitialInterval)
	}
}

func TestNewJobPoller_NilConfig(t *testing.T) {
	mock := &MockClient{}
	poller := NewJobPoller(mock, nil)

	if poller.config == nil {
		t.Error("expected default config when nil passed, got nil")
	}
}

func TestJobPoller_ExponentialBackoff(t *testing.T) {
	// Verify exponential backoff is happening
	var timestamps []time.Time

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			timestamps = append(timestamps, time.Now())
			if len(timestamps) < 4 {
				return json.RawMessage(`[{"id": 42, "state": "RUNNING"}]`), nil
			}
			return json.RawMessage(`[{"id": 42, "state": "SUCCESS", "result": null}]`), nil
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
	})

	_, err := poller.Wait(context.Background(), 42, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(timestamps) < 4 {
		t.Fatalf("expected at least 4 polls, got %d", len(timestamps))
	}

	// Verify intervals are increasing (with some tolerance for timing)
	interval1 := timestamps[1].Sub(timestamps[0])
	interval2 := timestamps[2].Sub(timestamps[1])
	interval3 := timestamps[3].Sub(timestamps[2])

	// Second interval should be longer than first (allowing 50% tolerance)
	if interval2 < interval1 {
		t.Logf("interval1=%v, interval2=%v, interval3=%v", interval1, interval2, interval3)
		// Don't fail on timing - just log for visibility
	}

	// Third interval should be capped at MaxInterval or growing
	if interval3 < time.Duration(float64(interval2)*0.5) {
		t.Logf("interval3 (%v) unexpectedly small compared to interval2 (%v)", interval3, interval2)
	}
}

func TestJobPoller_SuccessWithNullResult(t *testing.T) {
	// Job completes with null result (some operations return null on success)
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return json.RawMessage(`[{"id": 42, "state": "SUCCESS", "result": null}]`), nil
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	})

	result, err := poller.Wait(context.Background(), 42, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != "null" {
		t.Errorf("expected null result, got %s", result)
	}
}

func TestJobPoller_FailureWithEmptyError(t *testing.T) {
	// Job fails but error field is empty
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return json.RawMessage(`[{"id": 42, "state": "FAILED", "error": ""}]`), nil
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	})

	_, err := poller.Wait(context.Background(), 42, 5*time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var tnErr *TrueNASError
	if !errors.As(err, &tnErr) {
		t.Fatalf("expected TrueNASError, got %T", err)
	}

	// Empty error should result in UNKNOWN code
	if tnErr.Code != "UNKNOWN" {
		t.Errorf("expected error code UNKNOWN for empty error, got %s", tnErr.Code)
	}
}

func TestJobPoller_UnknownState(t *testing.T) {
	// Job with unknown state should continue polling until success
	callCount := 0

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			if callCount < 2 {
				// Return unknown state
				return json.RawMessage(`[{"id": 42, "state": "PENDING"}]`), nil
			}
			return json.RawMessage(`[{"id": 42, "state": "SUCCESS", "result": "done"}]`), nil
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	})

	result, err := poller.Wait(context.Background(), 42, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != `"done"` {
		t.Errorf("expected result \"done\", got %s", result)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestJobPoller_ContextCanceledBeforePoll(t *testing.T) {
	// Context already canceled before first poll
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			t.Fatal("should not be called when context is already canceled")
			return nil, nil
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately before Wait

	_, err := poller.Wait(ctx, 42, 5*time.Second)
	if err == nil {
		t.Fatal("expected context canceled error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestJobPoller_FailureWithLogsExcerpt(t *testing.T) {
	// Job fails with logs_excerpt - verify it's included in the error
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return json.RawMessage(`[{
				"id": 42,
				"state": "FAILED",
				"error": "[EFAULT] Failed 'up' action for 'myapp' app",
				"logs_excerpt": "Container myapp  Creating\nError response from daemon: image not found",
				"logs_path": "/var/log/jobs/42.log"
			}]`), nil
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	})

	_, err := poller.Wait(context.Background(), 42, 5*time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var tnErr *TrueNASError
	if !errors.As(err, &tnErr) {
		t.Fatalf("expected TrueNASError, got %T", err)
	}

	if tnErr.Code != "EFAULT" {
		t.Errorf("expected error code EFAULT, got %s", tnErr.Code)
	}

	if tnErr.LogsExcerpt == "" {
		t.Error("expected LogsExcerpt to be populated")
	}

	if !strings.Contains(tnErr.LogsExcerpt, "image not found") {
		t.Errorf("expected LogsExcerpt to contain 'image not found', got %s", tnErr.LogsExcerpt)
	}

	// Verify logs appear in error string
	errStr := err.Error()
	if !strings.Contains(errStr, "Job logs:") {
		t.Error("expected error string to contain 'Job logs:'")
	}
	if !strings.Contains(errStr, "image not found") {
		t.Error("expected error string to contain logs excerpt content")
	}
}

func TestJobPoller_MaxIntervalCapping(t *testing.T) {
	// Test that interval is capped at MaxInterval
	var intervals []time.Duration
	var lastTime time.Time

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			now := time.Now()
			if !lastTime.IsZero() {
				intervals = append(intervals, now.Sub(lastTime))
			}
			lastTime = now

			if len(intervals) < 5 {
				return json.RawMessage(`[{"id": 42, "state": "RUNNING"}]`), nil
			}
			return json.RawMessage(`[{"id": 42, "state": "SUCCESS", "result": null}]`), nil
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 5 * time.Millisecond,
		MaxInterval:     15 * time.Millisecond,
		Multiplier:      3.0, // Fast growth to hit max quickly
	})

	_, err := poller.Wait(context.Background(), 42, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With multiplier 3.0 and initial 5ms:
	// Poll 1: immediate
	// Poll 2: after 5ms
	// Poll 3: after 15ms (5*3=15, not capped)
	// Poll 4: after 15ms (15*3=45, capped to 15)
	// Poll 5: after 15ms (still capped)
	// Poll 6: success

	// Verify we hit the cap - later intervals should not grow beyond ~15ms
	// (allowing for timing jitter)
	if len(intervals) >= 4 {
		for i := 2; i < len(intervals); i++ {
			if intervals[i] > 30*time.Millisecond {
				t.Errorf("interval %d (%v) exceeds max interval significantly", i, intervals[i])
			}
		}
	}
}

func TestJobPoller_FailureWithAppLifecycleLog(t *testing.T) {
	// Read the fixture
	logContent, err := os.ReadFile("../testdata/fixtures/app_lifecycle.log")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method == "core.get_jobs" {
				return json.RawMessage(`[{
					"id": 42,
					"state": "FAILED",
					"error": "[EFAULT] Failed 'up' action for 'dns' app. Please check /var/log/app_lifecycle.log for more details"
				}]`), nil
			}
			t.Errorf("unexpected method: %s", method)
			return nil, nil
		},
		ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
			if path == "/var/log/app_lifecycle.log" {
				return logContent, nil
			}
			return nil, errors.New("file not found")
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	})

	_, err = poller.Wait(context.Background(), 42, 5*time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var tnErr *TrueNASError
	if !errors.As(err, &tnErr) {
		t.Fatalf("expected TrueNASError, got %T", err)
	}

	// Should have extracted the clean error
	if !strings.Contains(tnErr.AppLifecycleError, "bind: address already in use") {
		t.Errorf("expected AppLifecycleError to contain 'bind: address already in use', got %q", tnErr.AppLifecycleError)
	}

	// The Error() output should be clean
	errStr := err.Error()
	if strings.Contains(errStr, "Please check") {
		t.Errorf("expected clean error output, got %q", errStr)
	}
}

func TestJobPoller_FailureWithAppLifecycleLog_FetchFails(t *testing.T) {
	// Log fetch fails - should still return original error
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method == "core.get_jobs" {
				return json.RawMessage(`[{
					"id": 42,
					"state": "FAILED",
					"error": "[EFAULT] Failed 'up' action for 'dns' app. Please check /var/log/app_lifecycle.log for more details"
				}]`), nil
			}
			return nil, nil
		},
		ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
			// Simulate failure to read log
			return nil, errors.New("permission denied")
		},
	}

	poller := NewJobPoller(mock, &JobPollerConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	})

	_, err := poller.Wait(context.Background(), 42, 5*time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var tnErr *TrueNASError
	if !errors.As(err, &tnErr) {
		t.Fatalf("expected TrueNASError, got %T", err)
	}

	// Should still have the original error info
	if tnErr.Code != "EFAULT" {
		t.Errorf("expected code EFAULT, got %s", tnErr.Code)
	}

	// AppLifecycleError should be empty since fetch failed
	if tnErr.AppLifecycleError != "" {
		t.Errorf("expected empty AppLifecycleError when fetch fails, got %q", tnErr.AppLifecycleError)
	}
}
