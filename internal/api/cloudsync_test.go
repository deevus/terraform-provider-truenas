package api

import (
	"encoding/json"
	"testing"
)

func Test_parseCredentialV25(t *testing.T) {
	tests := []struct {
		name    string
		raw     credentialRaw
		want    CloudSyncCredentialResponse
		wantErr bool
	}{
		{
			name: "parses S3 provider",
			raw: credentialRaw{
				ID:   1,
				Name: "Test Cred",
				Provider: json.RawMessage(`{
					"type": "S3",
					"access_key_id": "AKIATEST",
					"secret_access_key": "secret123",
					"endpoint": "s3.example.com",
					"region": "us-east-1"
				}`),
			},
			want: CloudSyncCredentialResponse{
				ID:   1,
				Name: "Test Cred",
				Provider: CloudSyncCredentialProvider{
					Type:            "S3",
					AccessKeyID:     "AKIATEST",
					SecretAccessKey: "secret123",
					Endpoint:        "s3.example.com",
					Region:          "us-east-1",
				},
			},
		},
		{
			name: "invalid JSON in provider",
			raw: credentialRaw{
				ID:       2,
				Name:     "Invalid Cred",
				Provider: json.RawMessage(`{not valid json`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCredentialV25(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCredentialV25() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.ID != tt.want.ID {
				t.Errorf("ID = %v, want %v", got.ID, tt.want.ID)
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.Provider.Type != tt.want.Provider.Type {
				t.Errorf("Provider.Type = %v, want %v", got.Provider.Type, tt.want.Provider.Type)
			}
			if got.Provider.AccessKeyID != tt.want.Provider.AccessKeyID {
				t.Errorf("Provider.AccessKeyID = %v, want %v", got.Provider.AccessKeyID, tt.want.Provider.AccessKeyID)
			}
			if got.Provider.SecretAccessKey != tt.want.Provider.SecretAccessKey {
				t.Errorf("Provider.SecretAccessKey = %v, want %v", got.Provider.SecretAccessKey, tt.want.Provider.SecretAccessKey)
			}
			if got.Provider.Endpoint != tt.want.Provider.Endpoint {
				t.Errorf("Provider.Endpoint = %v, want %v", got.Provider.Endpoint, tt.want.Provider.Endpoint)
			}
			if got.Provider.Region != tt.want.Provider.Region {
				t.Errorf("Provider.Region = %v, want %v", got.Provider.Region, tt.want.Provider.Region)
			}
		})
	}
}

func Test_parseCredentialV24(t *testing.T) {
	tests := []struct {
		name    string
		raw     credentialRaw
		want    CloudSyncCredentialResponse
		wantErr bool
	}{
		{
			name: "parses S3 provider",
			raw: credentialRaw{
				ID:       2,
				Name:     "Legacy Cred",
				Provider: json.RawMessage(`"S3"`),
				Attributes: json.RawMessage(`{
					"access_key_id": "AKIALEGACY",
					"secret_access_key": "legacysecret",
					"endpoint": "s3.legacy.com",
					"region": "eu-west-1"
				}`),
			},
			want: CloudSyncCredentialResponse{
				ID:   2,
				Name: "Legacy Cred",
				Provider: CloudSyncCredentialProvider{
					Type:            "S3",
					AccessKeyID:     "AKIALEGACY",
					SecretAccessKey: "legacysecret",
					Endpoint:        "s3.legacy.com",
					Region:          "eu-west-1",
				},
			},
		},
		{
			name: "invalid JSON in provider",
			raw: credentialRaw{
				ID:       3,
				Name:     "Invalid Provider",
				Provider: json.RawMessage(`{not valid json`),
			},
			wantErr: true,
		},
		{
			name: "invalid JSON in attributes",
			raw: credentialRaw{
				ID:         4,
				Name:       "Invalid Attributes",
				Provider:   json.RawMessage(`"S3"`),
				Attributes: json.RawMessage(`{not valid json`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCredentialV24(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCredentialV24() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.ID != tt.want.ID {
				t.Errorf("ID = %v, want %v", got.ID, tt.want.ID)
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.Provider.Type != tt.want.Provider.Type {
				t.Errorf("Provider.Type = %v, want %v", got.Provider.Type, tt.want.Provider.Type)
			}
			if got.Provider.AccessKeyID != tt.want.Provider.AccessKeyID {
				t.Errorf("Provider.AccessKeyID = %v, want %v", got.Provider.AccessKeyID, tt.want.Provider.AccessKeyID)
			}
			if got.Provider.SecretAccessKey != tt.want.Provider.SecretAccessKey {
				t.Errorf("Provider.SecretAccessKey = %v, want %v", got.Provider.SecretAccessKey, tt.want.Provider.SecretAccessKey)
			}
			if got.Provider.Endpoint != tt.want.Provider.Endpoint {
				t.Errorf("Provider.Endpoint = %v, want %v", got.Provider.Endpoint, tt.want.Provider.Endpoint)
			}
			if got.Provider.Region != tt.want.Provider.Region {
				t.Errorf("Provider.Region = %v, want %v", got.Provider.Region, tt.want.Provider.Region)
			}
		})
	}
}

func TestBuildCredentialsParams(t *testing.T) {
	tests := []struct {
		name         string
		version      Version
		credName     string
		providerType string
		attributes   map[string]any
		wantV25      bool // true if expecting 25.x format
	}{
		{
			name:         "V25 format with S3 attributes",
			version:      Version{Major: 25, Minor: 4},
			credName:     "Test",
			providerType: "S3",
			attributes: map[string]any{
				"access_key_id":     "AKIATEST",
				"secret_access_key": "secret",
			},
			wantV25: true,
		},
		{
			name:         "V25 format with minimal version",
			version:      Version{Major: 25, Minor: 0},
			credName:     "MinimalV25",
			providerType: "B2",
			attributes: map[string]any{
				"account": "b2account",
				"key":     "b2key",
			},
			wantV25: true,
		},
		{
			name:         "V24 format with S3 attributes",
			version:      Version{Major: 24, Minor: 10},
			credName:     "Legacy",
			providerType: "S3",
			attributes: map[string]any{
				"access_key_id":     "AKIALEGACY",
				"secret_access_key": "legacysecret",
			},
			wantV25: false,
		},
		{
			name:         "V24 format with older minor version",
			version:      Version{Major: 24, Minor: 4},
			credName:     "OlderLegacy",
			providerType: "B2",
			attributes: map[string]any{
				"account": "legacyaccount",
			},
			wantV25: false,
		},
		{
			name:         "V25 format with empty attributes",
			version:      Version{Major: 25, Minor: 4},
			credName:     "EmptyAttrs",
			providerType: "S3",
			attributes:   map[string]any{},
			wantV25:      true,
		},
		{
			name:         "V24 format with empty attributes",
			version:      Version{Major: 24, Minor: 10},
			credName:     "EmptyAttrsLegacy",
			providerType: "S3",
			attributes:   map[string]any{},
			wantV25:      false,
		},
		{
			name:         "V25 format with nil attributes",
			version:      Version{Major: 25, Minor: 4},
			credName:     "NilAttrs",
			providerType: "GCS",
			attributes:   nil,
			wantV25:      true,
		},
		{
			name:         "V24 format with nil attributes",
			version:      Version{Major: 24, Minor: 10},
			credName:     "NilAttrsLegacy",
			providerType: "GCS",
			attributes:   nil,
			wantV25:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := BuildCredentialsParams(tt.version, tt.credName, tt.providerType, tt.attributes)

			// Check name is always present
			if params["name"] != tt.credName {
				t.Errorf("expected name %q, got %v", tt.credName, params["name"])
			}

			if tt.wantV25 {
				// 25.x format: provider is object with type + attributes
				provider, ok := params["provider"].(map[string]any)
				if !ok {
					t.Fatalf("expected provider to be map[string]any for 25.x, got %T", params["provider"])
				}
				if provider["type"] != tt.providerType {
					t.Errorf("expected provider.type %q, got %v", tt.providerType, provider["type"])
				}
				// Verify attributes are merged into provider
				for k, v := range tt.attributes {
					if provider[k] != v {
						t.Errorf("expected provider[%q] = %v, got %v", k, v, provider[k])
					}
				}
				// Should NOT have separate attributes field
				if _, hasAttrs := params["attributes"]; hasAttrs {
					t.Error("25.x format should not have separate attributes field")
				}
			} else {
				// 24.x format: provider is string, attributes is separate object
				provider, ok := params["provider"].(string)
				if !ok {
					t.Fatalf("expected provider to be string for 24.x, got %T", params["provider"])
				}
				if provider != tt.providerType {
					t.Errorf("expected provider %q, got %v", tt.providerType, provider)
				}

				attributes, ok := params["attributes"].(map[string]any)
				if !ok {
					t.Fatalf("expected attributes to be map[string]any, got %T", params["attributes"])
				}
				// Verify attributes are in attributes field
				for k, v := range tt.attributes {
					if attributes[k] != v {
						t.Errorf("expected attributes[%q] = %v, got %v", k, v, attributes[k])
					}
				}
			}
		})
	}
}

func TestParseCredentials(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		version Version
		want    []CloudSyncCredentialResponse
		wantErr bool
	}{
		{
			name: "V25 format single credential",
			data: []byte(`[{
				"id": 1,
				"name": "Modern Cred",
				"provider": {
					"type": "B2",
					"account": "b2account",
					"key": "b2key"
				}
			}]`),
			version: Version{Major: 25, Minor: 4},
			want: []CloudSyncCredentialResponse{
				{
					ID:   1,
					Name: "Modern Cred",
					Provider: CloudSyncCredentialProvider{
						Type:    "B2",
						Account: "b2account",
						Key:     "b2key",
					},
				},
			},
		},
		{
			name: "V24 format single credential",
			data: []byte(`[{
				"id": 2,
				"name": "Legacy Cred",
				"provider": "B2",
				"attributes": {
					"account": "legacyaccount",
					"key": "legacykey"
				}
			}]`),
			version: Version{Major: 24, Minor: 10},
			want: []CloudSyncCredentialResponse{
				{
					ID:   2,
					Name: "Legacy Cred",
					Provider: CloudSyncCredentialProvider{
						Type:    "B2",
						Account: "legacyaccount",
						Key:     "legacykey",
					},
				},
			},
		},
		{
			name: "V25 format multiple credentials",
			data: []byte(`[
				{"id": 1, "name": "First", "provider": {"type": "S3", "access_key_id": "key1"}},
				{"id": 2, "name": "Second", "provider": {"type": "B2", "account": "acc2"}}
			]`),
			version: Version{Major: 25, Minor: 0},
			want: []CloudSyncCredentialResponse{
				{
					ID:   1,
					Name: "First",
					Provider: CloudSyncCredentialProvider{
						Type:        "S3",
						AccessKeyID: "key1",
					},
				},
				{
					ID:   2,
					Name: "Second",
					Provider: CloudSyncCredentialProvider{
						Type:    "B2",
						Account: "acc2",
					},
				},
			},
		},
		{
			name: "V24 format multiple credentials",
			data: []byte(`[
				{"id": 1, "name": "First", "provider": "S3", "attributes": {"access_key_id": "key1"}},
				{"id": 2, "name": "Second", "provider": "B2", "attributes": {"account": "acc2"}}
			]`),
			version: Version{Major: 24, Minor: 4},
			want: []CloudSyncCredentialResponse{
				{
					ID:   1,
					Name: "First",
					Provider: CloudSyncCredentialProvider{
						Type:        "S3",
						AccessKeyID: "key1",
					},
				},
				{
					ID:   2,
					Name: "Second",
					Provider: CloudSyncCredentialProvider{
						Type:    "B2",
						Account: "acc2",
					},
				},
			},
		},
		{
			name:    "invalid JSON array",
			data:    []byte(`{not valid json`),
			version: Version{Major: 25, Minor: 0},
			wantErr: true,
		},
		{
			name: "V25 invalid provider JSON",
			data: []byte(`[{
				"id": 1,
				"name": "Invalid",
				"provider": {invalid json}
			}]`),
			version: Version{Major: 25, Minor: 0},
			wantErr: true,
		},
		{
			name: "V24 invalid provider string",
			data: []byte(`[{
				"id": 1,
				"name": "Invalid",
				"provider": {not a string}
			}]`),
			version: Version{Major: 24, Minor: 10},
			wantErr: true,
		},
		{
			name:    "empty array",
			data:    []byte(`[]`),
			version: Version{Major: 25, Minor: 0},
			want:    []CloudSyncCredentialResponse{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCredentials(tt.data, tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseCredentials() returned %d credentials, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].ID != tt.want[i].ID {
					t.Errorf("credential[%d].ID = %v, want %v", i, got[i].ID, tt.want[i].ID)
				}
				if got[i].Name != tt.want[i].Name {
					t.Errorf("credential[%d].Name = %v, want %v", i, got[i].Name, tt.want[i].Name)
				}
				if got[i].Provider.Type != tt.want[i].Provider.Type {
					t.Errorf("credential[%d].Provider.Type = %v, want %v", i, got[i].Provider.Type, tt.want[i].Provider.Type)
				}
				if got[i].Provider.Account != tt.want[i].Provider.Account {
					t.Errorf("credential[%d].Provider.Account = %v, want %v", i, got[i].Provider.Account, tt.want[i].Provider.Account)
				}
				if got[i].Provider.Key != tt.want[i].Provider.Key {
					t.Errorf("credential[%d].Provider.Key = %v, want %v", i, got[i].Provider.Key, tt.want[i].Provider.Key)
				}
				if got[i].Provider.AccessKeyID != tt.want[i].Provider.AccessKeyID {
					t.Errorf("credential[%d].Provider.AccessKeyID = %v, want %v", i, got[i].Provider.AccessKeyID, tt.want[i].Provider.AccessKeyID)
				}
			}
		})
	}
}

func TestCloudSyncTaskResponse_EmbeddedCredentials(t *testing.T) {
	tests := []struct {
		name       string
		jsonData   string
		wantCredID int64
		wantErr    bool
	}{
		{
			name: "25.x format - provider is object",
			jsonData: `[{
				"id": 1,
				"description": "Test Task",
				"path": "/mnt/tank/data",
				"credentials": {
					"id": 5,
					"name": "Scaleway",
					"provider": {"type": "S3", "access_key_id": "AKIATEST"}
				},
				"attributes": {"bucket": "my-bucket"},
				"schedule": {"minute": "0", "hour": "3", "dom": "*", "month": "*", "dow": "*"},
				"direction": "PUSH",
				"transfer_mode": "SYNC",
				"encryption": false,
				"snapshot": false,
				"transfers": 4,
				"bwlimit": [],
				"exclude": [],
				"follow_symlinks": false,
				"create_empty_src_dirs": false,
				"enabled": true
			}]`,
			wantCredID: 5,
		},
		{
			name: "24.x format - provider is string",
			jsonData: `[{
				"id": 1,
				"description": "Test Task",
				"path": "/mnt/tank/data",
				"credentials": {
					"id": 5,
					"name": "Scaleway",
					"provider": "S3",
					"attributes": {"access_key_id": "AKIATEST"}
				},
				"attributes": {"bucket": "my-bucket"},
				"schedule": {"minute": "0", "hour": "3", "dom": "*", "month": "*", "dow": "*"},
				"direction": "PUSH",
				"transfer_mode": "SYNC",
				"encryption": false,
				"snapshot": false,
				"transfers": 4,
				"bwlimit": [],
				"exclude": [],
				"follow_symlinks": false,
				"create_empty_src_dirs": false,
				"enabled": true
			}]`,
			wantCredID: 5,
		},
		{
			name: "minimal credentials - only id and name",
			jsonData: `[{
				"id": 2,
				"description": "Minimal Task",
				"path": "/mnt/tank/minimal",
				"credentials": {"id": 10, "name": "Minimal Cred"},
				"attributes": {"bucket": "test"},
				"schedule": {"minute": "*", "hour": "*", "dom": "*", "month": "*", "dow": "*"},
				"direction": "PULL",
				"transfer_mode": "COPY",
				"encryption": false,
				"snapshot": false,
				"transfers": 2,
				"bwlimit": [],
				"exclude": [],
				"follow_symlinks": false,
				"create_empty_src_dirs": false,
				"enabled": true
			}]`,
			wantCredID: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tasks []CloudSyncTaskResponse
			err := json.Unmarshal([]byte(tt.jsonData), &tasks)

			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if len(tasks) != 1 {
				t.Fatalf("expected 1 task, got %d", len(tasks))
			}

			if tasks[0].Credentials.ID != tt.wantCredID {
				t.Errorf("Credentials.ID = %d, want %d", tasks[0].Credentials.ID, tt.wantCredID)
			}
		})
	}
}
