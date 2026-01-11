package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestYAMLStringValue_SemanticEquals_IdenticalStrings(t *testing.T) {
	ctx := context.Background()
	yaml := "services:\n  web:\n    image: nginx"

	v1 := NewYAMLStringValue(yaml)
	v2 := NewYAMLStringValue(yaml)

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if !equal {
		t.Error("expected identical YAML strings to be semantically equal")
	}
}

func TestYAMLStringValue_SemanticEquals_DifferentKeyOrder(t *testing.T) {
	ctx := context.Background()
	yaml1 := "services:\n  web:\n    image: nginx\n    ports:\n      - 80:80"
	yaml2 := "services:\n  web:\n    ports:\n      - 80:80\n    image: nginx"

	v1 := NewYAMLStringValue(yaml1)
	v2 := NewYAMLStringValue(yaml2)

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if !equal {
		t.Error("expected YAML with different key order to be semantically equal")
	}
}

func TestYAMLStringValue_SemanticEquals_DifferentWhitespace(t *testing.T) {
	ctx := context.Background()
	yaml1 := "services:\n  web:\n    image: nginx"
	yaml2 := "services:\n  web:\n    image:   nginx\n"

	v1 := NewYAMLStringValue(yaml1)
	v2 := NewYAMLStringValue(yaml2)

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if !equal {
		t.Error("expected YAML with different whitespace to be semantically equal")
	}
}

func TestYAMLStringValue_SemanticEquals_DifferentValues(t *testing.T) {
	ctx := context.Background()
	yaml1 := "services:\n  web:\n    image: nginx"
	yaml2 := "services:\n  web:\n    image: apache"

	v1 := NewYAMLStringValue(yaml1)
	v2 := NewYAMLStringValue(yaml2)

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if equal {
		t.Error("expected YAML with different values to NOT be semantically equal")
	}
}

func TestYAMLStringValue_SemanticEquals_NullValues(t *testing.T) {
	ctx := context.Background()

	v1 := NewYAMLStringNull()
	v2 := NewYAMLStringNull()

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if !equal {
		t.Error("expected null values to be semantically equal")
	}
}

func TestYAMLStringValue_SemanticEquals_NullVsNonNull(t *testing.T) {
	ctx := context.Background()

	v1 := NewYAMLStringNull()
	v2 := NewYAMLStringValue("services:\n  web:\n    image: nginx")

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if equal {
		t.Error("expected null and non-null to NOT be semantically equal")
	}
}

func TestYAMLStringValue_SemanticEquals_InvalidYAML(t *testing.T) {
	ctx := context.Background()
	validYAML := "services:\n  web:\n    image: nginx"
	invalidYAML := "services:\n  web:\n    image: nginx\n  invalid: [unclosed"

	v1 := NewYAMLStringValue(validYAML)
	v2 := NewYAMLStringValue(invalidYAML)

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	// Invalid YAML should fall back to string comparison
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if equal {
		t.Error("expected invalid YAML to fall back to string comparison (not equal)")
	}
}

func TestYAMLStringValue_Type(t *testing.T) {
	v := NewYAMLStringValue("test")
	typ := v.Type(context.Background())

	if _, ok := typ.(YAMLStringType); !ok {
		t.Errorf("expected YAMLStringType, got %T", typ)
	}
}

func TestYAMLStringType_ValueFromString(t *testing.T) {
	ctx := context.Background()
	typ := YAMLStringType{}

	stringValue := basetypes.NewStringValue("services:\n  web:\n    image: nginx")
	result, diags := typ.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}

	yamlValue, ok := result.(YAMLStringValue)
	if !ok {
		t.Fatalf("expected YAMLStringValue, got %T", result)
	}

	if yamlValue.ValueString() != "services:\n  web:\n    image: nginx" {
		t.Errorf("unexpected value: %q", yamlValue.ValueString())
	}
}
