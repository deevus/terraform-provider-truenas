package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/deevus/terraform-provider-truenas/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &CloudSyncTaskResource{}
	_ resource.ResourceWithConfigure   = &CloudSyncTaskResource{}
	_ resource.ResourceWithImportState = &CloudSyncTaskResource{}
)

// CloudSyncTaskResourceModel describes the resource data model.
type CloudSyncTaskResourceModel struct {
	ID                 types.String     `tfsdk:"id"`
	Description        types.String     `tfsdk:"description"`
	Path               types.String     `tfsdk:"path"`
	Credentials        types.Int64      `tfsdk:"credentials"`
	Direction          types.String     `tfsdk:"direction"`
	TransferMode       types.String     `tfsdk:"transfer_mode"`
	Snapshot           types.Bool       `tfsdk:"snapshot"`
	Transfers          types.Int64      `tfsdk:"transfers"`
	BWLimit            types.String     `tfsdk:"bwlimit"`
	Exclude            types.List       `tfsdk:"exclude"`
	Include            types.List       `tfsdk:"include"`
	FollowSymlinks     types.Bool       `tfsdk:"follow_symlinks"`
	CreateEmptySrcDirs types.Bool       `tfsdk:"create_empty_src_dirs"`
	FastList           types.Bool       `tfsdk:"fast_list"`
	Enabled            types.Bool       `tfsdk:"enabled"`
	SyncOnChange       types.Bool       `tfsdk:"sync_on_change"`
	Schedule           *ScheduleBlock   `tfsdk:"schedule"`
	Encryption         *EncryptionBlock `tfsdk:"encryption"`
	S3                 *TaskS3Block     `tfsdk:"s3"`
	B2                 *TaskB2Block     `tfsdk:"b2"`
	GCS                *TaskGCSBlock    `tfsdk:"gcs"`
	Azure              *TaskAzureBlock  `tfsdk:"azure"`
}

// ScheduleBlock represents cron schedule settings.
type ScheduleBlock struct {
	Minute types.String `tfsdk:"minute"`
	Hour   types.String `tfsdk:"hour"`
	Dom    types.String `tfsdk:"dom"`
	Month  types.String `tfsdk:"month"`
	Dow    types.String `tfsdk:"dow"`
}

// EncryptionBlock represents encryption settings for cloud storage.
type EncryptionBlock struct {
	Password types.String `tfsdk:"password"`
	Salt     types.String `tfsdk:"salt"`
}

// TaskS3Block represents S3-compatible storage settings.
type TaskS3Block struct {
	Bucket types.String `tfsdk:"bucket"`
	Folder types.String `tfsdk:"folder"`
}

// TaskB2Block represents Backblaze B2 storage settings.
type TaskB2Block struct {
	Bucket types.String `tfsdk:"bucket"`
	Folder types.String `tfsdk:"folder"`
}

// TaskGCSBlock represents Google Cloud Storage settings.
type TaskGCSBlock struct {
	Bucket types.String `tfsdk:"bucket"`
	Folder types.String `tfsdk:"folder"`
}

// TaskAzureBlock represents Azure Blob Storage settings.
type TaskAzureBlock struct {
	Container types.String `tfsdk:"container"`
	Folder    types.String `tfsdk:"folder"`
}

// CloudSyncTaskResource defines the resource implementation.
type CloudSyncTaskResource struct {
	BaseResource
}

// NewCloudSyncTaskResource creates a new CloudSyncTaskResource.
func NewCloudSyncTaskResource() resource.Resource {
	return &CloudSyncTaskResource{}
}

func (r *CloudSyncTaskResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloudsync_task"
}

