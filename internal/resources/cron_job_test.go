package resources

import (
	"context"
	"errors"
	"testing"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/terraform-provider-truenas/internal/services"
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
	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(r.(*CronJobResource))
	_ = resource.ResourceWithImportState(r.(*CronJobResource))
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

	svc := &services.TrueNASServices{}

	req := resource.ConfigureRequest{
		ProviderData: svc,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
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
	if attrs["capture_stdout"] == nil {
		t.Error("expected 'capture_stdout' attribute")
	}
	if attrs["capture_stderr"] == nil {
		t.Error("expected 'capture_stderr' attribute")
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
	ID            interface{}
	User          interface{}
	Command       interface{}
	Description   interface{}
	Enabled       bool
	CaptureStdout bool
	CaptureStderr bool
	Schedule      *scheduleBlockParams
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
		"id":             tftypes.NewValue(tftypes.String, p.ID),
		"user":           tftypes.NewValue(tftypes.String, p.User),
		"command":        tftypes.NewValue(tftypes.String, p.Command),
		"description":    tftypes.NewValue(tftypes.String, p.Description),
		"enabled":        tftypes.NewValue(tftypes.Bool, p.Enabled),
		"capture_stdout": tftypes.NewValue(tftypes.Bool, p.CaptureStdout),
		"capture_stderr": tftypes.NewValue(tftypes.Bool, p.CaptureStderr),
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
			"id":             tftypes.String,
			"user":           tftypes.String,
			"command":        tftypes.String,
			"description":    tftypes.String,
			"enabled":        tftypes.Bool,
			"capture_stdout": tftypes.Bool,
			"capture_stderr": tftypes.Bool,
			"schedule":       scheduleType,
		},
	}

	return tftypes.NewValue(objectType, values)
}

