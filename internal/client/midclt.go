package client

import (
	"encoding/json"
	"fmt"
	"regexp"

	"al.essio.dev/pkg/shellescape"
)

// methodRegex validates that method names contain only safe characters.
// Valid methods: lowercase letters, digits, dots, and underscores (e.g., "pool.dataset.create").
var methodRegex = regexp.MustCompile(`^[a-z][a-z0-9_.]+$`)

// BuildCommand constructs a midclt command string.
func BuildCommand(method string, params any) string {
	// Validate method name to prevent command injection
	if !methodRegex.MatchString(method) {
		return fmt.Sprintf("midclt call %s", shellescape.Quote(method))
	}

	if params == nil {
		return fmt.Sprintf("midclt call %s", method)
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		// In practice this shouldn't happen with valid Go types
		return fmt.Sprintf("midclt call %s", method)
	}

	return fmt.Sprintf("midclt call %s %s", method, shellescape.Quote(string(paramsJSON)))
}

// AppCreateParams represents parameters for app.create.
type AppCreateParams struct {
	AppName                   string    `json:"app_name"`
	CustomApp                 bool      `json:"custom_app"`
	CustomComposeConfigString string    `json:"custom_compose_config_string,omitempty"`
	Values                    AppValues `json:"values"`
}

// AppValues represents the values section of app configuration.
type AppValues struct {
	Storage map[string]StorageConfig `json:"storage,omitempty"`
	Network map[string]NetworkConfig `json:"network,omitempty"`
	Labels  []string                 `json:"labels,omitempty"`
}

// StorageConfig represents volume storage configuration.
type StorageConfig struct {
	Type           string         `json:"type"`
	HostPathConfig HostPathConfig `json:"host_path_config"`
}

// HostPathConfig represents host path configuration.
type HostPathConfig struct {
	ACLEnable       bool   `json:"acl_enable"`
	AutoPermissions bool   `json:"auto_permissions"`
	Path            string `json:"path"`
}

// NetworkConfig represents network port configuration.
type NetworkConfig struct {
	BindMode   string   `json:"bind_mode"`
	HostIPs    []string `json:"host_ips"`
	PortNumber int      `json:"port_number"`
}

// DatasetCreateParams represents parameters for pool.dataset.create.
type DatasetCreateParams struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Compression string `json:"compression,omitempty"`
	Quota       int64  `json:"quota,omitempty"`
	RefQuota    int64  `json:"refquota,omitempty"`
	Atime       string `json:"atime,omitempty"`
}