func (r *CloudSyncTaskResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages cloud sync backup tasks.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Task ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Task description.",
				Required:    true,
			},
			"path": schema.StringAttribute{
				Description: "Local path to sync.",
				Required:    true,
			},
			"credentials": schema.Int64Attribute{
				Description: "Cloud sync credentials ID.",
				Required:    true,
			},
			"direction": schema.StringAttribute{
				Description: "Sync direction: push, pull, or sync.",
				Required:    true,
			},
			"transfer_mode": schema.StringAttribute{
				Description: "Transfer mode: sync, copy, or move.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("sync"),
			},
			"snapshot": schema.BoolAttribute{
				Description: "Take a snapshot before sync.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"transfers": schema.Int64Attribute{
				Description: "Number of simultaneous file transfers.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(4),
			},
			"bwlimit": schema.StringAttribute{
				Description: "Bandwidth limit in KB/s or schedule.",
				Optional:    true,
			},
			"exclude": schema.ListAttribute{
				Description: "Patterns to exclude from sync.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"include": schema.ListAttribute{
				Description: "Patterns to include in sync. Supports glob patterns like '/folder/**' or '*.jpg'.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"follow_symlinks": schema.BoolAttribute{
				Description: "Follow symbolic links.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"create_empty_src_dirs": schema.BoolAttribute{
				Description: "Create empty source directories on destination.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"fast_list": schema.BoolAttribute{
				Description: "Use fewer transactions in exchange for more memory. See rclone --fast-list.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"enabled": schema.BoolAttribute{
				Description: "Enable the task.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"sync_on_change": schema.BoolAttribute{
				Description: "Fire-and-forget sync after create or update.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"schedule": schema.SingleNestedBlock{
				Description: "Cron schedule for the task.",
				Attributes: map[string]schema.Attribute{
					"minute": schema.StringAttribute{
						Description: "Minute (0-59 or cron expression).",
						Required:    true,
					},
					"hour": schema.StringAttribute{
						Description: "Hour (0-23 or cron expression).",
						Required:    true,
					},
					"dom": schema.StringAttribute{
						Description: "Day of month (1-31 or cron expression).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("*"),
					},
					"month": schema.StringAttribute{
						Description: "Month (1-12 or cron expression).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("*"),
					},
					"dow": schema.StringAttribute{
						Description: "Day of week (0-6 or cron expression).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("*"),
					},
				},
			},
			"encryption": schema.SingleNestedBlock{
				Description: "Encryption settings for cloud storage.",
				Attributes: map[string]schema.Attribute{
					"password": schema.StringAttribute{
						Description: "Encryption password.",
						Optional:    true,
						Sensitive:   true,
					},
					"salt": schema.StringAttribute{
						Description: "Encryption salt.",
						Optional:    true,
						Computed:    true,
						Sensitive:   true,
					},
				},
			},
			"s3": schema.SingleNestedBlock{
				Description: "S3-compatible storage settings.",
				Attributes: map[string]schema.Attribute{
					"bucket": schema.StringAttribute{
						Description: "Bucket name.",
						Optional:    true,
					},
					"folder": schema.StringAttribute{
						Description: "Folder path within the bucket.",
						Optional:    true,
					},
				},
			},
			"b2": schema.SingleNestedBlock{
				Description: "Backblaze B2 storage settings.",
				Attributes: map[string]schema.Attribute{
					"bucket": schema.StringAttribute{
						Description: "Bucket name.",
						Optional:    true,
					},
					"folder": schema.StringAttribute{
						Description: "Folder path within the bucket.",
						Optional:    true,
					},
				},
			},
			"gcs": schema.SingleNestedBlock{
				Description: "Google Cloud Storage settings.",
				Attributes: map[string]schema.Attribute{
					"bucket": schema.StringAttribute{
						Description: "Bucket name.",
						Optional:    true,
					},
					"folder": schema.StringAttribute{
						Description: "Folder path within the bucket.",
						Optional:    true,
					},
				},
			},
			"azure": schema.SingleNestedBlock{
				Description: "Azure Blob Storage settings.",
				Attributes: map[string]schema.Attribute{
					"container": schema.StringAttribute{
						Description: "Container name.",
						Optional:    true,
					},
					"folder": schema.StringAttribute{
						Description: "Folder path within the container.",
						Optional:    true,
					},
				},
			},
		},
	}
}


func (r *CloudSyncTaskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CloudSyncTaskResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that exactly one provider block is specified
	count := 0
	if data.S3 != nil {
		count++
	}
	if data.B2 != nil {
		count++
	}
	if data.GCS != nil {
		count++
	}
	if data.Azure != nil {
		count++
	}
	if count != 1 {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Exactly one provider block (s3, b2, gcs, or azure) must be specified.",
		)
		return
	}

	// Validate required fields within provider blocks
	if validationErrors := validateTaskProviderBlock(&data); len(validationErrors) > 0 {
		for _, err := range validationErrors {
			resp.Diagnostics.AddError("Invalid Configuration", err)
		}
		return
	}

	// Build params
	params := buildCloudSyncTaskParams(ctx, &data)

	// Call API
	result, err := r.client.Call(ctx, "cloudsync.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Cloud Sync Task",
			fmt.Sprintf("Unable to create task: %s", err.Error()),
		)
		return
	}

	// Parse response to get ID
	var createResp struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(result, &createResp); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Response",
			fmt.Sprintf("Unable to parse create response: %s", err.Error()),
		)
		return
	}

	// Query to get full state
	task, err := r.queryTask(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Task",
			fmt.Sprintf("Task created but unable to read: %s", err.Error()),
		)
		return
	}

	if task == nil {
		resp.Diagnostics.AddError(
			"Task Not Found",
			"Task was created but could not be found.",
		)
		return
	}

	// Set state from response
	r.mapTaskToModel(task, &data)

	// Trigger sync if sync_on_change is true
	if data.SyncOnChange.ValueBool() {
		_, err := r.client.Call(ctx, "cloudsync.sync", createResp.ID)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Sync Trigger Failed",
				fmt.Sprintf("Task created but sync failed to trigger: %s", err.Error()),
			)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// queryTask queries a cloud sync task by ID and returns the response.
