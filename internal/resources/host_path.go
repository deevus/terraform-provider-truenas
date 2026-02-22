package resources

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	truenas "github.com/deevus/truenas-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &HostPathResource{}
var _ resource.ResourceWithConfigure = &HostPathResource{}
var _ resource.ResourceWithImportState = &HostPathResource{}

// HostPathResource defines the resource implementation.
type HostPathResource struct {
	BaseResource
}

// HostPathResourceModel describes the resource data model.
type HostPathResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Path         types.String `tfsdk:"path"`
	Mode         types.String `tfsdk:"mode"`
	UID          types.Int64  `tfsdk:"uid"`
	GID          types.Int64  `tfsdk:"gid"`
	ForceDestroy types.Bool   `tfsdk:"force_destroy"`
}

// NewHostPathResource creates a new HostPathResource.
func NewHostPathResource() resource.Resource {
	return &HostPathResource{}
}

func (r *HostPathResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_path"
}

func (r *HostPathResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:        "Manages a TrueNAS host path directory for app storage mounts.",
		DeprecationMessage: "Use truenas_dataset with nested datasets instead. host_path relies on SFTP which may not work with non-root SSH users. Datasets are created via the TrueNAS API and provide better ZFS integration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Host path identifier (the full path).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "Full path to the directory (e.g., '/mnt/tank/apps/myapp').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mode": schema.StringAttribute{
				Description: "Unix mode (e.g., '755').",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uid": schema.Int64Attribute{
				Description: "Owner user ID.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"gid": schema.Int64Attribute{
				Description: "Owner group ID.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"force_destroy": schema.BoolAttribute{
				Description: "Force deletion of non-empty directories (recursive delete).",
				Optional:    true,
			},
		},
	}
}


