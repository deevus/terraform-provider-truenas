package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	customtypes "github.com/deevus/terraform-provider-truenas/internal/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"
)

var _ resource.Resource = &AppResource{}
var _ resource.ResourceWithConfigure = &AppResource{}
var _ resource.ResourceWithImportState = &AppResource{}
var _ resource.ResourceWithValidateConfig = &AppResource{}

// AppResource defines the resource implementation.
type AppResource struct {
	BaseResource
}

// AppResourceModel describes the resource data model.
type AppResourceModel struct {
	ID              types.String                           `tfsdk:"id"`
	Name            types.String                           `tfsdk:"name"`
	CustomApp       types.Bool                             `tfsdk:"custom_app"`
	ComposeConfig   customtypes.YAMLStringValue            `tfsdk:"compose_config"`
	CatalogApp      types.String                           `tfsdk:"catalog_app"`
	Train           types.String                           `tfsdk:"train"`
	Version         types.String                           `tfsdk:"version"`
	Values          types.String                           `tfsdk:"values"`
	DesiredState    customtypes.CaseInsensitiveStringValue `tfsdk:"desired_state"`
	StateTimeout    types.Int64                            `tfsdk:"state_timeout"`
	State           types.String                           `tfsdk:"state"`
	RestartTriggers types.Map                              `tfsdk:"restart_triggers"`
}

// appAPIResponse represents the JSON response from app API calls.
type appAPIResponse struct {
	Name      string            `json:"name"`
	State     string            `json:"state"`
	CustomApp bool              `json:"custom_app"`
	Version   string            `json:"version"`
	Config    appConfigResponse `json:"config"`
	Metadata  appMetadata       `json:"metadata"`
}

