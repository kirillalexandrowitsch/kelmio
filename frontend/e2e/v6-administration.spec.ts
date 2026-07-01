import { expect, test, type Page } from "@playwright/test";

const adminLogin = process.env.E2E_ADMIN_LOGIN ?? "admin";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "admin12345";
const apiBaseURL = process.env.E2E_API_BASE_URL ?? "http://localhost:8080";

test("V6 browser: organization administration", async ({ page }) => {
  const runId = Date.now().toString().slice(-7);
  const groupName = `E2E Group ${runId}`;
  const orgName = `E2E Org ${runId}`;

  try {
    await page.goto("/");
    await page.getByLabel("Username or email").fill(adminLogin);
    await page.getByLabel("Password").fill(adminPassword);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.getByRole("heading", { name: /Good to see you/ })).toBeVisible();

    const mainNav = page.getByRole("navigation", { name: "Main navigation" });

    // The shell exposes the workspace switcher.
    await expect(page.getByRole("button", { name: "Switch workspace" })).toBeVisible();

    // Groups: create a group and add a member from the directory.
    await mainNav.getByRole("button", { name: "Groups", exact: true }).click();
    const groups = page.getByRole("region", { name: "Group administration" });
    await groups.getByLabel("Group name").fill(groupName);
    await groups.getByRole("button", { name: "Create group" }).click();
    const groupRow = groups.locator(".site-admin__item").filter({ hasText: groupName });
    await expect(groupRow).toBeVisible();

    await groupRow.getByRole("button", { name: "Members" }).click();
    const memberPanel = page.getByRole("region", { name: "Group members" });
    await expect(memberPanel).toBeVisible();
    await memberPanel.getByLabel("Add member").selectOption({ index: 1 });
    await memberPanel.getByRole("button", { name: "Add" }).click();
    await expect(groupRow.getByText("1 member")).toBeVisible();

    // Workspaces: assign the group a role on the first workspace, then remove it.
    await mainNav.getByRole("button", { name: "Workspaces", exact: true }).click();
    const workspaces = page.getByRole("region", { name: "Organization administration" });
    await workspaces.locator(".site-admin__item").first().getByRole("button", { name: "Roles" }).click();
    const rolesPanel = page.getByRole("region", { name: "Workspace roles" });
    await expect(rolesPanel).toBeVisible();
    await rolesPanel.getByLabel("Subject").selectOption("group");
    await rolesPanel.getByLabel("Group").selectOption({ label: groupName });
    await rolesPanel.getByLabel("Role").selectOption("member");
    await rolesPanel.getByRole("button", { name: "Assign" }).click();
    const assignmentRow = rolesPanel.locator(".site-admin__item").filter({ hasText: groupName });
    await expect(assignmentRow).toBeVisible();
    await assignmentRow.getByRole("button", { name: "Remove" }).click();
    await expect(assignmentRow).toHaveCount(0);

    // Site administration: create an organization, then archive it.
    await mainNav.getByRole("button", { name: "Administration", exact: true }).click();
    const site = page.getByRole("region", { name: "Site administration" });
    await site.getByLabel("Organization name").fill(orgName);
    await site.getByRole("button", { name: "Create organization" }).click();
    const orgRow = site.locator(".site-admin__item").filter({ hasText: orgName });
    await expect(orgRow).toBeVisible();
    await orgRow.getByRole("button", { name: "Archive" }).click();
    await expect(orgRow.getByText("archived")).toBeVisible();

    // Clean up the group through the UI.
    await mainNav.getByRole("button", { name: "Groups", exact: true }).click();
    const cleanupRow = groups.locator(".site-admin__item").filter({ hasText: groupName });
    await cleanupRow.getByRole("button", { name: "Delete" }).click();
    await cleanupRow.getByRole("button", { name: "Confirm" }).click();
    await expect(groups.locator(".site-admin__item").filter({ hasText: groupName })).toHaveCount(0);
  } finally {
    await cleanupGroup(page, groupName);
    await archiveOrganization(page, orgName);
  }
});

async function cleanupGroup(page: Page, name: string) {
  const response = await page.request.get(`${apiBaseURL}/api/v1/groups`);
  if (!response.ok()) {
    return;
  }
  const payload = (await response.json()) as { groups: Array<{ id: string; name: string }> };
  const group = payload.groups.find((current) => current.name === name);
  if (!group) {
    return;
  }
  await page.request.delete(`${apiBaseURL}/api/v1/groups/${group.id}`, {
    headers: await csrfHeaders(page),
  });
}

async function archiveOrganization(page: Page, name: string) {
  const response = await page.request.get(`${apiBaseURL}/api/v1/organizations`);
  if (!response.ok()) {
    return;
  }
  const payload = (await response.json()) as {
    organizations: Array<{ id: string; name: string; status: string }>;
  };
  const organization = payload.organizations.find(
    (current) => current.name === name && current.status === "active",
  );
  if (!organization) {
    return;
  }
  await page.request.patch(`${apiBaseURL}/api/v1/organizations/${organization.id}`, {
    headers: await csrfHeaders(page),
    data: { status: "archived" },
  });
}

async function csrfHeaders(page: Page) {
  const response = await page.request.get(`${apiBaseURL}/api/v1/auth/csrf-token`);
  expect(response.ok()).toBe(true);

  const payload = (await response.json()) as { csrf_token: string };
  return {
    "X-CSRF-Token": payload.csrf_token,
  };
}
