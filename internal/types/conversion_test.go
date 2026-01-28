package types_test

import (
	"context"
	"testing"

	customtypes "github.com/deevus/terraform-provider-truenas/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type TestModel struct {
	DesiredState customtypes.CaseInsensitiveStringValue `tfsdk:"desired_state"`
}

func TestCaseInsensitiveStringType_StateGetConversion(t *testing.T) {
	ctx := context.Background()

	// Create schema with custom type and default
	sch := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"desired_state": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				CustomType: customtypes.CaseInsensitiveStringType{},
				Default:    customtypes.CaseInsensitiveStringDefault("RUNNING"),
			},
		},
	}

	// Create a state with a raw string value (simulating existing state)
	stateValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"desired_state": tftypes.String,
		},
	}, map[string]tftypes.Value{
		"desired_state": tftypes.NewValue(tftypes.String, "RUNNING"),
	})

	state := tfsdk.State{
		Schema: sch,
		Raw:    stateValue,
	}

	// Try to get the model
	var model TestModel
	diags := state.Get(ctx, &model)

	if diags.HasError() {
		for _, d := range diags {
			t.Errorf("%s: %s", d.Summary(), d.Detail())
		}
		return
	}

	if model.DesiredState.ValueString() != "RUNNING" {
		t.Errorf("expected 'RUNNING', got %q", model.DesiredState.ValueString())
	}
}

func TestCaseInsensitiveStringType_DefaultValueConversion(t *testing.T) {
	ctx := context.Background()

	// Create schema with custom type, Computed, and Default (same as app.go)
	sch := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"desired_state": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				CustomType: customtypes.CaseInsensitiveStringType{},
				Default:    customtypes.CaseInsensitiveStringDefault("RUNNING"),
			},
		},
	}

	// Create a state with NULL value to trigger the default
	stateValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"desired_state": tftypes.String,
		},
	}, map[string]tftypes.Value{
		"desired_state": tftypes.NewValue(tftypes.String, nil), // NULL value
	})

	state := tfsdk.State{
		Schema: sch,
		Raw:    stateValue,
	}

	// Try to get the model
	var model TestModel
	diags := state.Get(ctx, &model)

	if diags.HasError() {
		for _, d := range diags {
			t.Errorf("%s: %s", d.Summary(), d.Detail())
		}
		return
	}

	// The state has NULL, so model should be NULL (default only applies during planning)
	if !model.DesiredState.IsNull() {
		t.Errorf("expected null, got %q", model.DesiredState.ValueString())
	}
}

func TestCaseInsensitiveStringType_StateSetConversion(t *testing.T) {
	ctx := context.Background()

	// Create schema with custom type and default
	sch := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"desired_state": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				CustomType: customtypes.CaseInsensitiveStringType{},
				Default:    customtypes.CaseInsensitiveStringDefault("RUNNING"),
			},
		},
	}

	state := tfsdk.State{
		Schema: sch,
	}

	// Create model with custom value
	model := TestModel{
		DesiredState: customtypes.NewCaseInsensitiveStringValue("stopped"),
	}

	// Try to set the state
	diags := state.Set(ctx, model)

	if diags.HasError() {
		for _, d := range diags {
			t.Errorf("%s: %s", d.Summary(), d.Detail())
		}
		return
	}

	// Read it back
	var readModel TestModel
	diags = state.Get(ctx, &readModel)

	if diags.HasError() {
		for _, d := range diags {
			t.Errorf("Get: %s: %s", d.Summary(), d.Detail())
		}
		return
	}

	if readModel.DesiredState.ValueString() != "stopped" {
		t.Errorf("expected 'stopped', got %q", readModel.DesiredState.ValueString())
	}
}
