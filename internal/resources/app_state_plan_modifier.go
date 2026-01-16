package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// caseInsensitiveStatePlanModifier returns a plan modifier that treats
// state values as equal regardless of case (e.g., "running" == "RUNNING").
func caseInsensitiveStatePlanModifier() planmodifier.String {
	return &caseInsensitiveStateModifier{}
}

type caseInsensitiveStateModifier struct{}

func (m *caseInsensitiveStateModifier) Description(ctx context.Context) string {
	return "Treats state values as equal regardless of case."
}

func (m *caseInsensitiveStateModifier) MarkdownDescription(ctx context.Context) string {
	return "Treats state values as equal regardless of case (e.g., `running` == `RUNNING`)."
}

func (m *caseInsensitiveStateModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If plan is null/unknown, don't modify
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	// Always normalize the plan value to uppercase to ensure consistency
	// between what Terraform plans and what the provider returns after apply.
	// This prevents "inconsistent result after apply" errors when users
	// specify lowercase values like "stopped" but the API returns "STOPPED".
	planNormalized := normalizeDesiredState(req.PlanValue.ValueString())
	resp.PlanValue = types.StringValue(planNormalized)
}

// computedStatePlanModifier returns a plan modifier for the computed `state` attribute.
// It preserves state when desired_state isn't effectively changing, otherwise marks unknown.
func computedStatePlanModifier() planmodifier.String {
	return &computedStateModifier{}
}

type computedStateModifier struct{}

func (m *computedStateModifier) Description(ctx context.Context) string {
	return "Preserves state value when desired_state isn't changing."
}

func (m *computedStateModifier) MarkdownDescription(ctx context.Context) string {
	return "Preserves `state` value when `desired_state` isn't effectively changing, otherwise marks as unknown."
}

func (m *computedStateModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// On resource destruction, state is null
	if req.StateValue.IsNull() {
		return
	}

	// Get desired_state from both state and plan
	var stateDesired, planDesired types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("desired_state"), &stateDesired)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("desired_state"), &planDesired)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Normalize both for comparison
	stateDesiredNorm := normalizeDesiredState(stateDesired.ValueString())
	planDesiredNorm := normalizeDesiredState(planDesired.ValueString())

	// If desired_state is effectively the same, preserve current state value
	if stateDesiredNorm == planDesiredNorm {
		resp.PlanValue = req.StateValue
	}
	// Otherwise leave as unknown (the default for computed attributes)
}
