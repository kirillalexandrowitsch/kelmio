import assert from "node:assert/strict";
import test from "node:test";

import { activityDescription, activityTitle } from "./activity-view.ts";
import { type IssueActivity, type TeamMember } from "./api-types.ts";

const members: TeamMember[] = [
  {
    id: "user-1",
    email: "one@example.com",
    username: "one",
    display_name: "User One",
    role: "member",
    is_active: true,
    joined_at: "2026-05-26T10:00:00Z",
  },
  {
    id: "user-2",
    email: "two@example.com",
    username: "two",
    display_name: "User Two",
    role: "member",
    is_active: true,
    joined_at: "2026-05-26T10:00:00Z",
  },
];

function makeActivity(overrides: Partial<IssueActivity>): IssueActivity {
  return {
    id: "activity-1",
    issue_id: "issue-1",
    action: "issue_created",
    actor_id: "user-1",
    actor_display_name: "User One",
    payload: {},
    created_at: "2026-05-26T10:00:00Z",
    ...overrides,
  };
}

test("formats known and unknown activity titles", () => {
  assert.equal(activityTitle(makeActivity({ action: "comment_added" })), "Added comment");
  assert.equal(
    activityTitle(makeActivity({ action: "issue_parent_changed" })),
    "Changed parent",
  );
  assert.equal(
    activityTitle(makeActivity({ action: "issue_link_created" })),
    "Linked issue",
  );
  assert.equal(
    activityTitle(makeActivity({ action: "custom_action" })),
    "custom action",
  );
});

test("formats activity descriptions with issue and member context", () => {
  assert.equal(
    activityDescription(
      makeActivity({
        action: "status_changed",
        payload: { from_status: "todo", to_status: "in_progress" },
      }),
      members,
    ),
    "Todo -> In progress",
  );
  assert.equal(
    activityDescription(
      makeActivity({
        action: "assignee_changed",
        payload: { from_assignee_id: "user-1", to_assignee_id: "user-2" },
      }),
      members,
    ),
    "User One -> User Two",
  );
  assert.equal(
    activityDescription(
      makeActivity({
        action: "comment_added",
        payload: { preview: "Looks good" },
      }),
      members,
    ),
    "\"Looks good\"",
  );
  assert.equal(
    activityDescription(
      makeActivity({
        action: "issue_parent_changed",
        payload: {
          from_parent_issue_id: "parent-1",
          to_parent_issue_id: "parent-2",
        },
      }),
      members,
    ),
    "parent-1 -> parent-2",
  );
  assert.equal(
    activityDescription(
      makeActivity({
        action: "issue_link_created",
        payload: {
          link_type: "blocks",
          source_issue_key: "WEB-1",
          target_issue_key: "WEB-2",
        },
      }),
      members,
    ),
    "WEB-1 blocks WEB-2",
  );
});
