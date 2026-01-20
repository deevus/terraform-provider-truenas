# App Lifecycle Error Handling Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Clean up messy TrueNAS app lifecycle errors by fetching and parsing `/var/log/app_lifecycle.log` to extract actionable Docker errors.

**Architecture:** Detect trigger pattern in middleware errors, SSH to fetch the remote log, parse to extract the actual Docker error from the noise, present clean error to user.

**Tech Stack:** Go, regexp, SSH via existing midclt client

**Date:** 2026-01-20
**Status:** Approved

---

## Problem

When TrueNAS app creation/update fails, the error message is a mess:

```
Unable to create app "dns": Process exited with status 1:
Status: (none)
Status: Initial validation completed for custom app creation 0.00%
Status: Setting up App directory___________________________] 25.00%
...50 more lines of progress bars and status updates...
[EFAULT] Failed 'up' action for 'dns' app. Please check /var/log/app_lifecycle.log for more details
Traceback (most recent call last):
  File "/usr/lib/python3/dist-packages/middlewared/job.py", line 515, in run
...Python traceback...
```

The actual useful error is buried in `/var/log/app_lifecycle.log` on the TrueNAS server.

## Solution

Automatically fetch and parse `/var/log/app_lifecycle.log` when app errors reference it, extracting the clean Docker error.

### Before

```
Unable to create app "dns": Process exited with status 1:
...100 lines of noise...
```

### After

```
Unable to create app "dns": Error starting userland proxy: listen tcp4 192.168.1.10:80: bind: address already in use

Suggestion: Container failed to start. Check compose_config and image availability.
```

## Design

### 1. Error Detection

Extend `TrueNASError` struct in `internal/client/errors.go`:

```go
type TrueNASError struct {
    Code              string
    Message           string
    Field             string
    JobID             int64
    Suggestion        string
    LogsExcerpt       string
    // New fields for app lifecycle log fetching
    AppAction         string  // "up", "down", etc.
    AppName           string  // app that failed
    LogPath           string  // "/var/log/app_lifecycle.log"
    AppLifecycleError string  // The extracted clean error
}
```

Detect trigger pattern in `ParseTrueNASError()`:

```go
// Pattern: Failed '<action>' action for '<app>' app.*app_lifecycle\.log
var appLifecyclePattern = regexp.MustCompile(`Failed '(\w+)' action for '([^']+)' app.*(/var/log/app_lifecycle\.log)`)
```

### 2. Log Fetching

In `internal/client/jobs.go`, fetch log after job failure:

```go
case JobStateFailed:
    err := ParseTrueNASError(job.Error)
    err.LogsExcerpt = job.LogsExcerpt

    // Fetch app lifecycle log if applicable
    if err.LogPath != "" && err.AppName != "" {
        appErr := p.fetchAppLifecycleError(ctx, err)
        if appErr != "" {
            err.AppLifecycleError = appErr
        }
    }
    return nil, err
