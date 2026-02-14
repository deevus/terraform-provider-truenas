package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/deevus/terraform-provider-truenas/internal/datasources"
	"github.com/deevus/terraform-provider-truenas/internal/resources"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &TrueNASProvider{}

// TrueNASProviderModel describes the provider data model.
type TrueNASProviderModel struct {
	Host       types.String         `tfsdk:"host"`
	AuthMethod types.String         `tfsdk:"auth_method"`
	SSH        *SSHBlockModel       `tfsdk:"ssh"`
	WebSocket  *WebSocketBlockModel `tfsdk:"websocket"`
	RateLimit  types.Int64          `tfsdk:"rate_limit"`
	MaxRetries types.Int64          `tfsdk:"max_retries"`
}

// SSHBlockModel describes the SSH configuration block.
type SSHBlockModel struct {
	Port               types.Int64  `tfsdk:"port"`
	User               types.String `tfsdk:"user"`
	PrivateKey         types.String `tfsdk:"private_key"`
	HostKeyFingerprint types.String `tfsdk:"host_key_fingerprint"`
	MaxSessions        types.Int64  `tfsdk:"max_sessions"`
}

// WebSocketBlockModel describes the WebSocket configuration block.
type WebSocketBlockModel struct {
	Username           types.String `tfsdk:"username"`
	APIKey             types.String `tfsdk:"api_key"`
	Port               types.Int64  `tfsdk:"port"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
	MaxConcurrent      types.Int64  `tfsdk:"max_concurrent"`
	ConnectTimeout     types.Int64  `tfsdk:"connect_timeout"`
	MaxRetries         types.Int64  `tfsdk:"max_retries"`
}

type TrueNASProvider struct {
	version string
	factory ClientFactory
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TrueNASProvider{
			version: version,
		}
	}
}

func (p *TrueNASProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "truenas"
	resp.Version = p.version
}

func (p *TrueNASProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for TrueNAS SCALE and Community Edition.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "TrueNAS server hostname or IP address.",
				Required:    true,
			},
			"auth_method": schema.StringAttribute{
				Description: "Authentication method: 'ssh' or 'websocket'. WebSocket requires both websocket and ssh blocks (ssh is used for fallback operations).",
				Required:    true,
			},
			"rate_limit": schema.Int64Attribute{
				Description: "Maximum API calls per minute. Default: 300 (5 per second). " +
					"Set to 0 to disable rate limiting.",
				Optional: true,
			},
			"max_retries": schema.Int64Attribute{
				Description: "Maximum retry attempts for transient connection errors. Default: 3. " +
					"Set to 0 to disable retries.",
				Optional: true,
			},
		},
		Blocks: map[string]schema.Block{
			"ssh": schema.SingleNestedBlock{
				Description: "SSH connection configuration.",
				Attributes: map[string]schema.Attribute{
					"port": schema.Int64Attribute{
						Description: "SSH port. Defaults to 22.",
						Optional:    true,
					},
					"user": schema.StringAttribute{
						Description: "SSH username. Defaults to 'root'.",
						Optional:    true,
					},
					"private_key": schema.StringAttribute{
						Description: "SSH private key content.",
						Required:    true,
						Sensitive:   true,
					},
					"host_key_fingerprint": schema.StringAttribute{
						Description: "SHA256 fingerprint of the TrueNAS server's SSH host key. " +
							"Get it with: ssh-keyscan <host> 2>/dev/null | ssh-keygen -lf -",
						Required:  true,
						Sensitive: false,
					},
					"max_sessions": schema.Int64Attribute{
						Description: "Maximum concurrent SSH sessions. Defaults to 5. " +
							"Increase for large deployments, decrease if you see connection errors.",
						Optional: true,
					},
				},
			},
			"websocket": schema.SingleNestedBlock{
				Description: "WebSocket connection configuration. Required when auth_method is 'websocket'.",
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						Description: "TrueNAS username associated with the API key. Usually 'root'.",
						Optional:    true,
					},
					"api_key": schema.StringAttribute{
						Description: "TrueNAS API key for authentication.",
						Optional:    true,
						Sensitive:   true,
					},
					"port": schema.Int64Attribute{
						Description: "WebSocket port. Defaults to 443.",
						Optional:    true,
					},
					"insecure_skip_verify": schema.BoolAttribute{
						Description: "Skip TLS certificate verification. Defaults to false.",
						Optional:    true,
					},
					"max_concurrent": schema.Int64Attribute{
						Description: "Maximum concurrent in-flight requests. Defaults to 20.",
						Optional:    true,
					},
					"connect_timeout": schema.Int64Attribute{
						Description: "Connection timeout in seconds. Defaults to 30.",
						Optional:    true,
					},
					"max_retries": schema.Int64Attribute{
						Description: "Maximum retry attempts for transient errors. Defaults to 3.",
						Optional:    true,
					},
				},
			},
		},
	}
}

func (p *TrueNASProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config TrueNASProviderModel

	// Parse configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve factory (use default if not set)
	factory := p.factory
	if factory == nil {
		factory = &DefaultClientFactory{}
	}

	var finalClient client.Client

	switch config.AuthMethod.ValueString() {
	case "websocket":
		// Validate websocket block
		if config.WebSocket == nil {
			resp.Diagnostics.AddError(
				"Missing WebSocket Configuration",
				"WebSocket block is required when auth_method is 'websocket'.",
			)
			return
		}

		// Validate required websocket attributes
		if config.WebSocket.Username.IsNull() || config.WebSocket.Username.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing WebSocket Username",
				"websocket.username is required when auth_method is 'websocket'.",
			)
			return
		}
		if config.WebSocket.APIKey.IsNull() || config.WebSocket.APIKey.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing WebSocket API Key",
				"websocket.api_key is required when auth_method is 'websocket'.",
			)
			return
		}

		// Validate SSH block (needed for fallback)
		if config.SSH == nil {
			resp.Diagnostics.AddError(
				"Missing SSH Configuration",
				"SSH block is required for fallback operations when auth_method is 'websocket'.",
			)
			return
		}

		// Create SSH client for fallback
		sshConfig := &client.SSHConfig{
			Host:               config.Host.ValueString(),
			PrivateKey:         config.SSH.PrivateKey.ValueString(),
			HostKeyFingerprint: config.SSH.HostKeyFingerprint.ValueString(),
		}
		if !config.SSH.Port.IsNull() {
			sshConfig.Port = int(config.SSH.Port.ValueInt64())
		}
		if !config.SSH.User.IsNull() {
			sshConfig.User = config.SSH.User.ValueString()
		}
		if !config.SSH.MaxSessions.IsNull() {
			sshConfig.MaxSessions = int(config.SSH.MaxSessions.ValueInt64())
		}

		sshClient, err := factory.NewSSHClient(sshConfig)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create SSH Client",
				err.Error(),
			)
			return
		}

		// Connect SSH client to detect version
		if err := sshClient.Connect(ctx); err != nil {
			resp.Diagnostics.AddError(
				"Unable to Connect to TrueNAS",
				err.Error(),
			)
			return
		}

		// Validate version for WebSocket mode
		if !sshClient.Version().AtLeast(25, 0) {
			resp.Diagnostics.AddError(
				"WebSocket Transport Requires TrueNAS 25.0+",
				fmt.Sprintf("Detected version %s. Use auth_method = \"ssh\" instead.",
					sshClient.Version().Raw),
			)
			return
		}

		// Create WebSocket client
		wsConfig := client.WebSocketConfig{
			Host:     config.Host.ValueString(),
			Username: config.WebSocket.Username.ValueString(),
			APIKey:   config.WebSocket.APIKey.ValueString(),
			Fallback: sshClient,
		}
		if !config.WebSocket.Port.IsNull() {
			wsConfig.Port = int(config.WebSocket.Port.ValueInt64())
		}
		if !config.WebSocket.InsecureSkipVerify.IsNull() {
			wsConfig.InsecureSkipVerify = config.WebSocket.InsecureSkipVerify.ValueBool()
		}
		if !config.WebSocket.MaxConcurrent.IsNull() {
			wsConfig.MaxConcurrent = int(config.WebSocket.MaxConcurrent.ValueInt64())
		}
		if !config.WebSocket.ConnectTimeout.IsNull() {
			wsConfig.ConnectTimeout = time.Duration(config.WebSocket.ConnectTimeout.ValueInt64()) * time.Second
		}
		if !config.WebSocket.MaxRetries.IsNull() {
			wsConfig.MaxRetries = int(config.WebSocket.MaxRetries.ValueInt64())
		}

		wsClient, err := factory.NewWebSocketClient(wsConfig)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create WebSocket Client",
				err.Error(),
			)
			return
		}

		// Connect WebSocket client (caches version from fallback)
		if err := wsClient.Connect(ctx); err != nil {
			resp.Diagnostics.AddError(
				"Unable to Connect WebSocket Client",
				err.Error(),
			)
			return
		}

		finalClient = wsClient

	case "ssh", "":
		// Validate SSH block is provided
		if config.SSH == nil {
			resp.Diagnostics.AddError(
				"Missing SSH Configuration",
				"SSH block is required when auth_method is 'ssh'.",
			)
			return
		}

		// Build SSH config with values from provider configuration
		sshConfig := &client.SSHConfig{
			Host:               config.Host.ValueString(),
			PrivateKey:         config.SSH.PrivateKey.ValueString(),
			HostKeyFingerprint: config.SSH.HostKeyFingerprint.ValueString(),
		}

		// Set optional values if provided
		if !config.SSH.Port.IsNull() {
			sshConfig.Port = int(config.SSH.Port.ValueInt64())
		}
		if !config.SSH.User.IsNull() {
			sshConfig.User = config.SSH.User.ValueString()
		}
		if !config.SSH.MaxSessions.IsNull() {
			sshConfig.MaxSessions = int(config.SSH.MaxSessions.ValueInt64())
		}

		// Create SSH client (validates config and applies defaults)
		sshClient, err := factory.NewSSHClient(sshConfig)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create SSH Client",
				err.Error(),
			)
			return
		}

		// Connect SSH client to detect version
		if err := sshClient.Connect(ctx); err != nil {
			resp.Diagnostics.AddError(
				"Unable to Connect to TrueNAS",
				err.Error(),
			)
			return
		}

		// Apply rate limiting defaults
		rateLimit := 0 // 0 means use default (18)
		if !config.RateLimit.IsNull() {
			rateLimit = int(config.RateLimit.ValueInt64())
		}

		maxRetries := -1 // -1 means use default (3)
		if !config.MaxRetries.IsNull() {
			maxRetries = int(config.MaxRetries.ValueInt64())
		}

		// Wrap client with rate limiting and retry
		finalClient = client.NewRateLimitedClient(
			sshClient,
			rateLimit,
			maxRetries,
			&client.SSHRetryClassifier{},
		)

	default:
		resp.Diagnostics.AddError(
			"Invalid Authentication Method",
			fmt.Sprintf("auth_method must be 'ssh' or 'websocket', got '%s'.", config.AuthMethod.ValueString()),
		)
		return
	}

	// Set client for data sources and resources
	resp.DataSourceData = finalClient
	resp.ResourceData = finalClient
}

func (p *TrueNASProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewPoolDataSource,
		datasources.NewDatasetDataSource,
		datasources.NewSnapshotsDataSource,
		datasources.NewCloudSyncCredentialsDataSource,
		datasources.NewVirtConfigDataSource,
	}
}

func (p *TrueNASProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewDatasetResource,
		resources.NewHostPathResource,
		resources.NewAppResource,
		resources.NewFileResource,
		resources.NewSnapshotResource,
		resources.NewCloudSyncCredentialsResource,
		resources.NewCloudSyncTaskResource,
		resources.NewCronJobResource,
		resources.NewVirtConfigResource,
		resources.NewVirtInstanceResource,
		resources.NewAppRegistryResource,
		resources.NewVMResource,
		resources.NewZvolResource,
	}
}
