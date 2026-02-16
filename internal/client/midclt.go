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
// If params is a []any slice, each element is passed as a separate positional argument.
// This is needed for TrueNAS CRUD update methods which expect (id, data) as separate args.
func BuildCommand(method string, params any) string {
	// Validate method name to prevent command injection
	if !methodRegex.MatchString(method) {
		return fmt.Sprintf("midclt call %s", shellescape.Quote(method))
	}

	if params == nil {
		return fmt.Sprintf("midclt call %s", method)
	}

	// Handle []any slices specially - each element becomes a separate argument
	if slice, ok := params.([]any); ok {
		args := fmt.Sprintf("midclt call %s", method)
		for _, arg := range slice {
			argJSON, err := json.Marshal(arg)
			if err != nil {
				continue
			}
			args += " " + shellescape.Quote(string(argJSON))
		}
		return args
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
	AppName                   string         `json:"app_name"`
	CustomApp                 bool           `json:"custom_app"`
	CustomComposeConfigString string         `json:"custom_compose_config_string,omitempty"`
	CatalogApp                string         `json:"catalog_app,omitempty"`
	Train                     string         `json:"train,omitempty"`
	Version                   string         `json:"version,omitempty"`
	Values                    map[string]any `json:"values,omitempty"`
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
