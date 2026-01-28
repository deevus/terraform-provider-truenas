package api

import "encoding/json"

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

// CloudSyncTaskResponse represents a cloud sync task from the API.
type CloudSyncTaskResponse struct {
	ID                 int64                       `json:"id"`
	Description        string                      `json:"description"`
	Path               string                      `json:"path"`
	Credentials        CloudSyncCredentialResponse `json:"credentials"`
	Attributes         json.RawMessage             `json:"attributes"` // Can be object or false
	Schedule           ScheduleResponse            `json:"schedule"`
	Direction          string                      `json:"direction"`
	TransferMode       string                      `json:"transfer_mode"`
	Encryption         bool                        `json:"encryption"`
	EncryptionPassword string                      `json:"encryption_password,omitempty"`
	EncryptionSalt     string                      `json:"encryption_salt,omitempty"`
	Snapshot           bool                        `json:"snapshot"`
	Transfers          int64                       `json:"transfers"`
	BWLimit            []BwLimit                   `json:"bwlimit"`
	Exclude            []string                    `json:"exclude"`
	Include            []string                    `json:"include"`
	FollowSymlinks     bool                        `json:"follow_symlinks"`
	CreateEmptySrcDirs bool                        `json:"create_empty_src_dirs"`
	Enabled            bool                        `json:"enabled"`
	Job                *JobStatus                  `json:"job,omitempty"`
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
