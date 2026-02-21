package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	truenas "github.com/deevus/truenas-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &CronJobResource{}
	_ resource.ResourceWithConfigure   = &CronJobResource{}
	_ resource.ResourceWithImportState = &CronJobResource{}
)

// CronJobResourceModel describes the resource data model.
type CronJobResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	User          types.String   `tfsdk:"user"`
	Command       types.String   `tfsdk:"command"`
	Description   types.String   `tfsdk:"description"`
	Enabled       types.Bool     `tfsdk:"enabled"`
	CaptureStdout types.Bool     `tfsdk:"capture_stdout"`
	CaptureStderr types.Bool     `tfsdk:"capture_stderr"`
	Schedule      *ScheduleBlock `tfsdk:"schedule"`
}

// CronJobResource defines the resource implementation.
type CronJobResource struct {
	BaseResource
}

// NewCronJobResource creates a new CronJobResource.
func NewCronJobResource() resource.Resource {
	return &CronJobResource{}
}

func (r *CronJobResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cron_job"
}

func (r *CronJobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages cron jobs for scheduled task execution.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Cron job ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user": schema.StringAttribute{
				Description: "User to run the command as.",
				Required:    true,
			},
			"command": schema.StringAttribute{
				Description: "Command to execute.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Job description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"enabled": schema.BoolAttribute{
				Description: "Enable the cron job.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"capture_stdout": schema.BoolAttribute{
				Description: "Capture standard output and mail to user account.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"capture_stderr": schema.BoolAttribute{
				Description: "Capture error output and mail to user account.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"schedule": schema.SingleNestedBlock{
				Description: "Cron schedule for the job.",
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
		},
	}
}


func (r *CronJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CronJobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build params
	params := buildCronJobParams(&data)

	// Call API
	result, err := r.client.Call(ctx, "cronjob.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Cron Job",
			fmt.Sprintf("Unable to create cron job: %s", err.Error()),
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
	job, err := r.queryCronJob(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Cron Job",
			fmt.Sprintf("Cron job created but unable to read: %s", err.Error()),
		)
		return
	}

	if job == nil {
		resp.Diagnostics.AddError(
			"Cron Job Not Found",
			"Cron job was created but could not be found.",
		)
		return
	}

	// Set state from response
	mapCronJobToModel(job, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CronJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CronJobResourceModel

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

	job, err := r.queryCronJob(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Cron Job",
			fmt.Sprintf("Unable to query cron job: %s", err.Error()),
		)
		return
	}

	if job == nil {
		// Cron job was deleted outside Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	mapCronJobToModel(job, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CronJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state CronJobResourceModel
	var plan CronJobResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
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

	// Build update params
	params := buildCronJobParams(&plan)

	// Call API with []any{id, params}
	_, err = r.client.Call(ctx, "cronjob.update", []any{id, params})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Cron Job",
			fmt.Sprintf("Unable to update cron job: %s", err.Error()),
		)
		return
	}

	// Query to get full state
	job, err := r.queryCronJob(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Cron Job",
			fmt.Sprintf("Cron job updated but unable to read: %s", err.Error()),
		)
		return
	}

	if job == nil {
		resp.Diagnostics.AddError(
			"Cron Job Not Found",
			"Cron job was updated but could not be found.",
		)
		return
	}

	// Set state from response
	mapCronJobToModel(job, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CronJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CronJobResourceModel

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

	_, err = r.client.Call(ctx, "cronjob.delete", id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Cron Job",
			fmt.Sprintf("Unable to delete cron job: %s", err.Error()),
		)
		return
	}
}


// queryCronJob queries a cron job by ID and returns the response.
func (r *CronJobResource) queryCronJob(ctx context.Context, id int64) (*truenas.CronJobResponse, error) {
	filter := [][]any{{"id", "=", id}}
	result, err := r.client.Call(ctx, "cronjob.query", filter)
	if err != nil {
		return nil, err
	}

	var jobs []truenas.CronJobResponse
	if err := json.Unmarshal(result, &jobs); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(jobs) == 0 {
		return nil, nil
	}

	return &jobs[0], nil
}

// buildCronJobParams builds the API params from the resource model.
func buildCronJobParams(data *CronJobResourceModel) map[string]any {
	params := map[string]any{
		"user":        data.User.ValueString(),
		"command":     data.Command.ValueString(),
		"description": data.Description.ValueString(),
		"enabled":     data.Enabled.ValueBool(),
		"stdout":      !data.CaptureStdout.ValueBool(),
		"stderr":      !data.CaptureStderr.ValueBool(),
	}

	if data.Schedule != nil {
		params["schedule"] = map[string]any{
			"minute": data.Schedule.Minute.ValueString(),
			"hour":   data.Schedule.Hour.ValueString(),
			"dom":    data.Schedule.Dom.ValueString(),
			"month":  data.Schedule.Month.ValueString(),
			"dow":    data.Schedule.Dow.ValueString(),
		}
	}

	return params
}

// mapCronJobToModel maps an API response to the resource model.
func mapCronJobToModel(job *truenas.CronJobResponse, data *CronJobResourceModel) {
	data.ID = types.StringValue(strconv.FormatInt(job.ID, 10))
	data.User = types.StringValue(job.User)
	data.Command = types.StringValue(job.Command)
	data.Description = types.StringValue(job.Description)
	data.Enabled = types.BoolValue(job.Enabled)
	data.CaptureStdout = types.BoolValue(!job.Stdout)
	data.CaptureStderr = types.BoolValue(!job.Stderr)

	if data.Schedule != nil {
		data.Schedule.Minute = types.StringValue(job.Schedule.Minute)
		data.Schedule.Hour = types.StringValue(job.Schedule.Hour)
		data.Schedule.Dom = types.StringValue(job.Schedule.Dom)
		data.Schedule.Month = types.StringValue(job.Schedule.Month)
		data.Schedule.Dow = types.StringValue(job.Schedule.Dow)
	}
}
