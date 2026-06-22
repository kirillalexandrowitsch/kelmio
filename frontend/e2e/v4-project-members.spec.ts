import {
  expect,
  test,
  type APIResponse,
  type Locator,
  type Page,
} from "@playwright/test";

import type { Project, TeamMember } from "../src/lib/api-types";

const adminLogin = process.env.E2E_ADMIN_LOGIN ?? "admin";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "admin12345";
const apiBaseURL = process.env.E2E_API_BASE_URL ?? "http://localhost:8080";

test("V4 project members UI: admin and lead manage access while viewer is read-only", async ({
  page,
}) => {
  const runId = newRunId();
  const projectKey = `PM${runId.toUpperCase()}`;
  const projectName = `V4 Project Members ${runId}`;
  const managed = memberInput(`v4_managed_${runId}`, `V4 Managed ${runId}`);
  const lead = memberInput(`v4_lead_${runId}`, `V4 Lead ${runId}`);
  const viewer = memberInput(`v4_viewer_${runId}`, `V4 Viewer ${runId}`);
  let projectId = "";
  const memberIds: string[] = [];

  try {
    await login(page);
    const project = await createProjectViaApi(page, projectKey, projectName);
    projectId = project.id;
    const managedMember = await createTeamMemberViaApi(page, managed);
    const leadMember = await createTeamMemberViaApi(page, lead);
    const viewerMember = await createTeamMemberViaApi(page, viewer);
    memberIds.push(managedMember.id, leadMember.id, viewerMember.id);

    await page.reload();
    await openProjectMembers(page, projectName);

    await addProjectMember(page, managedMember.id, "contributor");
    const managedCard = projectMemberCard(page, managed.email);
    await managedCard
      .getByLabel(`Project role for ${managed.displayName}`)
      .selectOption("viewer");
    await expect(
      managedCard.getByLabel(`Project role for ${managed.displayName}`),
    ).toHaveValue("viewer");
    await removeProjectMember(page, managedCard);

    await addProjectMember(page, leadMember.id, "lead");
    await addProjectMember(page, viewerMember.id, "viewer");

    await logoutViaApi(page);
    await login(page, lead.username, lead.password);
    await openProjectMembers(page, projectName);
    await expect(page.getByRole("tab", { name: "Workflow" })).toBeVisible();
    await expect(page.getByRole("tab", { name: "Automation" })).toBeVisible();

    const viewerCard = projectMemberCard(page, viewer.email);
    await viewerCard
      .getByLabel(`Project role for ${viewer.displayName}`)
      .selectOption("contributor");
    await expect(
      viewerCard.getByLabel(`Project role for ${viewer.displayName}`),
    ).toHaveValue("contributor");
    await viewerCard
      .getByLabel(`Project role for ${viewer.displayName}`)
      .selectOption("viewer");

    await logoutViaApi(page);
    await login(page, viewer.username, viewer.password);
    await openProjectDetail(page, projectName);
    await expect(page.getByRole("tab", { name: "Members" })).toHaveCount(0);
    await expect(page.getByRole("tab", { name: "Workflow" })).toHaveCount(0);
    await expect(page.getByRole("tab", { name: "Automation" })).toHaveCount(0);
    await expect(page.getByText("Viewer access")).toBeVisible();
    await expect(page.getByText(/This project is read-only/)).toBeVisible();

    await logoutViaApi(page);
    await login(page, lead.username, lead.password);
    await openProjectMembers(page, projectName);
    await removeProjectMember(page, projectMemberCard(page, viewer.email));

    await logoutViaApi(page);
    await login(page, viewer.username, viewer.password);
    await openNav(page, "Projects");
    await expect(page.locator(".project-row").filter({ hasText: projectName })).toHaveCount(
      0,
    );
  } finally {
    await ensureAdminSession(page);
    if (projectId) {
      await archiveProjectViaApi(page, projectId).catch(() => undefined);
    }
    for (const memberId of memberIds) {
      await deactivateMemberViaApi(page, memberId).catch(() => undefined);
    }
  }
});

async function openProjectMembers(page: Page, projectName: string) {
  await openProjectDetail(page, projectName);
  await page.getByRole("tab", { name: "Members" }).click();
  await expect(page.getByRole("region", { name: "Project members" })).toBeVisible();
}

