package resources

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewCronJobResource(t *testing.T) {
	r := NewCronJobResource()
	if r == nil {
		t.Fatal("NewCronJobResource returned nil")
	}

	_, ok := r.(*CronJobResource)
	if !ok {
		t.Fatalf("expected *CronJobResource, got %T", r)
	}

	// Verify interface implementations
	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*CronJobResource)
	var _ resource.ResourceWithImportState = r.(*CronJobResource)
}

func TestCronJobResource_Metadata(t *testing.T) {
	r := NewCronJobResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_cron_job" {
		t.Errorf("expected TypeName 'truenas_cron_job', got %q", resp.TypeName)
	}
}

func TestCronJobResource_Configure_Success(t *testing.T) {
	r := NewCronJobResource().(*CronJobResource)

	mockClient := &client.MockClient{}

	req := resource.ConfigureRequest{
		ProviderData: mockClient,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if r.client == nil {
		t.Error("expected client to be set")
	}
}

func TestCronJobResource_Configure_NilProviderData(t *testing.T) {
	r := NewCronJobResource().(*CronJobResource)

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestCronJobResource_Configure_WrongType(t *testing.T) {
	r := NewCronJobResource().(*CronJobResource)

	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

func TestCronJobResource_Schema(t *testing.T) {
	r := NewCronJobResource()

	ctx := context.Background()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}

	r.Schema(ctx, schemaReq, schemaResp)

	if schemaResp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify required attributes exist
	attrs := schemaResp.Schema.Attributes
	if attrs["id"] == nil {
		t.Error("expected 'id' attribute")
	}
	if attrs["user"] == nil {
		t.Error("expected 'user' attribute")
	}
	if attrs["command"] == nil {
		t.Error("expected 'command' attribute")
	}
	if attrs["description"] == nil {
		t.Error("expected 'description' attribute")
	}
	if attrs["enabled"] == nil {
		t.Error("expected 'enabled' attribute")
	}
	if attrs["stdout"] == nil {
		t.Error("expected 'stdout' attribute")
	}
	if attrs["stderr"] == nil {
		t.Error("expected 'stderr' attribute")
	}

	// Verify blocks exist
	blocks := schemaResp.Schema.Blocks
	if blocks["schedule"] == nil {
		t.Error("expected 'schedule' block")
	}
}

// Test helpers

func getCronJobResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewCronJobResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("failed to get schema: %v", schemaResp.Diagnostics)
	}
	return *schemaResp
}

// cronJobModelParams holds parameters for creating test model values.
type cronJobModelParams struct {
	ID          interface{}
	User        interface{}
	Command     interface{}
	Description interface{}
	Enabled     bool
	Stdout      bool
	Stderr      bool
	Schedule    *scheduleBlockParams
}

func createCronJobModelValue(p cronJobModelParams) tftypes.Value {
	// Define type structures
	scheduleType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"minute": tftypes.String,
			"hour":   tftypes.String,
			"dom":    tftypes.String,
			"month":  tftypes.String,
			"dow":    tftypes.String,
		},
	}

	// Build the values map
	values := map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, p.ID),
		"user":        tftypes.NewValue(tftypes.String, p.User),
		"command":     tftypes.NewValue(tftypes.String, p.Command),
		"description": tftypes.NewValue(tftypes.String, p.Description),
		"enabled":     tftypes.NewValue(tftypes.Bool, p.Enabled),
		"stdout":      tftypes.NewValue(tftypes.Bool, p.Stdout),
		"stderr":      tftypes.NewValue(tftypes.Bool, p.Stderr),
	}

	// Handle schedule block
	if p.Schedule != nil {
		values["schedule"] = tftypes.NewValue(scheduleType, map[string]tftypes.Value{
			"minute": tftypes.NewValue(tftypes.String, p.Schedule.Minute),
			"hour":   tftypes.NewValue(tftypes.String, p.Schedule.Hour),
			"dom":    tftypes.NewValue(tftypes.String, p.Schedule.Dom),
			"month":  tftypes.NewValue(tftypes.String, p.Schedule.Month),
			"dow":    tftypes.NewValue(tftypes.String, p.Schedule.Dow),
		})
	} else {
		values["schedule"] = tftypes.NewValue(scheduleType, nil)
	}

	// Create object type matching the schema
	objectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":          tftypes.String,
			"user":        tftypes.String,
			"command":     tftypes.String,
			"description": tftypes.String,
			"enabled":     tftypes.Bool,
			"stdout":      tftypes.Bool,
			"stderr":      tftypes.Bool,
			"schedule":    scheduleType,
		},
	}

	return tftypes.NewValue(objectType, values)
}

