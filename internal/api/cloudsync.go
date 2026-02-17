package api

import (
	"encoding/json"
	"fmt"
)

// CloudSyncCredentialResponse represents a cloud sync credential from the API.
type CloudSyncCredentialResponse struct {
	ID       int64                        `json:"id"`
	Name     string                       `json:"name"`
	Provider CloudSyncCredentialProvider  `json:"provider"`
}

// CloudSyncCredentialProvider contains the provider type and attributes.
type CloudSyncCredentialProvider struct {
	Type       string            `json:"type"`
	// S3 attributes
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	Endpoint        string `json:"endpoint,omitempty"`
	Region          string `json:"region,omitempty"`
	// B2 attributes
	Account string `json:"account,omitempty"`
	Key     string `json:"key,omitempty"`
	// GCS attributes
	ServiceAccountCredentials string `json:"service_account_credentials,omitempty"`
}

// CloudSyncTaskCredentialRef is a minimal struct for embedded credential references in tasks.
// The full credential parsing is handled by ParseCredentials when needed.
// This avoids version-specific parsing complexity since task responses embed credentials
// with provider in different formats (string in 24.x, object in 25.x).
type CloudSyncTaskCredentialRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// CloudSyncTaskResponse represents a cloud sync task from the API.
type CloudSyncTaskResponse struct {
	ID                 int64                      `json:"id"`
	Description        string                     `json:"description"`
	Path               string                     `json:"path"`
	Credentials        CloudSyncTaskCredentialRef `json:"credentials"`
	Attributes         json.RawMessage            `json:"attributes"` // Can be object or false
	Schedule           ScheduleResponse           `json:"schedule"`
	Direction          string                     `json:"direction"`
	TransferMode       string                     `json:"transfer_mode"`
	Encryption         bool                       `json:"encryption"`
	EncryptionPassword string                     `json:"encryption_password,omitempty"`
	EncryptionSalt     string                     `json:"encryption_salt,omitempty"`
	Snapshot           bool                       `json:"snapshot"`
	Transfers          int64                      `json:"transfers"`
	BWLimit            []BwLimit                  `json:"bwlimit"`
	Exclude            []string                   `json:"exclude"`
	Include            []string                   `json:"include"`
	FollowSymlinks     bool                       `json:"follow_symlinks"`
	CreateEmptySrcDirs bool                       `json:"create_empty_src_dirs"`
	Enabled            bool                       `json:"enabled"`
	Job                *JobStatus                 `json:"job,omitempty"`
}

// FastList returns the fast_list value from the task attributes.
// Returns false if attributes is not an object or fast_list is not set.
func (r *CloudSyncTaskResponse) FastList() bool {
	var attrs struct {
		FastList bool `json:"fast_list"`
	}
	if json.Unmarshal(r.Attributes, &attrs) != nil {
		return false
	}
	return attrs.FastList
}

// BwLimit represents a bandwidth limit entry.
type BwLimit struct {
	Time      string `json:"time"`
	Bandwidth *int64 `json:"bandwidth"` // null when unlimited
}

// ScheduleResponse represents a cron schedule from the API.
type ScheduleResponse struct {
	Minute string `json:"minute"`
	Hour   string `json:"hour"`
	Dom    string `json:"dom"`
	Month  string `json:"month"`
	Dow    string `json:"dow"`
}

// JobStatus represents the last job status for a task.
type JobStatus struct {
	ID    int64  `json:"id"`
	State string `json:"state"`
}

// credentialRaw is the intermediate struct for parsing API responses.
type credentialRaw struct {
	ID         int64           `json:"id"`
	Name       string          `json:"name"`
	Provider   json.RawMessage `json:"provider"`
	Attributes json.RawMessage `json:"attributes"` // 24.x only
}

// parseCredentialV25 parses a 25.x format credential response.
// Format: { "provider": { "type": "S3", "access_key_id": "...", ... } }
func parseCredentialV25(raw credentialRaw) (CloudSyncCredentialResponse, error) {
	var c CloudSyncCredentialResponse
	c.ID = raw.ID
	c.Name = raw.Name

	if err := json.Unmarshal(raw.Provider, &c.Provider); err != nil {
		return c, fmt.Errorf("parse provider object: %w", err)
	}
	return c, nil
}

// parseCredentialV24 parses a 24.x format credential response.
// Format: { "provider": "S3", "attributes": { ... } }
func parseCredentialV24(raw credentialRaw) (CloudSyncCredentialResponse, error) {
	var c CloudSyncCredentialResponse
	c.ID = raw.ID
	c.Name = raw.Name

	var providerType string
	if err := json.Unmarshal(raw.Provider, &providerType); err != nil {
		return c, fmt.Errorf("parse provider string: %w", err)
	}
	c.Provider.Type = providerType

	if len(raw.Attributes) > 0 {
		if err := json.Unmarshal(raw.Attributes, &c.Provider); err != nil {
			return c, fmt.Errorf("parse attributes: %w", err)
		}
	}
	return c, nil
}

// BuildCredentialsParams builds cloudsync credentials params for the appropriate API version.
// Pre-25.x uses separate "provider" (string) and "attributes" (object).
// 25.x+ uses "provider" as object with "type" merged with attributes.
func BuildCredentialsParams(v Version, name, providerType string, attributes map[string]any) map[string]any {
	if v.AtLeast(25, 0) {
		// 25.x format: provider is object with type + attributes
		providerObj := map[string]any{"type": providerType}
		for k, val := range attributes {
			providerObj[k] = val
		}
		return map[string]any{
			"name":     name,
			"provider": providerObj,
		}
	}
	// 24.x format: provider is string, attributes is separate object
	attrsCopy := make(map[string]any)
	for k, val := range attributes {
		attrsCopy[k] = val
	}
	return map[string]any{
		"name":       name,
		"provider":   providerType,
		"attributes": attrsCopy,
	}
}

// ParseCredentials parses credentials from API response using version-specific logic.
func ParseCredentials(data []byte, v Version) ([]CloudSyncCredentialResponse, error) {
	var raws []credentialRaw
	if err := json.Unmarshal(data, &raws); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	parse := parseCredentialV25
	if !v.AtLeast(25, 0) {
		parse = parseCredentialV24
	}

	results := make([]CloudSyncCredentialResponse, 0, len(raws))
	for _, raw := range raws {
		cred, err := parse(raw)
		if err != nil {
			return nil, err
		}
		results = append(results, cred)
	}
	return results, nil
}
