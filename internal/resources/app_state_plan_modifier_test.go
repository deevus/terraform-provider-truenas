package resources

import (
	"context"
	"testing"

	customtypes "github.com/deevus/terraform-provider-truenas/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestComputedStatePlanModifier_Description(t *testing.T) {
	modifier := computedStatePlanModifier()

	description := modifier.Description(context.Background())

	if description == "" {
		t.Error("expected non-empty description")
	}
	expected := "Predicts state value based on desired_state and drift detection."
	if description != expected {
		t.Errorf("expected description %q, got %q", expected, description)
	}
}

func TestComputedStatePlanModifier_MarkdownDescription(t *testing.T) {
	modifier := computedStatePlanModifier()

	description := modifier.MarkdownDescription(context.Background())

	if description == "" {
		t.Error("expected non-empty markdown description")
	}
	expected := "Predicts `state` value: if drift detected (state != desired_state), plans reconciliation to desired_state; otherwise preserves current state."
	if description != expected {
		t.Errorf("expected markdown description %q, got %q", expected, description)
	}
}

// Helper to create a simple schema for testing
func testAppSchema() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"desired_state": tftypes.String,
			"state":         tftypes.String,
		},
	}
}

func TestComputedStatePlanModifier_PlanModifyString(t *testing.T) {
	tests := []struct {
		name               string
		stateValue         types.String
		stateDesiredState  string
		planDesiredState   string
		expectedPreserved  bool // true if state should be preserved
	}{
		{
			name:              "desired_state unchanged - preserve state",
			stateValue:        types.StringValue("STOPPED"),
			stateDesiredState: "STOPPED",
			planDesiredState:  "STOPPED",
			expectedPreserved: true,
		},
		{
			name:              "desired_state unchanged case insensitive - preserve state",
			stateValue:        types.StringValue("STOPPED"),
			stateDesiredState: "STOPPED",
			planDesiredState:  "stopped",
			expectedPreserved: true,
		},
		{
			name:              "desired_state changing - mark unknown",
			stateValue:        types.StringValue("STOPPED"),
			stateDesiredState: "STOPPED",
			planDesiredState:  "RUNNING",
			expectedPreserved: false,
		},
		{
			name:              "desired_state changing case insensitive - mark unknown",
			stateValue:        types.StringValue("RUNNING"),
			stateDesiredState: "running",
			planDesiredState:  "stopped",
			expectedPreserved: false,
		},
	}

	schemaType := testAppSchema()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			modifier := computedStatePlanModifier()

			// Create state
			stateVal := tftypes.NewValue(schemaType, map[string]tftypes.Value{
				"desired_state": tftypes.NewValue(tftypes.String, tc.stateDesiredState),
				"state":         tftypes.NewValue(tftypes.String, tc.stateValue.ValueString()),
			})
			state := tfsdk.State{
				Raw:    stateVal,
				Schema: testAppSchemaFramework(),
			}

			// Create plan
			planVal := tftypes.NewValue(schemaType, map[string]tftypes.Value{
				"desired_state": tftypes.NewValue(tftypes.String, tc.planDesiredState),
				"state":         tftypes.NewValue(tftypes.String, nil), // unknown in plan
			})
			plan := tfsdk.Plan{
				Raw:    planVal,
				Schema: testAppSchemaFramework(),
			}

			req := planmodifier.StringRequest{
				StateValue: tc.stateValue,
				PlanValue:  types.StringUnknown(),
				State:      state,
				Plan:       plan,
			}
			resp := &planmodifier.StringResponse{
				PlanValue: types.StringUnknown(),
			}

			modifier.PlanModifyString(context.Background(), req, resp)

			if tc.expectedPreserved {
				if resp.PlanValue.ValueString() != tc.stateValue.ValueString() {
					t.Errorf("expected preserved value %q, got %q", tc.stateValue.ValueString(), resp.PlanValue.ValueString())
				}
			} else {
				if !resp.PlanValue.IsUnknown() {
					t.Errorf("expected unknown value, got %q", resp.PlanValue.ValueString())
				}
			}
		})
	}
}

