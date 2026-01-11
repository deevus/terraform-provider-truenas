package types

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"gopkg.in/yaml.v3"
)

// Ensure interfaces are implemented.
var (
	_ basetypes.StringTypable  = YAMLStringType{}
	_ basetypes.StringValuable = YAMLStringValue{}
)

// YAMLStringType is a custom type for YAML strings with semantic equality.
type YAMLStringType struct {
	basetypes.StringType
}

// Equal returns true if the given type is equivalent.
func (t YAMLStringType) Equal(o attr.Type) bool {
	other, ok := o.(YAMLStringType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

// String returns a human-readable string of the type.
func (t YAMLStringType) String() string {
	return "YAMLStringType"
}

// ValueType returns the value type.
func (t YAMLStringType) ValueType(ctx context.Context) attr.Value {
	return YAMLStringValue{}
}

// ValueFromString converts a StringValue to a YAMLStringValue.
func (t YAMLStringType) ValueFromString(ctx context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return YAMLStringValue{StringValue: in}, nil
}

// ValueFromTerraform converts a tftypes.Value to a YAMLStringValue.
func (t YAMLStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
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

	return stringValuable, nil
}

// YAMLStringValue is a custom string value that compares YAML semantically.
type YAMLStringValue struct {
	basetypes.StringValue
}

// Type returns the type of this value.
func (v YAMLStringValue) Type(ctx context.Context) attr.Type {
	return YAMLStringType{}
}

// Equal returns true if the values are equal (including null/unknown state).
func (v YAMLStringValue) Equal(o attr.Value) bool {
	other, ok := o.(YAMLStringValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals compares two YAML strings for semantic equality.
// If both are valid YAML, they're compared as parsed structures.
// If either is invalid YAML, falls back to string comparison.
func (v YAMLStringValue) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
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

	// Parse both as YAML
	var oldParsed, newParsed any
	if err := yaml.Unmarshal([]byte(v.ValueString()), &oldParsed); err != nil {
		// Invalid YAML, fall back to string comparison
		return v.ValueString() == newValue.ValueString(), diags
	}
	if err := yaml.Unmarshal([]byte(newValue.ValueString()), &newParsed); err != nil {
		// Invalid YAML, fall back to string comparison
		return v.ValueString() == newValue.ValueString(), diags
	}

	// Compare parsed structures
	return reflect.DeepEqual(oldParsed, newParsed), diags
}

// NewYAMLStringValue creates a new YAMLStringValue with the given string.
func NewYAMLStringValue(value string) YAMLStringValue {
	return YAMLStringValue{StringValue: basetypes.NewStringValue(value)}
}

// NewYAMLStringNull creates a new null YAMLStringValue.
func NewYAMLStringNull() YAMLStringValue {
	return YAMLStringValue{StringValue: basetypes.NewStringNull()}
}

// NewYAMLStringUnknown creates a new unknown YAMLStringValue.
func NewYAMLStringUnknown() YAMLStringValue {
	return YAMLStringValue{StringValue: basetypes.NewStringUnknown()}
}

// NewYAMLStringPointerValue creates a YAMLStringValue from a *string.
func NewYAMLStringPointerValue(value *string) YAMLStringValue {
	return YAMLStringValue{StringValue: basetypes.NewStringPointerValue(value)}
}
