package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/deevus/terraform-provider-truenas/internal/api"
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

	// Build params
	params := buildAppRegistryParams(&data)

	// Call API
	result, err := r.client.Call(ctx, "app.registry.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create App Registry",
			fmt.Sprintf("Unable to create app registry: %s", err.Error()),
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
	registry, err := r.queryAppRegistry(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read App Registry",
			fmt.Sprintf("App registry created but unable to read: %s", err.Error()),
		)
		return
	}

	if registry == nil {
		resp.Diagnostics.AddError(
			"App Registry Not Found",
			"App registry was created but could not be found.",
		)
		return
	}

	// Set state from response
	mapAppRegistryToModel(registry, &data)

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

	registry, err := r.queryAppRegistry(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read App Registry",
			fmt.Sprintf("Unable to query app registry: %s", err.Error()),
		)
		return
	}

	if registry == nil {
		// App registry was deleted outside Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	mapAppRegistryToModel(registry, &data)

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

	// Build update params
	params := buildAppRegistryParams(&plan)

	// Call API with []any{id, params}
	_, err = r.client.Call(ctx, "app.registry.update", []any{id, params})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update App Registry",
			fmt.Sprintf("Unable to update app registry: %s", err.Error()),
		)
		return
	}

	// Query to get full state
	registry, err := r.queryAppRegistry(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read App Registry",
			fmt.Sprintf("App registry updated but unable to read: %s", err.Error()),
		)
		return
	}

	if registry == nil {
		resp.Diagnostics.AddError(
			"App Registry Not Found",
			"App registry was updated but could not be found.",
		)
		return
	}

	// Set state from response
	mapAppRegistryToModel(registry, &plan)

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

	_, err = r.client.Call(ctx, "app.registry.delete", id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete App Registry",
			fmt.Sprintf("Unable to delete app registry: %s", err.Error()),
		)
		return
	}
}


// queryAppRegistry queries an app registry by ID and returns the response.
func (r *AppRegistryResource) queryAppRegistry(ctx context.Context, id int64) (*api.AppRegistryResponse, error) {
	filter := [][]any{{"id", "=", id}}
	result, err := r.client.Call(ctx, "app.registry.query", filter)
	if err != nil {
		return nil, err
	}

	var registries []api.AppRegistryResponse
	if err := json.Unmarshal(result, &registries); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(registries) == 0 {
		return nil, nil
	}

	return &registries[0], nil
}

// buildAppRegistryParams builds the API params from the resource model.
func buildAppRegistryParams(data *AppRegistryResourceModel) map[string]any {
	params := map[string]any{
		"name":     data.Name.ValueString(),
		"username": data.Username.ValueString(),
		"password": data.Password.ValueString(),
		"uri":      data.URI.ValueString(),
	}

	// Handle description - API expects null for empty, not empty string
	if !data.Description.IsNull() && data.Description.ValueString() != "" {
		params["description"] = data.Description.ValueString()
	} else {
		params["description"] = nil
	}

	return params
}

// mapAppRegistryToModel maps an API response to the resource model.
func mapAppRegistryToModel(registry *api.AppRegistryResponse, data *AppRegistryResourceModel) {
	data.ID = types.StringValue(strconv.FormatInt(registry.ID, 10))
	data.Name = types.StringValue(registry.Name)
	data.Username = types.StringValue(registry.Username)
	data.Password = types.StringValue(registry.Password)
	data.URI = types.StringValue(registry.URI)

	// Handle nullable description
	if registry.Description != nil {
		data.Description = types.StringValue(*registry.Description)
	} else {
		data.Description = types.StringValue("")
	}
}
