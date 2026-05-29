import assert from "node:assert/strict";
import test from "node:test";

import {
  appSectionFromPath,
  appSectionPath,
  currentAppSectionFromLocation,
  sprintIdFromPath,
} from "./routing.ts";

test("maps app sections to canonical paths", () => {
  assert.equal(appSectionPath("dashboard"), "/");
  assert.equal(appSectionPath("projects"), "/projects");
  assert.equal(appSectionPath("issues"), "/issues");
  assert.equal(appSectionPath("board"), "/board");
  assert.equal(appSectionPath("sprints"), "/sprints");
  assert.equal(appSectionPath("team"), "/team");
  assert.equal(appSectionPath("labels"), "/labels");
  assert.equal(appSectionPath("account"), "/account");
});

test("maps direct paths to app sections", () => {
  assert.equal(appSectionFromPath("/"), "dashboard");
  assert.equal(appSectionFromPath("/projects"), "projects");
  assert.equal(appSectionFromPath("/issues"), "issues");
  assert.equal(appSectionFromPath("/board"), "board");
  assert.equal(appSectionFromPath("/sprints"), "sprints");
  assert.equal(appSectionFromPath("/sprints/example-id"), "sprints");
  assert.equal(appSectionFromPath("/team"), "team");
  assert.equal(appSectionFromPath("/labels"), "labels");
  assert.equal(appSectionFromPath("/account"), "account");
});

test("falls back to dashboard for unknown paths", () => {
  assert.equal(appSectionFromPath("/unknown"), "dashboard");
  assert.equal(currentAppSectionFromLocation({ pathname: "/missing" }), "dashboard");
});

test("extracts sprint ids from direct sprint paths", () => {
  assert.equal(sprintIdFromPath("/sprints/sprint-1"), "sprint-1");
  assert.equal(sprintIdFromPath("/sprints/sprint%201"), "sprint 1");
  assert.equal(sprintIdFromPath("/sprints"), "");
  assert.equal(sprintIdFromPath("/issues/sprint-1"), "");
});
