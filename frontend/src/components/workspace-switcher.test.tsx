import assert from "node:assert/strict";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { test, vi } from "vitest";

import { type Workspace } from "../lib/api-types";
import { WorkspaceSwitcher } from "./workspace-switcher";

function workspace(overrides: Partial<Workspace> = {}): Workspace {
  return {
    id: "ws-1",
    name: "Alpha",
    slug: "alpha",
    status: "active",
    role: "admin",
    is_active: true,
    ...overrides,
  };
}

test("renders nothing when there are no workspaces", () => {
  const { container } = render(
    <WorkspaceSwitcher
      activeWorkspaceId=""
      isSwitching={false}
      onSwitch={vi.fn()}
      workspaces={[]}
    />,
  );
  assert.equal(container.firstChild, null);
});

test("shows the active workspace and disables switching with a single workspace", () => {
  render(
    <WorkspaceSwitcher
      activeWorkspaceId="ws-1"
      isSwitching={false}
      onSwitch={vi.fn()}
      workspaces={[workspace()]}
    />,
  );

  const trigger = screen.getByRole("button", { name: "Switch workspace" });
  assert.ok(trigger.textContent?.includes("Alpha"));
  assert.equal((trigger as HTMLButtonElement).disabled, true);
});

test("switches to another workspace and ignores reselecting the active one", async () => {
  const user = userEvent.setup();
  const onSwitch = vi.fn();
  render(
    <WorkspaceSwitcher
      activeWorkspaceId="ws-1"
      isSwitching={false}
      onSwitch={onSwitch}
      workspaces={[
        workspace(),
        workspace({ id: "ws-2", name: "Beta", slug: "beta", is_active: false }),
      ]}
    />,
  );

  await user.click(screen.getByRole("button", { name: "Switch workspace" }));
  await user.click(screen.getByRole("menuitem", { name: "Beta" }));
  assert.deepEqual(onSwitch.mock.calls[0], ["ws-2"]);

  onSwitch.mockClear();
  await user.click(screen.getByRole("button", { name: "Switch workspace" }));
  await user.click(screen.getByRole("menuitem", { name: "Alpha" }));
  assert.equal(onSwitch.mock.calls.length, 0);
});