func TestCronJobResource_Create_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cronjob.create" {
					capturedMethod = method
					capturedParams = params
					return json.RawMessage(`{"id": 5}`), nil
				}
				if method == "cronjob.query" {
					return json.RawMessage(`[{
						"id": 5,
						"user": "root",
						"command": "/usr/local/bin/backup.sh",
						"description": "Daily Backup",
						"enabled": true,
						"stdout": true,
						"stderr": false,
						"schedule": {"minute": "0", "hour": "3", "dom": "*", "month": "*", "dow": "*"}
					}]`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	planValue := createCronJobModelValue(cronJobModelParams{
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily Backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      false,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if capturedMethod != "cronjob.create" {
		t.Errorf("expected method 'cronjob.create', got %q", capturedMethod)
	}

	// Verify params
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["user"] != "root" {
		t.Errorf("expected user 'root', got %v", params["user"])
	}
	if params["command"] != "/usr/local/bin/backup.sh" {
		t.Errorf("expected command '/usr/local/bin/backup.sh', got %v", params["command"])
	}
	if params["description"] != "Daily Backup" {
		t.Errorf("expected description 'Daily Backup', got %v", params["description"])
	}
	if params["enabled"] != true {
		t.Errorf("expected enabled true, got %v", params["enabled"])
	}
	if params["stdout"] != true {
		t.Errorf("expected stdout true, got %v", params["stdout"])
	}
	if params["stderr"] != false {
		t.Errorf("expected stderr false, got %v", params["stderr"])
	}

	// Verify schedule
	schedule, ok := params["schedule"].(map[string]any)
	if !ok {
		t.Fatalf("expected schedule to be map[string]any, got %T", params["schedule"])
	}
	if schedule["minute"] != "0" {
		t.Errorf("expected schedule minute '0', got %v", schedule["minute"])
	}
	if schedule["hour"] != "3" {
		t.Errorf("expected schedule hour '3', got %v", schedule["hour"])
	}
	if schedule["dom"] != "*" {
		t.Errorf("expected schedule dom '*', got %v", schedule["dom"])
	}
	if schedule["month"] != "*" {
		t.Errorf("expected schedule month '*', got %v", schedule["month"])
	}
	if schedule["dow"] != "*" {
		t.Errorf("expected schedule dow '*', got %v", schedule["dow"])
	}

	// Verify state was set
	var resultData CronJobResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "5" {
		t.Errorf("expected ID '5', got %q", resultData.ID.ValueString())
	}
	if resultData.User.ValueString() != "root" {
		t.Errorf("expected user 'root', got %q", resultData.User.ValueString())
	}
	if resultData.Command.ValueString() != "/usr/local/bin/backup.sh" {
		t.Errorf("expected command '/usr/local/bin/backup.sh', got %q", resultData.Command.ValueString())
	}
	if resultData.Description.ValueString() != "Daily Backup" {
		t.Errorf("expected description 'Daily Backup', got %q", resultData.Description.ValueString())
	}
	if resultData.Enabled.ValueBool() != true {
		t.Errorf("expected enabled true, got %v", resultData.Enabled.ValueBool())
	}
	if resultData.Stdout.ValueBool() != true {
		t.Errorf("expected stdout true, got %v", resultData.Stdout.ValueBool())
	}
	if resultData.Stderr.ValueBool() != false {
		t.Errorf("expected stderr false, got %v", resultData.Stderr.ValueBool())
	}
}

func TestCronJobResource_Create_APIError(t *testing.T) {
	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	planValue := createCronJobModelValue(cronJobModelParams{
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily Backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      false,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}

	// Verify state was not set (should remain empty/null)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to not be set when API returns error")
	}
}

func TestCronJobResource_Read_Success(t *testing.T) {
	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": 5,
					"user": "root",
					"command": "/usr/local/bin/backup.sh",
					"description": "Daily Backup",
					"enabled": true,
					"stdout": true,
					"stderr": false,
					"schedule": {"minute": "0", "hour": "3", "dom": "*", "month": "*", "dow": "*"}
				}]`), nil
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:          "5",
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily Backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      false,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	})

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify state was updated
	var resultData CronJobResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "5" {
		t.Errorf("expected ID '5', got %q", resultData.ID.ValueString())
	}
	if resultData.User.ValueString() != "root" {
		t.Errorf("expected user 'root', got %q", resultData.User.ValueString())
	}
	if resultData.Command.ValueString() != "/usr/local/bin/backup.sh" {
		t.Errorf("expected command '/usr/local/bin/backup.sh', got %q", resultData.Command.ValueString())
	}
	if resultData.Description.ValueString() != "Daily Backup" {
		t.Errorf("expected description 'Daily Backup', got %q", resultData.Description.ValueString())
	}
	if resultData.Enabled.ValueBool() != true {
		t.Errorf("expected enabled true, got %v", resultData.Enabled.ValueBool())
	}
	if resultData.Stdout.ValueBool() != true {
		t.Errorf("expected stdout true, got %v", resultData.Stdout.ValueBool())
	}
	if resultData.Stderr.ValueBool() != false {
		t.Errorf("expected stderr false, got %v", resultData.Stderr.ValueBool())
	}
	if resultData.Schedule == nil {
		t.Fatal("expected schedule block to be set")
	}
	if resultData.Schedule.Minute.ValueString() != "0" {
		t.Errorf("expected schedule minute '0', got %q", resultData.Schedule.Minute.ValueString())
	}
	if resultData.Schedule.Hour.ValueString() != "3" {
		t.Errorf("expected schedule hour '3', got %q", resultData.Schedule.Hour.ValueString())
	}
	if resultData.Schedule.Dom.ValueString() != "*" {
		t.Errorf("expected schedule dom '*', got %q", resultData.Schedule.Dom.ValueString())
	}
	if resultData.Schedule.Month.ValueString() != "*" {
		t.Errorf("expected schedule month '*', got %q", resultData.Schedule.Month.ValueString())
	}
	if resultData.Schedule.Dow.ValueString() != "*" {
		t.Errorf("expected schedule dow '*', got %q", resultData.Schedule.Dow.ValueString())
	}
}

func TestCronJobResource_Read_NotFound(t *testing.T) {
	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:          "5",
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Deleted Job",
		Enabled:     true,
		Stdout:      true,
		Stderr:      false,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	})

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// State should be removed (resource not found)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed when resource not found")
	}
}

func TestCronJobResource_Read_APIError(t *testing.T) {
	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:          "5",
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily Backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      false,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	})

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestCronJobResource_Update_Success(t *testing.T) {
	var capturedMethod string
	var capturedID int64
	var capturedUpdateData map[string]any

	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cronjob.update" {
					capturedMethod = method
					args := params.([]any)
					capturedID = args[0].(int64)
					capturedUpdateData = args[1].(map[string]any)
					return json.RawMessage(`{"id": 5}`), nil
				}
				if method == "cronjob.query" {
					return json.RawMessage(`[{
						"id": 5,
						"user": "admin",
						"command": "/usr/local/bin/updated-backup.sh",
						"description": "Updated Daily Backup",
						"enabled": false,
						"stdout": false,
						"stderr": true,
						"schedule": {"minute": "30", "hour": "4", "dom": "1", "month": "*", "dow": "1-5"}
					}]`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)

	// Current state
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:          "5",
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily Backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      false,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	})

	// Updated plan
	planValue := createCronJobModelValue(cronJobModelParams{
		ID:          "5",
		User:        "admin",
		Command:     "/usr/local/bin/updated-backup.sh",
		Description: "Updated Daily Backup",
		Enabled:     false,
		Stdout:      false,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "30",
			Hour:   "4",
			Dom:    "1",
			Month:  "*",
			Dow:    "1-5",
		},
	})

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if capturedMethod != "cronjob.update" {
		t.Errorf("expected method 'cronjob.update', got %q", capturedMethod)
	}

	if capturedID != 5 {
		t.Errorf("expected ID 5, got %d", capturedID)
	}

	// Verify update params
	if capturedUpdateData["user"] != "admin" {
		t.Errorf("expected user 'admin', got %v", capturedUpdateData["user"])
	}
	if capturedUpdateData["command"] != "/usr/local/bin/updated-backup.sh" {
		t.Errorf("expected command '/usr/local/bin/updated-backup.sh', got %v", capturedUpdateData["command"])
	}
	if capturedUpdateData["description"] != "Updated Daily Backup" {
		t.Errorf("expected description 'Updated Daily Backup', got %v", capturedUpdateData["description"])
	}
	if capturedUpdateData["enabled"] != false {
		t.Errorf("expected enabled false, got %v", capturedUpdateData["enabled"])
	}
	if capturedUpdateData["stdout"] != false {
		t.Errorf("expected stdout false, got %v", capturedUpdateData["stdout"])
	}
	if capturedUpdateData["stderr"] != true {
		t.Errorf("expected stderr true, got %v", capturedUpdateData["stderr"])
	}

	// Verify schedule in update params
	schedule, ok := capturedUpdateData["schedule"].(map[string]any)
	if !ok {
		t.Fatalf("expected schedule to be map[string]any, got %T", capturedUpdateData["schedule"])
	}
	if schedule["minute"] != "30" {
		t.Errorf("expected schedule minute '30', got %v", schedule["minute"])
	}
	if schedule["hour"] != "4" {
		t.Errorf("expected schedule hour '4', got %v", schedule["hour"])
	}
	if schedule["dom"] != "1" {
		t.Errorf("expected schedule dom '1', got %v", schedule["dom"])
	}
	if schedule["dow"] != "1-5" {
		t.Errorf("expected schedule dow '1-5', got %v", schedule["dow"])
	}

	// Verify state was set
	var resultData CronJobResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "5" {
		t.Errorf("expected ID '5', got %q", resultData.ID.ValueString())
	}
	if resultData.User.ValueString() != "admin" {
		t.Errorf("expected user 'admin', got %q", resultData.User.ValueString())
	}
	if resultData.Command.ValueString() != "/usr/local/bin/updated-backup.sh" {
		t.Errorf("expected command '/usr/local/bin/updated-backup.sh', got %q", resultData.Command.ValueString())
	}
	if resultData.Description.ValueString() != "Updated Daily Backup" {
		t.Errorf("expected description 'Updated Daily Backup', got %q", resultData.Description.ValueString())
	}
	if resultData.Enabled.ValueBool() != false {
		t.Errorf("expected enabled false, got %v", resultData.Enabled.ValueBool())
	}
	if resultData.Stdout.ValueBool() != false {
		t.Errorf("expected stdout false, got %v", resultData.Stdout.ValueBool())
	}
	if resultData.Stderr.ValueBool() != true {
		t.Errorf("expected stderr true, got %v", resultData.Stderr.ValueBool())
	}
	if resultData.Schedule == nil {
		t.Fatal("expected schedule block to be set")
	}
	if resultData.Schedule.Minute.ValueString() != "30" {
		t.Errorf("expected schedule minute '30', got %q", resultData.Schedule.Minute.ValueString())
	}
	if resultData.Schedule.Hour.ValueString() != "4" {
		t.Errorf("expected schedule hour '4', got %q", resultData.Schedule.Hour.ValueString())
	}
	if resultData.Schedule.Dom.ValueString() != "1" {
		t.Errorf("expected schedule dom '1', got %q", resultData.Schedule.Dom.ValueString())
	}
	if resultData.Schedule.Dow.ValueString() != "1-5" {
		t.Errorf("expected schedule dow '1-5', got %q", resultData.Schedule.Dow.ValueString())
	}
}

func TestCronJobResource_Update_APIError(t *testing.T) {
	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)

	// Current state
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:          "5",
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily Backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      false,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	})

	// Updated plan
	planValue := createCronJobModelValue(cronJobModelParams{
		ID:          "5",
		User:        "admin",
		Command:     "/usr/local/bin/updated-backup.sh",
		Description: "Updated Daily Backup",
		Enabled:     false,
		Stdout:      false,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "30",
			Hour:   "4",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	})

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}
