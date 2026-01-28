package types

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type TestModelDebug struct {
	DesiredState CaseInsensitiveStringValue `tfsdk:"desired_state"`
}

func TestReflectStructField(t *testing.T) {
	model := TestModelDebug{}
	
	v := reflect.ValueOf(&model).Elem()
	field := v.FieldByName("DesiredState")
	
	t.Logf("Field type: %s", field.Type())
	t.Logf("Field type implements attr.Value: %v", 
		field.Type().Implements(reflect.TypeOf((*interface{ Type(any) any })(nil)).Elem()))
	
	// Check if CaseInsensitiveStringValue is the type
	if field.Type().String() != "types.CaseInsensitiveStringValue" {
		t.Errorf("expected field type 'types.CaseInsensitiveStringValue', got %q", field.Type().String())
	}
	
	// Create a value and check its type
	cisv := NewCaseInsensitiveStringValue("test")
	sv := basetypes.StringValue{}
	
	t.Logf("CaseInsensitiveStringValue: %s", reflect.TypeOf(cisv))
	t.Logf("basetypes.StringValue: %s", reflect.TypeOf(sv))
	
	// Check equality
	if reflect.TypeOf(cisv) == reflect.TypeOf(sv) {
		t.Error("PROBLEM: CaseInsensitiveStringValue type equals basetypes.StringValue type!")
	}
}
