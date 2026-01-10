package resources

import (
	"context"
	"encoding/json"
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

var _ resource.Resource = &AppResource{}
var _ resource.ResourceWithConfigure = &AppResource{}
var _ resource.ResourceWithImportState = &AppResource{}

// AppResource defines the resource implementation.
type AppResource struct {
	client client.Client
}

// AppResourceModel describes the resource data model.
type AppResourceModel struct {
	ID            types.String        `tfsdk:"id"`
	Name          types.String        `tfsdk:"name"`
	CustomApp     types.Bool          `tfsdk:"custom_app"`
	ComposeConfig types.String        `tfsdk:"compose_config"`
	Labels        types.List          `tfsdk:"labels"`
	State         types.String        `tfsdk:"state"`
	Storage       []StorageBlockModel `tfsdk:"storage"`
	Network       []NetworkBlockModel `tfsdk:"network"`
}

// StorageBlockModel describes a storage volume mount.
type StorageBlockModel struct {
	VolumeName      types.String `tfsdk:"volume_name"`
	Type            types.String `tfsdk:"type"`
	HostPath        types.String `tfsdk:"host_path"`
	ACLEnable       types.Bool   `tfsdk:"acl_enable"`
	AutoPermissions types.Bool   `tfsdk:"auto_permissions"`
}

// NetworkBlockModel describes a network port mapping.
type NetworkBlockModel struct {
	PortName   types.String `tfsdk:"port_name"`
	BindMode   types.String `tfsdk:"bind_mode"`
	HostIPs    types.List   `tfsdk:"host_ips"`
	PortNumber types.Int64  `tfsdk:"port_number"`
}

// appAPIResponse represents the JSON response from app API calls.
type appAPIResponse struct {
	Name      string                    `json:"name"`
	State     string                    `json:"state"`
	CustomApp bool                      `json:"custom_app"`
	Config    appConfigResponse         `json:"config"`
	Values    map[string]appValueEntry  `json:"active_workloads"`
}

// appConfigResponse contains config fields from the API.
type appConfigResponse struct {
	CustomComposeConfigString string `json:"custom_compose_config_string"`
}

// appValueEntry represents an entry in the values map.
type appValueEntry struct {
	Storage map[string]storageConfigResponse `json:"storage"`
	Network map[string]networkConfigResponse `json:"network"`
	Labels  []string                         `json:"labels"`
}

// storageConfigResponse represents storage configuration from the API.
type storageConfigResponse struct {
	Type           string                   `json:"type"`
	HostPathConfig hostPathConfigResponse   `json:"host_path_config"`
}

// hostPathConfigResponse represents host path configuration from the API.
type hostPathConfigResponse struct {
	Path            string `json:"path"`
	ACLEnable       bool   `json:"acl_enable"`
	AutoPermissions bool   `json:"auto_permissions"`
}

// networkConfigResponse represents network configuration from the API.
type networkConfigResponse struct {
	BindMode   string   `json:"bind_mode"`
	HostIPs    []string `json:"host_ips"`
	PortNumber int64    `json:"port_number"`
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
				Description: "Docker Compose YAML configuration string (for custom apps).",
				Optional:    true,
			},
			"labels": schema.ListAttribute{
				Description: "Container labels.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"state": schema.StringAttribute{
				Description: "Application state (RUNNING, STOPPED, etc.).",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"storage": schema.ListNestedBlock{
				Description: "Storage volume mounts.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"volume_name": schema.StringAttribute{
							Description: "Name/key for the storage volume.",
							Required:    true,
						},
						"type": schema.StringAttribute{
							Description: "Storage type (e.g., 'host_path').",
							Required:    true,
						},
						"host_path": schema.StringAttribute{
							Description: "Path on the host filesystem.",
							Required:    true,
						},
						"acl_enable": schema.BoolAttribute{
							Description: "Enable ACL support.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"auto_permissions": schema.BoolAttribute{
							Description: "Automatically set permissions.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
					},
				},
			},
			"network": schema.ListNestedBlock{
				Description: "Network port mappings.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"port_name": schema.StringAttribute{
							Description: "Name/key for the port mapping.",
							Required:    true,
						},
						"bind_mode": schema.StringAttribute{
							Description: "Port bind mode (e.g., 'published').",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("published"),
						},
						"host_ips": schema.ListAttribute{
							Description: "Host IPs to bind to.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"port_number": schema.Int64Attribute{
							Description: "Port number.",
							Required:    true,
						},
					},
				},
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

	// Call the TrueNAS API (app.create returns a job, use CallAndWait)
	result, err := r.client.CallAndWait(ctx, "app.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create App",
			fmt.Sprintf("Unable to create app %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Parse the response
	var appResp appAPIResponse
	if err := json.Unmarshal(result, &appResp); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse App Response",
			fmt.Sprintf("Unable to parse app create response: %s", err.Error()),
		)
		return
	}

	// Map response to model
	data.ID = types.StringValue(appResp.Name)
	data.State = types.StringValue(appResp.State)

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

	// Build filter params: [["name", "=", "appName"]]
	filter := [][]any{{"name", "=", appName}}

	// Call the TrueNAS API
	result, err := r.client.Call(ctx, "app.query", filter)
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
	if app.Config.CustomComposeConfigString != "" {
		data.ComposeConfig = types.StringValue(app.Config.CustomComposeConfigString)
	} else {
		data.ComposeConfig = types.StringNull()
	}

	// Sync storage, network, and labels from active_workloads
	// The API returns these in a map structure, we need to extract them
	r.syncValuesFromAPI(ctx, &data, app.Values, resp)
	if resp.Diagnostics.HasError() {
		return
	}

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

	result, err := r.client.CallAndWait(ctx, "app.update", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update App",
			fmt.Sprintf("Unable to update app %q: %s", appName, err.Error()),
		)
		return
	}

	// Parse the response
	var appResp appAPIResponse
	if err := json.Unmarshal(result, &appResp); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse App Response",
			fmt.Sprintf("Unable to parse app update response: %s", err.Error()),
		)
		return
	}

	// Map response to model
	data.ID = types.StringValue(appResp.Name)
	data.State = types.StringValue(appResp.State)

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
func (r *AppResource) buildCreateParams(ctx context.Context, data *AppResourceModel) client.AppCreateParams {
	params := client.AppCreateParams{
		AppName:   data.Name.ValueString(),
		CustomApp: data.CustomApp.ValueBool(),
	}

	// Add compose config if set
	if !data.ComposeConfig.IsNull() && !data.ComposeConfig.IsUnknown() {
		params.CustomComposeConfigString = data.ComposeConfig.ValueString()
	}

	// Build values
	params.Values = client.AppValues{}

	// Add storage
	if len(data.Storage) > 0 {
		params.Values.Storage = make(map[string]client.StorageConfig)
		for _, s := range data.Storage {
			params.Values.Storage[s.VolumeName.ValueString()] = client.StorageConfig{
				Type: s.Type.ValueString(),
				HostPathConfig: client.HostPathConfig{
					Path:            s.HostPath.ValueString(),
					ACLEnable:       s.ACLEnable.ValueBool(),
					AutoPermissions: s.AutoPermissions.ValueBool(),
				},
			}
		}
	}

	// Add network
	if len(data.Network) > 0 {
		params.Values.Network = make(map[string]client.NetworkConfig)
		for _, n := range data.Network {
			hostIPs := []string{}
			if !n.HostIPs.IsNull() && !n.HostIPs.IsUnknown() {
				n.HostIPs.ElementsAs(ctx, &hostIPs, false)
			}

			params.Values.Network[n.PortName.ValueString()] = client.NetworkConfig{
				BindMode:   n.BindMode.ValueString(),
				HostIPs:    hostIPs,
				PortNumber: int(n.PortNumber.ValueInt64()),
			}
		}
	}

	// Add labels
	if !data.Labels.IsNull() && !data.Labels.IsUnknown() {
		var labels []string
		data.Labels.ElementsAs(ctx, &labels, false)
		params.Values.Labels = labels
	}

	return params
}

