import assert from "node:assert/strict";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, test, vi } from "vitest";

import { type Workspace } from "../../lib/api-types";

const apiMocks = vi.hoisted(() => {
  class MockApiError extends Error {
    status: number;
    code: string;

    constructor(message: string, status: number, code = "") {
      super(message);
      this.name = "ApiError";
      this.status = status;
      this.code = code;
    }
  }

  return {
    ApiError: MockApiError,
    listOrganizationWorkspaces: vi.fn(),
    createWorkspace: vi.fn(),
    updateWorkspace: vi.fn(),
  };
});

vi.mock("../../lib/api", () => apiMocks);

import { OrganizationAdminSection } from "./organization-admin-section";

function workspace(overrides: Partial<Workspace> = {}): Workspace {
  return {
    id: "ws-1",
    name: "Home",
    slug: "home",
    status: "active",
    role: "admin",
    is_active: true,
    ...overrides,
  };
}

beforeEach(() => {
  apiMocks.listOrganizationWorkspaces.mockReset();
  apiMocks.createWorkspace.mockReset();
  apiMocks.updateWorkspace.mockReset();
});

test("lists organization workspaces once the section becomes active", async () => {
  apiMocks.listOrganizationWorkspaces.mockResolvedValue({
    workspaces: [
      workspace(),
      workspace({ id: "ws-2", name: "Retired", slug: "retired", status: "archived", is_active: false }),
    ],
  });

  render(<OrganizationAdminSection isActive />);

  await screen.findByText("Home");
  assert.ok(screen.getByText("Retired"));
  assert.equal(apiMocks.listOrganizationWorkspaces.mock.calls.length, 1);
});

test("does not load while inactive", () => {
  render(<OrganizationAdminSection isActive={false} />);
  assert.equal(apiMocks.listOrganizationWorkspaces.mock.calls.length, 0);
});

test("creates a workspace and prepends it", async () => {
  const user = userEvent.setup();
  apiMocks.listOrganizationWorkspaces.mockResolvedValue({ workspaces: [] });
  apiMocks.createWorkspace.mockResolvedValue(
    workspace({ id: "ws-new", name: "Marketing", slug: "marketing", is_active: false }),
  );

  render(<OrganizationAdminSection isActive />);
  await screen.findByText("No workspaces yet");

  await user.type(screen.getByLabelText("Workspace name"), "Marketing");
  await user.click(screen.getByRole("button", { name: "Create workspace" }));

  await screen.findByText("Marketing");
  assert.equal(apiMocks.createWorkspace.mock.calls[0]?.[0], "Marketing");
});

test("archives an active workspace", async () => {
  const user = userEvent.setup();
  apiMocks.listOrganizationWorkspaces.mockResolvedValue({
    workspaces: [workspace({ is_active: false })],
  });
  apiMocks.updateWorkspace.mockResolvedValue(
    workspace({ status: "archived", is_active: false }),
  );

  render(<OrganizationAdminSection isActive />);

  const item = (await screen.findByText("Home")).closest("li");
  assert.ok(item);
  await user.click(within(item as HTMLElement).getByRole("button", { name: "Archive" }));

  await waitFor(() => {
    assert.deepEqual(apiMocks.updateWorkspace.mock.calls[0], [
      "ws-1",
      { status: "archived" },
    ]);
  });
  await within(item as HTMLElement).findByRole("button", { name: "Restore" });
});
