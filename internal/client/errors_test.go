package client

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseError_EINVAL(t *testing.T) {
	raw := `[EINVAL] storage.config.host_path_config.path: Field was not expected`

	err := ParseTrueNASError(raw)

	if err.Code != "EINVAL" {
		t.Errorf("expected code EINVAL, got %s", err.Code)
	}
	if err.Field != "storage.config.host_path_config.path" {
		t.Errorf("expected field storage.config.host_path_config.path, got %s", err.Field)
	}
}

func TestParseError_ENOENT(t *testing.T) {
	raw := `[ENOENT] Unable to locate app "nonexistent"`

	err := ParseTrueNASError(raw)

	if err.Code != "ENOENT" {
		t.Errorf("expected code ENOENT, got %s", err.Code)
	}
	if err.Suggestion == "" {
		t.Error("expected suggestion for ENOENT error")
	}
}

func TestParseError_EFAULT(t *testing.T) {
	raw := `[EFAULT] Failed 'up' action for app caddy: image not found`

	err := ParseTrueNASError(raw)

	if err.Code != "EFAULT" {
		t.Errorf("expected code EFAULT, got %s", err.Code)
	}
}

func TestParseError_NoCode(t *testing.T) {
	raw := `Some random error message`

	err := ParseTrueNASError(raw)

	if err.Code != "UNKNOWN" {
		t.Errorf("expected code UNKNOWN, got %s", err.Code)
	}
	if err.Message != raw {
		t.Errorf("expected message to be preserved")
	}
}

func TestTrueNASError_Error(t *testing.T) {
	err := &TrueNASError{
		Code:       "EINVAL",
		Message:    "test message",
		Suggestion: "try this",
	}

	errStr := err.Error()

	if errStr == "" {
		t.Error("Error() should return non-empty string")
	}
}

func TestNewConnectionError(t *testing.T) {
	err := NewConnectionError("10.0.0.1", 22, fmt.Errorf("connection refused"))

	if err.Code != "ECONNREFUSED" {
		t.Errorf("expected code ECONNREFUSED, got %s", err.Code)
	}
	if err.Suggestion == "" {
		t.Error("expected suggestion for connection error")
	}
	if !strings.Contains(err.Message, "10.0.0.1") {
		t.Error("expected message to contain host")
	}
	if !strings.Contains(err.Message, "22") {
		t.Error("expected message to contain port")
	}
}

func TestNewTimeoutError(t *testing.T) {
	err := NewTimeoutError(12345, "5m")

	if err.Code != "ETIMEDOUT" {
		t.Errorf("expected code ETIMEDOUT, got %s", err.Code)
	}
	if err.JobID != 12345 {
		t.Errorf("expected JobID 12345, got %d", err.JobID)
	}
	if err.Suggestion == "" {
		t.Error("expected suggestion for timeout error")
	}
	if !strings.Contains(err.Message, "5m") {
		t.Error("expected message to contain duration")
	}
}

func TestNewHostKeyError(t *testing.T) {
	expected := "SHA256:expectedFingerprint123456789012345678901234"
	actual := "SHA256:actualFingerprint9876543210987654321098765"
	err := NewHostKeyError("truenas.local", expected, actual)

	if err.Code != "EHOSTKEY" {
		t.Errorf("expected code EHOSTKEY, got %s", err.Code)
	}
	if err.Suggestion == "" {
		t.Error("expected suggestion for host key error")
	}
	if !strings.Contains(err.Message, "truenas.local") {
		t.Error("expected message to contain host")
	}
	if !strings.Contains(err.Message, expected) {
		t.Error("expected message to contain expected fingerprint")
	}
	if !strings.Contains(err.Message, actual) {
		t.Error("expected message to contain actual fingerprint")
	}
}

func TestTrueNASError_Error_Format(t *testing.T) {
	err := &TrueNASError{
		Code:       "EINVAL",
		Message:    "test message",
		Suggestion: "try this",
	}

	errStr := err.Error()
	expected := "test message\n\nSuggestion: try this"
	if errStr != expected {
		t.Errorf("expected %q, got %q", expected, errStr)
	}
}

