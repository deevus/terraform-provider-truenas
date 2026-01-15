# Desired State Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `desired_state` attribute to `truenas_app` resource for declarative app lifecycle control (start/stop).

**Architecture:** Extend the existing `AppResource` with two new schema attributes (`desired_state`, `state_timeout`). Add helper functions for state normalization, polling, and waiting. Modify Create/Update lifecycle methods to reconcile desired vs actual state by calling `app.start`/`app.stop` APIs.

**Tech Stack:** Go, Terraform Plugin Framework, TrueNAS midclt API (`app.start`, `app.stop`, `app.query`)

---

## Task 1: Add State Constants and Normalization Helper

**Files:**
- Create: `internal/resources/app_state.go`
- Test: `internal/resources/app_state_test.go`

**Step 1: Write the failing test for normalizeDesiredState**

```go
// internal/resources/app_state_test.go
package resources

import "testing"

func TestNormalizeDesiredState(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"running", "RUNNING"},
		{"RUNNING", "RUNNING"},
		{"Running", "RUNNING"},
		{"stopped", "STOPPED"},
		{"STOPPED", "STOPPED"},
		{"  running  ", "RUNNING"},
	}

	for _, tc := range tests {
		got := normalizeDesiredState(tc.input)
		if got != tc.expected {
			t.Errorf("normalizeDesiredState(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/resources -run TestNormalizeDesiredState -v`
Expected: FAIL with "undefined: normalizeDesiredState"

**Step 3: Write minimal implementation**

```go
// internal/resources/app_state.go
package resources

import "strings"

// App state constants matching TrueNAS API values.
const (
	AppStateRunning   = "RUNNING"
	AppStateStopped   = "STOPPED"
	AppStateDeploying = "DEPLOYING"
	AppStateStarting  = "STARTING"
	AppStateStopping  = "STOPPING"
	AppStateCrashed   = "CRASHED"
)

// normalizeDesiredState converts user input to uppercase for comparison.
func normalizeDesiredState(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resources -run TestNormalizeDesiredState -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/app_state.go internal/resources/app_state_test.go
git commit -m "feat(app): add state constants and normalization helper"
```

---

## Task 2: Add isStableState and isValidDesiredState Helpers

**Files:**
- Modify: `internal/resources/app_state.go`
- Modify: `internal/resources/app_state_test.go`

**Step 1: Write the failing tests**

```go
// Add to internal/resources/app_state_test.go

func TestIsStableState(t *testing.T) {
	tests := []struct {
		state    string
		expected bool
	}{
		{AppStateRunning, true},
		{AppStateStopped, true},
		{AppStateCrashed, true},
		{AppStateDeploying, false},
		{AppStateStarting, false},
		{AppStateStopping, false},
		{"UNKNOWN", false},
	}

	for _, tc := range tests {
		got := isStableState(tc.state)
		if got != tc.expected {
			t.Errorf("isStableState(%q) = %v, want %v", tc.state, got, tc.expected)
		}
	}
}

func TestIsValidDesiredState(t *testing.T) {
	tests := []struct {
		state    string
		expected bool
	}{
		{"running", true},
		{"RUNNING", true},
		{"stopped", true},
		{"STOPPED", true},
		{"deploying", false},
		{"crashed", false},
		{"paused", false},
	}

	for _, tc := range tests {
		got := isValidDesiredState(tc.state)
		if got != tc.expected {
			t.Errorf("isValidDesiredState(%q) = %v, want %v", tc.state, got, tc.expected)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/resources -run "TestIsStableState|TestIsValidDesiredState" -v`
Expected: FAIL with "undefined: isStableState" and "undefined: isValidDesiredState"

**Step 3: Write minimal implementation**

```go
// Add to internal/resources/app_state.go

// isStableState returns true if the state is stable (not transitional).
func isStableState(state string) bool {
	switch state {
	case AppStateRunning, AppStateStopped, AppStateCrashed:
		return true
	default:
		return false
	}
}

// isValidDesiredState returns true if the state is valid for desired_state.
func isValidDesiredState(state string) bool {
	normalized := normalizeDesiredState(state)
	return normalized == AppStateRunning || normalized == AppStateStopped
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/resources -run "TestIsStableState|TestIsValidDesiredState" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/app_state.go internal/resources/app_state_test.go
git commit -m "feat(app): add isStableState and isValidDesiredState helpers"
```

---

## Task 3: Add Case-Insensitive Plan Modifier

