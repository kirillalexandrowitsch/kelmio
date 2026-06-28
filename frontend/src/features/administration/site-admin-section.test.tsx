import assert from "node:assert/strict";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, test, vi } from "vitest";

import { type Organization } from "../../lib/api-types";

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
    listOrganizations: vi.fn(),
    createOrganization: vi.fn(),
    updateOrganization: vi.fn(),
  };
});

vi.mock("../../lib/api", () => apiMocks);

import { SiteAdminSection } from "./site-admin-section";

function organization(overrides: Partial<Organization> = {}): Organization {
  return {
    id: "org-1",
    name: "Acme Inc.",
    slug: "acme-inc",
    status: "active",
    role: "org_admin",
    ...overrides,
  };
}

beforeEach(() => {
  apiMocks.listOrganizations.mockReset();
  apiMocks.createOrganization.mockReset();
  apiMocks.updateOrganization.mockReset();
});

test("lists organizations once the section becomes active", async () => {
  apiMocks.listOrganizations.mockResolvedValue({
    organizations: [
      organization(),
      organization({ id: "org-2", name: "Globex", slug: "globex", status: "archived" }),
    ],
  });

  render(<SiteAdminSection isActive />);

  await screen.findByText("Acme Inc.");
  assert.ok(screen.getByText("Globex"));
  assert.equal(apiMocks.listOrganizations.mock.calls.length, 1);
});

test("does not load organizations while inactive", () => {
  render(<SiteAdminSection isActive={false} />);
  assert.equal(apiMocks.listOrganizations.mock.calls.length, 0);
});

test("creates a new organization and prepends it to the list", async () => {
  const user = userEvent.setup();
  apiMocks.listOrganizations.mockResolvedValue({ organizations: [] });
  apiMocks.createOrganization.mockResolvedValue(
    organization({ id: "org-new", name: "Initech", slug: "initech" }),
  );

  render(<SiteAdminSection isActive />);
  await screen.findByText("No organizations yet");

  await user.type(screen.getByLabelText("Organization name"), "Initech");
  await user.click(screen.getByRole("button", { name: "Create organization" }));

  await screen.findByText("Initech");
  assert.equal(apiMocks.createOrganization.mock.calls[0]?.[0], "Initech");
});

test("archives an active organization", async () => {
  const user = userEvent.setup();
  apiMocks.listOrganizations.mockResolvedValue({
    organizations: [organization()],
  });
  apiMocks.updateOrganization.mockResolvedValue(
    organization({ status: "archived" }),
  );

  render(<SiteAdminSection isActive />);

  const item = (await screen.findByText("Acme Inc.")).closest("li");
  assert.ok(item);
  await user.click(within(item as HTMLElement).getByRole("button", { name: "Archive" }));

  await waitFor(() => {
    assert.deepEqual(apiMocks.updateOrganization.mock.calls[0], [
      "org-1",
      { status: "archived" },
    ]);
  });
  await within(item as HTMLElement).findByRole("button", { name: "Restore" });
});