func (r *HostPathResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data HostPathResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pathStr := data.Path.ValueString()

	// Determine the mode for the directory
	mode := fs.FileMode(0755) // default
	if !data.Mode.IsNull() && !data.Mode.IsUnknown() {
		mode = parseMode(data.Mode.ValueString())
	}

	// Create the directory
	if err := r.services.Filesystem.Client().MkdirAll(ctx, pathStr, mode); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Host Path",
			fmt.Sprintf("Cannot create directory %q: %s", pathStr, err.Error()),
		)
		return
	}

	// Set permissions if uid/gid are specified (uses TrueNAS API)
	if r.hasUIDGID(&data) {
		permOpts := r.buildPermOpts(&data)
		if err := r.services.Filesystem.SetPermissions(ctx, permOpts); err != nil {
			resp.Diagnostics.AddError(
				"Unable to Set Permissions",
				fmt.Sprintf("Cannot set permissions on %q: %s", pathStr, err.Error()),
			)
			return
		}
	}

	// Set the ID to the path
	data.ID = types.StringValue(pathStr)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HostPathResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data HostPathResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := data.Path.ValueString()

	// Call filesystem.stat to verify the path exists
	stat, err := r.services.Filesystem.Stat(ctx, path)
	if err != nil {
		// Path doesn't exist or API error - remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Sync state from API response
	data.ID = types.StringValue(path)

	// Only update optional+computed attributes if they were previously set
	// This prevents drift when user didn't specify these values
	if !data.Mode.IsNull() {
		data.Mode = types.StringValue(fmt.Sprintf("%o", stat.Mode))
	}
	if !data.UID.IsNull() {
		data.UID = types.Int64Value(stat.UID)
	}
	if !data.GID.IsNull() {
		data.GID = types.Int64Value(stat.GID)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HostPathResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data HostPathResourceModel
	var state HostPathResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read current state
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if any permissions changed
	permChanged := !data.Mode.Equal(state.Mode) ||
		!data.UID.Equal(state.UID) ||
		!data.GID.Equal(state.GID)

	if permChanged {
		permOpts := r.buildPermOpts(&data)
		if err := r.services.Filesystem.SetPermissions(ctx, permOpts); err != nil {
			resp.Diagnostics.AddError(
				"Unable to Update Permissions",
				fmt.Sprintf("Cannot update permissions on %q: %s", data.Path.ValueString(), err.Error()),
			)
			return
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HostPathResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data HostPathResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p := data.Path.ValueString()

	// Delete the directory using SFTP
	var err error
	if data.ForceDestroy.ValueBool() {
		err = r.forceDestroyHostPath(ctx, p, resp)
	} else {
		// Only remove empty directory when force_destroy is false
		err = r.services.Filesystem.Client().RemoveDir(ctx, p)
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Host Path",
			fmt.Sprintf("Cannot delete directory %q: %s", p, err.Error()),
		)
		return
	}
}

// forceDestroyHostPath handles deletion with force_destroy=true, including
// permission manipulation on parent directory with guaranteed restore via defer.
func (r *HostPathResource) forceDestroyHostPath(ctx context.Context, p string, resp *resource.DeleteResponse) error {
	parentPath := filepath.Dir(p)

	// Capture parent directory permissions before modification
	var originalParentMode string
	var originalParentUID, originalParentGID int64
	var parentModified bool

	parentStat, parentStatErr := r.services.Filesystem.Stat(ctx, parentPath)
	if parentStatErr == nil && parentStat != nil {
		originalParentMode = fmt.Sprintf("%o", parentStat.Mode)
		originalParentUID = parentStat.UID
		originalParentGID = parentStat.GID
	}

	// Best effort - fix permissions on target directory using TrueNAS API
	// This handles permission issues from apps that may have restricted access
	// Uses filesystem.setperm with stripacl to remove ACLs, set ownership to root,
	// and set permissive mode recursively - all in one API call
	targetOpts := truenas.SetPermOpts{
		Path:      p,
		UID:       truenas.Int64Ptr(0),
		GID:       truenas.Int64Ptr(0),
		Mode:      "777",
		StripACL:  true,
		Recursive: true,
		Traverse:  true,
	}
	permErr := r.services.Filesystem.SetPermissions(ctx, targetOpts)
	if permErr != nil {
		resp.Diagnostics.AddWarning(
			"Failed to Set Permissions Before Delete",
			fmt.Sprintf("filesystem.setperm failed for %q: %s. Will attempt deletion anyway.", p, permErr.Error()),
		)
	}

	// Set permissive permissions on parent directory to allow deletion
	// Need write permission on parent to remove an entry from it
	parentOpts := truenas.SetPermOpts{
		Path: parentPath,
		UID:  truenas.Int64Ptr(0),
		GID:  truenas.Int64Ptr(0),
		Mode: "777",
	}
	parentPermErr := r.services.Filesystem.SetPermissions(ctx, parentOpts)
	if parentPermErr != nil {
		resp.Diagnostics.AddWarning(
			"Failed to Set Parent Permissions Before Delete",
			fmt.Sprintf("filesystem.setperm failed for parent %q: %s. Will attempt deletion anyway.", parentPath, parentPermErr.Error()),
		)
	} else {
		parentModified = true
	}

	// Defer restoration of parent permissions - runs regardless of deletion success/failure
	defer func() {
		if !parentModified || parentStatErr != nil {
			return
		}
		restoreOpts := truenas.SetPermOpts{
			Path: parentPath,
			UID:  &originalParentUID,
			GID:  &originalParentGID,
			Mode: originalParentMode,
		}
		restoreErr := r.services.Filesystem.SetPermissions(ctx, restoreOpts)
		if restoreErr != nil {
			resp.Diagnostics.AddWarning(
				"Failed to Restore Parent Permissions",
				fmt.Sprintf("Could not restore original permissions on %q: %s", parentPath, restoreErr.Error()),
			)
		}
	}()

	// Recursive delete when force_destroy is true
	return r.services.Filesystem.Client().RemoveAll(ctx, p)
}

func (r *HostPathResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the path - set it to both id and path attributes
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), req.ID)...)
}

// hasUIDGID returns true if uid or gid are set (mode handled separately in mkdir).
func (r *HostPathResource) hasUIDGID(data *HostPathResourceModel) bool {
	return (!data.UID.IsNull() && !data.UID.IsUnknown()) ||
		(!data.GID.IsNull() && !data.GID.IsUnknown())
}

// buildPermOpts builds the options for filesystem.setperm.
func (r *HostPathResource) buildPermOpts(data *HostPathResourceModel) truenas.SetPermOpts {
	opts := truenas.SetPermOpts{
		Path: data.Path.ValueString(),
	}

	if !data.Mode.IsNull() && !data.Mode.IsUnknown() {
		opts.Mode = data.Mode.ValueString()
	}

	if !data.UID.IsNull() && !data.UID.IsUnknown() {
		uid := data.UID.ValueInt64()
		opts.UID = &uid
	}

	if !data.GID.IsNull() && !data.GID.IsUnknown() {
		gid := data.GID.ValueInt64()
		opts.GID = &gid
	}

	return opts
}
