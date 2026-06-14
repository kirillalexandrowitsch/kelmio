package automations

import (
	"encoding/json"
	"strings"
	"testing"
)

const testUUID = "6d5257d4-002e-44da-8925-d9108699c504"
const otherTestUUID = "4d4257d4-002e-44da-8925-d9108699c505"

func TestNormalizeCreateRule(t *testing.T) {
	t.Parallel()
	enabled := false
	input, err := normalizeCreateRule(createRuleRequest{
		Name:        "  Route critical bugs ",
		TriggerType: "issue_created",
		Conditions: json.RawMessage(`[
			{"type":"issue_type","value":"bug"},
			{"type":"label","label_id":"6D5257D4-002E-44DA-8925-D9108699C504"}
		]`),
		Actions: json.RawMessage(`[
			{"type":"change_priority","value":"critical"},
			{"type":"change_assignee","user_id":null},
			{"type":"add_label","label_id":"4D4257D4-002E-44DA-8925-D9108699C505"}
		]`),
		IsEnabled: &enabled,
	})
	if err != nil {
		t.Fatalf("normalize create rule: %v", err)
	}
	if input.Name != "Route critical bugs" || input.TriggerType != "issue_created" || input.IsEnabled {
		t.Fatalf("normalized create rule = %#v", input)
	}
	if got := string(input.Conditions); !strings.Contains(got, testUUID) {
		t.Fatalf("normalized conditions = %s", got)
	}
	if len(input.Definition.Dependencies) != 2 {
		t.Fatalf("dependencies = %#v, want two label dependencies", input.Definition.Dependencies)
	}
}

func TestNormalizeRuleValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		req  createRuleRequest
	}{
		{name: "missing name", req: validCreateRuleRequest()},
		{name: "invalid trigger", req: withCreateRule(validCreateRuleRequest(), func(req *createRuleRequest) { req.Name = "Rule"; req.TriggerType = "commented" })},
		{name: "missing actions", req: withCreateRule(validCreateRuleRequest(), func(req *createRuleRequest) { req.Name = "Rule"; req.Actions = nil })},
		{name: "empty actions", req: withCreateRule(validCreateRuleRequest(), func(req *createRuleRequest) { req.Name = "Rule"; req.Actions = json.RawMessage(`[]`) })},
		{name: "null conditions", req: withCreateRule(validCreateRuleRequest(), func(req *createRuleRequest) { req.Name = "Rule"; req.Conditions = json.RawMessage(`null`) })},
		{name: "trailing actions value", req: withCreateRule(validCreateRuleRequest(), func(req *createRuleRequest) {
			req.Name = "Rule"
			req.Actions = json.RawMessage(`[{"type":"change_priority","value":"high"}] []`)
		})},
		{name: "duplicate scalar condition", req: withCreateRule(validCreateRuleRequest(), func(req *createRuleRequest) {
			req.Name = "Rule"
			req.Conditions = json.RawMessage(`[{"type":"priority","value":"high"},{"type":"priority","value":"critical"}]`)
		})},
		{name: "duplicate label condition", req: withCreateRule(validCreateRuleRequest(), func(req *createRuleRequest) {
			req.Name = "Rule"
			req.Conditions = json.RawMessage(`[{"type":"label","label_id":"` + testUUID + `"},{"type":"label","label_id":"` + testUUID + `"}]`)
		})},
		{name: "reporter null", req: withCreateRule(validCreateRuleRequest(), func(req *createRuleRequest) {
			req.Name = "Rule"
			req.Conditions = json.RawMessage(`[{"type":"reporter","user_id":null}]`)
		})},
		{name: "unknown action field", req: withCreateRule(validCreateRuleRequest(), func(req *createRuleRequest) {
			req.Name = "Rule"
			req.Actions = json.RawMessage(`[{"type":"change_priority","value":"high","extra":true}]`)
		})},
		{name: "too many conditions", req: withCreateRule(validCreateRuleRequest(), func(req *createRuleRequest) {
			req.Name = "Rule"
			req.Conditions = json.RawMessage(`[` + strings.TrimSuffix(strings.Repeat(`{"type":"label","label_id":"`+testUUID+`"},`, 21), ",") + `]`)
		})},
	}
	tests[0].req.Name = ""
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := normalizeCreateRule(tt.req); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNormalizeUpdateAndOrderValidation(t *testing.T) {
	t.Parallel()
	if _, err := normalizeUpdateRule(updateRuleRequest{}); err == nil {
		t.Fatal("expected empty update error")
	}
	input, err := normalizeUpdateRule(updateRuleRequest{
		Conditions: json.RawMessage(`[{"type":"assignee","user_id":null}]`),
		Actions:    json.RawMessage(`[{"type":"change_assignee","user_id":null}]`),
	})
	if err != nil {
		t.Fatalf("normalize update rule: %v", err)
	}
	if !input.HasConditions || !input.HasActions {
		t.Fatalf("normalized update = %#v", input)
	}
	if _, err := normalizeRuleOrder(ruleOrderRequest{}); err == nil {
		t.Fatal("expected empty order error")
	}
	if _, err := normalizeRuleOrder(ruleOrderRequest{RuleIDs: []string{testUUID, testUUID}}); err == nil {
		t.Fatal("expected duplicate order error")
	}
	if _, err := normalizeRuleOrder(ruleOrderRequest{RuleIDs: []string{"bad-id"}}); err == nil {
		t.Fatal("expected malformed order id error")
	}
}

func validCreateRuleRequest() createRuleRequest {
	return createRuleRequest{
		Name:        "Rule",
		TriggerType: "status_changed",
		Conditions:  json.RawMessage(`[]`),
		Actions:     json.RawMessage(`[{"type":"change_priority","value":"high"}]`),
	}
}

func withCreateRule(req createRuleRequest, mutate func(*createRuleRequest)) createRuleRequest {
	mutate(&req)
	return req
}