// testCronJob returns a standard test cron job for use in tests.
func testCronJob() *truenas.CronJob {
	return &truenas.CronJob{
		ID:            5,
		User:          "root",
		Command:       "/usr/local/bin/backup.sh",
		Description:   "Daily Backup",
		Enabled:       true,
		CaptureStdout: false,
		CaptureStderr: true,
		Schedule: truenas.Schedule{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	}
}

func TestCronJobResource_Create_Success(t *testing.T) {
	var capturedOpts truenas.CreateCronJobOpts

	r := &CronJobResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Cron: &truenas.MockCronService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateCronJobOpts) (*truenas.CronJob, error) {
					capturedOpts = opts
					return testCronJob(), nil
				},
			},
		}},
	}

	schemaResp := getCronJobResourceSchema(t)
	planValue := createCronJobModelValue(cronJobModelParams{
		User:          "root",
		Command:       "/usr/local/bin/backup.sh",
		Description:   "Daily Backup",
		Enabled:       true,
		CaptureStdout: false,
		CaptureStderr: true,
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

	// Verify opts sent to service
	if capturedOpts.User != "root" {
		t.Errorf("expected user 'root', got %q", capturedOpts.User)
	}
	if capturedOpts.Command != "/usr/local/bin/backup.sh" {
		t.Errorf("expected command '/usr/local/bin/backup.sh', got %q", capturedOpts.Command)
	}
	if capturedOpts.Description != "Daily Backup" {
		t.Errorf("expected description 'Daily Backup', got %q", capturedOpts.Description)
	}
	if capturedOpts.Enabled != true {
		t.Errorf("expected enabled true, got %v", capturedOpts.Enabled)
	}
	if capturedOpts.CaptureStdout != false {
		t.Errorf("expected CaptureStdout false, got %v", capturedOpts.CaptureStdout)
	}
	if capturedOpts.CaptureStderr != true {
		t.Errorf("expected CaptureStderr true, got %v", capturedOpts.CaptureStderr)
	}

	// Verify schedule opts
	if capturedOpts.Schedule.Minute != "0" {
		t.Errorf("expected schedule minute '0', got %q", capturedOpts.Schedule.Minute)
	}
	if capturedOpts.Schedule.Hour != "3" {
		t.Errorf("expected schedule hour '3', got %q", capturedOpts.Schedule.Hour)
	}
	if capturedOpts.Schedule.Dom != "*" {
		t.Errorf("expected schedule dom '*', got %q", capturedOpts.Schedule.Dom)
	}
	if capturedOpts.Schedule.Month != "*" {
		t.Errorf("expected schedule month '*', got %q", capturedOpts.Schedule.Month)
	}
	if capturedOpts.Schedule.Dow != "*" {
		t.Errorf("expected schedule dow '*', got %q", capturedOpts.Schedule.Dow)
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
	// CronJob already handles inversion: CaptureStdout=false maps directly
	if resultData.CaptureStdout.ValueBool() != false {
		t.Errorf("expected capture_stdout false, got %v", resultData.CaptureStdout.ValueBool())
	}
	// CronJob already handles inversion: CaptureStderr=true maps directly
	if resultData.CaptureStderr.ValueBool() != true {
		t.Errorf("expected capture_stderr true, got %v", resultData.CaptureStderr.ValueBool())
	}
}

func TestCronJobResource_Create_APIError(t *testing.T) {
	r := &CronJobResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Cron: &truenas.MockCronService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateCronJobOpts) (*truenas.CronJob, error) {
					return nil, errors.New("connection refused")
				},
			},
		}},
	}

	schemaResp := getCronJobResourceSchema(t)
	planValue := createCronJobModelValue(cronJobModelParams{
		User:          "root",
		Command:       "/usr/local/bin/backup.sh",
		Description:   "Daily Backup",
		Enabled:       true,
		CaptureStdout: false,
		CaptureStderr: true,
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
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Cron: &truenas.MockCronService{
				GetFunc: func(ctx context.Context, id int64) (*truenas.CronJob, error) {
					return testCronJob(), nil
				},
			},
		}},
	}

	schemaResp := getCronJobResourceSchema(t)
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:            "5",
		User:          "root",
		Command:       "/usr/local/bin/backup.sh",
		Description:   "Daily Backup",
		Enabled:       true,
		CaptureStdout: false,
		CaptureStderr: true,
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
	if resultData.CaptureStdout.ValueBool() != false {
		t.Errorf("expected capture_stdout false, got %v", resultData.CaptureStdout.ValueBool())
	}
	if resultData.CaptureStderr.ValueBool() != true {
		t.Errorf("expected capture_stderr true, got %v", resultData.CaptureStderr.ValueBool())
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
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Cron: &truenas.MockCronService{
				GetFunc: func(ctx context.Context, id int64) (*truenas.CronJob, error) {
					return nil, nil
				},
			},
		}},
	}

	schemaResp := getCronJobResourceSchema(t)
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:            "5",
		User:          "root",
		Command:       "/usr/local/bin/backup.sh",
		Description:   "Deleted Job",
		Enabled:       true,
		CaptureStdout: false,
		CaptureStderr: true,
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
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Cron: &truenas.MockCronService{
				GetFunc: func(ctx context.Context, id int64) (*truenas.CronJob, error) {
					return nil, errors.New("connection refused")
				},
			},
		}},
	}

	schemaResp := getCronJobResourceSchema(t)
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:            "5",
		User:          "root",
		Command:       "/usr/local/bin/backup.sh",
		Description:   "Daily Backup",
		Enabled:       true,
		CaptureStdout: false,
		CaptureStderr: true,
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
	var capturedID int64
	var capturedOpts truenas.UpdateCronJobOpts

	r := &CronJobResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Cron: &truenas.MockCronService{
				UpdateFunc: func(ctx context.Context, id int64, opts truenas.UpdateCronJobOpts) (*truenas.CronJob, error) {
					capturedID = id
					capturedOpts = opts
					return &truenas.CronJob{
						ID:            5,
						User:          "admin",
						Command:       "/usr/local/bin/updated-backup.sh",
						Description:   "Updated Daily Backup",
						Enabled:       false,
						CaptureStdout: true,
						CaptureStderr: false,
						Schedule: truenas.Schedule{
							Minute: "30",
							Hour:   "4",
							Dom:    "1",
							Month:  "*",
							Dow:    "1-5",
						},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCronJobResourceSchema(t)

	// Current state
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:            "5",
		User:          "root",
		Command:       "/usr/local/bin/backup.sh",
		Description:   "Daily Backup",
		Enabled:       true,
		CaptureStdout: false,
		CaptureStderr: true,
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
		ID:            "5",
		User:          "admin",
		Command:       "/usr/local/bin/updated-backup.sh",
		Description:   "Updated Daily Backup",
		Enabled:       false,
		CaptureStdout: true,
		CaptureStderr: false,
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

	if capturedID != 5 {
		t.Errorf("expected ID 5, got %d", capturedID)
	}

	// Verify update opts
	if capturedOpts.User != "admin" {
		t.Errorf("expected user 'admin', got %q", capturedOpts.User)
	}
	if capturedOpts.Command != "/usr/local/bin/updated-backup.sh" {
		t.Errorf("expected command '/usr/local/bin/updated-backup.sh', got %q", capturedOpts.Command)
	}
	if capturedOpts.Description != "Updated Daily Backup" {
		t.Errorf("expected description 'Updated Daily Backup', got %q", capturedOpts.Description)
	}
	if capturedOpts.Enabled != false {
		t.Errorf("expected enabled false, got %v", capturedOpts.Enabled)
	}
	if capturedOpts.CaptureStdout != true {
		t.Errorf("expected CaptureStdout true, got %v", capturedOpts.CaptureStdout)
	}
	if capturedOpts.CaptureStderr != false {
		t.Errorf("expected CaptureStderr false, got %v", capturedOpts.CaptureStderr)
	}

	// Verify schedule in update opts
	if capturedOpts.Schedule.Minute != "30" {
		t.Errorf("expected schedule minute '30', got %q", capturedOpts.Schedule.Minute)
	}
	if capturedOpts.Schedule.Hour != "4" {
		t.Errorf("expected schedule hour '4', got %q", capturedOpts.Schedule.Hour)
	}
	if capturedOpts.Schedule.Dom != "1" {
		t.Errorf("expected schedule dom '1', got %q", capturedOpts.Schedule.Dom)
	}
	if capturedOpts.Schedule.Dow != "1-5" {
		t.Errorf("expected schedule dow '1-5', got %q", capturedOpts.Schedule.Dow)
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
	if resultData.CaptureStdout.ValueBool() != true {
		t.Errorf("expected capture_stdout true, got %v", resultData.CaptureStdout.ValueBool())
	}
	if resultData.CaptureStderr.ValueBool() != false {
		t.Errorf("expected capture_stderr false, got %v", resultData.CaptureStderr.ValueBool())
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
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Cron: &truenas.MockCronService{
				UpdateFunc: func(ctx context.Context, id int64, opts truenas.UpdateCronJobOpts) (*truenas.CronJob, error) {
					return nil, errors.New("connection refused")
				},
			},
		}},
	}

	schemaResp := getCronJobResourceSchema(t)

	// Current state
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:            "5",
		User:          "root",
		Command:       "/usr/local/bin/backup.sh",
		Description:   "Daily Backup",
		Enabled:       true,
		CaptureStdout: false,
		CaptureStderr: true,
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
		ID:            "5",
		User:          "admin",
		Command:       "/usr/local/bin/updated-backup.sh",
		Description:   "Updated Daily Backup",
		Enabled:       false,
		CaptureStdout: true,
		CaptureStderr: false,
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

func TestCronJobResource_Delete_Success(t *testing.T) {
	var deleteCalled bool
	var capturedID int64

	r := &CronJobResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Cron: &truenas.MockCronService{
				DeleteFunc: func(ctx context.Context, id int64) error {
					deleteCalled = true
					capturedID = id
					return nil
				},
			},
		}},
	}

	schemaResp := getCronJobResourceSchema(t)
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:            "5",
		User:          "root",
		Command:       "/usr/local/bin/backup.sh",
		Description:   "Daily Backup",
		Enabled:       true,
		CaptureStdout: false,
		CaptureStderr: true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !deleteCalled {
		t.Error("expected Delete to be called")
	}

	if capturedID != 5 {
		t.Errorf("expected ID 5, got %d", capturedID)
	}
}

