package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	customtypes "github.com/deevus/terraform-provider-truenas/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"
)

var _ resource.Resource = &AppResource{}
var _ resource.ResourceWithConfigure = &AppResource{}
var _ resource.ResourceWithImportState = &AppResource{}

// AppResource defines the resource implementation.
type AppResource struct {
	client client.Client
}

// AppResourceModel describes the resource data model.
// Simplified for custom Docker Compose apps only.
type AppResourceModel struct {
	ID            types.String              `tfsdk:"id"`
	Name          types.String              `tfsdk:"name"`
	CustomApp     types.Bool                `tfsdk:"custom_app"`
	ComposeConfig customtypes.YAMLStringValue `tfsdk:"compose_config"`
	State         types.String              `tfsdk:"state"`
}

// appAPIResponse represents the JSON response from app API calls.
// Simplified to only parse fields needed for custom Docker Compose apps.
type appAPIResponse struct {
	Name      string            `json:"name"`
	State     string            `json:"state"`
	CustomApp bool              `json:"custom_app"`
	Config    appConfigResponse `json:"config"`
	// active_workloads is ignored - it's runtime info, not configuration
}

// appConfigResponse contains config fields from the API.
// When retrieve_config is true, the API returns the parsed compose config as a map.
type appConfigResponse map[string]any

// NewAppResource creates a new AppResource.
func NewAppResource() resource.Resource {
	return &AppResource{}
}

func (r *AppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (r *AppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS application (custom Docker Compose app or catalog app).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Application identifier (the app name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Application name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"custom_app": schema.BoolAttribute{
				Description: "Whether this is a custom Docker Compose application.",
				Required:    true,
			},
			"compose_config": schema.StringAttribute{
				Description: "Docker Compose YAML configuration string (required for custom apps).",
				Optional:    true,
				CustomType:  customtypes.YAMLStringType{},
			},
			"state": schema.StringAttribute{
				Description: "Application state (RUNNING, STOPPED, etc.).",
				Computed:    true,
			},
		},
	}
}

func (r *AppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
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

func (r *AppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build create params
	params := r.buildCreateParams(ctx, &data)
	appName := data.Name.ValueString()

	// Call the TrueNAS API (app.create returns a job, use CallAndWait)
	// Ignore the response as it contains unparseable progress output mixed with JSON
	_, err := r.client.CallAndWait(ctx, "app.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create App",
			fmt.Sprintf("Unable to create app %q: %s", appName, err.Error()),
		)
		return
	}

	// Query the app to get current state
	filter := [][]any{{"name", "=", appName}}
	result, err := r.client.Call(ctx, "app.query", filter)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Query App After Create",
			fmt.Sprintf("Unable to query app %q after create: %s", appName, err.Error()),
		)
		return
	}

	var apps []appAPIResponse
	if err := json.Unmarshal(result, &apps); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse App Response",
			fmt.Sprintf("Unable to parse app query response: %s", err.Error()),
		)
		return
	}

	if len(apps) == 0 {
		resp.Diagnostics.AddError(
			"App Not Found After Create",
			fmt.Sprintf("App %q was not found after create", appName),
		)
		return
	}

	// Map response to model
	app := apps[0]
	data.ID = types.StringValue(app.Name)
	data.State = types.StringValue(app.State)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use the name to query the app
	appName := data.Name.ValueString()

	// Build query params: [filter, options]
	// Filter: [["name", "=", "appName"]]
	// Options: {"extra": {"retrieve_config": true}} to get compose config
	filter := [][]any{{"name", "=", appName}}
	options := map[string]any{
		"extra": map[string]any{
			"retrieve_config": true,
		},
	}
	queryParams := []any{filter, options}

	// Call the TrueNAS API
	result, err := r.client.Call(ctx, "app.query", queryParams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read App",
			fmt.Sprintf("Unable to read app %q: %s", appName, err.Error()),
		)
		return
	}

	// Parse the response
	var apps []appAPIResponse
	if err := json.Unmarshal(result, &apps); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse App Response",
			fmt.Sprintf("Unable to parse app response: %s", err.Error()),
		)
		return
	}

	// Check if app was found
	if len(apps) == 0 {
		// App was deleted outside of Terraform - remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	app := apps[0]

	// Map response to model - sync all fields from API
	data.ID = types.StringValue(app.Name)
	data.State = types.StringValue(app.State)
	data.CustomApp = types.BoolValue(app.CustomApp)

	// Sync compose_config if present
	// The API returns the parsed compose config as a map, convert back to YAML
	if len(app.Config) > 0 {
		yamlBytes, err := yaml.Marshal(app.Config)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Marshal Config",
				fmt.Sprintf("Unable to marshal app config to YAML: %s", err.Error()),
			)
			return
		}
		data.ComposeConfig = customtypes.NewYAMLStringValue(string(yamlBytes))
	} else {
		data.ComposeConfig = customtypes.NewYAMLStringNull()
	}

	// TODO: Sync storage, network, and labels from active_workloads
	// The API returns a different structure than expected, skipping for now

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build update params
	updateParams := r.buildUpdateParams(ctx, &data)

	// Call the TrueNAS API with positional args [name, params_object]
	appName := data.Name.ValueString()
	params := []any{appName, updateParams}

	// Call app.update and wait for completion - ignore the response as it contains
	// unparseable progress output mixed with JSON
	_, err := r.client.CallAndWait(ctx, "app.update", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update App",
			fmt.Sprintf("Unable to update app %q: %s", appName, err.Error()),
		)
		return
	}

	// Query the app to get current state
	filter := [][]any{{"name", "=", appName}}
	result, err := r.client.Call(ctx, "app.query", filter)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Query App After Update",
			fmt.Sprintf("Unable to query app %q after update: %s", appName, err.Error()),
		)
		return
	}

	var apps []appAPIResponse
	if err := json.Unmarshal(result, &apps); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse App Response",
			fmt.Sprintf("Unable to parse app query response: %s", err.Error()),
		)
		return
	}

	if len(apps) == 0 {
		resp.Diagnostics.AddError(
			"App Not Found After Update",
			fmt.Sprintf("App %q was not found after update", appName),
		)
		return
	}

	// Map response to model
	app := apps[0]
	data.ID = types.StringValue(app.Name)
	data.State = types.StringValue(app.State)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call the TrueNAS API
	appName := data.Name.ValueString()
	_, err := r.client.CallAndWait(ctx, "app.delete", appName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete App",
			fmt.Sprintf("Unable to delete app %q: %s", appName, err.Error()),
		)
		return
	}
}

func (r *AppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the app name - set it to both id and name attributes
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

// buildCreateParams builds the AppCreateParams from the model.
func (r *AppResource) buildCreateParams(_ context.Context, data *AppResourceModel) client.AppCreateParams {
	params := client.AppCreateParams{
		AppName:   data.Name.ValueString(),
		CustomApp: data.CustomApp.ValueBool(),
	}

	// Add compose config if set
	if !data.ComposeConfig.IsNull() && !data.ComposeConfig.IsUnknown() {
		params.CustomComposeConfigString = data.ComposeConfig.ValueString()
	}

	return params
}

// buildUpdateParams builds the update parameters from the model.
func (r *AppResource) buildUpdateParams(_ context.Context, data *AppResourceModel) map[string]any {
	params := map[string]any{}

	// Add compose config if set
	if !data.ComposeConfig.IsNull() && !data.ComposeConfig.IsUnknown() {
		params["custom_compose_config_string"] = data.ComposeConfig.ValueString()
	}

	return params
}

