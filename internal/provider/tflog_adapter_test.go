package provider

import (
	"context"
	"testing"

	"github.com/deevus/truenas-go/client"
)

func TestTFLogAdapter_ImplementsLogger(t *testing.T) {
	var _ client.Logger = TFLogAdapter{}
}

func TestTFLogAdapter_Debug_DoesNotPanic(t *testing.T) {
	adapter := TFLogAdapter{}
	// tflog requires a configured context; with a bare context it's a no-op
	adapter.Debug(context.Background(), "test", map[string]any{"key": "value"})
	adapter.Debug(context.Background(), "test", nil)
}
