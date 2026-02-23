package resources

import (
	"context"
	"fmt"
	"time"

	truenas "github.com/deevus/truenas-go"
	customtypes "github.com/deevus/terraform-provider-truenas/internal/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

// AppResource defines the resource implementation.
type AppResource struct {
	BaseResource
}

// AppResourceModel describes the resource data model.
// Simplified for custom Docker Compose apps only.
type AppResourceModel struct {
	ID              types.String                            `tfsdk:"id"`
	Name            types.String                            `tfsdk:"name"`
	CustomApp       types.Bool                              `tfsdk:"custom_app"`
	ComposeConfig   customtypes.YAMLStringValue             `tfsdk:"compose_config"`
	DesiredState    customtypes.CaseInsensitiveStringValue  `tfsdk:"desired_state"`
	StateTimeout    types.Int64                             `tfsdk:"state_timeout"`
	State           types.String                            `tfsdk:"state"`
	RestartTriggers types.Map                               `tfsdk:"restart_triggers"`
}

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


func (r *AppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build create opts
	opts := r.buildCreateOpts(ctx, &data)
	appName := data.Name.ValueString()

	// Call the TrueNAS API (CreateApp handles CallAndWait + GetApp internally)
	app, err := r.services.App.CreateApp(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create App",
			fmt.Sprintf("Unable to create app %q: %s", appName, err.Error()),
		)
		return
	}

	// Map response to model
	data.ID = types.StringValue(app.Name)
	data.State = types.StringValue(app.State)

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
		if normalizedDesired == AppStateRunning {
			err = r.services.App.StartApp(ctx, appName)
		} else {
			err = r.services.App.StopApp(ctx, appName)
		}
		if err != nil {
			action := "start"
			if normalizedDesired != AppStateRunning {
				action = "stop"
			}
			resp.Diagnostics.AddError(
				"Unable to Set App State",
				fmt.Sprintf("Unable to %s app %q: %s", action, appName, err.Error()),
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

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve user-specified values from prior state (these are not returned by API)
	priorDesiredState := data.DesiredState
	priorStateTimeout := data.StateTimeout
	priorRestartTriggers := data.RestartTriggers

	// Use the name to query the app
	appName := data.Name.ValueString()

	// Call the TrueNAS API with config retrieval
	app, err := r.services.App.GetAppWithConfig(ctx, appName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read App",
			fmt.Sprintf("Unable to read app %q: %s", appName, err.Error()),
		)
		return
	}

	// Check if app was found
	if app == nil {
		// App was deleted outside of Terraform - remove from state
		resp.State.RemoveResource(ctx)
		return
	}

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

	// Restore user-specified values from prior state
	data.DesiredState = priorDesiredState
	data.StateTimeout = priorStateTimeout
	data.RestartTriggers = priorRestartTriggers

	// Default desired_state if null/unknown (e.g., after import)
	if data.DesiredState.IsNull() || data.DesiredState.IsUnknown() {
		data.DesiredState = customtypes.NewCaseInsensitiveStringValue(app.State)
	}

	// Default state_timeout if null/unknown
	if data.StateTimeout.IsNull() || data.StateTimeout.IsUnknown() {
		data.StateTimeout = types.Int64Value(120)
	}

	// Save updated data into Terraform state
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

	// Handle compose_config changes first (if any)
	// Check if compose_config changed by comparing old and new values
	composeConfigChanged := !data.ComposeConfig.Equal(stateData.ComposeConfig)
	if composeConfigChanged {
		updateOpts := r.buildUpdateOpts(ctx, &data)

		// Call app.update and wait for completion
		_, err := r.services.App.UpdateApp(ctx, appName, updateOpts)
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
		err := r.services.App.StopApp(ctx, appName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Stop App for Restart",
				fmt.Sprintf("Unable to stop app %q for restart triggered by restart_triggers change: %s", appName, err.Error()),
			)
			return
		}

		err = r.services.App.StartApp(ctx, appName)
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
	err := r.services.App.DeleteApp(ctx, appName)
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

// buildCreateOpts builds the CreateAppOpts from the model.
func (r *AppResource) buildCreateOpts(_ context.Context, data *AppResourceModel) truenas.CreateAppOpts {
	opts := truenas.CreateAppOpts{
		Name:      data.Name.ValueString(),
		CustomApp: data.CustomApp.ValueBool(),
	}

	// Add compose config if set
	if !data.ComposeConfig.IsNull() && !data.ComposeConfig.IsUnknown() {
		opts.CustomComposeConfig = data.ComposeConfig.ValueString()
	}

	return opts
}

// buildUpdateOpts builds the UpdateAppOpts from the model.
func (r *AppResource) buildUpdateOpts(_ context.Context, data *AppResourceModel) truenas.UpdateAppOpts {
	opts := truenas.UpdateAppOpts{}

	// Add compose config if set
	if !data.ComposeConfig.IsNull() && !data.ComposeConfig.IsUnknown() {
		opts.CustomComposeConfig = data.ComposeConfig.ValueString()
	}

	return opts
}

// queryAppState queries the TrueNAS API for the current state of an app.
func (r *AppResource) queryAppState(ctx context.Context, name string) (string, error) {
	app, err := r.services.App.GetApp(ctx, name)
	if err != nil {
		return "", err
	}

	if app == nil {
		return "", fmt.Errorf("app %q not found", name)
	}

	return app.State, nil
}

// reconcileDesiredState ensures the app is in the desired state.
// It calls StartApp or StopApp as needed and waits for the state to stabilize.
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

	// Determine which action to take and call the API
	if normalizedDesired == AppStateRunning {
		if err := r.services.App.StartApp(ctx, name); err != nil {
			return fmt.Errorf("failed to start app %q: %w", name, err)
		}
	} else {
		if err := r.services.App.StopApp(ctx, name); err != nil {
			return fmt.Errorf("failed to stop app %q: %w", name, err)
		}
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
