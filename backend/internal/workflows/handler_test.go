package workflows

import (
	"strings"
	"testing"
)

const testWorkflowID = "6d5257d4-002e-44da-8925-d9108699c504"
const otherWorkflowID = "807513ce-18b0-4ed6-a500-e65bb4fef444"

func TestNormalizeCreateStatus(t *testing.T) {
	t.Parallel()

	input, err := normalizeCreateStatus(createStatusRequest{
		Key:      " Review_Ready ",
		Name:     " Review ready ",
		Color:    " #ABCDEF ",
		Category: "in_progress",
	})
	if err != nil {
		t.Fatalf("normalize create status: %v", err)
	}
	if input.Key != "review_ready" || input.Name != "Review ready" || input.Color != "#abcdef" {
		t.Fatalf("normalized create status = %#v", input)
	}
}

func TestNormalizeCreateStatusValidation(t *testing.T) {
	t.Parallel()

	tests := []createStatusRequest{
		{Key: "Invalid-Key", Name: "Valid", Color: "#123456", Category: "todo"},
		{Key: "valid", Name: "", Color: "#123456", Category: "todo"},
		{Key: "valid", Name: strings.Repeat("a", 61), Color: "#123456", Category: "todo"},
		{Key: "valid", Name: "Valid", Color: "red", Category: "todo"},
		{Key: "valid", Name: "Valid", Color: "#123456", Category: "paused"},
	}
	for _, req := range tests {
		if _, err := normalizeCreateStatus(req); err == nil {
			t.Fatalf("expected validation error for %#v", req)
		}
	}
}

func TestNormalizeUpdateStatusRequiresChange(t *testing.T) {
	t.Parallel()

	if _, err := normalizeUpdateStatus(updateStatusRequest{}); err == nil {
		t.Fatal("expected empty update error")
	}
}

func TestNormalizeStatusOrder(t *testing.T) {
	t.Parallel()

	got, err := normalizeStatusOrder(statusOrderRequest{StatusIDs: []string{" " + testWorkflowID + " ", otherWorkflowID}})
	if err != nil {
		t.Fatalf("normalize order: %v", err)
	}
	if got[0] != testWorkflowID || got[1] != otherWorkflowID {
		t.Fatalf("normalized order = %#v", got)
	}

	if _, err := normalizeStatusOrder(statusOrderRequest{StatusIDs: []string{testWorkflowID, testWorkflowID}}); err == nil {
		t.Fatal("expected duplicate order id error")
	}
	if _, err := normalizeStatusOrder(statusOrderRequest{StatusIDs: []string{"bad-id"}}); err == nil {
		t.Fatal("expected malformed order id error")
	}
}

func TestNormalizeTransitions(t *testing.T) {
	t.Parallel()

	got, err := normalizeTransitions(replaceTransitionsRequest{Transitions: []transitionRequest{{
		FromStatusID: testWorkflowID,
		ToStatusID:   otherWorkflowID,
	}}})
	if err != nil {
		t.Fatalf("normalize transitions: %v", err)
	}
	if len(got) != 1 || got[0].FromStatusID != testWorkflowID || got[0].ToStatusID != otherWorkflowID {
		t.Fatalf("normalized transitions = %#v", got)
	}

	if _, err := normalizeTransitions(replaceTransitionsRequest{Transitions: []transitionRequest{{
		FromStatusID: testWorkflowID,
		ToStatusID:   testWorkflowID,
	}}}); err == nil {
		t.Fatal("expected self transition error")
	}
	if _, err := normalizeTransitions(replaceTransitionsRequest{Transitions: []transitionRequest{
		{FromStatusID: testWorkflowID, ToStatusID: otherWorkflowID},
		{FromStatusID: testWorkflowID, ToStatusID: otherWorkflowID},
	}}); err == nil {
		t.Fatal("expected duplicate transition error")
	}
	if _, err := normalizeTransitions(replaceTransitionsRequest{Transitions: []transitionRequest{{
		FromStatusID: "bad-id",
		ToStatusID:   otherWorkflowID,
	}}}); err == nil {
		t.Fatal("expected malformed transition id error")
	}
}
