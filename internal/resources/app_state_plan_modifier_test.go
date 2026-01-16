package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