func (r *CloudSyncTaskResource) queryTask(ctx context.Context, id int64) (*api.CloudSyncTaskResponse, error) {
	filter := [][]any{{"id", "=", id}}
	result, err := r.client.Call(ctx, "cloudsync.query", filter)
	if err != nil {
		return nil, err
	}

	var tasks []api.CloudSyncTaskResponse
	if err := json.Unmarshal(result, &tasks); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(tasks) == 0 {
		return nil, nil
	}

	return &tasks[0], nil
}

// buildCloudSyncTaskParams builds the API params from the resource model.
func buildCloudSyncTaskParams(ctx context.Context, data *CloudSyncTaskResourceModel) map[string]any {
	params := map[string]any{
		"description":   data.Description.ValueString(),
		"path":          data.Path.ValueString(),
		"credentials":   data.Credentials.ValueInt64(),
		"direction":     strings.ToUpper(data.Direction.ValueString()),
		"transfer_mode": strings.ToUpper(data.TransferMode.ValueString()),
		"snapshot":      data.Snapshot.ValueBool(),
		"transfers":     data.Transfers.ValueInt64(),
		"enabled":       data.Enabled.ValueBool(),
	}

	// Add optional fields
	if !data.BWLimit.IsNull() && !data.BWLimit.IsUnknown() {
		params["bwlimit"] = data.BWLimit.ValueString()
	}

	params["follow_symlinks"] = data.FollowSymlinks.ValueBool()
	params["create_empty_src_dirs"] = data.CreateEmptySrcDirs.ValueBool()

	// Handle exclude list
	if !data.Exclude.IsNull() && !data.Exclude.IsUnknown() {
		var excludeItems []string
		data.Exclude.ElementsAs(ctx, &excludeItems, false)
		params["exclude"] = excludeItems
	}

	// Handle include list
	if !data.Include.IsNull() && !data.Include.IsUnknown() {
		var includeItems []string
		data.Include.ElementsAs(ctx, &includeItems, false)
		params["include"] = includeItems
	}

	// Build schedule
	if data.Schedule != nil {
		params["schedule"] = map[string]any{
			"minute": data.Schedule.Minute.ValueString(),
			"hour":   data.Schedule.Hour.ValueString(),
			"dom":    data.Schedule.Dom.ValueString(),
			"month":  data.Schedule.Month.ValueString(),
			"dow":    data.Schedule.Dow.ValueString(),
		}
	}

	// Build attributes from provider block
	attributes := getTaskAttributes(data)
	params["attributes"] = attributes

	// Handle encryption
	if data.Encryption != nil {
		params["encryption"] = true
		params["encryption_password"] = data.Encryption.Password.ValueString()
		if !data.Encryption.Salt.IsNull() && !data.Encryption.Salt.IsUnknown() {
			params["encryption_salt"] = data.Encryption.Salt.ValueString()
		}
	} else {
		params["encryption"] = false
	}

	return params
}

// validateTaskProviderBlock validates that required fields are present in the specified provider block.
func validateTaskProviderBlock(data *CloudSyncTaskResourceModel) []string {
	var errors []string

	if data.S3 != nil {
		if data.S3.Bucket.IsNull() || data.S3.Bucket.ValueString() == "" {
			errors = append(errors, "s3.bucket is required when s3 block is specified")
		}
	}
	if data.B2 != nil {
		if data.B2.Bucket.IsNull() || data.B2.Bucket.ValueString() == "" {
			errors = append(errors, "b2.bucket is required when b2 block is specified")
		}
	}
	if data.GCS != nil {
		if data.GCS.Bucket.IsNull() || data.GCS.Bucket.ValueString() == "" {
			errors = append(errors, "gcs.bucket is required when gcs block is specified")
		}
	}
	if data.Azure != nil {
		if data.Azure.Container.IsNull() || data.Azure.Container.ValueString() == "" {
			errors = append(errors, "azure.container is required when azure block is specified")
		}
	}
	if data.Encryption != nil {
		if data.Encryption.Password.IsNull() || data.Encryption.Password.ValueString() == "" {
			errors = append(errors, "encryption.password is required when encryption block is specified")
		}
	}

	return errors
}