**Files:**
- Create: `internal/resources/app_state_plan_modifier.go`
- Create: `internal/resources/app_state_plan_modifier_test.go`

**Step 1: Write the failing test**

```go
// internal/resources/app_state_plan_modifier_test.go
package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCaseInsensitiveStatePlanModifier_PlanModifyString(t *testing.T) {
	tests := []struct {
		name          string
		stateValue    string
		planValue     string
		expectedPlan  string
		expectUnknown bool
	}{
		{
			name:         "lowercase to uppercase - no change needed",
			stateValue:   "RUNNING",
			planValue:    "running",
			expectedPlan: "RUNNING", // Should preserve state value
		},
		{
			name:         "same case - no change",
			stateValue:   "STOPPED",
			planValue:    "STOPPED",
			expectedPlan: "STOPPED",
		},
		{
			name:         "actual state change",
			stateValue:   "RUNNING",
			planValue:    "stopped",
			expectedPlan: "stopped", // Different state, keep plan value
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			modifier := caseInsensitiveStatePlanModifier()

			req := planmodifier.StringRequest{
				StateValue: types.StringValue(tc.stateValue),
				PlanValue:  types.StringValue(tc.planValue),
			}
			resp := &planmodifier.StringResponse{
				PlanValue: types.StringValue(tc.planValue),
			}

			modifier.PlanModifyString(context.Background(), req, resp)

			if resp.PlanValue.ValueString() != tc.expectedPlan {
				t.Errorf("expected plan value %q, got %q", tc.expectedPlan, resp.PlanValue.ValueString())
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/resources -run TestCaseInsensitiveStatePlanModifier -v`
Expected: FAIL with "undefined: caseInsensitiveStatePlanModifier"

**Step 3: Write minimal implementation**

```go
// internal/resources/app_state_plan_modifier.go
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resources -run TestCaseInsensitiveStatePlanModifier -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/app_state_plan_modifier.go internal/resources/app_state_plan_modifier_test.go
git commit -m "feat(app): add case-insensitive state plan modifier"
```

---

## Task 4: Update AppResourceModel and Schema

**Files:**
- Modify: `internal/resources/app.go:30-36` (model)
- Modify: `internal/resources/app.go:61-94` (schema)
- Modify: `internal/resources/app_test.go`

**Step 1: Write the failing test for new schema attributes**

```go
// Add to internal/resources/app_test.go after existing schema tests

func TestAppResource_Schema_DesiredStateAttribute(t *testing.T) {
	r := NewAppResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify desired_state attribute exists and is optional
	desiredStateAttr, ok := resp.Schema.Attributes["desired_state"]
	if !ok {
		t.Fatal("expected 'desired_state' attribute in schema")
	}
	if !desiredStateAttr.IsOptional() {
		t.Error("expected 'desired_state' attribute to be optional")
	}

	// Verify state_timeout attribute exists and is optional
	stateTimeoutAttr, ok := resp.Schema.Attributes["state_timeout"]
	if !ok {
		t.Fatal("expected 'state_timeout' attribute in schema")
	}
	if !stateTimeoutAttr.IsOptional() {
		t.Error("expected 'state_timeout' attribute to be optional")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/resources -run TestAppResource_Schema_DesiredStateAttribute -v`
Expected: FAIL with "expected 'desired_state' attribute in schema"

**Step 3: Update the model and schema**

Update `internal/resources/app.go` model (around line 30):

```go
// AppResourceModel describes the resource data model.
// Simplified for custom Docker Compose apps only.
type AppResourceModel struct {
	ID            types.String                `tfsdk:"id"`
	Name          types.String                `tfsdk:"name"`
	CustomApp     types.Bool                  `tfsdk:"custom_app"`
	ComposeConfig customtypes.YAMLStringValue `tfsdk:"compose_config"`
	DesiredState  types.String                `tfsdk:"desired_state"`
	StateTimeout  types.Int64                 `tfsdk:"state_timeout"`
	State         types.String                `tfsdk:"state"`
}
```

Update schema (around line 61) - add after `compose_config` attribute:

```go
"desired_state": schema.StringAttribute{
	Description: "Desired application state: 'running' or 'stopped' (case-insensitive). Defaults to 'RUNNING'.",
	Optional:    true,
	Computed:    true,
	Default:     stringdefault.StaticString("RUNNING"),
	PlanModifiers: []planmodifier.String{
		caseInsensitiveStatePlanModifier(),
	},
	Validators: []validator.String{
		stringvalidator.Any(
			stringvalidator.OneOfCaseInsensitive("running", "stopped"),
		),
	},
},
"state_timeout": schema.Int64Attribute{
	Description: "Timeout in seconds to wait for state transitions. Defaults to 120. Range: 30-600.",
	Optional:    true,
	Computed:    true,
	Default:     int64default.StaticInt64(120),
	Validators: []validator.Int64{
		int64validator.Between(30, 600),
	},
},
```

Add imports at top of file:

```go
"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
"github.com/hashicorp/terraform-plugin-framework/schema/validator"
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resources -run TestAppResource_Schema_DesiredStateAttribute -v`
Expected: PASS

**Step 5: Update test helper to include new attributes**

Update `createAppResourceModelValue` in `app_test.go`:

```go
func createAppResourceModelValue(
	id, name interface{},
	customApp interface{},
	composeConfig interface{},
	desiredState interface{},
	stateTimeout interface{},
	state interface{},
) tftypes.Value {
	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":             tftypes.String,
			"name":           tftypes.String,
			"custom_app":     tftypes.Bool,
			"compose_config": tftypes.String,
			"desired_state":  tftypes.String,
			"state_timeout":  tftypes.Number,
			"state":          tftypes.String,
		},
	}, map[string]tftypes.Value{
		"id":             tftypes.NewValue(tftypes.String, id),
		"name":           tftypes.NewValue(tftypes.String, name),
		"custom_app":     tftypes.NewValue(tftypes.Bool, customApp),
		"compose_config": tftypes.NewValue(tftypes.String, composeConfig),
		"desired_state":  tftypes.NewValue(tftypes.String, desiredState),
		"state_timeout":  tftypes.NewValue(tftypes.Number, stateTimeout),
		"state":          tftypes.NewValue(tftypes.String, state),
	})
}
```

**Step 6: Update all existing test call sites**

Update all calls to `createAppResourceModelValue` throughout `app_test.go` to include the two new parameters. For most existing tests, use `"RUNNING"` for desired_state and `120` (as float64) for state_timeout.

Example update:
```go
// Before:
planValue := createAppResourceModelValue(nil, "myapp", true, nil, nil)

// After:
planValue := createAppResourceModelValue(nil, "myapp", true, nil, "RUNNING", float64(120), nil)
```

**Step 7: Run all app tests to verify they pass**

Run: `go test ./internal/resources -run TestAppResource -v`
Expected: PASS

**Step 8: Commit**

```bash
git add internal/resources/app.go internal/resources/app_test.go
git commit -m "feat(app): add desired_state and state_timeout schema attributes"
```

---

## Task 5: Add waitForStableState Helper

**Files:**
- Modify: `internal/resources/app_state.go`
- Modify: `internal/resources/app_state_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/resources/app_state_test.go

func TestWaitForStableState_AlreadyStable(t *testing.T) {
	callCount := 0
	queryFunc := func(ctx context.Context, name string) (string, error) {
		callCount++
		return AppStateRunning, nil
	}

	ctx := context.Background()
	state, err := waitForStableState(ctx, "myapp", 30*time.Second, queryFunc)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != AppStateRunning {
		t.Errorf("expected state %q, got %q", AppStateRunning, state)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestWaitForStableState_TransitionsToStable(t *testing.T) {
	callCount := 0
	queryFunc := func(ctx context.Context, name string) (string, error) {
		callCount++
		if callCount < 3 {
			return AppStateStarting, nil
		}
		return AppStateRunning, nil
	}

	ctx := context.Background()
	state, err := waitForStableState(ctx, "myapp", 30*time.Second, queryFunc)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != AppStateRunning {
		t.Errorf("expected state %q, got %q", AppStateRunning, state)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestWaitForStableState_Timeout(t *testing.T) {
	queryFunc := func(ctx context.Context, name string) (string, error) {
		return AppStateDeploying, nil // Never becomes stable
	}

	ctx := context.Background()
	_, err := waitForStableState(ctx, "myapp", 100*time.Millisecond, queryFunc)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "DEPLOYING") {
		t.Errorf("expected error to mention DEPLOYING state, got: %v", err)
	}
}
```

Add import: `"time"`

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/resources -run TestWaitForStableState -v`
Expected: FAIL with "undefined: waitForStableState"

**Step 3: Write minimal implementation**

```go
// Add to internal/resources/app_state.go

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// stateQueryFunc is a function type for querying app state.
type stateQueryFunc func(ctx context.Context, name string) (string, error)

