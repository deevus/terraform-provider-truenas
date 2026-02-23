package resources

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// -- Shared identity helpers --

// poolDatasetFullName builds the full dataset/zvol name from pool+path or parent+path.
// Returns "" if the configuration is invalid.
//
// Modes:
//   - pool + path: "tank" + "vms/disk0" -> "tank/vms/disk0"
//   - parent + path: "tank/vms" + "disk0" -> "tank/vms/disk0"
func poolDatasetFullName(pool, path, parent, name types.String) string {
	hasPool := !pool.IsNull() && !pool.IsUnknown() && pool.ValueString() != ""
	hasPath := !path.IsNull() && !path.IsUnknown() && path.ValueString() != ""

	if hasPool && hasPath {
		return fmt.Sprintf("%s/%s", pool.ValueString(), path.ValueString())
	}

	hasParent := !parent.IsNull() && !parent.IsUnknown() && parent.ValueString() != ""
	hasName := !name.IsNull() && !name.IsUnknown() && name.ValueString() != ""

	if hasParent {
		if hasPath {
			return fmt.Sprintf("%s/%s", parent.ValueString(), path.ValueString())
		}
		if hasName {
			return fmt.Sprintf("%s/%s", parent.ValueString(), name.ValueString())
		}
	}

	return ""
}

// poolDatasetIDToParts splits a dataset ID like "tank/vms/disk0" into pool="tank", path="vms/disk0".
func poolDatasetIDToParts(id string) (pool, path string) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return id, ""
}

// -- Shared schema attributes --

// poolDatasetIdentitySchema returns the common pool/path/parent identity attributes
// shared by both dataset and zvol resources.
func poolDatasetIdentitySchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "Dataset identifier (pool/path).",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"pool": schema.StringAttribute{
			Description: "Pool name. Use with 'path' attribute.",
			Optional:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"path": schema.StringAttribute{
			Description: "Path within the pool (e.g., 'vms/disk0').",
			Optional:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"parent": schema.StringAttribute{
			Description: "Parent dataset ID (e.g., 'tank/vms'). Use with 'path' attribute.",
			Optional:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
	}
}
