package types

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Ensure interfaces are implemented.
var (
	_ basetypes.StringTypable  = CaseInsensitiveStringType{}
	_ basetypes.StringValuable = CaseInsensitiveStringValue{}
)

// CaseInsensitiveStringType is a custom type for case-insensitive string comparison.
type CaseInsensitiveStringType struct {
	basetypes.StringType
}

// Equal returns true if the given type is equivalent.
func (t CaseInsensitiveStringType) Equal(o attr.Type) bool {
	other, ok := o.(CaseInsensitiveStringType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

// String returns a human-readable string of the type.
func (t CaseInsensitiveStringType) String() string {
	return "CaseInsensitiveStringType"
}

// ValueType returns the value type.
func (t CaseInsensitiveStringType) ValueType(ctx context.Context) attr.Value {
	return CaseInsensitiveStringValue{}
}

// ValueFromString converts a StringValue to a CaseInsensitiveStringValue.
func (t CaseInsensitiveStringType) ValueFromString(ctx context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return CaseInsensitiveStringValue{StringValue: in}, nil
}

// ValueFromTerraform converts a tftypes.Value to a CaseInsensitiveStringValue.
func (t CaseInsensitiveStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}

	return stringValuable.(CaseInsensitiveStringValue), nil
}

// CaseInsensitiveStringValue is a custom string value that compares case-insensitively.
type CaseInsensitiveStringValue struct {
	basetypes.StringValue
}

// Type returns the type of this value.
func (v CaseInsensitiveStringValue) Type(ctx context.Context) attr.Type {
	return CaseInsensitiveStringType{}
}

// Equal returns true if the values are equal (including null/unknown state).
func (v CaseInsensitiveStringValue) Equal(o attr.Value) bool {
	other, ok := o.(CaseInsensitiveStringValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals compares two strings for case-insensitive equality.
func (v CaseInsensitiveStringValue) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, d := newValuable.ToStringValue(ctx)
	diags.Append(d...)
	if diags.HasError() {
		return false, diags
	}

	// Handle null/unknown cases
	if v.IsNull() && newValue.IsNull() {
		return true, diags
	}
	if v.IsNull() || newValue.IsNull() {
		return false, diags
	}
	if v.IsUnknown() || newValue.IsUnknown() {
		return false, diags
	}

	// Case-insensitive comparison
	return strings.EqualFold(v.ValueString(), newValue.ValueString()), diags
}

// NewCaseInsensitiveStringValue creates a new CaseInsensitiveStringValue with the given string.
func NewCaseInsensitiveStringValue(value string) CaseInsensitiveStringValue {
	return CaseInsensitiveStringValue{StringValue: basetypes.NewStringValue(value)}
}

// NewCaseInsensitiveStringNull creates a new null CaseInsensitiveStringValue.
func NewCaseInsensitiveStringNull() CaseInsensitiveStringValue {
	return CaseInsensitiveStringValue{StringValue: basetypes.NewStringNull()}
}

// NewCaseInsensitiveStringUnknown creates a new unknown CaseInsensitiveStringValue.
func NewCaseInsensitiveStringUnknown() CaseInsensitiveStringValue {
	return CaseInsensitiveStringValue{StringValue: basetypes.NewStringUnknown()}
}

// NewCaseInsensitiveStringPointerValue creates a CaseInsensitiveStringValue from a *string.
func NewCaseInsensitiveStringPointerValue(value *string) CaseInsensitiveStringValue {
	return CaseInsensitiveStringValue{StringValue: basetypes.NewStringPointerValue(value)}
}

// Ensure caseInsensitiveStringDefault implements defaults.String.
var _ defaults.String = caseInsensitiveStringDefault{}

// caseInsensitiveStringDefault is a custom default implementation for CaseInsensitiveStringValue.
// This is necessary because stringdefault.StaticString returns a basetypes.StringValue,
// but when a CustomType is specified in the schema, the framework requires the default
// to return the custom value type (CaseInsensitiveStringValue).
type caseInsensitiveStringDefault struct {
	value string
}

// Description returns a plain text description of the default's behavior.
func (d caseInsensitiveStringDefault) Description(ctx context.Context) string {
	return fmt.Sprintf("value defaults to %q", d.value)
}

// MarkdownDescription returns a markdown formatted description of the default's behavior.
func (d caseInsensitiveStringDefault) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("value defaults to `%s`", d.value)
}

// DefaultString sets the default CaseInsensitiveStringValue for the attribute.
// Note: The interface requires types.String for PlanValue, but the framework will
// convert this through the custom type's ValueFromTerraform method when populating
// the model. However, for semantic equality checking during plan diffs, we need to
// ensure the underlying value is properly handled.
func (d caseInsensitiveStringDefault) DefaultString(ctx context.Context, req defaults.StringRequest, resp *defaults.StringResponse) {
	// The default interface requires types.String, which the framework will convert
	// to CaseInsensitiveStringValue via ValueFromTerraform when populating models.
	resp.PlanValue = NewCaseInsensitiveStringValue(d.value).StringValue
}

// CaseInsensitiveStringDefault returns a default value for a CaseInsensitiveStringValue attribute.
// Use this instead of stringdefault.StaticString when the attribute has CustomType set to
// CaseInsensitiveStringType, as the framework requires the default to return the custom value type.
func CaseInsensitiveStringDefault(value string) defaults.String {
	return caseInsensitiveStringDefault{value: value}
}
