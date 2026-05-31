import assert from "node:assert/strict";
import test from "node:test";

import {
  sprintDisplayName,
  sprintDateRange,
  sprintMatchesFilters,
  sprintOptionLabel,
  sprintStatusCounts,
} from "./sprint-model.ts";
import { type Sprint } from "./api-types.ts";

const baseSprint: Sprint = {
  id: "sprint-1",
  workspace_id: "workspace-1",
  project_id: "project-1",
  project_key: "CORE",
  project_name: "Core",
  name: "Sprint 1",
  goal: "Ship useful work",
  status: "planned",
  start_date: "2026-06-01",
  end_date: "2026-06-14",
  created_by: "user-1",
  created_at: "2026-05-29T10:00:00Z",
  completed_at: null,
  issue_count: 0,
  done_count: 0,
  points_total: 0,
  points_done: 0,
  points_open: 0,
};

test("matches sprint project and status filters", () => {
  assert.equal(sprintMatchesFilters(baseSprint, "", ""), true);
  assert.equal(sprintMatchesFilters(baseSprint, "project-1", ""), true);
  assert.equal(sprintMatchesFilters(baseSprint, "project-2", ""), false);
  assert.equal(sprintMatchesFilters(baseSprint, "", "planned"), true);
  assert.equal(sprintMatchesFilters(baseSprint, "", "active"), false);
});

test("formats sprint date ranges", () => {
  assert.equal(sprintDateRange(baseSprint), "2026-06-01 to 2026-06-14");
  assert.equal(
    sprintDateRange({ start_date: "2026-06-01", end_date: null }),
    "Starts 2026-06-01",
  );
  assert.equal(
    sprintDateRange({ start_date: null, end_date: "2026-06-14" }),
    "Ends 2026-06-14",
  );
  assert.equal(
    sprintDateRange({ start_date: null, end_date: null }),
    "No dates planned",
  );
});

test("formats sprint labels for filters and issue rows", () => {
  assert.equal(sprintOptionLabel(baseSprint), "Sprint 1 (CORE)");
  assert.equal(sprintDisplayName([baseSprint], baseSprint.id), "Sprint 1");
  assert.equal(sprintDisplayName([baseSprint], "missing"), "missing");
  assert.equal(sprintDisplayName([baseSprint], null), "No sprint");
});

test("counts sprints by status", () => {
  const counts = sprintStatusCounts([
    baseSprint,
    { ...baseSprint, id: "sprint-2", status: "active" },
    { ...baseSprint, id: "sprint-3", status: "completed" },
    { ...baseSprint, id: "sprint-4", status: "completed" },
  ]);

  assert.deepEqual(counts, {
    planned: 1,
    active: 1,
    completed: 2,
  });
});
