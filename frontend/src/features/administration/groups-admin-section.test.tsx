import assert from "node:assert/strict";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, test, vi } from "vitest";

import { type Group } from "../../lib/api-types";

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
    listGroups: vi.fn(),
    createGroup: vi.fn(),
    updateGroup: vi.fn(),
    deleteGroup: vi.fn(),
    listGroupMembers: vi.fn(),
    addGroupMember: vi.fn(),
    removeGroupMember: vi.fn(),
    listDirectory: vi.fn(),
  };
});

vi.mock("../../lib/api", () => apiMocks);

import { GroupsAdminSection } from "./groups-admin-section";

function group(overrides: Partial<Group> = {}): Group {
  return {
    id: "group-1",
    name: "Engineers",
    description: "Builders",
    member_count: 3,
    ...overrides,
  };
}

beforeEach(() => {
  apiMocks.listGroups.mockReset();
  apiMocks.createGroup.mockReset();
  apiMocks.updateGroup.mockReset();
  apiMocks.deleteGroup.mockReset();
  apiMocks.listGroupMembers.mockReset();
  apiMocks.addGroupMember.mockReset();
  apiMocks.removeGroupMember.mockReset();
  apiMocks.listDirectory.mockReset();
});

test("lists groups once active", async () => {
  apiMocks.listGroups.mockResolvedValue({
    groups: [group(), group({ id: "group-2", name: "Designers", member_count: 1 })],
  });

  render(<GroupsAdminSection isActive />);

  await screen.findByText("Engineers");
  assert.ok(screen.getByText("Designers"));
  assert.ok(screen.getByText("1 member"));
  assert.equal(apiMocks.listGroups.mock.calls.length, 1);
});

test("does not load while inactive", () => {
  render(<GroupsAdminSection isActive={false} />);
  assert.equal(apiMocks.listGroups.mock.calls.length, 0);
});

test("creates a group with a description and prepends it", async () => {
  const user = userEvent.setup();
  apiMocks.listGroups.mockResolvedValue({ groups: [] });
  apiMocks.createGroup.mockResolvedValue(
    group({ id: "group-new", name: "Leads", description: "Team leads", member_count: 0 }),
  );

  render(<GroupsAdminSection isActive />);
  await screen.findByText("No groups yet");

  await user.type(screen.getByLabelText("Group name"), "Leads");
  await user.type(screen.getByLabelText("Description"), "Team leads");
  await user.click(screen.getByRole("button", { name: "Create group" }));

  await screen.findByText("Leads");
  assert.deepEqual(apiMocks.createGroup.mock.calls[0], ["Leads", "Team leads"]);
});

test("adds a member from the directory and removes a member", async () => {
  const user = userEvent.setup();
  apiMocks.listGroups.mockResolvedValue({ groups: [group({ member_count: 1 })] });
  apiMocks.listGroupMembers.mockResolvedValue({
    members: [
      {
        user_id: "user-1",
        username: "ada",
        display_name: "Ada Lovelace",
        email: "ada@example.com",
        added_at: "2026-01-01T00:00:00Z",
      },
    ],
  });
  apiMocks.listDirectory.mockResolvedValue({
    users: [
      { user_id: "user-1", username: "ada", display_name: "Ada Lovelace", email: "ada@example.com", role: "org_member" },
      { user_id: "user-2", username: "alan", display_name: "Alan Turing", email: "alan@example.com", role: "org_member" },
    ],
  });
  apiMocks.addGroupMember.mockResolvedValue({
    user_id: "user-2",
    username: "alan",
    display_name: "Alan Turing",
    email: "alan@example.com",
    added_at: "2026-01-02T00:00:00Z",
  });
  apiMocks.removeGroupMember.mockResolvedValue(undefined);

  render(<GroupsAdminSection isActive />);

  const groupItem = (await screen.findByText("Engineers")).closest("li");
  await user.click(within(groupItem as HTMLElement).getByRole("button", { name: "Members" }));

  const panel = await screen.findByRole("region", { name: "Group members" });
  await within(panel).findByText("Ada Lovelace");

  // The directory picker only offers people who are not already members.
  assert.equal(within(panel).queryByRole("option", { name: /Ada Lovelace/ }), null);
  await user.selectOptions(within(panel).getByLabelText("Add member"), "user-2");
  await user.click(within(panel).getByRole("button", { name: "Add" }));
  await within(panel).findByText("Alan Turing");
  assert.deepEqual(apiMocks.addGroupMember.mock.calls[0], ["group-1", "user-2"]);

  await user.click(
    within((await within(panel).findByText("Ada Lovelace")).closest("li") as HTMLElement).getByRole(
      "button",
      { name: "Remove" },
    ),
  );
  await waitFor(() => {
    assert.deepEqual(apiMocks.removeGroupMember.mock.calls[0], ["group-1", "user-1"]);
  });
});

test("deletes a group after confirmation", async () => {
  const user = userEvent.setup();
  apiMocks.listGroups.mockResolvedValue({ groups: [group()] });
  apiMocks.deleteGroup.mockResolvedValue(undefined);

  render(<GroupsAdminSection isActive />);

  const item = (await screen.findByText("Engineers")).closest("li");
  assert.ok(item);
  await user.click(within(item as HTMLElement).getByRole("button", { name: "Delete" }));
  await user.click(within(item as HTMLElement).getByRole("button", { name: "Confirm" }));

  await waitFor(() => {
    assert.equal(apiMocks.deleteGroup.mock.calls[0]?.[0], "group-1");
  });
  await waitFor(() => {
    assert.equal(screen.queryByText("Engineers"), null);
  });
});
