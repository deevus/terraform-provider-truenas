package provider

import (
	"context"
	"errors"
	"testing"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/truenas-go/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// mockClientFactory returns pre-configured mock clients for testing
type mockClientFactory struct {
	sshClient client.Client
	sshErr    error
	wsClient  client.Client
	wsErr     error
}

func (f *mockClientFactory) NewSSHClient(cfg *client.SSHConfig) (client.Client, error) {
	if f.sshErr != nil {
		return nil, f.sshErr
	}
	return f.sshClient, nil
}

func (f *mockClientFactory) NewWebSocketClient(cfg client.WebSocketConfig) (client.Client, error) {
	if f.wsErr != nil {
		return nil, f.wsErr
	}
	return f.wsClient, nil
}

// newTestMockClient creates a MockClient configured for testing
func newTestMockClient(version truenas.Version) *client.MockClient {
	return &client.MockClient{
		VersionVal:  version,
		ConnectFunc: func(ctx context.Context) error { return nil },
	}
}

func TestProvider_Metadata(t *testing.T) {
	p := New("1.0.0")()

	req := provider.MetadataRequest{}
	resp := &provider.MetadataResponse{}

	p.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas" {
		t.Errorf("expected TypeName 'truenas', got %q", resp.TypeName)
	}

	if resp.Version != "1.0.0" {
		t.Errorf("expected Version '1.0.0', got %q", resp.Version)
	}
}

func TestProvider_Schema(t *testing.T) {
	p := New("1.0.0")()

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}

	p.Schema(context.Background(), req, resp)

	// Verify schema has expected description
	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify host attribute exists and is required
	hostAttr, ok := resp.Schema.Attributes["host"]
	if !ok {
		t.Fatal("expected 'host' attribute in schema")
	}
	if !hostAttr.IsRequired() {
		t.Error("expected 'host' attribute to be required")
	}

	// Verify auth_method attribute exists and is required
	authMethodAttr, ok := resp.Schema.Attributes["auth_method"]
	if !ok {
		t.Fatal("expected 'auth_method' attribute in schema")
	}
	if !authMethodAttr.IsRequired() {
		t.Error("expected 'auth_method' attribute to be required")
	}

	// Verify ssh block exists
	_, ok = resp.Schema.Blocks["ssh"]
	if !ok {
		t.Fatal("expected 'ssh' block in schema")
	}
}

func TestProvider_Schema_SSHBlock(t *testing.T) {
	p := New("1.0.0")()

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}

	p.Schema(context.Background(), req, resp)

	// Get the SSH block
	sshBlock, ok := resp.Schema.Blocks["ssh"]
	if !ok {
		t.Fatal("expected 'ssh' block in schema")
	}

	// Cast to SingleNestedBlock to access attributes
	singleBlock, ok := sshBlock.(schema.SingleNestedBlock)
	if !ok {
		t.Fatalf("expected ssh block to be SingleNestedBlock, got %T", sshBlock)
	}

	// Verify port attribute exists and is optional
	portAttr, ok := singleBlock.Attributes["port"]
	if !ok {
		t.Fatal("expected 'port' attribute in ssh block")
	}
	if portAttr.IsRequired() {
		t.Error("expected 'port' attribute to be optional")
	}

	// Verify user attribute exists and is optional
	userAttr, ok := singleBlock.Attributes["user"]
	if !ok {
		t.Fatal("expected 'user' attribute in ssh block")
	}
	if userAttr.IsRequired() {
		t.Error("expected 'user' attribute to be optional")
	}

	// Verify private_key attribute exists, is required, and is sensitive
	privateKeyAttr, ok := singleBlock.Attributes["private_key"]
	if !ok {
		t.Fatal("expected 'private_key' attribute in ssh block")
	}
	if !privateKeyAttr.IsRequired() {
		t.Error("expected 'private_key' attribute to be required")
	}
	if !privateKeyAttr.IsSensitive() {
		t.Error("expected 'private_key' attribute to be sensitive")
	}
}

func TestProvider_Schema_SSHBlock_HostKeyFingerprint(t *testing.T) {
	p := New("1.0.0")()

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}

	p.Schema(context.Background(), req, resp)

	// Get the SSH block
	sshBlock, ok := resp.Schema.Blocks["ssh"]
	if !ok {
		t.Fatal("expected 'ssh' block in schema")
	}

	// Cast to SingleNestedBlock to access attributes
	singleBlock, ok := sshBlock.(schema.SingleNestedBlock)
	if !ok {
		t.Fatalf("expected ssh block to be SingleNestedBlock, got %T", sshBlock)
	}

	// Verify host_key_fingerprint attribute exists
	hostKeyFingerprintAttr, ok := singleBlock.Attributes["host_key_fingerprint"]
	if !ok {
		t.Fatal("expected 'host_key_fingerprint' attribute in ssh block")
	}

	// Verify it is required
	if !hostKeyFingerprintAttr.IsRequired() {
		t.Error("expected 'host_key_fingerprint' attribute to be required")
	}

	// Verify it is NOT sensitive (fingerprints are not secrets)
	if hostKeyFingerprintAttr.IsSensitive() {
		t.Error("expected 'host_key_fingerprint' attribute to NOT be sensitive")
	}
}