func TestComputedStatePlanModifier_DriftReconciliation(t *testing.T) {
	// Bug: When the actual state differs from desired_state (drift scenario),
	// the plan modifier should predict state will change to match desired_state
	// because reconciliation will occur during Apply.
	//
	// Scenario: App was externally stopped (state=STOPPED) but desired_state=RUNNING.
	// The plan modifier should predict state=RUNNING (not preserve STOPPED).
	tests := []struct {
		name              string
		stateValue        types.String // actual state from TrueNAS
		stateDesiredState string       // desired_state in terraform state
		planDesiredState  string       // desired_state in plan
		expectedPlanValue string       // what state should be planned as
	}{
		{
			name:              "state STOPPED but desired RUNNING - should plan RUNNING",
			stateValue:        types.StringValue("STOPPED"),
			stateDesiredState: "RUNNING",
			planDesiredState:  "RUNNING",
			expectedPlanValue: "RUNNING", // reconciliation will start the app
		},
		{
			name:              "state RUNNING but desired STOPPED - should plan STOPPED",
			stateValue:        types.StringValue("RUNNING"),
			stateDesiredState: "STOPPED",
			planDesiredState:  "STOPPED",
			expectedPlanValue: "STOPPED", // reconciliation will stop the app
		},
		{
			name:              "state matches desired - should preserve state",
			stateValue:        types.StringValue("RUNNING"),
			stateDesiredState: "RUNNING",
			planDesiredState:  "RUNNING",
			expectedPlanValue: "RUNNING", // no reconciliation needed
		},
		{
			name:              "CRASHED when desired STOPPED - should preserve CRASHED",
			stateValue:        types.StringValue("CRASHED"),
			stateDesiredState: "STOPPED",
			planDesiredState:  "STOPPED",
			expectedPlanValue: "CRASHED", // CRASHED is "stopped enough"
		},
	}

	schemaType := testAppSchema()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			modifier := computedStatePlanModifier()

			// Create state
			stateVal := tftypes.NewValue(schemaType, map[string]tftypes.Value{
				"desired_state": tftypes.NewValue(tftypes.String, tc.stateDesiredState),
				"state":         tftypes.NewValue(tftypes.String, tc.stateValue.ValueString()),
			})
			state := tfsdk.State{
				Raw:    stateVal,
				Schema: testAppSchemaFramework(),
			}

			// Create plan
			planVal := tftypes.NewValue(schemaType, map[string]tftypes.Value{
				"desired_state": tftypes.NewValue(tftypes.String, tc.planDesiredState),
				"state":         tftypes.NewValue(tftypes.String, nil), // unknown in plan
			})
			plan := tfsdk.Plan{
				Raw:    planVal,
				Schema: testAppSchemaFramework(),
			}

			req := planmodifier.StringRequest{
				StateValue: tc.stateValue,
				PlanValue:  types.StringUnknown(),
				State:      state,
				Plan:       plan,
			}
			resp := &planmodifier.StringResponse{
				PlanValue: types.StringUnknown(),
			}

			modifier.PlanModifyString(context.Background(), req, resp)

			if resp.PlanValue.ValueString() != tc.expectedPlanValue {
				t.Errorf("expected plan value %q, got %q", tc.expectedPlanValue, resp.PlanValue.ValueString())
			}
		})
	}
}

func TestComputedStatePlanModifier_NullState(t *testing.T) {
	modifier := computedStatePlanModifier()

	req := planmodifier.StringRequest{
		StateValue: types.StringNull(),
		PlanValue:  types.StringUnknown(),
	}
	resp := &planmodifier.StringResponse{
		PlanValue: types.StringUnknown(),
	}

	modifier.PlanModifyString(context.Background(), req, resp)

	// Should return early without modifying
	if !resp.PlanValue.IsUnknown() {
		t.Errorf("expected unknown value for null state, got %v", resp.PlanValue)
	}
}

// Helper to create framework schema for tests
func testAppSchemaFramework() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"desired_state": schema.StringAttribute{
				CustomType: customtypes.CaseInsensitiveStringType{},
			},
			"state": schema.StringAttribute{},
		},
	}
}