// appMetadata contains catalog metadata from the API response.
type appMetadata struct {
	Name  string `json:"name"`
	Train string `json:"train"`
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
				Description: "Whether this is a custom Docker Compose application. Computed from whether compose_config is set.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"compose_config": schema.StringAttribute{
				Description: "Docker Compose YAML configuration string. Mutually exclusive with catalog_app.",
				Optional:    true,
				CustomType:  customtypes.YAMLStringType{},
			},
			"catalog_app": schema.StringAttribute{
				Description: "Catalog application name (e.g., 'plex', 'tailscale'). Mutually exclusive with compose_config.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"train": schema.StringAttribute{
				Description: "Catalog train. Defaults to 'stable'.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.StringAttribute{
				Description: "Catalog app version. Defaults to 'latest' on creation, then tracks the installed version.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"values": schema.StringAttribute{
				Description: "JSON-encoded configuration values for catalog apps.",
				Optional:    true,
			},
			"desired_state": schema.StringAttribute{
				Description: "Desired application state: 'running' or 'stopped' (case-insensitive). Defaults to 'RUNNING'.",
				Optional:    true,
				Computed:    true,
				CustomType:  customtypes.CaseInsensitiveStringType{},
				Default:     customtypes.CaseInsensitiveStringDefault("RUNNING"),
				Validators: []validator.String{
					stringvalidator.Any(
						stringvalidator.OneOfCaseInsensitive("running", "stopped"),
					),
				},
			},
			"state_timeout": schema.Int64Attribute{
				Description: "Timeout in seconds to wait for state transitions. Defaults to 120. Range: 30-600.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(120),
				Validators: []validator.Int64{
					int64validator.Between(30, 600),
				},
			},
			"state": schema.StringAttribute{
				Description: "Application state (RUNNING, STOPPED, etc.).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					computedStatePlanModifier(),
				},
			},
			"restart_triggers": schema.MapAttribute{
				Description: "Map of values that, when changed, trigger an app restart. " +
					"Use this to restart the app when dependent resources change, e.g., " +
					"`restart_triggers = { config_checksum = truenas_file.config.checksum }`.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *AppResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data AppResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.CatalogApp.IsUnknown() || data.ComposeConfig.IsUnknown() || data.Values.IsUnknown() {
		return
	}

	hasCatalog := !data.CatalogApp.IsNull()
	hasCompose := !data.ComposeConfig.IsNull()

	if hasCatalog && hasCompose {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"catalog_app and compose_config are mutually exclusive. Use catalog_app for catalog apps or compose_config for custom Docker Compose apps.",
		)
		return
	}

	if !hasCatalog && !hasCompose {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Either catalog_app or compose_config must be set.",
		)
		return
	}

	hasValues := !data.Values.IsNull()
	if hasValues && hasCompose {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"values can only be used with catalog_app, not compose_config.",
		)
	}
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
	_, err := r.client.CallAndWait(ctx, "app.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create App",
			fmt.Sprintf("Unable to create app %q: %s", appName, err.Error()),
		)
		return
	}

	// Query the app to get current state and metadata
	app, err := r.queryApp(ctx, appName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Query App After Create",
			fmt.Sprintf("Unable to query app %q after create: %s", appName, err.Error()),
		)
		return
	}
	if app == nil {
		resp.Diagnostics.AddError(
			"App Not Found After Create",
			fmt.Sprintf("App %q was not found after create", appName),
		)
		return
	}

	// Map response to model
	data.ID = types.StringValue(app.Name)
	data.State = types.StringValue(app.State)
	data.CustomApp = types.BoolValue(app.CustomApp)

	// For catalog apps, populate computed fields from API
	if !app.CustomApp {
		if data.Train.IsNull() || data.Train.IsUnknown() {
			data.Train = types.StringValue(app.Metadata.Train)
		}
		data.Version = types.StringValue(app.Version)
	}

	// Handle desired_state - if user wants STOPPED but app started as RUNNING
	desiredState := data.DesiredState.ValueString()
	if desiredState == "" {
		desiredState = AppStateRunning
	}
	normalizedDesired := normalizeDesiredState(desiredState)

	if app.State != normalizedDesired {
		timeout := time.Duration(data.StateTimeout.ValueInt64()) * time.Second
		if timeout == 0 {
			timeout = 120 * time.Second
		}

		// For Create, we don't warn about drift - it's expected that we may need to stop
		var method string
		if normalizedDesired == AppStateRunning {
			method = "app.start"
		} else {
			method = "app.stop"
		}

		_, err := r.client.CallAndWait(ctx, method, appName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Set App State",
				fmt.Sprintf("Unable to %s app %q: %s", method, appName, err.Error()),
			)
			return
		}

		// Wait for stable state and query final state
		queryFunc := func(ctx context.Context, n string) (string, error) {
			return r.queryAppState(ctx, n)
		}

		finalState, err := waitForStableState(ctx, appName, timeout, queryFunc)
		if err != nil {
			resp.Diagnostics.AddError(
				"Timeout Waiting for App State",
				err.Error(),
			)
			return
		}

		data.State = types.StringValue(finalState)
	}

	// Preserve user's original desired_state value (semantic equality handles case differences)
	// Only set if it was empty (defaulting to RUNNING)
	if data.DesiredState.IsNull() || data.DesiredState.ValueString() == "" {
		data.DesiredState = customtypes.NewCaseInsensitiveStringValue(AppStateRunning)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve user-specified values from prior state (not returned by API)
	priorDesiredState := data.DesiredState
	priorStateTimeout := data.StateTimeout
	priorRestartTriggers := data.RestartTriggers
	priorValues := data.Values

	appName := data.Name.ValueString()

	// Query with retrieve_config for custom apps (compose config)
	filter := [][]any{{"name", "=", appName}}
	options := map[string]any{
		"extra": map[string]any{
			"retrieve_config": true,
		},
	}
	queryParams := []any{filter, options}

	result, err := r.client.Call(ctx, "app.query", queryParams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read App",
			fmt.Sprintf("Unable to read app %q: %s", appName, err.Error()),
		)
		return
	}

	var apps []appAPIResponse
	if err := json.Unmarshal(result, &apps); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse App Response",
			fmt.Sprintf("Unable to parse app response: %s", err.Error()),
		)
		return
	}

	if len(apps) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	app := apps[0]

	data.ID = types.StringValue(app.Name)
	data.State = types.StringValue(app.State)
	data.CustomApp = types.BoolValue(app.CustomApp)

	if app.CustomApp {
		// Custom app: sync compose_config from API
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
		data.CatalogApp = types.StringNull()
		data.Train = types.StringNull()
		data.Version = types.StringNull()
		data.Values = types.StringNull()
	} else {
		// Catalog app: sync metadata from API, preserve user values
		data.ComposeConfig = customtypes.NewYAMLStringNull()
		data.CatalogApp = types.StringValue(app.Metadata.Name)
		data.Train = types.StringValue(app.Metadata.Train)
		data.Version = types.StringValue(app.Version)
		data.Values = priorValues
	}

	// Restore user-specified values from prior state
	data.DesiredState = priorDesiredState
	data.StateTimeout = priorStateTimeout
	data.RestartTriggers = priorRestartTriggers

	if data.DesiredState.IsNull() || data.DesiredState.IsUnknown() {
		data.DesiredState = customtypes.NewCaseInsensitiveStringValue(app.State)
	}

	if data.StateTimeout.IsNull() || data.StateTimeout.IsUnknown() {
		data.StateTimeout = types.Int64Value(120)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppResourceModel
	var stateData AppResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read current state data to detect compose_config changes
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appName := data.Name.ValueString()

	// Detect config changes and update if needed
	configChanged := false
	if data.CustomApp.ValueBool() {
		// Custom app: check compose_config changes
		configChanged = !data.ComposeConfig.Equal(stateData.ComposeConfig)
	} else {
		// Catalog app: check values changes
		configChanged = !data.Values.Equal(stateData.Values)
	}

	if configChanged {
		updateParams := r.buildUpdateParams(ctx, &data)
		params := []any{appName, updateParams}

		_, err := r.client.CallAndWait(ctx, "app.update", params)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Update App",
				fmt.Sprintf("Unable to update app %q: %s", appName, err.Error()),
			)
			return
		}
	}

	// Check if restart_triggers changed - if so, we need to restart the app
	restartTriggersChanged := !data.RestartTriggers.Equal(stateData.RestartTriggers)
	needsRestart := restartTriggersChanged && !data.RestartTriggers.IsNull() && !stateData.RestartTriggers.IsNull()

	// Query the app to get current state
	currentState, err := r.queryAppState(ctx, appName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Query App State",
			fmt.Sprintf("Unable to query app %q state: %s", appName, err.Error()),
		)
		return
	}

	// Get timeout from plan
	timeout := time.Duration(data.StateTimeout.ValueInt64()) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	// Wait for transitional states to complete before reconciling
	if !isStableState(currentState) {
		queryFunc := func(ctx context.Context, n string) (string, error) {
			return r.queryAppState(ctx, n)
		}

		stableState, err := waitForStableState(ctx, appName, timeout, queryFunc)
		if err != nil {
			resp.Diagnostics.AddError(
				"Timeout Waiting for App State",
				err.Error(),
			)
			return
		}
		currentState = stableState
	}

	// Handle restart_triggers: if triggers changed and app is running, restart it
	if needsRestart && currentState == AppStateRunning {
		// Restart by stopping then starting the app
		_, err := r.client.CallAndWait(ctx, "app.stop", appName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Stop App for Restart",
				fmt.Sprintf("Unable to stop app %q for restart triggered by restart_triggers change: %s", appName, err.Error()),
			)
			return
		}

		_, err = r.client.CallAndWait(ctx, "app.start", appName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Start App for Restart",
				fmt.Sprintf("Unable to start app %q for restart triggered by restart_triggers change: %s", appName, err.Error()),
			)
			return
		}

		// Wait for stable state after restart
		queryFunc := func(ctx context.Context, n string) (string, error) {
			return r.queryAppState(ctx, n)
		}

		stableState, err := waitForStableState(ctx, appName, timeout, queryFunc)
		if err != nil {
			resp.Diagnostics.AddError(
				"Timeout Waiting for App State After Restart",
				err.Error(),
			)
			return
		}
		currentState = stableState
	}

	// Reconcile desired_state - this adds drift warnings if state was externally changed
	desiredState := data.DesiredState.ValueString()
	if desiredState == "" {
		desiredState = AppStateRunning
	}
	normalizedDesired := normalizeDesiredState(desiredState)

	if currentState != normalizedDesired {
		err := r.reconcileDesiredState(ctx, appName, currentState, normalizedDesired, timeout, resp)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Reconcile App State",
				err.Error(),
			)
			return
		}
		// Query final state after reconciliation
		currentState, err = r.queryAppState(ctx, appName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Query App State After Reconciliation",
				fmt.Sprintf("Unable to query app %q state: %s", appName, err.Error()),
			)
			return
		}
	}

	// Map final state to model
	data.ID = types.StringValue(appName)
	data.State = types.StringValue(currentState)
	// DesiredState is preserved from plan - don't overwrite user's value

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
	isCatalog := !data.CatalogApp.IsNull() && !data.CatalogApp.IsUnknown()

	params := client.AppCreateParams{
		AppName:   data.Name.ValueString(),
		CustomApp: !isCatalog,
	}

	if isCatalog {
		params.CatalogApp = data.CatalogApp.ValueString()
		if !data.Train.IsNull() && !data.Train.IsUnknown() {
			params.Train = data.Train.ValueString()
		}
		if !data.Version.IsNull() && !data.Version.IsUnknown() {
			params.Version = data.Version.ValueString()
		}
		if !data.Values.IsNull() && !data.Values.IsUnknown() {
			var values map[string]any
			if err := json.Unmarshal([]byte(data.Values.ValueString()), &values); err == nil {
				params.Values = values
			}
		}
	} else {
		if !data.ComposeConfig.IsNull() && !data.ComposeConfig.IsUnknown() {
			params.CustomComposeConfigString = data.ComposeConfig.ValueString()
		}
	}

	return params
}