func TestProvider_Schema_WebSocketBlock(t *testing.T) {
	ctx := context.Background()
	p := New("test")()

	var resp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &resp)

	// Check websocket block exists
	wsBlock, ok := resp.Schema.Blocks["websocket"]
	if !ok {
		t.Fatal("websocket block not found in schema")
	}

	// Cast to SingleNestedBlock to access attributes
	singleBlock, ok := wsBlock.(schema.SingleNestedBlock)
	if !ok {
		t.Fatal("websocket block is not a SingleNestedBlock")
	}

	// Check that username and api_key attributes exist
	// (validation happens in Configure, not schema, to allow SSH-only configs)
	coreAttrs := []string{"username", "api_key"}
	for _, attr := range coreAttrs {
		_, ok := singleBlock.Attributes[attr]
		if !ok {
			t.Errorf("attribute %q not found in websocket block", attr)
		}
	}

	// Check api_key is sensitive
	apiKeyAttr := singleBlock.Attributes["api_key"]
	if !apiKeyAttr.IsSensitive() {
		t.Error("api_key attribute should be sensitive")
	}

	// Check optional attributes
	optionalAttrs := []string{"port", "insecure_skip_verify", "max_concurrent", "connect_timeout", "max_retries"}
	for _, attr := range optionalAttrs {
		a, ok := singleBlock.Attributes[attr]
		if !ok {
			t.Errorf("optional attribute %q not found in websocket block", attr)
		} else if a.IsRequired() {
			t.Errorf("attribute %q should be optional", attr)
		}
	}
}

func TestProvider_DataSources(t *testing.T) {
	p := &TrueNASProvider{version: "1.0.0"}

	dataSources := p.DataSources(context.Background())

	// Verify it returns a slice
	if dataSources == nil {
		t.Error("expected non-nil data sources slice")
	}

	// Collect all registered data source type names
	registered := make(map[string]bool)
	for _, factory := range dataSources {
		ds := factory()
		if ds == nil {
			continue
		}
		req := datasource.MetadataRequest{ProviderTypeName: "truenas"}
		resp := &datasource.MetadataResponse{}
		ds.Metadata(context.Background(), req, resp)
		registered[resp.TypeName] = true
	}

	// Verify expected data sources are registered
	expected := []string{
		"truenas_pool",
		"truenas_dataset",
		"truenas_snapshots",
		"truenas_cloudsync_credentials",
		"truenas_virt_config",
	}
	for _, name := range expected {
		if !registered[name] {
			t.Errorf("expected data source %q to be registered", name)
		}
	}
}

func TestProvider_Resources(t *testing.T) {
	p := &TrueNASProvider{version: "1.0.0"}

	resources := p.Resources(context.Background())

	// Verify it returns a slice
	if resources == nil {
		t.Error("expected non-nil resources slice")
	}

	// Collect all registered resource type names
	registered := make(map[string]bool)
	for _, factory := range resources {
		r := factory()
		if r == nil {
			continue
		}
		req := resource.MetadataRequest{ProviderTypeName: "truenas"}
		resp := &resource.MetadataResponse{}
		r.Metadata(context.Background(), req, resp)
		registered[resp.TypeName] = true
	}

	// Verify expected resources are registered
	expected := []string{
		"truenas_dataset",
		"truenas_host_path",
		"truenas_app",
		"truenas_file",
		"truenas_snapshot",
		"truenas_cloudsync_credentials",
		"truenas_cloudsync_task",
		"truenas_cron_job",
		"truenas_virt_config",
		"truenas_virt_instance",
	}
	for _, name := range expected {
		if !registered[name] {
			t.Errorf("expected resource %q to be registered", name)
		}
	}
}

// Test ED25519 key for testing (same as in client tests)
const testHostKeyFingerprint = "SHA256:uVW+XYZ0123456789ABCDEFghijklmnopqrstuv"

const testPrivateKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACCtws8zNrmNWDx+nxb26zA2iTVTn4TZQyK1yANm0XiawgAAAJjjXr/4416/
+AAAAAtzc2gtZWQyNTUxOQAAACCtws8zNrmNWDx+nxb26zA2iTVTn4TZQyK1yANm0Xiawg
AAAEARU6QyekrrGEM7eyo5JKVU08PPAbbO19sp/dB3xMSpaq3CzzM2uY1YPH6fFvbrMDaJ
NVOfhNlDIrXIA2bReJrCAAAAEnRlc3RAZXhhbXBsZS5sb2NhbAECAw==
-----END OPENSSH PRIVATE KEY-----`

// createTestConfigureRequest creates a provider.ConfigureRequest with the given config values
func createTestConfigureRequest(t *testing.T, host, authMethod string, ssh *SSHBlockModel) provider.ConfigureRequest {
	t.Helper()

	// Get the provider schema
	p := New("1.0.0")()
	schemaReq := provider.SchemaRequest{}
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), schemaReq, schemaResp)

	// Build SSH block value
	var sshValue tftypes.Value
	sshObjectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"port":                 tftypes.Number,
			"user":                 tftypes.String,
			"private_key":          tftypes.String,
			"host_key_fingerprint": tftypes.String,
			"max_sessions":         tftypes.Number,
		},
	}
	if ssh == nil {
		sshValue = tftypes.NewValue(sshObjectType, nil)
	} else {
		var portValue tftypes.Value
		if ssh.Port.IsNull() {
			portValue = tftypes.NewValue(tftypes.Number, nil)
		} else {
			portValue = tftypes.NewValue(tftypes.Number, ssh.Port.ValueInt64())
		}

		var userValue tftypes.Value
		if ssh.User.IsNull() {
			userValue = tftypes.NewValue(tftypes.String, nil)
		} else {
			userValue = tftypes.NewValue(tftypes.String, ssh.User.ValueString())
		}

		var privateKeyValue tftypes.Value
		if ssh.PrivateKey.IsNull() {
			privateKeyValue = tftypes.NewValue(tftypes.String, nil)
		} else {
			privateKeyValue = tftypes.NewValue(tftypes.String, ssh.PrivateKey.ValueString())
		}

		var hostKeyFingerprintValue tftypes.Value
		if ssh.HostKeyFingerprint.IsNull() {
			hostKeyFingerprintValue = tftypes.NewValue(tftypes.String, nil)
		} else {
			hostKeyFingerprintValue = tftypes.NewValue(tftypes.String, ssh.HostKeyFingerprint.ValueString())
		}

		var maxSessionsValue tftypes.Value
		if ssh.MaxSessions.IsNull() {
			maxSessionsValue = tftypes.NewValue(tftypes.Number, nil)
		} else {
			maxSessionsValue = tftypes.NewValue(tftypes.Number, ssh.MaxSessions.ValueInt64())
		}

		sshValue = tftypes.NewValue(sshObjectType, map[string]tftypes.Value{
			"port":                 portValue,
			"user":                 userValue,
			"private_key":          privateKeyValue,
			"host_key_fingerprint": hostKeyFingerprintValue,
			"max_sessions":         maxSessionsValue,
		})
	}

	// Build WebSocket block value (null for these tests)
	websocketObjectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"api_key":              tftypes.String,
			"port":                 tftypes.Number,
			"insecure_skip_verify": tftypes.Bool,
			"max_concurrent":       tftypes.Number,
			"connect_timeout":      tftypes.Number,
			"max_retries":          tftypes.Number,
		},
	}
	websocketValue := tftypes.NewValue(websocketObjectType, nil)

	// Build config value
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"host":        tftypes.String,
			"auth_method": tftypes.String,
			"ssh":         sshObjectType,
			"websocket":   websocketObjectType,
			"rate_limit":  tftypes.Number,
			"max_retries": tftypes.Number,
		},
	}, map[string]tftypes.Value{
		"host":        tftypes.NewValue(tftypes.String, host),
		"auth_method": tftypes.NewValue(tftypes.String, authMethod),
		"ssh":         sshValue,
		"websocket":   websocketValue,
		"rate_limit":  tftypes.NewValue(tftypes.Number, nil),
		"max_retries": tftypes.NewValue(tftypes.Number, nil),
	})

	config, diags := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    configValue,
	}, diag.Diagnostics{}

	if diags.HasError() {
		t.Fatalf("unexpected error creating config: %v", diags)
	}

	return provider.ConfigureRequest{
		Config: config,
	}
}

func TestProvider_Configure_InvalidAuthMethod(t *testing.T) {
	p := &TrueNASProvider{version: "1.0.0"}

	ssh := &SSHBlockModel{
		Port:               types.Int64Null(),
		User:               types.StringNull(),
		PrivateKey:         types.StringValue(testPrivateKey),
		HostKeyFingerprint: types.StringValue(testHostKeyFingerprint),
		MaxSessions:        types.Int64Null(),
	}

	req := createTestConfigureRequest(t, "truenas.local", "api", ssh)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid auth_method")
	}

	// Check that the error message mentions unsupported auth method
	found := false
	for _, d := range resp.Diagnostics {
		if d.Severity() == diag.SeverityError && (containsString(d.Summary(), "ssh") || containsString(d.Detail(), "ssh")) {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error message to mention 'ssh'")
	}
}

func TestProvider_Configure_MissingSSHBlock(t *testing.T) {
	p := &TrueNASProvider{version: "1.0.0"}

	req := createTestConfigureRequest(t, "truenas.local", "ssh", nil)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for missing SSH block")
	}
}

func TestProvider_Configure_Success(t *testing.T) {
	mock := newTestMockClient(truenas.Version{Major: 24, Minor: 10})

	p := &TrueNASProvider{
		version: "1.0.0",
		factory: &mockClientFactory{sshClient: mock},
	}

	ssh := &SSHBlockModel{
		Port:               types.Int64Null(),
		User:               types.StringNull(),
		PrivateKey:         types.StringValue(testPrivateKey),
		HostKeyFingerprint: types.StringValue(testHostKeyFingerprint),
		MaxSessions:        types.Int64Null(),
	}

	req := createTestConfigureRequest(t, "truenas.local", "ssh", ssh)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify client is set
	if resp.DataSourceData == nil {
		t.Error("expected DataSourceData to be set")
	}
	if resp.ResourceData == nil {
		t.Error("expected ResourceData to be set")
	}
}

func TestProvider_Configure_WithCustomPortAndUser(t *testing.T) {
	mock := newTestMockClient(truenas.Version{Major: 24, Minor: 10})

	p := &TrueNASProvider{
		version: "1.0.0",
		factory: &mockClientFactory{sshClient: mock},
	}

	ssh := &SSHBlockModel{
		Port:               types.Int64Value(2222),
		User:               types.StringValue("admin"),
		PrivateKey:         types.StringValue(testPrivateKey),
		HostKeyFingerprint: types.StringValue(testHostKeyFingerprint),
		MaxSessions:        types.Int64Null(),
	}

	req := createTestConfigureRequest(t, "truenas.local", "ssh", ssh)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestProvider_Configure_ConfigParseError(t *testing.T) {
	p := &TrueNASProvider{version: "1.0.0"}

	// Get the provider schema
	schemaReq := provider.SchemaRequest{}
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), schemaReq, schemaResp)

	// Create an invalid config value with wrong type (number instead of string for host)
	sshObjectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"port":                 tftypes.Number,
			"user":                 tftypes.String,
			"private_key":          tftypes.String,
			"host_key_fingerprint": tftypes.String,
			"max_sessions":         tftypes.Number,
		},
	}
	websocketObjectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"api_key":              tftypes.String,
			"port":                 tftypes.Number,
			"insecure_skip_verify": tftypes.Bool,
			"max_concurrent":       tftypes.Number,
			"connect_timeout":      tftypes.Number,
			"max_retries":          tftypes.Number,
		},
	}
	invalidConfigValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"host":        tftypes.Number, // Wrong type!
			"auth_method": tftypes.String,
			"ssh":         sshObjectType,
			"websocket":   websocketObjectType,
			"rate_limit":  tftypes.Number,
			"max_retries": tftypes.Number,
		},
	}, map[string]tftypes.Value{
		"host":        tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"auth_method": tftypes.NewValue(tftypes.String, "ssh"),
		"ssh": tftypes.NewValue(sshObjectType, map[string]tftypes.Value{
			"port":                 tftypes.NewValue(tftypes.Number, nil),
			"user":                 tftypes.NewValue(tftypes.String, nil),
			"private_key":          tftypes.NewValue(tftypes.String, testPrivateKey),
			"host_key_fingerprint": tftypes.NewValue(tftypes.String, testHostKeyFingerprint),
			"max_sessions":         tftypes.NewValue(tftypes.Number, nil),
		}),
		"websocket":   tftypes.NewValue(websocketObjectType, nil),
		"rate_limit":  tftypes.NewValue(tftypes.Number, nil),
		"max_retries": tftypes.NewValue(tftypes.Number, nil),
	})

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    invalidConfigValue,
	}

	req := provider.ConfigureRequest{
		Config: config,
	}
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	// Config parsing should fail due to type mismatch
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for config parse error")
	}
}

func TestProvider_Configure_InvalidSSHClient(t *testing.T) {
	p := &TrueNASProvider{version: "1.0.0"}

	// Create an SSH block with an empty private_key (will fail NewSSHClient validation)
	// We need to bypass schema validation, so we'll create the config manually
	// Get the provider schema
	schemaReq := provider.SchemaRequest{}
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), schemaReq, schemaResp)

	// Create config with empty private_key (this will fail client validation)
	sshObjectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"port":                 tftypes.Number,
			"user":                 tftypes.String,
			"private_key":          tftypes.String,
			"host_key_fingerprint": tftypes.String,
			"max_sessions":         tftypes.Number,
		},
	}
	websocketObjectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"api_key":              tftypes.String,
			"port":                 tftypes.Number,
			"insecure_skip_verify": tftypes.Bool,
			"max_concurrent":       tftypes.Number,
			"connect_timeout":      tftypes.Number,
			"max_retries":          tftypes.Number,
		},
	}
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"host":        tftypes.String,
			"auth_method": tftypes.String,
			"ssh":         sshObjectType,
			"websocket":   websocketObjectType,
			"rate_limit":  tftypes.Number,
			"max_retries": tftypes.Number,
		},
	}, map[string]tftypes.Value{
		"host":        tftypes.NewValue(tftypes.String, "truenas.local"),
		"auth_method": tftypes.NewValue(tftypes.String, "ssh"),
		"ssh": tftypes.NewValue(sshObjectType, map[string]tftypes.Value{
			"port":                 tftypes.NewValue(tftypes.Number, nil),
			"user":                 tftypes.NewValue(tftypes.String, nil),
			"private_key":          tftypes.NewValue(tftypes.String, ""), // Empty private key
			"host_key_fingerprint": tftypes.NewValue(tftypes.String, testHostKeyFingerprint),
			"max_sessions":         tftypes.NewValue(tftypes.Number, nil),
		}),
		"websocket":   tftypes.NewValue(websocketObjectType, nil),
		"rate_limit":  tftypes.NewValue(tftypes.Number, nil),
		"max_retries": tftypes.NewValue(tftypes.Number, nil),
	})

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    configValue,
	}

	req := provider.ConfigureRequest{
		Config: config,
	}
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid SSH client configuration")
	}

	// Check that the error mentions the client issue
	found := false
	for _, d := range resp.Diagnostics {
		if d.Severity() == diag.SeverityError {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error diagnostic")
	}
}

func TestTrueNASProvider_Resources_IncludesFile(t *testing.T) {
	p := New("test")()

	resources := p.Resources(context.Background())

	// Find file resource
	found := false
	for _, rf := range resources {
		r := rf()
		req := resource.MetadataRequest{ProviderTypeName: "truenas"}
		resp := &resource.MetadataResponse{}
		r.Metadata(context.Background(), req, resp)

		if resp.TypeName == "truenas_file" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected truenas_file resource to be registered")
	}
}

func TestProviderSchema_RateLimitAttributes(t *testing.T) {
	ctx := context.Background()
	p := New("test")()

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}
	p.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Check rate_limit attribute exists
	rateLimitAttr, ok := resp.Schema.Attributes["rate_limit"]
	if !ok {
		t.Error("expected rate_limit attribute in schema")
	} else {
		if rateLimitAttr.IsRequired() {
			t.Error("rate_limit should be optional")
		}
	}

	// Check max_retries attribute exists
	maxRetriesAttr, ok := resp.Schema.Attributes["max_retries"]
	if !ok {
		t.Error("expected max_retries attribute in schema")
	} else {
		if maxRetriesAttr.IsRequired() {
			t.Error("max_retries should be optional")
		}
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// createTestConfigureRequestWithWebSocket creates a provider.ConfigureRequest with SSH and WebSocket config
func createTestConfigureRequestWithWebSocket(t *testing.T, host, authMethod string, ssh *SSHBlockModel, ws *WebSocketBlockModel) provider.ConfigureRequest {
	t.Helper()

	// Get the provider schema
	p := New("1.0.0")()
	schemaReq := provider.SchemaRequest{}
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), schemaReq, schemaResp)

	// Build SSH block value
	var sshValue tftypes.Value
	sshObjectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"port":                 tftypes.Number,
			"user":                 tftypes.String,
			"private_key":          tftypes.String,
			"host_key_fingerprint": tftypes.String,
			"max_sessions":         tftypes.Number,
		},
	}
	if ssh == nil {
		sshValue = tftypes.NewValue(sshObjectType, nil)
	} else {
		var portValue tftypes.Value
		if ssh.Port.IsNull() {
			portValue = tftypes.NewValue(tftypes.Number, nil)
		} else {
			portValue = tftypes.NewValue(tftypes.Number, ssh.Port.ValueInt64())
		}

		var userValue tftypes.Value
		if ssh.User.IsNull() {
			userValue = tftypes.NewValue(tftypes.String, nil)
		} else {
			userValue = tftypes.NewValue(tftypes.String, ssh.User.ValueString())
		}

		var privateKeyValue tftypes.Value
		if ssh.PrivateKey.IsNull() {
			privateKeyValue = tftypes.NewValue(tftypes.String, nil)
		} else {
			privateKeyValue = tftypes.NewValue(tftypes.String, ssh.PrivateKey.ValueString())
		}

		var hostKeyFingerprintValue tftypes.Value
		if ssh.HostKeyFingerprint.IsNull() {
			hostKeyFingerprintValue = tftypes.NewValue(tftypes.String, nil)
		} else {
			hostKeyFingerprintValue = tftypes.NewValue(tftypes.String, ssh.HostKeyFingerprint.ValueString())
		}

		var maxSessionsValue tftypes.Value
		if ssh.MaxSessions.IsNull() {
			maxSessionsValue = tftypes.NewValue(tftypes.Number, nil)
		} else {
			maxSessionsValue = tftypes.NewValue(tftypes.Number, ssh.MaxSessions.ValueInt64())
		}

		sshValue = tftypes.NewValue(sshObjectType, map[string]tftypes.Value{
			"port":                 portValue,
			"user":                 userValue,
			"private_key":          privateKeyValue,
			"host_key_fingerprint": hostKeyFingerprintValue,
			"max_sessions":         maxSessionsValue,
		})
	}

	// Build WebSocket block value
	var websocketValue tftypes.Value
	websocketObjectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"username":             tftypes.String,
			"api_key":              tftypes.String,
			"port":                 tftypes.Number,
			"insecure_skip_verify": tftypes.Bool,
			"max_concurrent":       tftypes.Number,
			"connect_timeout":      tftypes.Number,
			"max_retries":          tftypes.Number,
		},
	}
	if ws == nil {
		websocketValue = tftypes.NewValue(websocketObjectType, nil)
	} else {
		var usernameValue tftypes.Value
		if ws.Username.IsNull() {
			usernameValue = tftypes.NewValue(tftypes.String, nil)
		} else {
			usernameValue = tftypes.NewValue(tftypes.String, ws.Username.ValueString())
		}

		var apiKeyValue tftypes.Value
		if ws.APIKey.IsNull() {
			apiKeyValue = tftypes.NewValue(tftypes.String, nil)
		} else {
			apiKeyValue = tftypes.NewValue(tftypes.String, ws.APIKey.ValueString())
		}

		var portValue tftypes.Value
		if ws.Port.IsNull() {
			portValue = tftypes.NewValue(tftypes.Number, nil)
		} else {
			portValue = tftypes.NewValue(tftypes.Number, ws.Port.ValueInt64())
		}

		var insecureSkipVerifyValue tftypes.Value
		if ws.InsecureSkipVerify.IsNull() {
			insecureSkipVerifyValue = tftypes.NewValue(tftypes.Bool, nil)
		} else {
			insecureSkipVerifyValue = tftypes.NewValue(tftypes.Bool, ws.InsecureSkipVerify.ValueBool())
		}

		var maxConcurrentValue tftypes.Value
		if ws.MaxConcurrent.IsNull() {
			maxConcurrentValue = tftypes.NewValue(tftypes.Number, nil)
		} else {
			maxConcurrentValue = tftypes.NewValue(tftypes.Number, ws.MaxConcurrent.ValueInt64())
		}

		var connectTimeoutValue tftypes.Value
		if ws.ConnectTimeout.IsNull() {
			connectTimeoutValue = tftypes.NewValue(tftypes.Number, nil)
		} else {
			connectTimeoutValue = tftypes.NewValue(tftypes.Number, ws.ConnectTimeout.ValueInt64())
		}

		var maxRetriesValue tftypes.Value
		if ws.MaxRetries.IsNull() {
			maxRetriesValue = tftypes.NewValue(tftypes.Number, nil)
		} else {
			maxRetriesValue = tftypes.NewValue(tftypes.Number, ws.MaxRetries.ValueInt64())
		}

		websocketValue = tftypes.NewValue(websocketObjectType, map[string]tftypes.Value{
			"username":             usernameValue,
			"api_key":              apiKeyValue,
			"port":                 portValue,
			"insecure_skip_verify": insecureSkipVerifyValue,
			"max_concurrent":       maxConcurrentValue,
			"connect_timeout":      connectTimeoutValue,
			"max_retries":          maxRetriesValue,
		})
	}

	// Build config value
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"host":        tftypes.String,
			"auth_method": tftypes.String,
			"ssh":         sshObjectType,
			"websocket":   websocketObjectType,
			"rate_limit":  tftypes.Number,
			"max_retries": tftypes.Number,
		},
	}, map[string]tftypes.Value{
		"host":        tftypes.NewValue(tftypes.String, host),
		"auth_method": tftypes.NewValue(tftypes.String, authMethod),
		"ssh":         sshValue,
		"websocket":   websocketValue,
		"rate_limit":  tftypes.NewValue(tftypes.Number, nil),
		"max_retries": tftypes.NewValue(tftypes.Number, nil),
	})

	config, diags := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    configValue,
	}, diag.Diagnostics{}

	if diags.HasError() {
		t.Fatalf("unexpected error creating config: %v", diags)
	}

	return provider.ConfigureRequest{
		Config: config,
	}
}

func TestProvider_Configure_WebSocketAuthMethod_MissingWebSocketBlock(t *testing.T) {
	p := &TrueNASProvider{version: "1.0.0"}

	ssh := &SSHBlockModel{
		Port:               types.Int64Null(),
		User:               types.StringNull(),
		PrivateKey:         types.StringValue(testPrivateKey),
		HostKeyFingerprint: types.StringValue(testHostKeyFingerprint),
		MaxSessions:        types.Int64Null(),
	}

	req := createTestConfigureRequestWithWebSocket(t, "truenas.local", "websocket", ssh, nil)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for missing websocket block")
	}

	// Check error message mentions WebSocket
	found := false
	for _, d := range resp.Diagnostics {
		if d.Severity() == diag.SeverityError && containsString(d.Summary(), "WebSocket") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error message to mention 'WebSocket'")
	}
}

func TestProvider_Configure_WebSocketAuthMethod_MissingSSHBlock(t *testing.T) {
	p := &TrueNASProvider{version: "1.0.0"}

	ws := &WebSocketBlockModel{
		Username:           types.StringValue("root"),
		APIKey:             types.StringValue("test-api-key"),
		Port:               types.Int64Null(),
		InsecureSkipVerify: types.BoolNull(),
		MaxConcurrent:      types.Int64Null(),
		ConnectTimeout:     types.Int64Null(),
		MaxRetries:         types.Int64Null(),
	}

	req := createTestConfigureRequestWithWebSocket(t, "truenas.local", "websocket", nil, ws)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for missing ssh block when using websocket")
	}

	// Check error message mentions SSH and fallback
	found := false
	for _, d := range resp.Diagnostics {
		if d.Severity() == diag.SeverityError && containsString(d.Detail(), "fallback") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error message to mention 'fallback'")
	}
}

func TestProvider_Configure_WebSocketAuthMethod_Success(t *testing.T) {
	sshMock := newTestMockClient(truenas.Version{Major: 25, Minor: 0})
	wsMock := newTestMockClient(truenas.Version{Major: 25, Minor: 0})

	p := &TrueNASProvider{
		version: "1.0.0",
		factory: &mockClientFactory{
			sshClient: sshMock,
			wsClient:  wsMock,
		},
	}

	ssh := &SSHBlockModel{
		Port:               types.Int64Null(),
		User:               types.StringNull(),
		PrivateKey:         types.StringValue(testPrivateKey),
		HostKeyFingerprint: types.StringValue(testHostKeyFingerprint),
		MaxSessions:        types.Int64Null(),
	}

	ws := &WebSocketBlockModel{
		Username:           types.StringValue("root"),
		APIKey:             types.StringValue("test-api-key"),
		Port:               types.Int64Null(),
		InsecureSkipVerify: types.BoolNull(),
		MaxConcurrent:      types.Int64Null(),
		ConnectTimeout:     types.Int64Null(),
		MaxRetries:         types.Int64Null(),
	}

	req := createTestConfigureRequestWithWebSocket(t, "truenas.local", "websocket", ssh, ws)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify client is set
	if resp.DataSourceData == nil {
		t.Error("expected DataSourceData to be set")
	}
	if resp.ResourceData == nil {
		t.Error("expected ResourceData to be set")
	}
}

func TestProvider_Configure_WebSocketAuthMethod_WithAllOptions(t *testing.T) {
	sshMock := newTestMockClient(truenas.Version{Major: 25, Minor: 0})
	wsMock := newTestMockClient(truenas.Version{Major: 25, Minor: 0})

	p := &TrueNASProvider{
		version: "1.0.0",
		factory: &mockClientFactory{
			sshClient: sshMock,
			wsClient:  wsMock,
		},
	}

	ssh := &SSHBlockModel{
		Port:               types.Int64Value(2222),
		User:               types.StringValue("admin"),
		PrivateKey:         types.StringValue(testPrivateKey),
		HostKeyFingerprint: types.StringValue(testHostKeyFingerprint),
		MaxSessions:        types.Int64Value(10),
	}

	ws := &WebSocketBlockModel{
		Username:           types.StringValue("root"),
		APIKey:             types.StringValue("test-api-key"),
		Port:               types.Int64Value(8443),
		InsecureSkipVerify: types.BoolValue(true),
		MaxConcurrent:      types.Int64Value(30),
		ConnectTimeout:     types.Int64Value(60),
		MaxRetries:         types.Int64Value(5),
	}

	req := createTestConfigureRequestWithWebSocket(t, "truenas.local", "websocket", ssh, ws)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify client is set
	if resp.DataSourceData == nil {
		t.Error("expected DataSourceData to be set")
	}
	if resp.ResourceData == nil {
		t.Error("expected ResourceData to be set")
	}
}

func TestProvider_Configure_InvalidAuthMethod_MentionsWebSocket(t *testing.T) {
	p := &TrueNASProvider{version: "1.0.0"}

	ssh := &SSHBlockModel{
		Port:               types.Int64Null(),
		User:               types.StringNull(),
		PrivateKey:         types.StringValue(testPrivateKey),
		HostKeyFingerprint: types.StringValue(testHostKeyFingerprint),
		MaxSessions:        types.Int64Null(),
	}

	req := createTestConfigureRequestWithWebSocket(t, "truenas.local", "invalid", ssh, nil)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid auth_method")
	}

	// Check error message mentions both ssh and websocket options
	found := false
	for _, d := range resp.Diagnostics {
		if d.Severity() == diag.SeverityError {
			detail := d.Detail()
			if containsString(detail, "ssh") && containsString(detail, "websocket") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("expected error message to mention both 'ssh' and 'websocket' as valid options")
	}
}

func TestProvider_Configure_EmptyAuthMethod_DefaultsToSSH(t *testing.T) {
	mock := newTestMockClient(truenas.Version{Major: 24, Minor: 10})

	p := &TrueNASProvider{
		version: "1.0.0",
		factory: &mockClientFactory{sshClient: mock},
	}

	ssh := &SSHBlockModel{
		Port:               types.Int64Null(),
		User:               types.StringNull(),
		PrivateKey:         types.StringValue(testPrivateKey),
		HostKeyFingerprint: types.StringValue(testHostKeyFingerprint),
		MaxSessions:        types.Int64Null(),
	}

	req := createTestConfigureRequestWithWebSocket(t, "truenas.local", "", ssh, nil)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify client is set (should work with empty auth_method defaulting to ssh)
	if resp.DataSourceData == nil {
		t.Error("expected DataSourceData to be set")
	}
	if resp.ResourceData == nil {
		t.Error("expected ResourceData to be set")
	}
}

func TestProvider_Configure_WebSocketAuthMethod_EmptySSHKey(t *testing.T) {
	p := &TrueNASProvider{version: "1.0.0"}

	// Use empty SSH key to trigger SSH client creation error
	ssh := &SSHBlockModel{
		Port:               types.Int64Null(),
		User:               types.StringNull(),
		PrivateKey:         types.StringValue(""),
		HostKeyFingerprint: types.StringValue(testHostKeyFingerprint),
		MaxSessions:        types.Int64Null(),
	}

	ws := &WebSocketBlockModel{
		Username:           types.StringValue("root"),
		APIKey:             types.StringValue("test-api-key"),
		Port:               types.Int64Null(),
		InsecureSkipVerify: types.BoolNull(),
		MaxConcurrent:      types.Int64Null(),
		ConnectTimeout:     types.Int64Null(),
		MaxRetries:         types.Int64Null(),
	}

	req := createTestConfigureRequestWithWebSocket(t, "truenas.local", "websocket", ssh, ws)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for empty SSH key in websocket mode")
	}

	// Check error message mentions SSH
	found := false
	for _, d := range resp.Diagnostics {
		if d.Severity() == diag.SeverityError && containsString(d.Summary(), "SSH") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error message to mention 'SSH'")
	}
}

func TestProvider_Configure_WebSocketAuthMethod_OldVersionRejected(t *testing.T) {
	// TrueNAS 24.x should be rejected for WebSocket mode
	sshMock := newTestMockClient(truenas.Version{Major: 24, Minor: 10, Raw: "24.10"})

	p := &TrueNASProvider{
		version: "1.0.0",
		factory: &mockClientFactory{sshClient: sshMock},
	}

	ssh := &SSHBlockModel{
		Port:               types.Int64Null(),
		User:               types.StringNull(),
		PrivateKey:         types.StringValue(testPrivateKey),
		HostKeyFingerprint: types.StringValue(testHostKeyFingerprint),
		MaxSessions:        types.Int64Null(),
	}

	ws := &WebSocketBlockModel{
		Username:           types.StringValue("root"),
		APIKey:             types.StringValue("test-api-key"),
		Port:               types.Int64Null(),
		InsecureSkipVerify: types.BoolNull(),
		MaxConcurrent:      types.Int64Null(),
		ConnectTimeout:     types.Int64Null(),
		MaxRetries:         types.Int64Null(),
	}

	req := createTestConfigureRequestWithWebSocket(t, "truenas.local", "websocket", ssh, ws)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for TrueNAS < 25.0 with WebSocket mode")
	}

	// Verify error message mentions version requirement
	found := false
	for _, d := range resp.Diagnostics {
		if d.Severity() == diag.SeverityError && containsString(d.Summary(), "25.0") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error message to mention '25.0' version requirement")
	}
}

func TestProvider_Configure_SSHConnectError(t *testing.T) {
	mock := &client.MockClient{
		ConnectFunc: func(ctx context.Context) error {
			return errors.New("connection refused")
		},
	}

	p := &TrueNASProvider{
		version: "1.0.0",
		factory: &mockClientFactory{sshClient: mock},
	}

	ssh := &SSHBlockModel{
		Port:               types.Int64Null(),
		User:               types.StringNull(),
		PrivateKey:         types.StringValue(testPrivateKey),
		HostKeyFingerprint: types.StringValue(testHostKeyFingerprint),
		MaxSessions:        types.Int64Null(),
	}

	req := createTestConfigureRequest(t, "truenas.local", "ssh", ssh)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for connection failure")
	}

	// Verify diagnostics contains connection error
	found := false
	for _, d := range resp.Diagnostics {
		if d.Severity() == diag.SeverityError && containsString(d.Detail(), "connection refused") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error detail to contain 'connection refused'")
	}
}

func TestProvider_Configure_WebSocketConnectError(t *testing.T) {
	sshMock := newTestMockClient(truenas.Version{Major: 25, Minor: 0})
	wsMock := &client.MockClient{
		ConnectFunc: func(ctx context.Context) error {
			return errors.New("websocket handshake failed")
		},
	}

	p := &TrueNASProvider{
		version: "1.0.0",
		factory: &mockClientFactory{
			sshClient: sshMock,
			wsClient:  wsMock,
		},
	}

	ssh := &SSHBlockModel{
		Port:               types.Int64Null(),
		User:               types.StringNull(),
		PrivateKey:         types.StringValue(testPrivateKey),
		HostKeyFingerprint: types.StringValue(testHostKeyFingerprint),
		MaxSessions:        types.Int64Null(),
	}

	ws := &WebSocketBlockModel{
		Username:           types.StringValue("root"),
		APIKey:             types.StringValue("test-api-key"),
		Port:               types.Int64Null(),
		InsecureSkipVerify: types.BoolNull(),
		MaxConcurrent:      types.Int64Null(),
		ConnectTimeout:     types.Int64Null(),
		MaxRetries:         types.Int64Null(),
	}

	req := createTestConfigureRequestWithWebSocket(t, "truenas.local", "websocket", ssh, ws)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for WebSocket connection failure")
	}

	// Verify diagnostics contains connection error
	found := false
	for _, d := range resp.Diagnostics {
		if d.Severity() == diag.SeverityError && containsString(d.Detail(), "websocket handshake failed") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error detail to contain 'websocket handshake failed'")
	}
}

func TestProvider_Configure_WebSocketSSHConnectError(t *testing.T) {
	// Test that SSH Connect error in WebSocket mode is properly handled
	sshMock := &client.MockClient{
		ConnectFunc: func(ctx context.Context) error {
			return errors.New("ssh connection refused")
		},
	}

	p := &TrueNASProvider{
		version: "1.0.0",
		factory: &mockClientFactory{sshClient: sshMock},
	}

	ssh := &SSHBlockModel{
		Port:               types.Int64Null(),
		User:               types.StringNull(),
		PrivateKey:         types.StringValue(testPrivateKey),
		HostKeyFingerprint: types.StringValue(testHostKeyFingerprint),
		MaxSessions:        types.Int64Null(),
	}

	ws := &WebSocketBlockModel{
		Username:           types.StringValue("root"),
		APIKey:             types.StringValue("test-api-key"),
		Port:               types.Int64Null(),
		InsecureSkipVerify: types.BoolNull(),
		MaxConcurrent:      types.Int64Null(),
		ConnectTimeout:     types.Int64Null(),
		MaxRetries:         types.Int64Null(),
	}

	req := createTestConfigureRequestWithWebSocket(t, "truenas.local", "websocket", ssh, ws)
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for SSH connection failure in WebSocket mode")
	}

	// Verify diagnostics contains connection error
	found := false
	for _, d := range resp.Diagnostics {
		if d.Severity() == diag.SeverityError && containsString(d.Detail(), "ssh connection refused") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error detail to contain 'ssh connection refused'")
	}
}