// waitForStableState polls until the app reaches a stable state or timeout.
// Returns the final state or an error if timeout is reached.
func waitForStableState(ctx context.Context, name string, timeout time.Duration, queryState stateQueryFunc) (string, error) {
	const pollInterval = 5 * time.Second

	deadline := time.Now().Add(timeout)

	for {
		state, err := queryState(ctx, name)
		if err != nil {
			return "", fmt.Errorf("failed to query app state: %w", err)
		}

		if isStableState(state) {
			return state, nil
		}

		if time.Now().After(deadline) {
			return "", fmt.Errorf("timeout waiting for app state: app %q is stuck in %s state after %v", name, state, timeout)
		}

		// For testing, use shorter interval if timeout is very short
		sleepDuration := pollInterval
		if timeout < pollInterval {
			sleepDuration = timeout / 10
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(sleepDuration):
			// Continue polling
		}
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/resources -run TestWaitForStableState -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/app_state.go internal/resources/app_state_test.go
git commit -m "feat(app): add waitForStableState polling helper"
```

---

## Task 6: Add queryAppState Helper Method

**Files:**
- Modify: `internal/resources/app.go`
- Modify: `internal/resources/app_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/resources/app_test.go

func TestAppResource_queryAppState_Success(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{"name": "myapp", "state": "RUNNING"}]`), nil
			},
		},
	}

	state, err := r.queryAppState(context.Background(), "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != "RUNNING" {
		t.Errorf("expected state RUNNING, got %q", state)
	}
}

func TestAppResource_queryAppState_NotFound(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		},
	}

	_, err := r.queryAppState(context.Background(), "myapp")
	if err == nil {
		t.Fatal("expected error for app not found")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/resources -run TestAppResource_queryAppState -v`
Expected: FAIL with "r.queryAppState undefined"

**Step 3: Write minimal implementation**

```go
// Add to internal/resources/app.go

// queryAppState queries the TrueNAS API for the current state of an app.
func (r *AppResource) queryAppState(ctx context.Context, name string) (string, error) {
	filter := [][]any{{"name", "=", name}}
	result, err := r.client.Call(ctx, "app.query", filter)
	if err != nil {
		return "", err
	}

	var apps []appAPIResponse
	if err := json.Unmarshal(result, &apps); err != nil {
		return "", err
	}

	if len(apps) == 0 {
		return "", fmt.Errorf("app %q not found", name)
	}

	return apps[0].State, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/resources -run TestAppResource_queryAppState -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/app.go internal/resources/app_test.go
git commit -m "feat(app): add queryAppState helper method"
```

---

## Task 7: Add reconcileDesiredState Method

**Files:**
- Modify: `internal/resources/app.go`
- Modify: `internal/resources/app_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/resources/app_test.go

func TestAppResource_reconcileDesiredState_StartApp(t *testing.T) {
	var calledMethod string
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				calledMethod = method
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{"name": "myapp", "state": "RUNNING"}]`), nil
			},
		},
	}

	resp := &resource.UpdateResponse{}
	err := r.reconcileDesiredState(context.Background(), "myapp", "STOPPED", "RUNNING", 30*time.Second, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledMethod != "app.start" {
		t.Errorf("expected app.start to be called, got %q", calledMethod)
	}
}

func TestAppResource_reconcileDesiredState_StopApp(t *testing.T) {
	var calledMethod string
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				calledMethod = method
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{"name": "myapp", "state": "STOPPED"}]`), nil
			},
		},
	}

	resp := &resource.UpdateResponse{}
	err := r.reconcileDesiredState(context.Background(), "myapp", "RUNNING", "STOPPED", 30*time.Second, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledMethod != "app.stop" {
		t.Errorf("expected app.stop to be called, got %q", calledMethod)
	}
}

func TestAppResource_reconcileDesiredState_NoChangeNeeded(t *testing.T) {
	callCount := 0
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				callCount++
				return nil, nil
			},
		},
	}

	resp := &resource.UpdateResponse{}
	err := r.reconcileDesiredState(context.Background(), "myapp", "RUNNING", "RUNNING", 30*time.Second, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 0 {
		t.Errorf("expected no API calls when state matches, got %d calls", callCount)
	}
}
```

