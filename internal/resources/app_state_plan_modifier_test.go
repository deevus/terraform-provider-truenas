package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestCaseInsensitiveStatePlanModifier_Description(t *testing.T) {
	modifier := caseInsensitiveStatePlanModifier()

	description := modifier.Description(context.Background())

	if description == "" {
		t.Error("expected non-empty description")
	}
	expected := "Treats state values as equal regardless of case."
	if description != expected {
		t.Errorf("expected description %q, got %q", expected, description)
	}
}

func TestCaseInsensitiveStatePlanModifier_MarkdownDescription(t *testing.T) {
	modifier := caseInsensitiveStatePlanModifier()

	description := modifier.MarkdownDescription(context.Background())

	if description == "" {
		t.Error("expected non-empty markdown description")
	}
	expected := "Treats state values as equal regardless of case (e.g., `running` == `RUNNING`)."
	if description != expected {
		t.Errorf("expected markdown description %q, got %q", expected, description)
	}
}

func TestCaseInsensitiveStatePlanModifier_PlanModifyString(t *testing.T) {
	tests := []struct {
		name         string
		stateValue   types.String
		planValue    types.String
		expectedPlan string
	}{
		{
			name:         "lowercase to uppercase normalization",
			stateValue:   types.StringValue("RUNNING"),
			planValue:    types.StringValue("running"),
			expectedPlan: "RUNNING", // Normalized to uppercase
		},
		{
			name:         "same case - no change",
			stateValue:   types.StringValue("STOPPED"),
			planValue:    types.StringValue("STOPPED"),
			expectedPlan: "STOPPED",
		},
		{
			name:         "state change - normalized to uppercase",
			stateValue:   types.StringValue("RUNNING"),
			planValue:    types.StringValue("stopped"),
			expectedPlan: "STOPPED", // Normalized to uppercase
		},
		{
			name:         "initial create - null state",
			stateValue:   types.StringNull(),
			planValue:    types.StringValue("stopped"),
			expectedPlan: "STOPPED", // Normalized to uppercase
		},
		{
			name:         "null plan - no change",
			stateValue:   types.StringValue("RUNNING"),
			planValue:    types.StringNull(),
			expectedPlan: "", // Null stays null
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			modifier := caseInsensitiveStatePlanModifier()

			req := planmodifier.StringRequest{
				StateValue: tc.stateValue,
				PlanValue:  tc.planValue,
			}
			resp := &planmodifier.StringResponse{
				PlanValue: tc.planValue,
			}

			modifier.PlanModifyString(context.Background(), req, resp)

			if tc.planValue.IsNull() {
				if !resp.PlanValue.IsNull() {
					t.Errorf("expected null plan value, got %q", resp.PlanValue.ValueString())
				}
			} else if resp.PlanValue.ValueString() != tc.expectedPlan {
				t.Errorf("expected plan value %q, got %q", tc.expectedPlan, resp.PlanValue.ValueString())
			}
		})
	}
}

func TestComputedStatePlanModifier_Description(t *testing.T) {
	modifier := computedStatePlanModifier()

	description := modifier.Description(context.Background())

	if description == "" {
		t.Error("expected non-empty description")
	}
	expected := "Preserves state value when desired_state isn't changing."
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
	expected := "Preserves `state` value when `desired_state` isn't effectively changing, otherwise marks as unknown."
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
			"desired_state": schema.StringAttribute{},
			"state":         schema.StringAttribute{},
		},
	}
}