func TestCronJobResource_Delete_APIError(t *testing.T) {
	r := &CronJobResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Cron: &truenas.MockCronService{
				DeleteFunc: func(ctx context.Context, id int64) error {
					return errors.New("cron job in use")
				},
			},
		}},
	}

	schemaResp := getCronJobResourceSchema(t)
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:            "5",
		User:          "root",
		Command:       "/usr/local/bin/backup.sh",
		Description:   "Daily Backup",
		Enabled:       true,
		CaptureStdout: false,
		CaptureStderr: true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestCronJobResource_Create_CustomSchedule(t *testing.T) {
	var capturedOpts truenas.CreateCronJobOpts

	r := &CronJobResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Cron: &truenas.MockCronService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateCronJobOpts) (*truenas.CronJob, error) {
					capturedOpts = opts
					return &truenas.CronJob{
						ID:            10,
						User:          "admin",
						Command:       "/usr/local/bin/report.sh",
						Description:   "Business Hours Report",
						Enabled:       true,
						CaptureStdout: false,
						CaptureStderr: false,
						Schedule: truenas.Schedule{
							Minute: "*/15",
							Hour:   "9-17",
							Dom:    "1,15",
							Month:  "*",
							Dow:    "1-5",
						},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCronJobResourceSchema(t)
	planValue := createCronJobModelValue(cronJobModelParams{
		User:          "admin",
		Command:       "/usr/local/bin/report.sh",
		Description:   "Business Hours Report",
		Enabled:       true,
		CaptureStdout: false,
		CaptureStderr: false,
		Schedule: &scheduleBlockParams{
			Minute: "*/15",
			Hour:   "9-17",
			Dom:    "1,15",
			Month:  "*",
			Dow:    "1-5",
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

	// Verify schedule opts
	if capturedOpts.Schedule.Minute != "*/15" {
		t.Errorf("expected schedule minute '*/15', got %q", capturedOpts.Schedule.Minute)
	}
	if capturedOpts.Schedule.Hour != "9-17" {
		t.Errorf("expected schedule hour '9-17', got %q", capturedOpts.Schedule.Hour)
	}
	if capturedOpts.Schedule.Dom != "1,15" {
		t.Errorf("expected schedule dom '1,15', got %q", capturedOpts.Schedule.Dom)
	}
	if capturedOpts.Schedule.Month != "*" {
		t.Errorf("expected schedule month '*', got %q", capturedOpts.Schedule.Month)
	}
	if capturedOpts.Schedule.Dow != "1-5" {
		t.Errorf("expected schedule dow '1-5', got %q", capturedOpts.Schedule.Dow)
	}

	// Verify state was set correctly
	var resultData CronJobResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "10" {
		t.Errorf("expected ID '10', got %q", resultData.ID.ValueString())
	}
	if resultData.Schedule == nil {
		t.Fatal("expected schedule block to be set")
	}
	if resultData.Schedule.Minute.ValueString() != "*/15" {
		t.Errorf("expected schedule minute '*/15', got %q", resultData.Schedule.Minute.ValueString())
	}
	if resultData.Schedule.Hour.ValueString() != "9-17" {
		t.Errorf("expected schedule hour '9-17', got %q", resultData.Schedule.Hour.ValueString())
	}
	if resultData.Schedule.Dom.ValueString() != "1,15" {
		t.Errorf("expected schedule dom '1,15', got %q", resultData.Schedule.Dom.ValueString())
	}
	if resultData.Schedule.Dow.ValueString() != "1-5" {
		t.Errorf("expected schedule dow '1-5', got %q", resultData.Schedule.Dow.ValueString())
	}
}

func TestCronJobResource_Update_ScheduleOnly(t *testing.T) {
	var capturedOpts truenas.UpdateCronJobOpts

	r := &CronJobResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Cron: &truenas.MockCronService{
				UpdateFunc: func(ctx context.Context, id int64, opts truenas.UpdateCronJobOpts) (*truenas.CronJob, error) {
					capturedOpts = opts
					return &truenas.CronJob{
						ID:            7,
						User:          "root",
						Command:       "/usr/local/bin/backup.sh",
						Description:   "Daily Backup",
						Enabled:       true,
						CaptureStdout: false,
						CaptureStderr: true,
						Schedule: truenas.Schedule{
							Minute: "*/30",
							Hour:   "*/2",
							Dom:    "*",
							Month:  "1-6",
							Dow:    "0,6",
						},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCronJobResourceSchema(t)

	// Current state with original schedule
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:            "7",
		User:          "root",
		Command:       "/usr/local/bin/backup.sh",
		Description:   "Daily Backup",
		Enabled:       true,
		CaptureStdout: false,
		CaptureStderr: true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	})

	// Updated plan with only schedule changes
	planValue := createCronJobModelValue(cronJobModelParams{
		ID:            "7",
		User:          "root",
		Command:       "/usr/local/bin/backup.sh",
		Description:   "Daily Backup",
		Enabled:       true,
		CaptureStdout: false,
		CaptureStderr: true,
		Schedule: &scheduleBlockParams{
			Minute: "*/30",
			Hour:   "*/2",
			Dom:    "*",
			Month:  "1-6",
			Dow:    "0,6",
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

	// Verify non-schedule opts remain unchanged
	if capturedOpts.User != "root" {
		t.Errorf("expected user 'root', got %q", capturedOpts.User)
	}
	if capturedOpts.Command != "/usr/local/bin/backup.sh" {
		t.Errorf("expected command '/usr/local/bin/backup.sh', got %q", capturedOpts.Command)
	}
	if capturedOpts.Description != "Daily Backup" {
		t.Errorf("expected description 'Daily Backup', got %q", capturedOpts.Description)
	}
	if capturedOpts.Enabled != true {
		t.Errorf("expected enabled true, got %v", capturedOpts.Enabled)
	}
	if capturedOpts.CaptureStdout != false {
		t.Errorf("expected CaptureStdout false, got %v", capturedOpts.CaptureStdout)
	}
	if capturedOpts.CaptureStderr != true {
		t.Errorf("expected CaptureStderr true, got %v", capturedOpts.CaptureStderr)
	}

	// Verify schedule opts changed
	if capturedOpts.Schedule.Minute != "*/30" {
		t.Errorf("expected schedule minute '*/30', got %q", capturedOpts.Schedule.Minute)
	}
	if capturedOpts.Schedule.Hour != "*/2" {
		t.Errorf("expected schedule hour '*/2', got %q", capturedOpts.Schedule.Hour)
	}
	if capturedOpts.Schedule.Dom != "*" {
		t.Errorf("expected schedule dom '*', got %q", capturedOpts.Schedule.Dom)
	}
	if capturedOpts.Schedule.Month != "1-6" {
		t.Errorf("expected schedule month '1-6', got %q", capturedOpts.Schedule.Month)
	}
	if capturedOpts.Schedule.Dow != "0,6" {
		t.Errorf("expected schedule dow '0,6', got %q", capturedOpts.Schedule.Dow)
	}

	// Verify state was set correctly
	var resultData CronJobResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.Schedule == nil {
		t.Fatal("expected schedule block to be set")
	}
	if resultData.Schedule.Minute.ValueString() != "*/30" {
		t.Errorf("expected schedule minute '*/30', got %q", resultData.Schedule.Minute.ValueString())
	}
	if resultData.Schedule.Hour.ValueString() != "*/2" {
		t.Errorf("expected schedule hour '*/2', got %q", resultData.Schedule.Hour.ValueString())
	}
	if resultData.Schedule.Month.ValueString() != "1-6" {
		t.Errorf("expected schedule month '1-6', got %q", resultData.Schedule.Month.ValueString())
	}
	if resultData.Schedule.Dow.ValueString() != "0,6" {
		t.Errorf("expected schedule dow '0,6', got %q", resultData.Schedule.Dow.ValueString())
	}
}

func TestCronJobResource_Create_DisabledJob(t *testing.T) {
	var capturedOpts truenas.CreateCronJobOpts

	r := &CronJobResource{
		BaseResource: BaseResource{services: &services.TrueNASServices{
			Cron: &truenas.MockCronService{
				CreateFunc: func(ctx context.Context, opts truenas.CreateCronJobOpts) (*truenas.CronJob, error) {
					capturedOpts = opts
					return &truenas.CronJob{
						ID:            12,
						User:          "root",
						Command:       "/usr/local/bin/maintenance.sh",
						Description:   "Disabled Maintenance Job",
						Enabled:       false,
						CaptureStdout: true,
						CaptureStderr: true,
						Schedule: truenas.Schedule{
							Minute: "0",
							Hour:   "0",
							Dom:    "*",
							Month:  "*",
							Dow:    "*",
						},
					}, nil
				},
			},
		}},
	}

	schemaResp := getCronJobResourceSchema(t)
	planValue := createCronJobModelValue(cronJobModelParams{
		User:          "root",
		Command:       "/usr/local/bin/maintenance.sh",
		Description:   "Disabled Maintenance Job",
		Enabled:       false,
		CaptureStdout: true,
		CaptureStderr: true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "0",
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

	// Verify enabled flag is false in opts
	if capturedOpts.Enabled != false {
		t.Errorf("expected enabled false, got %v", capturedOpts.Enabled)
	}
	if capturedOpts.CaptureStdout != true {
		t.Errorf("expected CaptureStdout true, got %v", capturedOpts.CaptureStdout)
	}
	if capturedOpts.CaptureStderr != true {
		t.Errorf("expected CaptureStderr true, got %v", capturedOpts.CaptureStderr)
	}

	// Verify state was set correctly with enabled=false
	var resultData CronJobResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "12" {
		t.Errorf("expected ID '12', got %q", resultData.ID.ValueString())
	}
	if resultData.Enabled.ValueBool() != false {
		t.Errorf("expected enabled false, got %v", resultData.Enabled.ValueBool())
	}
	if resultData.Description.ValueString() != "Disabled Maintenance Job" {
		t.Errorf("expected description 'Disabled Maintenance Job', got %q", resultData.Description.ValueString())
	}
}
