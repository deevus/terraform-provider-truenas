package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCaseInsensitiveStatePlanModifier_PlanModifyString(t *testing.T) {
	tests := []struct {
		name          string
		stateValue    string
		planValue     string
		expectedPlan  string
		expectUnknown bool
	}{
		{
			name:         "lowercase to uppercase - no change needed",
			stateValue:   "RUNNING",
			planValue:    "running",
			expectedPlan: "RUNNING", // Should preserve state value
		},
		{
			name:         "same case - no change",
			stateValue:   "STOPPED",
			planValue:    "STOPPED",
			expectedPlan: "STOPPED",
		},
		{
			name:         "actual state change",
			stateValue:   "RUNNING",
			planValue:    "stopped",
			expectedPlan: "stopped", // Different state, keep plan value
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			modifier := caseInsensitiveStatePlanModifier()

			req := planmodifier.StringRequest{
				StateValue: types.StringValue(tc.stateValue),
				PlanValue:  types.StringValue(tc.planValue),
			}
			resp := &planmodifier.StringResponse{
				PlanValue: types.StringValue(tc.planValue),
			}

			modifier.PlanModifyString(context.Background(), req, resp)

			if resp.PlanValue.ValueString() != tc.expectedPlan {
				t.Errorf("expected plan value %q, got %q", tc.expectedPlan, resp.PlanValue.ValueString())
			}
		})
	}
}
