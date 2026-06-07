import assert from "node:assert/strict";
import { test } from "vitest";

import {
  notificationDescription,
  notificationTitle,
  unreadBadgeLabel,
} from "./notification-view.ts";
import { type AppNotification } from "./api-types.ts";

function makeNotification(
  overrides: Partial<AppNotification> = {},
): AppNotification {
  return {
    id: "notification-1",
    workspace_id: "workspace-1",
    user_id: "user-1",
    actor_id: "actor-1",
    actor_display_name: "Admin",
    issue_id: "issue-1",
    issue_key: "CORE-1",
    issue_title: "Build notifications",
    notification_type: "issue_assigned",
    payload: {},
    read_at: null,
    created_at: "2026-06-01T08:00:00Z",
    ...overrides,
  };
}

test("formats unread badge labels", () => {
  assert.equal(unreadBadgeLabel(0), "No unread notifications");
  assert.equal(unreadBadgeLabel(1), "1 unread notification");
  assert.equal(unreadBadgeLabel(5), "5 unread notifications");
});

test("formats notification titles", () => {
  assert.equal(
    notificationTitle(makeNotification()),
    "Admin assigned you an issue",
  );
  assert.equal(
    notificationTitle(
      makeNotification({ notification_type: "issue_mentioned" }),
    ),
    "Admin mentioned you",
  );
  assert.equal(
    notificationTitle(
      makeNotification({ notification_type: "sprint_completed" }),
    ),
    "Admin completed a sprint",
  );
});

test("describes issue and sprint notifications", () => {
  assert.equal(
    notificationDescription(makeNotification()),
    "CORE-1 · Build notifications",
  );
  assert.equal(
    notificationDescription(
      makeNotification({
        issue_id: null,
        issue_key: null,
        issue_title: null,
        notification_type: "sprint_started",
        payload: { sprint_name: "Sprint 1", project_key: "CORE" },
      }),
    ),
    "Sprint 1 · CORE",
  );
});