// buildUpdateParams builds the update parameters from the model.
func (r *AppResource) buildUpdateParams(_ context.Context, data *AppResourceModel) map[string]any {
	params := map[string]any{}

	if !data.ComposeConfig.IsNull() && !data.ComposeConfig.IsUnknown() {
		params["custom_compose_config_string"] = data.ComposeConfig.ValueString()
	}

	if !data.Values.IsNull() && !data.Values.IsUnknown() {
		var values map[string]any
		if err := json.Unmarshal([]byte(data.Values.ValueString()), &values); err == nil {
			params["values"] = values
		}
	}

	return params
}

// queryApp queries the TrueNAS API for an app by name, returning nil if not found.
func (r *AppResource) queryApp(ctx context.Context, name string) (*appAPIResponse, error) {
	filter := [][]any{{"name", "=", name}}
	result, err := r.client.Call(ctx, "app.query", filter)
	if err != nil {
		return nil, err
	}

	var apps []appAPIResponse
	if err := json.Unmarshal(result, &apps); err != nil {
		return nil, err
	}

	if len(apps) == 0 {
		return nil, nil
	}

	return &apps[0], nil
}

// queryAppState queries the TrueNAS API for the current state of an app.
func (r *AppResource) queryAppState(ctx context.Context, name string) (string, error) {
	filter := [][]any{{"name", "=", name}}
	result, err := r.client.Call(ctx, "app.query", filter)
	if err != nil {
		return "", err
	}

	var apps []appAPIResponse
	if err := json.Unmarshal(result, &apps); err != nil {
		return "", err
	}

	if len(apps) == 0 {
		return "", fmt.Errorf("app %q not found", name)
	}

	return apps[0].State, nil
}

