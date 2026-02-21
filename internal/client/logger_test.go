package client

import (
	"context"
	"testing"
)

func TestNopLogger_DoesNotPanic(t *testing.T) {
	var l Logger = NopLogger{}
	// Should not panic with any inputs
	l.Debug(context.Background(), "test message", map[string]any{
		"key": "value",
	})
}

func TestNopLogger_NilFields(t *testing.T) {
	var l Logger = NopLogger{}
	l.Debug(context.Background(), "test", nil)
}