```

Fetch via SSH using existing `cat` command:

```go
func (p *JobPoller) fetchAppLifecycleError(ctx context.Context, err *TrueNASError) string {
    result, cmdErr := p.client.Call(ctx, "core.run", []any{
        "cat", []string{err.LogPath},
    })
    if cmdErr != nil {
        return "" // Silently fail, don't make error worse
    }
    logContent := extractStdout(result)
    return parseAppLifecycleLog(logContent, err.AppAction, err.AppName)
}
```

### 3. Log Parsing

The app lifecycle log format:

```
[YYYY/MM/DD HH:MM:SS] (ERROR) app_lifecycle.compose_action():56 - Failed '<action>' action for '<app>' app: <docker_compose_output>\n
```

Docker compose output contains literal `\n` separators. The actual error is always at the end.

Parsing algorithm:

```go
func parseAppLifecycleLog(content, action, appName string) string {
    // Build search pattern
    pattern := fmt.Sprintf(`Failed '%s' action for '%s' app: (.+)`,
        regexp.QuoteMeta(action), regexp.QuoteMeta(appName))
    re := regexp.MustCompile(pattern)

    // Find all matches, take the last one (most recent)
    matches := re.FindAllStringSubmatch(content, -1)
    if len(matches) == 0 {
        return ""
    }
    lastMatch := matches[len(matches)-1][1]

    // Extract actual error - split by literal \n, take last non-empty segment
    parts := strings.Split(lastMatch, `\n`)
    for i := len(parts) - 1; i >= 0; i-- {
        trimmed := strings.TrimSpace(parts[i])
        if trimmed != "" {
            return trimmed
        }
    }
    return lastMatch
}
```

### 4. Error Presentation

Updated `Error()` method:

```go
func (e *TrueNASError) Error() string {
    var sb strings.Builder

    // Lead with the clean app lifecycle error if available
    if e.AppLifecycleError != "" {
        sb.WriteString(e.AppLifecycleError)
    } else {
        sb.WriteString(e.Message)
    }

    if e.Suggestion != "" {
        sb.WriteString("\n\nSuggestion: ")
        sb.WriteString(e.Suggestion)
    }

    return sb.String()
}
```

## Testing

### Test Fixture

Use real app lifecycle log as fixture: `internal/testdata/fixtures/app_lifecycle.log`

Example errors covered:

| App | Action | Error |
|-----|--------|-------|
| dns | up | `Error response from daemon: ...bind: address already in use` |
| caddy | up | `Error response from daemon: error while creating mount source path '/config': mkdir /config: read-only file system` |
| caddy | down | `service "caddy" refers to undefined volume config: invalid compose project` |
| nextcloud | up | `dependency failed to start: container ix-nextcloud-nextcloud-1 is unhealthy` |
| hello-world | up | `Error response from daemon: ...bind: address already in use` |

### Unit Tests

**Error detection:**

```go
func TestParseTrueNASError_DetectsAppLifecyclePattern(t *testing.T) {
    raw := `[EFAULT] Failed 'up' action for 'dns' app. Please check /var/log/app_lifecycle.log for more details`

    err := ParseTrueNASError(raw)

    assert.Equal(t, "EFAULT", err.Code)
    assert.Equal(t, "up", err.AppAction)
    assert.Equal(t, "dns", err.AppName)
    assert.Equal(t, "/var/log/app_lifecycle.log", err.LogPath)
}
```

**Log parsing:**

```go
func TestParseAppLifecycleLog(t *testing.T) {
    content, _ := os.ReadFile("testdata/fixtures/app_lifecycle.log")

    tests := []struct {
        action   string
        appName  string
        contains string
    }{
        {"up", "dns", "bind: address already in use"},
        {"up", "caddy", "read-only file system"},
        {"down", "caddy", "invalid compose project"},
        {"up", "nextcloud", "unhealthy"},
    }

    for _, tt := range tests {
        t.Run(tt.appName+"_"+tt.action, func(t *testing.T) {
            result := parseAppLifecycleLog(string(content), tt.action, tt.appName)
            assert.Contains(t, result, tt.contains)
        })
    }
}
```

## Files to Modify

- `internal/client/errors.go` - Add detection pattern, new struct fields, update Error()
- `internal/client/jobs.go` - Add log fetching after job failure
- `internal/client/errors_test.go` - Test error detection
- `internal/client/jobs_test.go` - Test log parsing

## Files to Add

- `internal/testdata/fixtures/app_lifecycle.log` - Test fixture (already added)

---

## Implementation Tasks

### Task 1: Add New Fields to TrueNASError Struct

**Files:**
- Modify: `internal/client/errors.go:9-17`

**Step 1: Add new fields to struct**

```go
// TrueNASError represents a parsed error from the TrueNAS middleware.
type TrueNASError struct {
	Code              string // e.g., "EINVAL", "ENOENT", "EFAULT"
	Message           string // Raw error from middleware
	Field             string // Which field caused error (if applicable)
	JobID             int64  // For job-related errors
	Suggestion        string // Actionable guidance
	LogsExcerpt       string // Job log excerpt for debugging
	// App lifecycle log fields
	AppAction         string // "up", "down", etc. - extracted from error
	AppName           string // App that failed - extracted from error
	LogPath           string // Path to log file mentioned in error
	AppLifecycleError string // Clean error extracted from app_lifecycle.log
}
```

**Step 2: Run tests to verify no regression**

Run: `go test ./internal/client/... -v -run TestTrueNASError`
Expected: All existing tests PASS

**Step 3: Commit**

```bash
git add internal/client/errors.go
git commit -m "feat(errors): add app lifecycle fields to TrueNASError struct"
```

---

### Task 2: Write Tests for App Lifecycle Pattern Detection

**Files:**
- Modify: `internal/client/errors_test.go`

**Step 1: Write failing tests for pattern detection**

Add to `internal/client/errors_test.go`:

```go
func TestParseTrueNASError_DetectsAppLifecyclePattern(t *testing.T) {
	raw := `[EFAULT] Failed 'up' action for 'dns' app. Please check /var/log/app_lifecycle.log for more details`

	err := ParseTrueNASError(raw)

	if err.Code != "EFAULT" {
		t.Errorf("expected code EFAULT, got %s", err.Code)
	}
	if err.AppAction != "up" {
		t.Errorf("expected AppAction 'up', got %q", err.AppAction)
	}
	if err.AppName != "dns" {
		t.Errorf("expected AppName 'dns', got %q", err.AppName)
	}
	if err.LogPath != "/var/log/app_lifecycle.log" {
		t.Errorf("expected LogPath '/var/log/app_lifecycle.log', got %q", err.LogPath)
	}
}

