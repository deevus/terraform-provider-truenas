package client

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// FileReader is a function type for reading file contents.
// This abstraction allows error enrichment to work with different client types.
type FileReader func(ctx context.Context, path string) (string, error)

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

var (
	// Matches [CODE] at start of error message
	errorCodeRegex = regexp.MustCompile(`\[([A-Z]+)\]\s*(.*)`)
	// Matches field path before colon
	fieldRegex = regexp.MustCompile(`^([\w.]+):\s*(.*)`)
	// Matches "Process exited with status N: " prefix
	processExitRegex = regexp.MustCompile(`^Process exited with status \d+:\s*`)
	// Matches app lifecycle error pattern: Failed '<action>' action for '<app>' app ... /var/log/app_lifecycle.log
	appLifecycleRegex = regexp.MustCompile(`Failed '(\w+)' action for '([^']+)' app.*(/var/log/app_lifecycle\.log)`)
)

// errorSuggestions maps error codes to helpful suggestions.
var errorSuggestions = map[string]string{
	"EINVAL":    "Check the configuration schema. A field may be invalid or unexpected.",
	"ENOENT":    "Resource not found. It may have been deleted outside Terraform.",
	"EFAULT":    "Container failed to start. Check compose_config and image availability.",
	"EEXIST":    "Resource already exists. Import it or choose a different name.",
	"ENOTEMPTY": "Directory or dataset has children. Use force_destroy = true or delete children first.",
}

// ParseTrueNASError parses a raw error string from midclt into a structured error.
func ParseTrueNASError(raw string) *TrueNASError {
	err := &TrueNASError{
		Code:    "UNKNOWN",
		Message: raw,
	}

	// Strip "Process exited with status N: " prefix
	cleaned := processExitRegex.ReplaceAllString(raw, "")

	// Strip Python traceback - take only the first line or content before "Traceback"
	if idx := strings.Index(cleaned, "\nTraceback"); idx != -1 {
		cleaned = strings.TrimSpace(cleaned[:idx])
	}

	// Also handle case where traceback is on same line
	if idx := strings.Index(cleaned, "Traceback (most recent call last)"); idx != -1 {
		cleaned = strings.TrimSpace(cleaned[:idx])
	}

	err.Message = cleaned

	// Try to extract error code
	if matches := errorCodeRegex.FindStringSubmatch(cleaned); len(matches) == 3 {
		err.Code = matches[1]
		err.Message = strings.TrimSpace(matches[2])

		// Try to extract field from remainder
		if fieldMatches := fieldRegex.FindStringSubmatch(err.Message); len(fieldMatches) == 3 {
			err.Field = fieldMatches[1]
		}
	}

	// Add suggestion based on code
	if suggestion, ok := errorSuggestions[err.Code]; ok {
		err.Suggestion = suggestion
	}

	// Check for app lifecycle pattern
	if matches := appLifecycleRegex.FindStringSubmatch(raw); len(matches) == 4 {
		err.AppAction = matches[1]
		err.AppName = matches[2]
		err.LogPath = matches[3]
	}

	return err
}

// NewConnectionError creates an error for SSH connection failures.
func NewConnectionError(host string, port int, cause error) *TrueNASError {
	return &TrueNASError{
		Code:       "ECONNREFUSED",
		Message:    fmt.Sprintf("Cannot connect to %s:%d: %v", host, port, cause),
		Suggestion: "Verify SSH credentials, network connectivity, and that the TrueNAS server is running.",
	}
}

// NewTimeoutError creates an error for job timeouts.
func NewTimeoutError(jobID int64, duration string) *TrueNASError {
	return &TrueNASError{
		Code:       "ETIMEDOUT",
		Message:    fmt.Sprintf("Operation timed out after %s", duration),
		JobID:      jobID,
		Suggestion: "Increase the timeout or check the TrueNAS server for issues.",
	}
}

// NewHostKeyError creates an error for SSH host key verification failures.
func NewHostKeyError(host string, expected, actual string) *TrueNASError {
	return &TrueNASError{
		Code:       "EHOSTKEY",
		Message:    fmt.Sprintf("host key verification failed for %s: expected %s, got %s", host, expected, actual),
		Suggestion: "Verify the fingerprint: ssh-keyscan <host> 2>/dev/null | ssh-keygen -lf -",
	}
}

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

// EnrichAppLifecycleError fetches the app lifecycle log and enriches the error
// with the extracted Docker error message. The readFile function should read
// the file contents (e.g., via cat command). This function silently fails if
// the log cannot be fetched - it never makes the error worse.
func EnrichAppLifecycleError(ctx context.Context, err *TrueNASError, readFile FileReader) {
	if err.LogPath == "" || err.AppName == "" || err.AppAction == "" {
		return
	}

	// Read the log file
	content, readErr := readFile(ctx, err.LogPath)
	if readErr != nil {
		return
	}

	if appErr := ParseAppLifecycleLog(content, err.AppAction, err.AppName); appErr != "" {
		err.AppLifecycleError = appErr
	}
}
