package types

import (
	"context"
	"fmt"

	truenas "github.com/deevus/truenas-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Ensure interfaces are implemented.
var (
	_ basetypes.StringTypable  = SizeStringType{}
	_ basetypes.StringValuable = SizeStringValue{}
)

// SizeStringType is a custom type for size strings that compares by byte value.
// This allows "2T", "2TB", and "2000000000000" to be considered equal.
type SizeStringType struct {
	basetypes.StringType
}

// Equal returns true if the given type is equivalent.
func (t SizeStringType) Equal(o attr.Type) bool {
	other, ok := o.(SizeStringType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

// String returns a human-readable string of the type.
func (t SizeStringType) String() string {
	return "SizeStringType"
}

// ValueType returns the value type.
func (t SizeStringType) ValueType(ctx context.Context) attr.Value {
	return SizeStringValue{}
}

// ValueFromString converts a StringValue to a SizeStringValue.
func (t SizeStringType) ValueFromString(ctx context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return SizeStringValue{StringValue: in}, nil
}

// ValueFromTerraform converts a tftypes.Value to a SizeStringValue.
func (t SizeStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
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

	return stringValuable.(SizeStringValue), nil
}

// SizeStringValue is a custom string value that compares sizes by byte value.
type SizeStringValue struct {
	basetypes.StringValue
}

// Type returns the type of this value.
func (v SizeStringValue) Type(ctx context.Context) attr.Type {
	return SizeStringType{}
}

// Equal returns true if the values are equal (including null/unknown state).
func (v SizeStringValue) Equal(o attr.Value) bool {
	other, ok := o.(SizeStringValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals compares two size strings by their byte values.
// This allows "2T", "2TB", and "2000000000000" to be considered equal.
func (v SizeStringValue) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
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

	// Parse both values to bytes and compare
	oldBytes, err := truenas.ParseSize(v.ValueString())
	if err != nil {
		diags.AddError("Invalid Size", fmt.Sprintf("Unable to parse size %q: %s", v.ValueString(), err))
		return false, diags
	}

	newBytes, err := truenas.ParseSize(newValue.ValueString())
	if err != nil {
		diags.AddError("Invalid Size", fmt.Sprintf("Unable to parse size %q: %s", newValue.ValueString(), err))
		return false, diags
	}

	return oldBytes == newBytes, diags
}

// NewSizeStringValue creates a new SizeStringValue with the given string.
func NewSizeStringValue(value string) SizeStringValue {
	return SizeStringValue{StringValue: basetypes.NewStringValue(value)}
}

// NewSizeStringNull creates a new null SizeStringValue.
func NewSizeStringNull() SizeStringValue {
	return SizeStringValue{StringValue: basetypes.NewStringNull()}
}

// NewSizeStringUnknown creates a new unknown SizeStringValue.
func NewSizeStringUnknown() SizeStringValue {
	return SizeStringValue{StringValue: basetypes.NewStringUnknown()}
}