// reconcileDesiredState ensures the app is in the desired state.
// It calls app.start or app.stop as needed and waits for the state to stabilize.
// Returns an error if the reconciliation fails.
func (r *AppResource) reconcileDesiredState(
	ctx context.Context,
	name string,
	currentState string,
	desiredState string,
	timeout time.Duration,
	resp *resource.UpdateResponse,
) error {
	normalizedDesired := normalizeDesiredState(desiredState)

	// Check if reconciliation is needed
	if currentState == normalizedDesired {
		return nil
	}

	// CRASHED is "stopped enough" when desired is STOPPED - no action needed
	if normalizedDesired == AppStateStopped && currentState == AppStateCrashed {
		return nil
	}

	// Add warning about drift
	resp.Diagnostics.AddWarning(
		"App state was externally changed",
		fmt.Sprintf(
			"The app %q was found in state %s but desired_state is %s. "+
				"Reconciling to desired state. To stop this app intentionally, set desired_state = \"stopped\".",
			name, currentState, normalizedDesired,
		),
	)

	// Determine which action to take
	var method string
	if normalizedDesired == AppStateRunning {
		method = "app.start"
	} else {
		method = "app.stop"
	}

	// Call the API
	_, err := r.client.CallAndWait(ctx, method, name)
	if err != nil {
		return fmt.Errorf("failed to %s app %q: %w", method, name, err)
	}

	// Wait for stable state
	queryFunc := func(ctx context.Context, n string) (string, error) {
		return r.queryAppState(ctx, n)
	}

	finalState, err := waitForStableState(ctx, name, timeout, queryFunc)
	if err != nil {
		return err
	}

	// Verify we reached the desired state
	if finalState != normalizedDesired {
		return fmt.Errorf("app %q reached state %s instead of desired %s", name, finalState, normalizedDesired)
	}

	return nil
}

