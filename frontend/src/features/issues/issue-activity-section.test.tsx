import assert from "node:assert/strict";
import { render, screen } from "@testing-library/react";
import { test } from "vitest";

import { IssueActivitySection } from "./issue-activity-section";

test("renders automation activity as a readable System entry", () => {
  render(
    <IssueActivitySection
      activity={[
        {
          id: "activity-automation",
          issue_id: "issue-1",
          action: "automation_applied",
          actor_id: null,
          actor_display_name: null,
          payload: {
            rule_name: "Assign reviewer",
            changed_fields: "status,assignee",
            from_status: "todo",
            to_status: "review",
            from_assignee_id: "",
            to_assignee_id: "member-1",
          },
          created_at: "2026-06-15T10:00:00Z",
        },
      ]}
      activityError=""
      isLoadingActivity={false}
      labels={[]}
      teamMembers={[
        {
          id: "member-1",
          email: "reviewer@example.com",
          username: "reviewer",
          display_name: "Reviewer",
          role: "member",
          is_active: true,
          joined_at: "2026-06-15T09:00:00Z",
        },
      ]}
    />,
  );

  assert.ok(screen.getByText("Automation applied"));
  assert.ok(
    screen.getByText(
      "System · Assign reviewer · status Todo -> Review; assignee Unassigned -> Reviewer",
    ),
  );
});