// buildUpdateParams builds the update parameters from the model.
func (r *AppResource) buildUpdateParams(ctx context.Context, data *AppResourceModel) map[string]any {
	params := map[string]any{}

	// Add compose config if set
	if !data.ComposeConfig.IsNull() && !data.ComposeConfig.IsUnknown() {
		params["custom_compose_config_string"] = data.ComposeConfig.ValueString()
	}

	// Build values
	values := map[string]any{}

	// Add storage
	if len(data.Storage) > 0 {
		storage := make(map[string]any)
		for _, s := range data.Storage {
			storage[s.VolumeName.ValueString()] = map[string]any{
				"type": s.Type.ValueString(),
				"host_path_config": map[string]any{
					"path":             s.HostPath.ValueString(),
					"acl_enable":       s.ACLEnable.ValueBool(),
					"auto_permissions": s.AutoPermissions.ValueBool(),
				},
			}
		}
		values["storage"] = storage
	}

	// Add network
	if len(data.Network) > 0 {
		network := make(map[string]any)
		for _, n := range data.Network {
			hostIPs := []string{}
			if !n.HostIPs.IsNull() && !n.HostIPs.IsUnknown() {
				n.HostIPs.ElementsAs(ctx, &hostIPs, false)
			}

			network[n.PortName.ValueString()] = map[string]any{
				"bind_mode":   n.BindMode.ValueString(),
				"host_ips":    hostIPs,
				"port_number": int(n.PortNumber.ValueInt64()),
			}
		}
		values["network"] = network
	}

	// Add labels
	if !data.Labels.IsNull() && !data.Labels.IsUnknown() {
		var labels []string
		data.Labels.ElementsAs(ctx, &labels, false)
		values["labels"] = labels
	}

	if len(values) > 0 {
		params["values"] = values
	}

	return params
}