Add import: `"time"`

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/resources -run TestAppResource_reconcileDesiredState -v`
Expected: FAIL with "r.reconcileDesiredState undefined"

**Step 3: Write minimal implementation**

```go
// Add to internal/resources/app.go

import "time"

// reconcileDesiredState ensures the app is in the desired state.
// It calls app.start or app.stop as needed and waits for the state to stabilize.
// Returns an error if the reconciliation fails.
func (r *AppResource) reconcileDesiredState(
	ctx context.Context,
	name string,
	currentState string,
	desiredState string,
	timeout time.Duration,
	resp *resource.UpdateResponse,
) error {
	normalizedDesired := normalizeDesiredState(desiredState)

	// Check if reconciliation is needed
	if currentState == normalizedDesired {
		return nil
	}

	// Add warning about drift
	resp.Diagnostics.AddWarning(
		"App state was externally changed",
		fmt.Sprintf(
			"The app %q was found in state %s but desired_state is %s. "+
				"Reconciling to desired state. To stop this app intentionally, set desired_state = \"stopped\".",
			name, currentState, normalizedDesired,
		),
	)

	// Determine which action to take
	var method string
	if normalizedDesired == AppStateRunning {
		method = "app.start"
	} else {
		method = "app.stop"
	}

	// Call the API
	_, err := r.client.CallAndWait(ctx, method, name)
	if err != nil {
		return fmt.Errorf("failed to %s app %q: %w", method, name, err)
	}

	// Wait for stable state
	queryFunc := func(ctx context.Context, n string) (string, error) {
		return r.queryAppState(ctx, n)
	}

	finalState, err := waitForStableState(ctx, name, timeout, queryFunc)
	if err != nil {
		return err
	}

	// Verify we reached the desired state
	if finalState != normalizedDesired {
		return fmt.Errorf("app %q reached state %s instead of desired %s", name, finalState, normalizedDesired)
	}

	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/resources -run TestAppResource_reconcileDesiredState -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/app.go internal/resources/app_test.go
git commit -m "feat(app): add reconcileDesiredState method"
```

---

## Task 8: Update Create Method for desired_state

**Files:**
- Modify: `internal/resources/app.go:114-173` (Create method)
- Modify: `internal/resources/app_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/resources/app_test.go

func TestAppResource_Create_WithDesiredStateStopped(t *testing.T) {
	var methods []string
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// Return RUNNING initially, then STOPPED after stop is called
				if len(methods) == 1 {
					return json.RawMessage(`[{"name": "myapp", "state": "RUNNING"}]`), nil
				}
				return json.RawMessage(`[{"name": "myapp", "state": "STOPPED"}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Plan with desired_state = "stopped"
	planValue := createAppResourceModelValue(nil, "myapp", true, nil, "stopped", float64(120), nil)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify app.create was called, then app.stop
	if len(methods) < 2 {
		t.Fatalf("expected at least 2 API calls, got %d: %v", len(methods), methods)
	}
	if methods[0] != "app.create" {
		t.Errorf("expected first call to be app.create, got %q", methods[0])
	}
	if methods[1] != "app.stop" {
		t.Errorf("expected second call to be app.stop, got %q", methods[1])
	}

	// Verify final state
	var model AppResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.State.ValueString() != "STOPPED" {
		t.Errorf("expected final state STOPPED, got %q", model.State.ValueString())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/resources -run TestAppResource_Create_WithDesiredStateStopped -v`
Expected: FAIL (app.stop not called)

**Step 3: Update Create method**

Modify the Create method in `app.go` to handle desired_state after app creation:

```go
func (r *AppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build create params
	params := r.buildCreateParams(ctx, &data)
	appName := data.Name.ValueString()

	// Call the TrueNAS API (app.create returns a job, use CallAndWait)
	_, err := r.client.CallAndWait(ctx, "app.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create App",
			fmt.Sprintf("Unable to create app %q: %s", appName, err.Error()),
		)
		return
	}

	// Query the app to get current state
	filter := [][]any{{"name", "=", appName}}
	result, err := r.client.Call(ctx, "app.query", filter)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Query App After Create",
			fmt.Sprintf("Unable to query app %q after create: %s", appName, err.Error()),
		)
		return
	}

	var apps []appAPIResponse
	if err := json.Unmarshal(result, &apps); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse App Response",
			fmt.Sprintf("Unable to parse app query response: %s", err.Error()),
		)
		return
	}

	if len(apps) == 0 {
		resp.Diagnostics.AddError(
			"App Not Found After Create",
			fmt.Sprintf("App %q was not found after create", appName),
		)
		return
	}

	// Map response to model
	app := apps[0]
	data.ID = types.StringValue(app.Name)
	data.State = types.StringValue(app.State)

	// Handle desired_state - if user wants STOPPED but app started as RUNNING
	desiredState := data.DesiredState.ValueString()
	if desiredState == "" {
		desiredState = AppStateRunning
	}
	normalizedDesired := normalizeDesiredState(desiredState)

	if app.State != normalizedDesired {
		timeout := time.Duration(data.StateTimeout.ValueInt64()) * time.Second
		if timeout == 0 {
			timeout = 120 * time.Second
		}

		// For Create, we don't warn about drift - it's expected that we may need to stop
		var method string
		if normalizedDesired == AppStateRunning {
			method = "app.start"
		} else {
			method = "app.stop"
		}

		_, err := r.client.CallAndWait(ctx, method, appName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Set App State",
				fmt.Sprintf("Unable to %s app %q: %s", method, appName, err.Error()),
			)
			return
		}

		// Wait for stable state and query final state
		queryFunc := func(ctx context.Context, n string) (string, error) {
			return r.queryAppState(ctx, n)
		}

		finalState, err := waitForStableState(ctx, appName, timeout, queryFunc)
		if err != nil {
			resp.Diagnostics.AddError(
				"Timeout Waiting for App State",
				err.Error(),
			)
			return
		}

		data.State = types.StringValue(finalState)
	}

	// Ensure desired_state is normalized in state
	data.DesiredState = types.StringValue(normalizedDesired)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resources -run TestAppResource_Create_WithDesiredStateStopped -v`
Expected: PASS

**Step 5: Run all Create tests to ensure no regressions**

Run: `go test ./internal/resources -run "TestAppResource_Create" -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/resources/app.go internal/resources/app_test.go
git commit -m "feat(app): handle desired_state in Create lifecycle"
```

---

## Task 9: Update Update Method for State Reconciliation

**Files:**
- Modify: `internal/resources/app.go:255-317` (Update method)
- Modify: `internal/resources/app_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/resources/app_test.go

func TestAppResource_Update_ReconcileStateFromStoppedToRunning(t *testing.T) {
	var methods []string
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// After start, return RUNNING
				return json.RawMessage(`[{"name": "myapp", "state": "RUNNING"}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Current state: STOPPED, desired: RUNNING
	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "STOPPED")
	planValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), nil)

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify app.start was called
	foundStart := false
	for _, m := range methods {
		if m == "app.start" {
			foundStart = true
			break
		}
	}
	if !foundStart {
		t.Errorf("expected app.start to be called, got methods: %v", methods)
	}

	// Verify warning was added about drift
	hasWarning := false
	for _, d := range resp.Diagnostics.Warnings() {
		if strings.Contains(d.Summary(), "externally changed") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected drift warning to be added")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/resources -run TestAppResource_Update_ReconcileStateFromStoppedToRunning -v`
Expected: FAIL (app.start not called, no warning)

**Step 3: Update the Update method**

Modify the Update method in `app.go` to handle state reconciliation:

```go
func (r *AppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppResourceModel
	var stateData AppResourceModel

	// Read Terraform plan and state data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appName := data.Name.ValueString()

	// Check if compose_config changed - if so, call app.update
	composeChanged := !data.ComposeConfig.Equal(stateData.ComposeConfig)
	if composeChanged {
		updateParams := r.buildUpdateParams(ctx, &data)
		params := []any{appName, updateParams}

		_, err := r.client.CallAndWait(ctx, "app.update", params)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Update App",
				fmt.Sprintf("Unable to update app %q: %s", appName, err.Error()),
			)
			return
		}
	}

	// Query current state
	currentState, err := r.queryAppState(ctx, appName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Query App State",
			fmt.Sprintf("Unable to query app %q state: %s", appName, err.Error()),
		)
		return
	}

	// Handle desired_state reconciliation
	desiredState := data.DesiredState.ValueString()
	if desiredState == "" {
		desiredState = AppStateRunning
	}
	normalizedDesired := normalizeDesiredState(desiredState)

	timeout := time.Duration(data.StateTimeout.ValueInt64()) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	// Wait for any transitional state to complete first
	if !isStableState(currentState) {
		queryFunc := func(ctx context.Context, n string) (string, error) {
			return r.queryAppState(ctx, n)
		}
		currentState, err = waitForStableState(ctx, appName, timeout, queryFunc)
		if err != nil {
			resp.Diagnostics.AddError(
				"Timeout Waiting for App State",
				err.Error(),
			)
			return
		}
	}

	// Reconcile if needed
	if currentState != normalizedDesired {
		// Use a temporary response to capture warnings
		tempResp := &resource.UpdateResponse{}
		err := r.reconcileDesiredState(ctx, appName, currentState, normalizedDesired, timeout, tempResp)

		// Copy warnings to actual response
		resp.Diagnostics.Append(tempResp.Diagnostics...)

		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Reconcile App State",
				err.Error(),
			)
			return
		}
		currentState = normalizedDesired
	}

	// Set final state
	data.ID = types.StringValue(appName)
	data.State = types.StringValue(currentState)
	data.DesiredState = types.StringValue(normalizedDesired)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resources -run TestAppResource_Update_ReconcileStateFromStoppedToRunning -v`
Expected: PASS

**Step 5: Run all Update tests to ensure no regressions**

Run: `go test ./internal/resources -run "TestAppResource_Update" -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/resources/app.go internal/resources/app_test.go
git commit -m "feat(app): handle desired_state reconciliation in Update lifecycle"
```

---

## Task 10: Update Read Method to Preserve desired_state

**Files:**
- Modify: `internal/resources/app.go:175-253` (Read method)
- Modify: `internal/resources/app_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/resources/app_test.go

func TestAppResource_Read_PreservesDesiredState(t *testing.T) {
	r := &AppResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"name": "myapp",
					"state": "STOPPED",
					"custom_app": true,
					"config": {}
				}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// State has desired_state = "STOPPED" (user intentionally stopped it)
	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "STOPPED", float64(120), "STOPPED")

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model AppResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	// desired_state should be preserved from prior state
	if model.DesiredState.ValueString() != "STOPPED" {
		t.Errorf("expected desired_state 'STOPPED' to be preserved, got %q", model.DesiredState.ValueString())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/resources -run TestAppResource_Read_PreservesDesiredState -v`
Expected: FAIL (desired_state not preserved or wrong value)

**Step 3: Update the Read method**

Modify the Read method in `app.go` to preserve `desired_state` and `state_timeout`:

```go
func (r *AppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve desired_state and state_timeout from prior state
	priorDesiredState := data.DesiredState
	priorStateTimeout := data.StateTimeout

	// Use the name to query the app
	appName := data.Name.ValueString()

	// Build query params
	filter := [][]any{{"name", "=", appName}}
	options := map[string]any{
		"extra": map[string]any{
			"retrieve_config": true,
		},
	}
	queryParams := []any{filter, options}

	// Call the TrueNAS API
	result, err := r.client.Call(ctx, "app.query", queryParams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read App",
			fmt.Sprintf("Unable to read app %q: %s", appName, err.Error()),
		)
		return
	}

	// Parse the response
	var apps []appAPIResponse
	if err := json.Unmarshal(result, &apps); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse App Response",
			fmt.Sprintf("Unable to parse app response: %s", err.Error()),
		)
		return
	}

	// Check if app was found
	if len(apps) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	app := apps[0]

	// Map response to model
	data.ID = types.StringValue(app.Name)
	data.State = types.StringValue(app.State)
	data.CustomApp = types.BoolValue(app.CustomApp)

	// Restore preserved values
	data.DesiredState = priorDesiredState
	data.StateTimeout = priorStateTimeout

	// If desired_state is null/unknown, default to current state (assume user wants current state)
	if data.DesiredState.IsNull() || data.DesiredState.IsUnknown() {
		data.DesiredState = types.StringValue(app.State)
	}
	if data.StateTimeout.IsNull() || data.StateTimeout.IsUnknown() {
		data.StateTimeout = types.Int64Value(120)
	}

	// Sync compose_config if present
	if len(app.Config) > 0 {
		yamlBytes, err := yaml.Marshal(app.Config)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Marshal Config",
				fmt.Sprintf("Unable to marshal app config to YAML: %s", err.Error()),
			)
			return
		}
		data.ComposeConfig = customtypes.NewYAMLStringValue(string(yamlBytes))
	} else {
		data.ComposeConfig = customtypes.NewYAMLStringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resources -run TestAppResource_Read_PreservesDesiredState -v`
Expected: PASS

**Step 5: Run all Read tests to ensure no regressions**

Run: `go test ./internal/resources -run "TestAppResource_Read" -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/resources/app.go internal/resources/app_test.go
git commit -m "feat(app): preserve desired_state in Read lifecycle"
```

---

## Task 11: Add CRASHED State Handling Test

**Files:**
- Modify: `internal/resources/app_test.go`

**Step 1: Write the test for CRASHED state handling**

```go
// Add to internal/resources/app_test.go

func TestAppResource_Update_CrashedAppStartAttempt(t *testing.T) {
	var calledMethod string
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				calledMethod = method
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				// Return RUNNING after start attempt
				return json.RawMessage(`[{"name": "myapp", "state": "RUNNING"}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Current state: CRASHED, desired: RUNNING
	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), "CRASHED")
	planValue := createAppResourceModelValue("myapp", "myapp", true, nil, "RUNNING", float64(120), nil)

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify app.start was called to recover from CRASHED
	if calledMethod != "app.start" {
		t.Errorf("expected app.start to be called for CRASHED app, got %q", calledMethod)
	}
}

