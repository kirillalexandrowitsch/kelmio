import assert from "node:assert/strict";
import { test } from "vitest";

import {
  appSectionFromPath,
  appSectionPath,
  boardPath,
  boardProjectIdFromLocation,
  currentAppSectionFromLocation,
  forgotPasswordRouteFromLocation,
  inviteAcceptTokenFromLocation,
  passwordResetTokenFromLocation,
  sprintIdFromPath,
} from "./routing.ts";

test("maps app sections to canonical paths", () => {
  assert.equal(appSectionPath("dashboard"), "/");
  assert.equal(appSectionPath("projects"), "/projects");
  assert.equal(appSectionPath("issues"), "/issues");
  assert.equal(appSectionPath("board"), "/board");
  assert.equal(appSectionPath("sprints"), "/sprints");
  assert.equal(appSectionPath("notifications"), "/notifications");
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
  assert.equal(appSectionFromPath("/notifications"), "notifications");
  assert.equal(appSectionFromPath("/team"), "team");
  assert.equal(appSectionFromPath("/labels"), "labels");
  assert.equal(appSectionFromPath("/account"), "account");
});

test("falls back to dashboard for unknown paths", () => {
  assert.equal(appSectionFromPath("/unknown"), "dashboard");
  assert.equal(currentAppSectionFromLocation({ pathname: "/missing" }), "dashboard");
});

test("extracts invite accept tokens from public route", () => {
  assert.equal(
    inviteAcceptTokenFromLocation({
      pathname: "/accept-invite",
      search: "?token=invite-token",
    }),
    "invite-token",
  );
  assert.equal(
    inviteAcceptTokenFromLocation({
      pathname: "/accept-invite",
      search: "",
    }),
    "",
  );
  assert.equal(
    inviteAcceptTokenFromLocation({
      pathname: "/team",
      search: "?token=invite-token",
    }),
    null,
  );
});

test("detects forgot password public route", () => {
  assert.equal(forgotPasswordRouteFromLocation({ pathname: "/forgot-password" }), true);
  assert.equal(forgotPasswordRouteFromLocation({ pathname: "/" }), false);
});

test("extracts password reset tokens from public route", () => {
  assert.equal(
    passwordResetTokenFromLocation({
      pathname: "/reset-password",
      search: "?token=reset-token",
    }),
    "reset-token",
  );
  assert.equal(
    passwordResetTokenFromLocation({
      pathname: "/reset-password",
      search: "",
    }),
    "",
  );
  assert.equal(
    passwordResetTokenFromLocation({
      pathname: "/account",
      search: "?token=reset-token",
    }),
    null,
  );
});

test("extracts sprint ids from direct sprint paths", () => {
  assert.equal(sprintIdFromPath("/sprints/sprint-1"), "sprint-1");
  assert.equal(sprintIdFromPath("/sprints/sprint%201"), "sprint 1");
  assert.equal(sprintIdFromPath("/sprints"), "");
  assert.equal(sprintIdFromPath("/issues/sprint-1"), "");
});

test("builds and reads project-specific board routes", () => {
  assert.equal(boardPath(), "/board");
  assert.equal(boardPath("project one"), "/board?projectId=project%20one");
  assert.equal(
    boardProjectIdFromLocation({
      pathname: "/board",
      search: "?projectId=project-1",
    }),
    "project-1",
  );
  assert.equal(
    boardProjectIdFromLocation({
      pathname: "/issues",
      search: "?projectId=project-1",
    }),
    "",
  );
});