async function openProjectDetail(page: Page, projectName: string) {
  await openNav(page, "Projects");
  const projectRow = page.locator(".project-row").filter({ hasText: projectName });
  await expect(projectRow).toBeVisible();
  await projectRow.getByRole("button", { name: "Details" }).click();
  await expect(
    page
      .getByRole("complementary", { name: "Project details" })
      .getByRole("heading", {
        name: new RegExp(projectName),
      }),
  ).toBeVisible();
}

async function addProjectMember(page: Page, userId: string, role: string) {
  const panel = page.getByRole("region", { name: "Project members" });
  await panel.getByLabel("Workspace member").selectOption(userId);
  await panel.getByLabel("New project member role").selectOption(role);
  await panel.getByRole("button", { name: "Add member" }).click();
  await expect(panel.getByLabel("Workspace member")).toHaveValue("");
}

function projectMemberCard(page: Page, email: string) {
  return page.locator(".project-member-card").filter({ hasText: email });
}

async function removeProjectMember(page: Page, card: Locator) {
  page.once("dialog", (dialog) => dialog.accept());
  await card.getByRole("button", { name: "Remove" }).click();
  await expect(card).toHaveCount(0);
}

async function login(page: Page, loginValue = adminLogin, password = adminPassword) {
  await page.goto("/");
  await page.getByLabel("Username or email").fill(loginValue);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("heading", { name: /Good to see you/ })).toBeVisible();
}

async function openNav(page: Page, sectionName: string) {
  await page
    .getByRole("navigation", { name: "Main navigation" })
    .getByRole("button", { name: sectionName, exact: true })
    .click();
}

async function ensureAdminSession(page: Page) {
  await logoutViaApi(page).catch(() => undefined);
  await login(page);
}

async function createProjectViaApi(page: Page, key: string, name: string) {
  return expectJson<Project>(
    await page.request.post(`${apiBaseURL}/api/v1/projects`, {
      headers: await csrfHeaders(page),
      data: {
        key,
        name,
        description: "Created by V4 project member browser e2e.",
      },
    }),
  );
}

async function createTeamMemberViaApi(
  page: Page,
  input: ReturnType<typeof memberInput>,
) {
  return expectJson<TeamMember>(
    await page.request.post(`${apiBaseURL}/api/v1/team/members`, {
      headers: await csrfHeaders(page),
      data: {
        email: input.email,
        username: input.username,
        display_name: input.displayName,
        password: input.password,
        role: "member",
      },
    }),
  );
}

async function archiveProjectViaApi(page: Page, projectId: string) {
  await expectOk(
    await page.request.post(`${apiBaseURL}/api/v1/projects/${projectId}/archive`, {
      headers: await csrfHeaders(page),
    }),
  );
}

async function deactivateMemberViaApi(page: Page, memberId: string) {
  await expectJson<TeamMember>(
    await page.request.patch(`${apiBaseURL}/api/v1/team/members/${memberId}`, {
      headers: await csrfHeaders(page),
      data: {
        role: "member",
        is_active: false,
      },
    }),
  );
}

async function logoutViaApi(page: Page) {
  await expectOk(
    await page.request.post(`${apiBaseURL}/api/v1/auth/logout`, {
      headers: await csrfHeaders(page),
    }),
  );
}

async function csrfHeaders(page: Page) {
  const response = await page.request.get(`${apiBaseURL}/api/v1/auth/csrf-token`);
  await expectOk(response);
  const payload = (await response.json()) as { csrf_token: string };
  return { "X-CSRF-Token": payload.csrf_token };
}

async function expectJson<T>(response: APIResponse) {
  await expectOk(response);
  return (await response.json()) as T;
}

async function expectOk(response: APIResponse) {
  if (!response.ok()) {
    throw new Error(
      `Expected API request to succeed, got ${response.status()}: ${await response.text()}`,
    );
  }
}

function memberInput(username: string, displayName: string) {
  return {
    username,
    displayName,
    email: `${username}@example.com`,
    password: "member12345",
  };
}

function newRunId() {
  return `${Date.now().toString(36).slice(-4)}${Math.random()
    .toString(36)
    .slice(2, 4)}`;
}
