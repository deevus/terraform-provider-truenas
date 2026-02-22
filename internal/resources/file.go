package resources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	truenas "github.com/deevus/truenas-go"
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
	BaseResource
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
	ForceDestroy types.Bool   `tfsdk:"force_destroy"`
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
			"force_destroy": schema.BoolAttribute{
				Description: "Change file ownership to root before deletion to handle permission issues from app containers.",
				Optional:    true,
			},
		},
	}
}


func (r *FileResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data FileResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Skip validation if any path-related values are unknown (e.g., referencing
	// another resource's output). Validation will occur at apply time when
	// values are known.
	if data.HostPath.IsUnknown() || data.RelativePath.IsUnknown() || data.Path.IsUnknown() {
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

	// Validate path is absolute and does not contain path traversal
	if hasPath {
		p := data.Path.ValueString()
		if !strings.HasPrefix(p, "/") {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"'path' must be an absolute path (start with '/').",
			)
			return
		}
		if strings.Contains(p, "..") {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"'path' must not contain '..' (path traversal not allowed).",
			)
			return
		}
	}
}

// computeChecksum calculates SHA256 checksum of content.
func computeChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// resolvePath resolves the full path from host_path + relative_path or standalone path.
func (r *FileResource) resolvePath(data *FileResourceModel) string {
	if !data.HostPath.IsNull() && !data.HostPath.IsUnknown() {
		return filepath.Join(data.HostPath.ValueString(), data.RelativePath.ValueString())
	}
	return data.Path.ValueString()
}

// parseMode converts mode string to fs.FileMode.
func parseMode(mode string) fs.FileMode {
	if mode == "" {
		return 0644
	}
	m, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return 0644
	}
	return fs.FileMode(m)
}

func (r *FileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fullPath := r.resolvePath(&data)
	content := data.Content.ValueString()
	mode := parseMode(data.Mode.ValueString())

	// If using host_path + relative_path, create parent directories
	if !data.HostPath.IsNull() && !data.HostPath.IsUnknown() {
		parentDir := filepath.Dir(fullPath)
		if err := r.client.MkdirAll(ctx, parentDir, 0755); err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create Parent Directory",
				fmt.Sprintf("Unable to create directory %q: %s", parentDir, err.Error()),
			)
			return
		}
	}

	// Build write file params
	params := truenas.DefaultWriteFileParams([]byte(content))
	params.Mode = mode
	if !data.UID.IsNull() && !data.UID.IsUnknown() {
		params.UID = truenas.IntPtr(int(data.UID.ValueInt64()))
	}
	if !data.GID.IsNull() && !data.GID.IsUnknown() {
		params.GID = truenas.IntPtr(int(data.GID.ValueInt64()))
	}

	// Write the file with ownership
	if err := r.client.WriteFile(ctx, fullPath, params); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create File",
			fmt.Sprintf("Unable to write file %q: %s", fullPath, err.Error()),
		)
		return
	}

	// Set computed values
	data.ID = types.StringValue(fullPath)
	data.Path = types.StringValue(fullPath)
	data.Checksum = types.StringValue(computeChecksum(content))

	// Set defaults for mode/uid/gid if not specified
	if data.Mode.IsNull() || data.Mode.IsUnknown() {
		data.Mode = types.StringValue("0644")
	}
	if data.UID.IsNull() || data.UID.IsUnknown() {
		data.UID = types.Int64Value(0)
	}
	if data.GID.IsNull() || data.GID.IsUnknown() {
		data.GID = types.Int64Value(0)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use path if set, otherwise fall back to ID (for import scenarios where
	// only ID is populated by ImportStatePassthroughID)
	fullPath := data.Path.ValueString()
	if fullPath == "" {
		fullPath = data.ID.ValueString()
	}

	// Check if file exists
	exists, err := r.client.FileExists(ctx, fullPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Check File",
			fmt.Sprintf("Unable to check if file %q exists: %s", fullPath, err.Error()),
		)
		return
	}

	if !exists {
		// File was deleted externally, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Read file content for checksum calculation
	content, err := r.client.ReadFile(ctx, fullPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read File",
			fmt.Sprintf("Unable to read file %q: %s", fullPath, err.Error()),
		)
		return
	}

	// Update computed values to reflect actual remote state
	// This also ensures path is set after import (where only ID is populated)
	data.Path = types.StringValue(fullPath)
	data.Checksum = types.StringValue(computeChecksum(string(content)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fullPath := r.resolvePath(&data)
	content := data.Content.ValueString()
	mode := parseMode(data.Mode.ValueString())

	// Build write file params
	params := truenas.DefaultWriteFileParams([]byte(content))
	params.Mode = mode
	if !data.UID.IsNull() && !data.UID.IsUnknown() {
		params.UID = truenas.IntPtr(int(data.UID.ValueInt64()))
	}
	if !data.GID.IsNull() && !data.GID.IsUnknown() {
		params.GID = truenas.IntPtr(int(data.GID.ValueInt64()))
	}

	// Write the updated file with ownership
	if err := r.client.WriteFile(ctx, fullPath, params); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update File",
			fmt.Sprintf("Unable to write file %q: %s", fullPath, err.Error()),
		)
		return
	}

	// Update computed values - explicitly set for consistency with Create
	data.ID = types.StringValue(fullPath)
	data.Path = types.StringValue(fullPath)
	data.Checksum = types.StringValue(computeChecksum(content))

	// Set defaults for mode/uid/gid if not specified (same as Create)
	if data.Mode.IsNull() || data.Mode.IsUnknown() {
		data.Mode = types.StringValue("0644")
	}
	if data.UID.IsNull() || data.UID.IsUnknown() {
		data.UID = types.Int64Value(0)
	}
	if data.GID.IsNull() || data.GID.IsUnknown() {
		data.GID = types.Int64Value(0)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fullPath := data.Path.ValueString()

	// If force_destroy is true, change ownership to root before deleting
	// This handles permission issues from app containers that may have modified the file
	if data.ForceDestroy.ValueBool() {
		// Best effort - continue even if chown fails (file might already be deletable)
		_ = r.client.Chown(ctx, fullPath, 0, 0)
	}

	if err := r.client.DeleteFile(ctx, fullPath); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete File",
			fmt.Sprintf("Unable to delete file %q: %s", fullPath, err.Error()),
		)
		return
	}
}