func TestParseTrueNASError_DetectsDownAction(t *testing.T) {
	raw := `[EFAULT] Failed 'down' action for 'caddy' app. Please check /var/log/app_lifecycle.log for more details`

	err := ParseTrueNASError(raw)

	if err.AppAction != "down" {
		t.Errorf("expected AppAction 'down', got %q", err.AppAction)
	}
	if err.AppName != "caddy" {
		t.Errorf("expected AppName 'caddy', got %q", err.AppName)
	}
}

func TestParseTrueNASError_NoAppLifecyclePattern(t *testing.T) {
	raw := `[EINVAL] Invalid configuration`

	err := ParseTrueNASError(raw)

	if err.AppAction != "" {
		t.Errorf("expected empty AppAction, got %q", err.AppAction)
	}
	if err.AppName != "" {
		t.Errorf("expected empty AppName, got %q", err.AppName)
	}
	if err.LogPath != "" {
		t.Errorf("expected empty LogPath, got %q", err.LogPath)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/client/... -v -run TestParseTrueNASError_Detects`
Expected: FAIL - fields not populated

**Step 3: Commit failing tests**

```bash
git add internal/client/errors_test.go
git commit -m "test(errors): add tests for app lifecycle pattern detection"
```

---

### Task 3: Implement App Lifecycle Pattern Detection

**Files:**
- Modify: `internal/client/errors.go:33-40` (add regex)
- Modify: `internal/client/errors.go:51-90` (update ParseTrueNASError)

**Step 1: Add the regex pattern**

Add after line 39 in `internal/client/errors.go`:

```go
	// Matches app lifecycle error pattern: Failed '<action>' action for '<app>' app ... /var/log/app_lifecycle.log
	appLifecycleRegex = regexp.MustCompile(`Failed '(\w+)' action for '([^']+)' app.*(/var/log/app_lifecycle\.log)`)
```

**Step 2: Update ParseTrueNASError to extract app lifecycle info**

Add before the `return err` at end of ParseTrueNASError (around line 88):

```go
	// Check for app lifecycle pattern
	if matches := appLifecycleRegex.FindStringSubmatch(raw); len(matches) == 4 {
		err.AppAction = matches[1]
		err.AppName = matches[2]
		err.LogPath = matches[3]
	}
```

**Step 3: Run tests to verify they pass**

Run: `go test ./internal/client/... -v -run TestParseTrueNASError`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add internal/client/errors.go
git commit -m "feat(errors): detect app lifecycle pattern in error messages"
```

---

### Task 4: Write Tests for App Lifecycle Log Parsing

**Files:**
- Modify: `internal/client/errors_test.go`
- Read: `internal/testdata/fixtures/app_lifecycle.log`

**Step 1: Write failing tests for log parsing**

Add to `internal/client/errors_test.go`:

```go
func TestParseAppLifecycleLog(t *testing.T) {
	content, err := os.ReadFile("testdata/fixtures/app_lifecycle.log")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	tests := []struct {
		name     string
		action   string
		appName  string
		contains string
	}{
		{
			name:     "dns_up_port_conflict",
			action:   "up",
			appName:  "dns",
			contains: "bind: address already in use",
		},
		{
			name:     "caddy_up_readonly",
			action:   "up",
			appName:  "caddy",
			contains: "read-only file system",
		},
		{
			name:     "caddy_down_invalid_compose",
			action:   "down",
			appName:  "caddy",
			contains: "invalid compose project",
		},
		{
			name:     "nextcloud_up_unhealthy",
			action:   "up",
			appName:  "nextcloud",
			contains: "unhealthy",
		},
		{
			name:     "hello_world_port_conflict",
			action:   "up",
			appName:  "hello-world",
			contains: "bind: address already in use",
		},
		{
			name:     "uptime_kuma_network_mode",
			action:   "up",
			appName:  "uptime-kuma",
			contains: "conflicting options: dns and the network mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseAppLifecycleLog(string(content), tt.action, tt.appName)
			if result == "" {
				t.Errorf("expected non-empty result for %s/%s", tt.action, tt.appName)
				return
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected result to contain %q, got %q", tt.contains, result)
			}
		})
	}
}

func TestParseAppLifecycleLog_NoMatch(t *testing.T) {
	content := `[2026/01/20 17:00:00] (ERROR) app_lifecycle.compose_action():56 - Failed 'up' action for 'other' app: some error\n`

	result := ParseAppLifecycleLog(content, "up", "nonexistent")

	if result != "" {
		t.Errorf("expected empty result for non-matching app, got %q", result)
	}
}

func TestParseAppLifecycleLog_EmptyContent(t *testing.T) {
	result := ParseAppLifecycleLog("", "up", "dns")

	if result != "" {
		t.Errorf("expected empty result for empty content, got %q", result)
	}
}
```

**Step 2: Add os import if not present**

Ensure `"os"` is in the imports.

**Step 3: Run tests to verify they fail**

Run: `go test ./internal/client/... -v -run TestParseAppLifecycleLog`
Expected: FAIL - function not defined

**Step 4: Commit failing tests**

```bash
git add internal/client/errors_test.go
git commit -m "test(errors): add tests for app lifecycle log parsing"
```

---

### Task 5: Implement App Lifecycle Log Parsing

**Files:**
- Modify: `internal/client/errors.go`

**Step 1: Add ParseAppLifecycleLog function**

Add at end of `internal/client/errors.go`:

```go
// ParseAppLifecycleLog extracts the actual Docker error from the app lifecycle log.
// It searches for the most recent matching entry and extracts the error from the end.
func ParseAppLifecycleLog(content, action, appName string) string {
	if content == "" || action == "" || appName == "" {
		return ""
	}

	// Build pattern to match log entries for this app/action
	// Format: Failed '<action>' action for '<app>' app: <content>
	pattern := fmt.Sprintf(`Failed '%s' action for '%s' app: (.+)`,
		regexp.QuoteMeta(action), regexp.QuoteMeta(appName))
	re := regexp.MustCompile(pattern)

	// Find all matches, take the last one (most recent)
	matches := re.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return ""
	}

	// Get the captured content from the last match
	lastMatch := matches[len(matches)-1][1]

	// The content contains literal \n separators from Docker output
	// The actual error is the last non-empty segment
	parts := strings.Split(lastMatch, `\n`)
	for i := len(parts) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(parts[i])
		if trimmed != "" {
			return trimmed
		}
	}

	return lastMatch
}
```

**Step 2: Run tests to verify they pass**

Run: `go test ./internal/client/... -v -run TestParseAppLifecycleLog`
Expected: All tests PASS

**Step 3: Commit**

```bash
git add internal/client/errors.go
git commit -m "feat(errors): implement app lifecycle log parsing"
```

---

### Task 6: Update Error() Method to Prefer AppLifecycleError

**Files:**
- Modify: `internal/client/errors.go:19-31`
- Modify: `internal/client/errors_test.go`

**Step 1: Write failing test for new Error() behavior**

Add to `internal/client/errors_test.go`:

```go
func TestTrueNASError_Error_WithAppLifecycleError(t *testing.T) {
	err := &TrueNASError{
		Code:              "EFAULT",
		Message:           "Failed 'up' action for 'dns' app. Please check /var/log/app_lifecycle.log for more details",
		AppLifecycleError: "Error response from daemon: bind: address already in use",
		Suggestion:        "Container failed to start. Check compose_config and image availability.",
	}

	errStr := err.Error()

	// Should lead with the clean app lifecycle error, not the messy message
	if !strings.HasPrefix(errStr, "Error response from daemon:") {
		t.Errorf("expected error to start with app lifecycle error, got %q", errStr)
	}
	// Should NOT contain the "Please check" noise
	if strings.Contains(errStr, "Please check") {
		t.Error("expected error to NOT contain 'Please check' when AppLifecycleError is set")
	}
	// Should still have suggestion
	if !strings.Contains(errStr, "Suggestion:") {
		t.Error("expected error to contain suggestion")
	}
}

func TestTrueNASError_Error_FallbackToMessage(t *testing.T) {
	err := &TrueNASError{
		Code:              "EFAULT",
		Message:           "Some other error",
		AppLifecycleError: "", // Not set
		Suggestion:        "Try something",
	}

	errStr := err.Error()

	if !strings.HasPrefix(errStr, "Some other error") {
		t.Errorf("expected error to fall back to Message, got %q", errStr)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/client/... -v -run TestTrueNASError_Error_WithAppLifecycleError`
Expected: FAIL

**Step 3: Update Error() method**

Replace `internal/client/errors.go:19-31`:

```go
func (e *TrueNASError) Error() string {
	var sb strings.Builder

	// Prefer the clean app lifecycle error if available
	if e.AppLifecycleError != "" {
		sb.WriteString(e.AppLifecycleError)
	} else {
		sb.WriteString(e.Message)
		if e.LogsExcerpt != "" {
			sb.WriteString("\n\nJob logs:\n")
			sb.WriteString(e.LogsExcerpt)
		}
	}

	if e.Suggestion != "" {
		sb.WriteString("\n\nSuggestion: ")
		sb.WriteString(e.Suggestion)
	}
	return sb.String()
}
```

**Step 4: Run all error tests**

Run: `go test ./internal/client/... -v -run TestTrueNASError`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/client/errors.go internal/client/errors_test.go
git commit -m "feat(errors): prefer AppLifecycleError in Error() output"
```

---

### Task 7: Add Log Fetching to JobPoller

**Files:**
- Modify: `internal/client/jobs.go`
- Modify: `internal/client/jobs_test.go`

**Step 1: Write failing test for log fetching integration**

Add to `internal/client/jobs_test.go`:

```go
func TestJobPoller_FailureWithAppLifecycleLog(t *testing.T) {
	// Read the fixture
	logContent, err := os.ReadFile("testdata/fixtures/app_lifecycle.log")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	callCount := 0
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			if method == "core.get_jobs" {
				return json.RawMessage(`[{
					"id": 42,
					"state": "FAILED",
					"error": "[EFAULT] Failed 'up' action for 'dns' app. Please check /var/log/app_lifecycle.log for more details"
				}]`), nil
			}
			if method == "filesystem.file_get_contents" {
				// Return the log file content
				contentJSON, _ := json.Marshal(string(logContent))
				return contentJSON, nil
			}
			t.Errorf("unexpected method: %s", method)
			return nil, nil
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
```

**Step 2: Add os import if not present**

**Step 3: Run test to verify it fails**

Run: `go test ./internal/client/... -v -run TestJobPoller_FailureWithAppLifecycleLog`
Expected: FAIL

**Step 4: Commit failing test**

```bash
git add internal/client/jobs_test.go
git commit -m "test(jobs): add test for app lifecycle log fetching"
```

---

### Task 8: Implement Log Fetching in JobPoller

**Files:**
- Modify: `internal/client/jobs.go`

**Step 1: Add fetchAppLifecycleLog method**

Add at end of `internal/client/jobs.go`:

```go
// fetchAppLifecycleLog fetches the app lifecycle log and extracts the error.
func (p *JobPoller) fetchAppLifecycleLog(ctx context.Context, err *TrueNASError) string {
	if err.LogPath == "" || err.AppName == "" || err.AppAction == "" {
		return ""
	}

	// Use filesystem.file_get_contents to read the log
	result, callErr := p.client.Call(ctx, "filesystem.file_get_contents", []any{err.LogPath})
	if callErr != nil {
		// Silently fail - don't make the error worse
		return ""
	}

	// Result is a JSON string
	var content string
	if jsonErr := json.Unmarshal(result, &content); jsonErr != nil {
		return ""
	}

	return ParseAppLifecycleLog(content, err.AppAction, err.AppName)
}
```

**Step 2: Update Wait() to call fetchAppLifecycleLog**

Modify the `JobStateFailed` case in `Wait()` (around line 92-95):

```go
		case JobStateFailed:
			err := ParseTrueNASError(job.Error)
			err.LogsExcerpt = job.LogsExcerpt

			// Fetch app lifecycle log if applicable
			if err.LogPath != "" {
				if appErr := p.fetchAppLifecycleLog(ctx, err); appErr != "" {
					err.AppLifecycleError = appErr
				}
			}

			return nil, err
```

**Step 3: Run test to verify it passes**

Run: `go test ./internal/client/... -v -run TestJobPoller_FailureWithAppLifecycleLog`
Expected: PASS

**Step 4: Run all tests**

Run: `go test ./internal/client/... -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/client/jobs.go
git commit -m "feat(jobs): fetch and parse app lifecycle log on failure"
```

---

### Task 9: Add Test for Log Fetch Failure Graceful Handling

**Files:**
- Modify: `internal/client/jobs_test.go`

**Step 1: Write test for graceful failure**

Add to `internal/client/jobs_test.go`:

```go
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
			if method == "filesystem.file_get_contents" {
				// Simulate failure to read log
				return nil, errors.New("permission denied")
			}
			return nil, nil
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
```

**Step 2: Run test**

Run: `go test ./internal/client/... -v -run TestJobPoller_FailureWithAppLifecycleLog`
Expected: All PASS

**Step 3: Commit**

```bash
git add internal/client/jobs_test.go
git commit -m "test(jobs): verify graceful handling when log fetch fails"
```

---

### Task 10: Run Full Test Suite and Final Commit

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

**Step 2: Run linter**

Run: `golangci-lint run`
Expected: No errors

**Step 3: Create final commit if any cleanup needed**

**Step 4: Verify git log**

Run: `git log --oneline -10`
Expected: See all commits from this implementation