// getTaskAttributes extracts attributes from the provider block.
func getTaskAttributes(data *CloudSyncTaskResourceModel) map[string]any {
	var attrs map[string]any

	if data.S3 != nil {
		attrs = map[string]any{
			"bucket": data.S3.Bucket.ValueString(),
		}
		if !data.S3.Folder.IsNull() && !data.S3.Folder.IsUnknown() {
			attrs["folder"] = data.S3.Folder.ValueString()
		}
	} else if data.B2 != nil {
		attrs = map[string]any{
			"bucket": data.B2.Bucket.ValueString(),
		}
		if !data.B2.Folder.IsNull() && !data.B2.Folder.IsUnknown() {
			attrs["folder"] = data.B2.Folder.ValueString()
		}
	} else if data.GCS != nil {
		attrs = map[string]any{
			"bucket": data.GCS.Bucket.ValueString(),
		}
		if !data.GCS.Folder.IsNull() && !data.GCS.Folder.IsUnknown() {
			attrs["folder"] = data.GCS.Folder.ValueString()
		}
	} else if data.Azure != nil {
		attrs = map[string]any{
			"container": data.Azure.Container.ValueString(),
		}
		if !data.Azure.Folder.IsNull() && !data.Azure.Folder.IsUnknown() {
			attrs["folder"] = data.Azure.Folder.ValueString()
		}
	} else {
		attrs = map[string]any{}
	}

	attrs["fast_list"] = data.FastList.ValueBool()

	return attrs
}

// mapTaskToModel maps an API response to the resource model.
func (r *CloudSyncTaskResource) mapTaskToModel(task *api.CloudSyncTaskResponse, data *CloudSyncTaskResourceModel) {
	data.ID = types.StringValue(strconv.FormatInt(task.ID, 10))
	data.Description = types.StringValue(task.Description)
	data.Path = types.StringValue(task.Path)
	data.Credentials = types.Int64Value(task.Credentials.ID)
	data.Direction = types.StringValue(strings.ToLower(task.Direction))
	data.TransferMode = types.StringValue(strings.ToLower(task.TransferMode))
	data.Snapshot = types.BoolValue(task.Snapshot)
	data.Transfers = types.Int64Value(task.Transfers)
	data.FollowSymlinks = types.BoolValue(task.FollowSymlinks)
	data.CreateEmptySrcDirs = types.BoolValue(task.CreateEmptySrcDirs)
	data.FastList = types.BoolValue(task.FastList())
	data.Enabled = types.BoolValue(task.Enabled)

	// BWLimit is preserved from plan since API returns array format
	// and we accept string format in Terraform

	// Map schedule
	if data.Schedule != nil {
		data.Schedule.Minute = types.StringValue(task.Schedule.Minute)
		data.Schedule.Hour = types.StringValue(task.Schedule.Hour)
		data.Schedule.Dom = types.StringValue(task.Schedule.Dom)
		data.Schedule.Month = types.StringValue(task.Schedule.Month)
		data.Schedule.Dow = types.StringValue(task.Schedule.Dow)
	}

	// Note: encryption, provider blocks, and exclude are preserved from plan
	// since API may not return complete information
}

func (r *CloudSyncTaskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CloudSyncTaskResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	task, err := r.queryTask(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Task",
			fmt.Sprintf("Unable to query task: %s", err.Error()),
		)
		return
	}

	if task == nil {
		// Task was deleted outside Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	r.mapTaskToModel(task, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CloudSyncTaskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state CloudSyncTaskResourceModel
	var plan CloudSyncTaskResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate required fields within provider blocks
	if validationErrors := validateTaskProviderBlock(&plan); len(validationErrors) > 0 {
		for _, err := range validationErrors {
			resp.Diagnostics.AddError("Invalid Configuration", err)
		}
		return
	}

	// Parse ID from state
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	// Build update params (same as create params)
	params := buildCloudSyncTaskParams(ctx, &plan)

	// Call API with []any{id, params}
	_, err = r.client.Call(ctx, "cloudsync.update", []any{id, params})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Cloud Sync Task",
			fmt.Sprintf("Unable to update task: %s", err.Error()),
		)
		return
	}

	// Query to get full state
	task, err := r.queryTask(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Task",
			fmt.Sprintf("Task updated but unable to read: %s", err.Error()),
		)
		return
	}

	if task == nil {
		resp.Diagnostics.AddError(
			"Task Not Found",
			"Task was updated but could not be found.",
		)
		return
	}

	// Set state from response
	r.mapTaskToModel(task, &plan)

	// Trigger sync if sync_on_change is true
	if plan.SyncOnChange.ValueBool() {
		_, err := r.client.Call(ctx, "cloudsync.sync", id)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Sync Trigger Failed",
				fmt.Sprintf("Task updated but sync failed to trigger: %s", err.Error()),
			)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CloudSyncTaskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CloudSyncTaskResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	_, err = r.client.Call(ctx, "cloudsync.delete", id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Task",
			fmt.Sprintf("Unable to delete task: %s", err.Error()),
		)
		return
	}
}

