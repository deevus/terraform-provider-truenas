package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &FileResource{}
var _ resource.ResourceWithConfigure = &FileResource{}
var _ resource.ResourceWithImportState = &FileResource{}
var _ resource.ResourceWithValidateConfig = &FileResource{}

// FileResource defines the resource implementation.
type FileResource struct {
	client client.Client
}

// FileResourceModel describes the resource data model.
type FileResourceModel struct {
	ID           types.String `tfsdk:"id"`
	HostPath     types.String `tfsdk:"host_path"`
	RelativePath types.String `tfsdk:"relative_path"`
	Path         types.String `tfsdk:"path"`
	Content      types.String `tfsdk:"content"`
	Mode         types.String `tfsdk:"mode"`
	UID          types.Int64  `tfsdk:"uid"`
	GID          types.Int64  `tfsdk:"gid"`
	Checksum     types.String `tfsdk:"checksum"`
}

// NewFileResource creates a new FileResource.
func NewFileResource() resource.Resource {
	return &FileResource{}
}

func (r *FileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

func (r *FileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a file on TrueNAS for configuration deployment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "File identifier (the full path).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host_path": schema.StringAttribute{
				Description: "ID of a truenas_host_path resource. Mutually exclusive with 'path'.",
				Optional:    true,
			},
			"relative_path": schema.StringAttribute{
				Description: "Path relative to host_path. Can include subdirectories (e.g., 'config/app.conf').",
				Optional:    true,
			},
			"path": schema.StringAttribute{
				Description: "Absolute path to the file. Mutually exclusive with 'host_path'/'relative_path'.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"content": schema.StringAttribute{
				Description: "Content of the file. Use templatefile() or file() to load from disk.",
				Required:    true,
				Sensitive:   true,
			},
			"mode": schema.StringAttribute{
				Description: "Unix mode (e.g., '0644'). Inherits from host_path if not specified.",
				Optional:    true,
				Computed:    true,
			},
			"uid": schema.Int64Attribute{
				Description: "Owner user ID. Inherits from host_path if not specified.",
				Optional:    true,
				Computed:    true,
			},
			"gid": schema.Int64Attribute{
				Description: "Owner group ID. Inherits from host_path if not specified.",
				Optional:    true,
				Computed:    true,
			},
			"checksum": schema.StringAttribute{
				Description: "SHA256 checksum of the file content.",
				Computed:    true,
			},
		},
	}
}

func (r *FileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *FileResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data FileResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasHostPath := !data.HostPath.IsNull() && !data.HostPath.IsUnknown()
	hasRelativePath := !data.RelativePath.IsNull() && !data.RelativePath.IsUnknown()
	hasPath := !data.Path.IsNull() && !data.Path.IsUnknown()

	// Must provide either host_path+relative_path OR path
	if hasHostPath && hasPath {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Cannot specify both 'host_path' and 'path'. Use one or the other.",
		)
		return
	}

	if !hasHostPath && !hasPath {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Must specify either 'host_path' with 'relative_path', or 'path'.",
		)
		return
	}

	// If host_path is specified, relative_path is required
	if hasHostPath && !hasRelativePath {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"'relative_path' is required when 'host_path' is specified.",
		)
		return
	}

	// If relative_path is specified, host_path is required
	if hasRelativePath && !hasHostPath {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"'host_path' is required when 'relative_path' is specified.",
		)
		return
	}

	// Validate relative_path format
	if hasRelativePath {
		relativePath := data.RelativePath.ValueString()

		if strings.HasPrefix(relativePath, "/") {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"'relative_path' must not start with '/'. It should be relative to host_path.",
			)
			return
		}

		if strings.Contains(relativePath, "..") {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"'relative_path' must not contain '..' (path traversal not allowed).",
			)
			return
		}
	}

	// Validate path is absolute
	if hasPath {
		p := data.Path.ValueString()
		if !strings.HasPrefix(p, "/") {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"'path' must be an absolute path (start with '/').",
			)
			return
		}
	}
}

func (r *FileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Will be implemented in Phase 3
}

func (r *FileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Will be implemented in Phase 3
}

func (r *FileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Will be implemented in Phase 3
}

func (r *FileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Will be implemented in Phase 3
}

func (r *FileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
