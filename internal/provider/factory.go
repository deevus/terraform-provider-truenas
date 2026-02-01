package provider

import "github.com/deevus/terraform-provider-truenas/internal/client"

// ClientFactory abstracts client creation for testability.
type ClientFactory interface {
	NewSSHClient(cfg *client.SSHConfig) (client.Client, error)
	NewWebSocketClient(cfg client.WebSocketConfig) (client.Client, error)
}

// DefaultClientFactory creates real clients for production use.
type DefaultClientFactory struct{}

func (f *DefaultClientFactory) NewSSHClient(cfg *client.SSHConfig) (client.Client, error) {
	return client.NewSSHClient(cfg)
}

func (f *DefaultClientFactory) NewWebSocketClient(cfg client.WebSocketConfig) (client.Client, error) {
	return client.NewWebSocketClient(cfg)
}
