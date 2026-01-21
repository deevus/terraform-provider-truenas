package resources

import (
	"context"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	ID          types.String   `tfsdk:"id"`
	User        types.String   `tfsdk:"user"`
	Command     types.String   `tfsdk:"command"`
	Description types.String   `tfsdk:"description"`
	Enabled     types.Bool     `tfsdk:"enabled"`
	Stdout      types.Bool     `tfsdk:"stdout"`
	Stderr      types.Bool     `tfsdk:"stderr"`
	Schedule    *ScheduleBlock `tfsdk:"schedule"`
}

// CronJobResource defines the resource implementation.
type CronJobResource struct {
	client client.Client
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
			"stdout": schema.BoolAttribute{
				Description: "Redirect stdout to syslog.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"stderr": schema.BoolAttribute{
				Description: "Redirect stderr to syslog.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
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

func (r *CronJobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *CronJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// TODO: implement in next task
}

func (r *CronJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// TODO: implement in next task
}

func (r *CronJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO: implement in next task
}

func (r *CronJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: implement in next task
}

func (r *CronJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
