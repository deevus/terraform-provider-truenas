package resources

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNormalizeDesiredState(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"running", "RUNNING"},
		{"RUNNING", "RUNNING"},
		{"Running", "RUNNING"},
		{"stopped", "STOPPED"},
		{"STOPPED", "STOPPED"},
		{"  running  ", "RUNNING"},
	}

	for _, tc := range tests {
		got := normalizeDesiredState(tc.input)
		if got != tc.expected {
			t.Errorf("normalizeDesiredState(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestIsStableState(t *testing.T) {
	tests := []struct {
		state    string
		expected bool
	}{
		{AppStateRunning, true},
		{AppStateStopped, true},
		{AppStateCrashed, true},
		{AppStateDeploying, false},
		{AppStateStarting, false},
		{AppStateStopping, false},
		{"UNKNOWN", false},
	}

	for _, tc := range tests {
		got := isStableState(tc.state)
		if got != tc.expected {
			t.Errorf("isStableState(%q) = %v, want %v", tc.state, got, tc.expected)
		}
	}
}

func TestIsValidDesiredState(t *testing.T) {
	tests := []struct {
		state    string
		expected bool
	}{
		{"running", true},
		{"RUNNING", true},
		{"stopped", true},
		{"STOPPED", true},
		{"deploying", false},
		{"crashed", false},
		{"paused", false},
	}

	for _, tc := range tests {
		got := isValidDesiredState(tc.state)
		if got != tc.expected {
			t.Errorf("isValidDesiredState(%q) = %v, want %v", tc.state, got, tc.expected)
		}
	}
}

func TestWaitForStableState_AlreadyStable(t *testing.T) {
	callCount := 0
	queryFunc := func(ctx context.Context, name string) (string, error) {
		callCount++
		return AppStateRunning, nil
	}

	ctx := context.Background()
	state, err := waitForStableState(ctx, "myapp", 30*time.Second, queryFunc)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != AppStateRunning {
		t.Errorf("expected state %q, got %q", AppStateRunning, state)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestWaitForStableState_TransitionsToStable(t *testing.T) {
	callCount := 0
	queryFunc := func(ctx context.Context, name string) (string, error) {
		callCount++
		if callCount < 3 {
			return AppStateStarting, nil
		}
		return AppStateRunning, nil
	}

	ctx := context.Background()
	state, err := waitForStableState(ctx, "myapp", 30*time.Second, queryFunc)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != AppStateRunning {
		t.Errorf("expected state %q, got %q", AppStateRunning, state)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestWaitForStableState_Timeout(t *testing.T) {
	queryFunc := func(ctx context.Context, name string) (string, error) {
		return AppStateDeploying, nil // Never becomes stable
	}

	ctx := context.Background()
	_, err := waitForStableState(ctx, "myapp", 100*time.Millisecond, queryFunc)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "DEPLOYING") {
		t.Errorf("expected error to mention DEPLOYING state, got: %v", err)
	}
}

func TestWaitForStableState_ContextCancelled(t *testing.T) {
	callCount := 0
	queryFunc := func(ctx context.Context, name string) (string, error) {
		callCount++
		return AppStateStarting, nil // Never becomes stable
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately before first poll

	_, err := waitForStableState(ctx, "myapp", 30*time.Second, queryFunc)

	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestWaitForStableState_ShortTimeoutUsesShortPollInterval(t *testing.T) {
	callCount := 0
	queryFunc := func(ctx context.Context, name string) (string, error) {
		callCount++
		if callCount < 3 {
			return AppStateStarting, nil
		}
		return AppStateRunning, nil
	}

	ctx := context.Background()
	// Use a timeout shorter than the default 5s poll interval
	// This triggers the timeout < pollInterval branch (line 67-68)
	timeout := 100 * time.Millisecond

	start := time.Now()
	state, err := waitForStableState(ctx, "myapp", timeout, queryFunc)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != AppStateRunning {
		t.Errorf("expected state %q, got %q", AppStateRunning, state)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
	// Verify it used the short poll interval (timeout/10 = 10ms)
	// Should complete much faster than 5s default poll interval
	if elapsed > 500*time.Millisecond {
		t.Errorf("expected fast completion with short poll interval, took %v", elapsed)
	}
}