// syncValuesFromAPI syncs storage, network, and labels from the API response.
func (r *AppResource) syncValuesFromAPI(ctx context.Context, data *AppResourceModel, values map[string]appValueEntry, resp *resource.ReadResponse) {
	// Aggregate all storage, network, and labels from all workload entries
	allStorage := make(map[string]storageConfigResponse)
	allNetwork := make(map[string]networkConfigResponse)
	var allLabels []string

	for _, entry := range values {
		for name, storage := range entry.Storage {
			allStorage[name] = storage
		}
		for name, network := range entry.Network {
			allNetwork[name] = network
		}
		allLabels = append(allLabels, entry.Labels...)
	}

	// Sync storage - convert map to list
	if len(allStorage) > 0 {
		data.Storage = make([]StorageBlockModel, 0, len(allStorage))
		for name, storage := range allStorage {
			data.Storage = append(data.Storage, StorageBlockModel{
				VolumeName:      types.StringValue(name),
				Type:            types.StringValue(storage.Type),
				HostPath:        types.StringValue(storage.HostPathConfig.Path),
				ACLEnable:       types.BoolValue(storage.HostPathConfig.ACLEnable),
				AutoPermissions: types.BoolValue(storage.HostPathConfig.AutoPermissions),
			})
		}
	} else {
		data.Storage = nil
	}

	// Sync network - convert map to list
	if len(allNetwork) > 0 {
		data.Network = make([]NetworkBlockModel, 0, len(allNetwork))
		for name, network := range allNetwork {
			hostIPsValue, diags := types.ListValueFrom(ctx, types.StringType, network.HostIPs)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			data.Network = append(data.Network, NetworkBlockModel{
				PortName:   types.StringValue(name),
				BindMode:   types.StringValue(network.BindMode),
				HostIPs:    hostIPsValue,
				PortNumber: types.Int64Value(network.PortNumber),
			})
		}
	} else {
		data.Network = nil
	}

	// Sync labels
	if len(allLabels) > 0 {
		labelsValue, diags := types.ListValueFrom(ctx, types.StringType, allLabels)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Labels = labelsValue
	} else {
		data.Labels = types.ListNull(types.StringType)
	}
}
