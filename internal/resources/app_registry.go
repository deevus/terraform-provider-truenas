package resources

import (
	"context"
	"fmt"
	"strconv"

	truenas "github.com/deevus/truenas-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &AppRegistryResource{}
	_ resource.ResourceWithConfigure   = &AppRegistryResource{}
	_ resource.ResourceWithImportState = &AppRegistryResource{}
)

// AppRegistryResourceModel describes the resource data model.
type AppRegistryResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
	URI         types.String `tfsdk:"uri"`
}

// AppRegistryResource defines the resource implementation.
type AppRegistryResource struct {
	BaseResource
}

// NewAppRegistryResource creates a new AppRegistryResource.
func NewAppRegistryResource() resource.Resource {
	return &AppRegistryResource{}
}

func (r *AppRegistryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_registry"
}

func (r *AppRegistryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Docker registry credentials for pulling images from private container registries.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Registry ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Registry name (identifier).",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"username": schema.StringAttribute{
				Description: "Registry username.",
				Required:    true,
			},
			"password": schema.StringAttribute{
				Description: "Registry password or token.",
				Required:    true,
				Sensitive:   true,
			},
			"uri": schema.StringAttribute{
				Description: "Registry URL.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("https://index.docker.io/v1/"),
			},
		},
	}
}


func (r *AppRegistryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppRegistryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := buildRegistryOpts(&data)

	reg, err := r.services.App.CreateRegistry(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create App Registry",
			fmt.Sprintf("Unable to create app registry: %s", err.Error()),
		)
		return
	}

	mapAppRegistryToModel(reg, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppRegistryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppRegistryResourceModel

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

	reg, err := r.services.App.GetRegistry(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read App Registry",
			fmt.Sprintf("Unable to query app registry: %s", err.Error()),
		)
		return
	}

	if reg == nil {
		// App registry was deleted outside Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	mapAppRegistryToModel(reg, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppRegistryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state AppRegistryResourceModel
	var plan AppRegistryResourceModel

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

	opts := buildRegistryOpts(&plan)

	reg, err := r.services.App.UpdateRegistry(ctx, id, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update App Registry",
			fmt.Sprintf("Unable to update app registry: %s", err.Error()),
		)
		return
	}

	mapAppRegistryToModel(reg, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppRegistryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppRegistryResourceModel

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

	err = r.services.App.DeleteRegistry(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete App Registry",
			fmt.Sprintf("Unable to delete app registry: %s", err.Error()),
		)
		return
	}
}

// buildRegistryOpts builds CreateRegistryOpts from the resource model.
func buildRegistryOpts(data *AppRegistryResourceModel) truenas.CreateRegistryOpts {
	return truenas.CreateRegistryOpts{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Username:    data.Username.ValueString(),
		Password:    data.Password.ValueString(),
		URI:         data.URI.ValueString(),
	}
}

// mapAppRegistryToModel maps a Registry to the resource model.
func mapAppRegistryToModel(registry *truenas.Registry, data *AppRegistryResourceModel) {
	data.ID = types.StringValue(strconv.FormatInt(registry.ID, 10))
	data.Name = types.StringValue(registry.Name)
	data.Description = types.StringValue(registry.Description)
	data.Username = types.StringValue(registry.Username)
	data.Password = types.StringValue(registry.Password)
	data.URI = types.StringValue(registry.URI)
}
