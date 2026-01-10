package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

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

func TestProvider_DataSources(t *testing.T) {
	p := &TrueNASProvider{version: "1.0.0"}

	dataSources := p.DataSources(context.Background())

	// Verify it returns a slice
	if dataSources == nil {
		t.Error("expected non-nil data sources slice")
	}

	// Verify we have the expected data sources
	if len(dataSources) != 2 {
		t.Errorf("expected 2 data sources, got %d", len(dataSources))
	}

	// Verify the return type
	var _ []func() datasource.DataSource = dataSources

	// Verify each factory function returns a valid data source
	for i, factory := range dataSources {
		ds := factory()
		if ds == nil {
			t.Errorf("data source factory %d returned nil", i)
		}
	}
}

func TestProvider_Resources(t *testing.T) {
	p := &TrueNASProvider{version: "1.0.0"}

	resources := p.Resources(context.Background())

	// Verify it returns a slice (can be empty for now)
	if resources == nil {
		t.Error("expected non-nil resources slice")
	}

	// Currently no resources are implemented
	if len(resources) != 0 {
		t.Errorf("expected 0 resources, got %d", len(resources))
	}

	// Verify the return type
	var _ []func() resource.Resource = resources
}

// Test ED25519 key for testing (same as in client tests)
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
	if ssh == nil {
		sshValue = tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"port":        tftypes.Number,
				"user":        tftypes.String,
				"private_key": tftypes.String,
			},
		}, nil)
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

		sshValue = tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"port":        tftypes.Number,
				"user":        tftypes.String,
				"private_key": tftypes.String,
			},
		}, map[string]tftypes.Value{
			"port":        portValue,
			"user":        userValue,
			"private_key": privateKeyValue,
		})
	}

	// Build config value
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"host":        tftypes.String,
			"auth_method": tftypes.String,
			"ssh": tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"port":        tftypes.Number,
					"user":        tftypes.String,
					"private_key": tftypes.String,
				},
			},
		},
	}, map[string]tftypes.Value{
		"host":        tftypes.NewValue(tftypes.String, host),
		"auth_method": tftypes.NewValue(tftypes.String, authMethod),
		"ssh":         sshValue,
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
		Port:       types.Int64Null(),
		User:       types.StringNull(),
		PrivateKey: types.StringValue(testPrivateKey),
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
	p := &TrueNASProvider{version: "1.0.0"}

	ssh := &SSHBlockModel{
		Port:       types.Int64Null(),
		User:       types.StringNull(),
		PrivateKey: types.StringValue(testPrivateKey),
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
	p := &TrueNASProvider{version: "1.0.0"}

	ssh := &SSHBlockModel{
		Port:       types.Int64Value(2222),
		User:       types.StringValue("admin"),
		PrivateKey: types.StringValue(testPrivateKey),
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
	invalidConfigValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"host":        tftypes.Number, // Wrong type!
			"auth_method": tftypes.String,
			"ssh": tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"port":        tftypes.Number,
					"user":        tftypes.String,
					"private_key": tftypes.String,
				},
			},
		},
	}, map[string]tftypes.Value{
		"host":        tftypes.NewValue(tftypes.Number, 123), // Wrong type!
		"auth_method": tftypes.NewValue(tftypes.String, "ssh"),
		"ssh": tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"port":        tftypes.Number,
				"user":        tftypes.String,
				"private_key": tftypes.String,
			},
		}, map[string]tftypes.Value{
			"port":        tftypes.NewValue(tftypes.Number, nil),
			"user":        tftypes.NewValue(tftypes.String, nil),
			"private_key": tftypes.NewValue(tftypes.String, testPrivateKey),
		}),
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
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"host":        tftypes.String,
			"auth_method": tftypes.String,
			"ssh": tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"port":        tftypes.Number,
					"user":        tftypes.String,
					"private_key": tftypes.String,
				},
			},
		},
	}, map[string]tftypes.Value{
		"host":        tftypes.NewValue(tftypes.String, "truenas.local"),
		"auth_method": tftypes.NewValue(tftypes.String, "ssh"),
		"ssh": tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"port":        tftypes.Number,
				"user":        tftypes.String,
				"private_key": tftypes.String,
			},
		}, map[string]tftypes.Value{
			"port":        tftypes.NewValue(tftypes.Number, nil),
			"user":        tftypes.NewValue(tftypes.String, nil),
			"private_key": tftypes.NewValue(tftypes.String, ""), // Empty private key
		}),
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
