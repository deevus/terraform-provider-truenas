package resources

import (
	"context"

	customtypes "github.com/deevus/terraform-provider-truenas/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// computedStatePlanModifier returns a plan modifier for the computed `state` attribute.
// It preserves state when desired_state isn't effectively changing, otherwise marks unknown.
func computedStatePlanModifier() planmodifier.String {
	return &computedStateModifier{}
}

type computedStateModifier struct{}

func (m *computedStateModifier) Description(ctx context.Context) string {
	return "Predicts state value based on desired_state and drift detection."
}

func (m *computedStateModifier) MarkdownDescription(ctx context.Context) string {
	return "Predicts `state` value: if drift detected (state != desired_state), plans reconciliation to desired_state; otherwise preserves current state."
}

func (m *computedStateModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// On resource destruction, state is null
	if req.StateValue.IsNull() {
		return
	}

	// Get desired_state from both state and plan
	// Note: desired_state uses CaseInsensitiveStringType, so we must use the matching value type
	var stateDesired, planDesired customtypes.CaseInsensitiveStringValue
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("desired_state"), &stateDesired)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("desired_state"), &planDesired)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Normalize for comparison
	stateDesiredNorm := normalizeDesiredState(stateDesired.ValueString())
	planDesiredNorm := normalizeDesiredState(planDesired.ValueString())
	currentState := req.StateValue.ValueString()

	// If desired_state is changing, leave as unknown
	if stateDesiredNorm != planDesiredNorm {
		return
	}

	// Check for drift: current state doesn't match desired state
	// In this case, reconciliation will occur during Apply
	if currentState != planDesiredNorm {
		// Special case: CRASHED is "stopped enough" when desired is STOPPED
		if currentState == AppStateCrashed && planDesiredNorm == AppStateStopped {
			resp.PlanValue = req.StateValue
			return
		}
		// Predict that state will change to match desired_state after reconciliation
		resp.PlanValue = types.StringValue(planDesiredNorm)
		return
	}

	// No drift, no desired_state change - preserve current state value
	resp.PlanValue = req.StateValue
}
