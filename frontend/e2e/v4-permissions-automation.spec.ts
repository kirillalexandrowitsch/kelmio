import { expect, test, type APIResponse, type Page } from "@playwright/test";

import type {
  Issue,
  Project,
  ProjectWorkflow,
  ProjectWorkflowStatus,
  TeamMember,
} from "../src/lib/api-types";

const adminLogin = process.env.E2E_ADMIN_LOGIN ?? "admin";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "admin12345";
const apiBaseURL = process.env.E2E_API_BASE_URL ?? "http://localhost:8080";

test.setTimeout(90_000);

test("V4 permissions enforce lead workflow contributor transitions and viewer read-only UI", async ({
  page,
}) => {
  const runId = newRunId();
  const projectName = `V4 Permissions ${runId}`;
  const lead = memberInput(`v4_flow_lead_${runId}`, `V4 Flow Lead ${runId}`);
  const contributor = memberInput(
    `v4_flow_contributor_${runId}`,
    `V4 Flow Contributor ${runId}`,
  );
  const viewer = memberInput(`v4_flow_viewer_${runId}`, `V4 Flow Viewer ${runId}`);
  let projectId = "";
  const memberIds: string[] = [];

  try {
    await login(page);
    const project = await createProjectViaApi(
      page,
      `VP${runId.toUpperCase()}`,
      projectName,
    );
    projectId = project.id;
    const leadMember = await createTeamMemberViaApi(page, lead);
    const contributorMember = await createTeamMemberViaApi(page, contributor);
    const viewerMember = await createTeamMemberViaApi(page, viewer);
    memberIds.push(leadMember.id, contributorMember.id, viewerMember.id);
    await putProjectMemberViaApi(page, project.id, leadMember.id, "lead");
    await putProjectMemberViaApi(page, project.id, contributorMember.id, "contributor");
    await putProjectMemberViaApi(page, project.id, viewerMember.id, "viewer");

    await logoutViaApi(page);
    await login(page, lead.username, lead.password);
    const panel = await openWorkflowSettings(page, projectName);
    const statusKey = `review_${runId}`;
    await panel.getByLabel("New status key").fill(statusKey);
    await panel.getByLabel("New status name").fill("Ready for review");
    await panel.getByLabel("New status category").selectOption("in_progress");
    await panel.getByRole("button", { name: "Create status" }).click();
    await expect(panel.getByLabel("Name for Ready for review")).toBeVisible();

    const checkedTransitions = panel.locator('input[type="checkbox"]:checked');
    while ((await checkedTransitions.count()) > 0) {
      await checkedTransitions.first().uncheck();
    }
    await panel.getByLabel("Allow Todo to Ready for review").check();
    await panel.getByLabel("Allow Ready for review to Done").check();
    await panel.getByRole("button", { name: "Save transitions" }).click();

    const workflow = await getWorkflowViaApi(page, project.id);
    const todo = requiredStatus(workflow, "todo");
    const review = requiredStatus(workflow, statusKey);

    await logoutViaApi(page);
    await login(page, contributor.username, contributor.password);
    const issue = await createIssueViaApi(
      page,
      project,
      `Contributor issue ${runId}`,
      todo.id,
    );
    await page.goto(`/board?projectId=${encodeURIComponent(project.id)}`);
    const card = page.locator(".issue-card").filter({ hasText: issue.title });
    const boardStatus = card.getByLabel(`Status for ${issue.issue_key}`);
    await expect(
      boardStatus.getByRole("option", { name: "Ready for review" }),
    ).toHaveCount(1);
    await expect(boardStatus.getByRole("option", { name: "Done" })).toHaveCount(0);
    await boardStatus.selectOption(review.id);
    await expect(boardStatus).toHaveValue(review.id);

    const forbidden = await page.request.post(
      `${apiBaseURL}/api/v1/issues/${issue.id}/transition`,
      {
        headers: await csrfHeaders(page),
        data: { workflow_status_id: todo.id },
      },
    );
    expect(forbidden.status()).toBe(409);
    expect((await forbidden.json()).error.code).toBe("transition_not_allowed");

    await logoutViaApi(page);
    await login(page, viewer.username, viewer.password);
    await page.goto(`/board?projectId=${encodeURIComponent(project.id)}`);
    await expect(
      page.getByText("This board is read-only for your project role."),
    ).toBeVisible();
    const viewerStatus = page
      .locator(".issue-card")
      .filter({ hasText: issue.title })
      .getByLabel(`Status for ${issue.issue_key}`);
    await expect(viewerStatus).toBeDisabled();

    await openIssueFromList(page, project.id, issue.title);
    await expect(
      page.locator(".issue-detail-status").first().getByRole("combobox"),
    ).toBeDisabled();
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

test("V4 automation result is visible in issue activity and recipient notifications", async ({
  page,
}) => {
  const runId = newRunId();
  const projectName = `V4 Automation Result ${runId}`;
  const recipient = memberInput(
    `v4_automation_recipient_${runId}`,
    `V4 Automation Recipient ${runId}`,
  );
  let projectId = "";
  let recipientId = "";

  try {
    await login(page);
    const project = await createProjectViaApi(
      page,
      `VA${runId.toUpperCase()}`,
      projectName,
    );
    projectId = project.id;
    const recipientMember = await createTeamMemberViaApi(page, recipient);
    recipientId = recipientMember.id;
    await putProjectMemberViaApi(page, project.id, recipientMember.id, "contributor");
    const workflow = await getWorkflowViaApi(page, project.id);
    const todo = requiredStatus(workflow, "todo");

    await createAutomationRuleViaApi(page, project.id, `Assign recipient ${runId}`, [
      { type: "change_assignee", user_id: recipientMember.id },
      { type: "change_priority", value: "critical" },
    ]);
    const issue = await createIssueViaApi(
      page,
      project,
      `Automated activity ${runId}`,
      todo.id,
    );
    expect(issue.assignee_id).toBe(recipientMember.id);
    expect(issue.priority).toBe("critical");

    await page.reload();
    await openIssueFromList(page, project.id, issue.title);
    const activity = page.getByRole("region", { name: "Issue activity" });
    await expect(activity.getByText("Automation applied")).toBeVisible();
    await expect(activity.getByText(/System · Assign recipient/)).toBeVisible();

    await logoutViaApi(page);
    await login(page, recipient.username, recipient.password);
    await openNav(page, "Notifications");
    const notifications = page.getByRole("region", { name: "Notifications" });
    await expect(
      notifications.getByRole("heading", {
        name: "Automation assigned you an issue",
      }),
    ).toBeVisible();
    await expect(notifications.getByText(`Assign recipient ${runId}`)).toBeVisible();
  } finally {
    await ensureAdminSession(page);
    if (projectId) {
      await archiveProjectViaApi(page, projectId).catch(() => undefined);
    }
    if (recipientId) {
      await deactivateMemberViaApi(page, recipientId).catch(() => undefined);
    }
  }
});

async function openWorkflowSettings(page: Page, projectName: string) {
  await openNav(page, "Projects");
  const projectRow = page.locator(".project-row").filter({ hasText: projectName });
  await expect(projectRow).toBeVisible();
  await projectRow.getByRole("button", { name: "Details" }).click();
  await page.getByRole("tab", { name: "Workflow" }).click();
  const panel = page.getByRole("region", { name: "Workflow settings" });
  await expect(panel).toBeVisible();
  await expect(panel.getByText("Refreshing")).toBeHidden();
  return panel;
}

async function openIssueFromList(page: Page, projectId: string, title: string) {
  await openNav(page, "Issues");
  const filters = page.locator(".issue-filters");
  await filters.getByLabel("Project").selectOption(projectId);
  const row = page.locator(".issue-row").filter({ hasText: title });
  await expect(row).toBeVisible();
  await row.getByRole("button", { name: "Open" }).click();
  await expect(page.getByRole("region", { name: "Issue details" })).toBeVisible();
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
      data: { key, name, description: "V4 permissions and automation browser e2e." },
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

async function putProjectMemberViaApi(
  page: Page,
  projectId: string,
  userId: string,
  role: "lead" | "contributor" | "viewer",
) {
  await expectOk(
    await page.request.put(
      `${apiBaseURL}/api/v1/projects/${projectId}/members/${userId}`,
      {
        headers: await csrfHeaders(page),
        data: { role },
      },
    ),
  );
}

async function getWorkflowViaApi(page: Page, projectId: string) {
  return expectJson<ProjectWorkflow>(
    await page.request.get(`${apiBaseURL}/api/v1/projects/${projectId}/workflow`),
  );
}

async function createIssueViaApi(
  page: Page,
  project: Project,
  title: string,
  workflowStatusId: string,
) {
  return expectJson<Issue>(
    await page.request.post(`${apiBaseURL}/api/v1/issues`, {
      headers: await csrfHeaders(page),
      data: {
        project_id: project.id,
        title,
        description: "Created by V4 permissions and automation browser e2e.",
        issue_type: "task",
        workflow_status_id: workflowStatusId,
        priority: "medium",
        story_points: 2,
        assignee_id: "",
        due_date: "",
        label_ids: [],
      },
    }),
  );
}

async function createAutomationRuleViaApi(
  page: Page,
  projectId: string,
  name: string,
  actions: Array<Record<string, string>>,
) {
  await expectOk(
    await page.request.post(
      `${apiBaseURL}/api/v1/projects/${projectId}/automation-rules`,
      {
        headers: await csrfHeaders(page),
        data: {
          name,
          trigger_type: "issue_created",
          conditions: [],
          actions,
          is_enabled: true,
        },
      },
    ),
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
  await expectOk(
    await page.request.patch(`${apiBaseURL}/api/v1/team/members/${memberId}`, {
      headers: await csrfHeaders(page),
      data: { role: "member", is_active: false },
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
    throw new Error(`${response.status()} ${await response.text()}`);
  }
}

function requiredStatus(workflow: ProjectWorkflow, key: string) {
  const status = workflow.statuses.find((candidate) => candidate.key === key);
  if (!status) {
    throw new Error(`Workflow status ${key} is missing`);
  }
  return status as ProjectWorkflowStatus;
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