func TestAppResource_Update_CrashedAppDesiredStopped(t *testing.T) {
	callCount := 0
	r := &AppResource{
		client: &client.MockClient{
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				callCount++
				return nil, nil
			},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{"name": "myapp", "state": "CRASHED"}]`), nil
			},
		},
	}

	schemaResp := getAppResourceSchema(t)

	// Current state: CRASHED, desired: STOPPED - no action needed
	stateValue := createAppResourceModelValue("myapp", "myapp", true, nil, "STOPPED", float64(120), "CRASHED")
	planValue := createAppResourceModelValue("myapp", "myapp", true, nil, "STOPPED", float64(120), nil)

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	// Should not error - CRASHED is "stopped enough"
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// No start/stop should be called
	if callCount > 0 {
		t.Errorf("expected no API calls for CRASHED->STOPPED, got %d", callCount)
	}
}
```

**Step 2: Run tests**

Run: `go test ./internal/resources -run "TestAppResource_Update_Crashed" -v`
Expected: PASS (if reconcileDesiredState handles CRASHED correctly)

If FAIL, update reconcileDesiredState to handle CRASHED state:

```go
// In reconcileDesiredState, add special handling for CRASHED
if normalizedDesired == AppStateStopped && currentState == AppStateCrashed {
	// CRASHED is "stopped enough" - no action needed
	return nil
}
```

**Step 3: Commit**

```bash
git add internal/resources/app.go internal/resources/app_test.go
git commit -m "test(app): add CRASHED state handling tests"
```

---

## Task 12: Run Full Test Suite and Fix Any Issues

**Files:**
- All modified files

**Step 1: Run all app resource tests**

Run: `go test ./internal/resources -v`
Expected: All PASS

**Step 2: Run all provider tests**

Run: `go test ./... -v`
Expected: All PASS

**Step 3: Build the provider**

Run: `go build ./...`
Expected: Success with no errors

**Step 4: Run linter (if configured)**

Run: `mise run lint` or `golangci-lint run`
Expected: No new issues

**Step 5: Commit any fixes**

```bash
git add .
git commit -m "fix(app): address test and lint issues"
```

---

## Task 13: Update Documentation

**Files:**
- Modify: `docs/resources/app.md` (if exists, or will be auto-generated)

**Step 1: Regenerate provider documentation**

Run: `mise run docs` or `go generate ./...`
Expected: Documentation updated

**Step 2: Verify documentation includes new attributes**

Check that `docs/resources/app.md` includes:
- `desired_state` attribute with description
- `state_timeout` attribute with description
- Updated examples showing desired_state usage

**Step 3: Commit documentation**

```bash
git add docs/
git commit -m "docs(app): update documentation for desired_state attribute"
```

---

## Summary

This plan implements the `desired_state` attribute in 13 tasks:

1. State constants and normalization helper
2. State validation helpers (isStableState, isValidDesiredState)
3. Case-insensitive plan modifier
4. Schema and model updates
5. waitForStableState polling helper
6. queryAppState helper method
7. reconcileDesiredState method
8. Create lifecycle updates
9. Update lifecycle updates
10. Read lifecycle updates
11. CRASHED state handling
12. Full test suite verification
13. Documentation updates

Each task follows TDD: write failing test → implement → verify → commit.

---

Plan complete and saved to `docs/plans/2026-01-15-desired-state-implementation.md`. Two execution options:

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach?
