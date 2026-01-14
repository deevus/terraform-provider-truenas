package provider

import (
	"context"

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
	Host       types.String   `tfsdk:"host"`
	AuthMethod types.String   `tfsdk:"auth_method"`
	SSH        *SSHBlockModel `tfsdk:"ssh"`
}

// SSHBlockModel describes the SSH configuration block.
type SSHBlockModel struct {
	Port               types.Int64  `tfsdk:"port"`
	User               types.String `tfsdk:"user"`
	PrivateKey         types.String `tfsdk:"private_key"`
	HostKeyFingerprint types.String `tfsdk:"host_key_fingerprint"`
	MaxSessions        types.Int64  `tfsdk:"max_sessions"`
}

type TrueNASProvider struct {
	version string
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
				Description: "Authentication method. Currently only 'ssh' is supported.",
				Required:    true,
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

	// Validate auth_method is "ssh"
	if config.AuthMethod.ValueString() != "ssh" {
		resp.Diagnostics.AddError(
			"Invalid Authentication Method",
			"Only 'ssh' authentication method is currently supported.",
		)
		return
	}

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
	sshClient, err := client.NewSSHClient(sshConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create SSH Client",
			err.Error(),
		)
		return
	}

	// Set client for data sources and resources
	resp.DataSourceData = sshClient
	resp.ResourceData = sshClient
}

func (p *TrueNASProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewPoolDataSource,
		datasources.NewDatasetDataSource,
	}
}

func (p *TrueNASProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewDatasetResource,
		resources.NewHostPathResource,
		resources.NewAppResource,
		resources.NewFileResource,
	}
}