func TestTrueNASError_Error_NoSuggestion(t *testing.T) {
	err := &TrueNASError{
		Code:    "UNKNOWN",
		Message: "simple message",
	}

	errStr := err.Error()
	if errStr != "simple message" {
		t.Errorf("expected %q, got %q", "simple message", errStr)
	}
}

func TestParseError_StripProcessExitPrefix(t *testing.T) {
	raw := `Process exited with status 1: [EINVAL] Invalid config`

	err := ParseTrueNASError(raw)

	if err.Code != "EINVAL" {
		t.Errorf("expected code EINVAL, got %s", err.Code)
	}
	if strings.Contains(err.Message, "Process exited") {
		t.Error("expected Process exited prefix to be stripped")
	}
}

func TestParseError_StripTracebackNewline(t *testing.T) {
	raw := `[EINVAL] Invalid config
Traceback (most recent call last):
  File "middleware.py", line 100
    raise ValidationError`

	err := ParseTrueNASError(raw)

	if err.Code != "EINVAL" {
		t.Errorf("expected code EINVAL, got %s", err.Code)
	}
	if strings.Contains(err.Message, "Traceback") {
		t.Error("expected Traceback to be stripped")
	}
	if err.Message != "Invalid config" {
		t.Errorf("expected 'Invalid config', got %q", err.Message)
	}
}

func TestParseError_StripTracebackSameLine(t *testing.T) {
	raw := `Some error Traceback (most recent call last): in file.py`

	err := ParseTrueNASError(raw)

	if strings.Contains(err.Message, "Traceback") {
		t.Error("expected Traceback to be stripped when on same line")
	}
	if err.Message != "Some error" {
		t.Errorf("expected 'Some error', got %q", err.Message)
	}
}

func TestTrueNASError_Error_WithLogsExcerpt(t *testing.T) {
	err := &TrueNASError{
		Code:        "EFAULT",
		Message:     "Failed 'up' action for 'myapp' app",
		LogsExcerpt: "Container myapp  Creating\nError response from daemon: image not found",
	}

	errStr := err.Error()

	if !strings.Contains(errStr, "Failed 'up' action") {
		t.Error("expected error string to contain message")
	}
	if !strings.Contains(errStr, "Job logs:") {
		t.Error("expected error string to contain 'Job logs:'")
	}
	if !strings.Contains(errStr, "image not found") {
		t.Error("expected error string to contain logs excerpt")
	}
}

func TestTrueNASError_Error_WithLogsExcerptAndSuggestion(t *testing.T) {
	err := &TrueNASError{
		Code:        "EFAULT",
		Message:     "Failed 'up' action for 'myapp' app",
		LogsExcerpt: "Error response from daemon: image not found",
		Suggestion:  "Check compose_config and image availability.",
	}

	errStr := err.Error()
	expected := "Failed 'up' action for 'myapp' app\n\nJob logs:\nError response from daemon: image not found\n\nSuggestion: Check compose_config and image availability."

	if errStr != expected {
		t.Errorf("expected:\n%q\n\ngot:\n%q", expected, errStr)
	}
}

func TestTrueNASError_Error_EmptyLogsExcerpt(t *testing.T) {
	err := &TrueNASError{
		Code:        "EFAULT",
		Message:     "Failed 'up' action",
		LogsExcerpt: "",
		Suggestion:  "try this",
	}

	errStr := err.Error()

	// Should not contain "Job logs:" when excerpt is empty
	if strings.Contains(errStr, "Job logs:") {
		t.Error("expected error string to NOT contain 'Job logs:' when excerpt is empty")
	}

	expected := "Failed 'up' action\n\nSuggestion: try this"
	if errStr != expected {
		t.Errorf("expected %q, got %q", expected, errStr)
	}
}
