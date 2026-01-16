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
