package resources

import (
	"context"

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
	// If state is null/unknown or plan is null/unknown, don't modify
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() ||
		req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	// If normalized values are equal, use state value to prevent spurious diffs
	stateNormalized := normalizeDesiredState(req.StateValue.ValueString())
	planNormalized := normalizeDesiredState(req.PlanValue.ValueString())

	if stateNormalized == planNormalized {
		resp.PlanValue = types.StringValue(req.StateValue.ValueString())
	}
}
