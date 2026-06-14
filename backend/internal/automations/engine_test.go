package automations

import (
	"reflect"
	"testing"
)

func TestMatchesConditionsUsesFrozenSnapshotValues(t *testing.T) {
	t.Parallel()
	assigneeID := testUUID
	reporterID := otherTestUUID
	issue := runtimeIssue{
		IssueType:        "bug",
		WorkflowStatusID: testUUID,
		Priority:         "high",
		ReporterID:       reporterID,
		AssigneeID:       assigneeID,
		LabelIDs:         map[string]bool{testUUID: true, otherTestUUID: true},
	}
	conditions := []runtimeItem{
		{Type: "issue_type", Value: "bug"},
		{Type: "workflow_status", WorkflowStatusID: testUUID},
		{Type: "priority", Value: "high"},
		{Type: "assignee", UserID: &assigneeID},
		{Type: "reporter", UserID: &reporterID},
		{Type: "label", LabelID: testUUID},
		{Type: "label", LabelID: otherTestUUID},
	}
	if !matchesConditions(issue, conditions) {
		t.Fatal("expected all ordered AND conditions to match")
	}
	conditions = append(conditions, runtimeItem{Type: "label", LabelID: "missing"})
	if matchesConditions(issue, conditions) {
		t.Fatal("expected missing label condition to reject snapshot")
	}
}

func TestMatchesConditionsSupportsUnassignedAndRejectsUnknown(t *testing.T) {
	t.Parallel()
	issue := runtimeIssue{LabelIDs: map[string]bool{}}
	if !matchesConditions(issue, []runtimeItem{{Type: "assignee", UserID: nil}}) {
		t.Fatal("expected null assignee condition to match unassigned issue")
	}
	if matchesConditions(issue, []runtimeItem{{Type: "unknown"}}) {
		t.Fatal("expected unknown condition to reject issue")
	}
}

func TestAutomationActivityPayloadTracksFinalRuleChanges(t *testing.T) {
	t.Parallel()
	before := runtimeIssue{
		Status: "todo", WorkflowStatusID: testUUID, Priority: "medium", AssigneeID: "",
		LabelIDs: map[string]bool{testUUID: true},
	}
	after := runtimeIssue{
		Status: "review", WorkflowStatusID: otherTestUUID, Priority: "critical", AssigneeID: testUUID,
		LabelIDs: map[string]bool{otherTestUUID: true},
	}
	payload := automationActivityPayload(
		runtimeRule{ID: testUUID, Name: "Route issue"},
		ExecuteRequest{TriggerType: "issue_created", InitiatedByUserID: otherTestUUID},
		before,
		after,
	)
	if payload["changed_fields"] != "status,assignee,priority,labels" {
		t.Fatalf("changed_fields = %q", payload["changed_fields"])
	}
	if payload["from_status"] != "todo" || payload["to_status"] != "review" {
		t.Fatalf("status payload = %#v", payload)
	}
	if payload["added_label_ids"] != otherTestUUID || payload["removed_label_ids"] != testUUID {
		t.Fatalf("label payload = %#v", payload)
	}
}

func TestAutomationActivityPayloadOmitsNoOpRule(t *testing.T) {
	t.Parallel()
	issue := runtimeIssue{Priority: "medium", LabelIDs: map[string]bool{testUUID: true}}
	if payload := automationActivityPayload(runtimeRule{}, ExecuteRequest{}, issue, cloneRuntimeIssue(issue)); payload != nil {
		t.Fatalf("no-op payload = %#v, want nil", payload)
	}
}

func TestChangedLabelsIsStable(t *testing.T) {
	t.Parallel()
	added, removed := changedLabels(
		map[string]bool{"b": true, "a": true},
		map[string]bool{"d": true, "c": true},
	)
	if !reflect.DeepEqual(added, []string{"c", "d"}) || !reflect.DeepEqual(removed, []string{"a", "b"}) {
		t.Fatalf("changed labels = %#v/%#v", added, removed)
	}
}
