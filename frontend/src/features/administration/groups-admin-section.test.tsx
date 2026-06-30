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
