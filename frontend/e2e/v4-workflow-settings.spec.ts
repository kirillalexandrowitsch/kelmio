import { expect, test, type APIResponse, type Locator, type Page } from "@playwright/test";

import type {
  Issue,
  Project,
  ProjectWorkflow,
  ProjectWorkflowStatus,
} from "../src/lib/api-types";

const adminLogin = process.env.E2E_ADMIN_LOGIN ?? "admin";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "admin12345";
const apiBaseURL = process.env.E2E_API_BASE_URL ?? "http://localhost:8080";

test.setTimeout(75_000);

test("V4 workflow settings UI manages statuses transitions and archive replacement", async ({
  page,
}) => {
  const runId = newRunId();
  const projectName = `V4 Workflow Settings ${runId}`;
  const issueTitle = `Workflow settings issue ${runId}`;
  const statusKey = `qa_${runId}`;
  let projectId = "";

  try {
    await login(page);
    const project = await createProjectViaApi(
      page,
      `WS${runId.toUpperCase()}`,
      projectName,
    );
    projectId = project.id;
    await page.reload();
    const panel = await openWorkflowSettings(page, projectName);

    await panel.getByLabel("New status key").fill(statusKey);
    await panel.getByLabel("New status name").fill("Ready for QA");
    await panel.getByLabel("New status category").selectOption("in_progress");
    await panel.getByRole("button", { name: "Create status" }).click();
    await expect(panel.getByLabel("Name for Ready for QA")).toBeVisible();

    const qaCard = statusCard(panel, statusKey);
    await qaCard.getByLabel("Name for Ready for QA").fill("QA review");
    await qaCard.getByRole("button", { name: "Save" }).click();
    await expect(panel.getByLabel("Name for QA review")).toBeVisible();
    await panel.getByRole("button", { name: "Move QA review up" }).click();

    const workflow = await getWorkflowViaApi(page, project.id);
    const todo = requiredStatus(workflow, "todo");
    const done = requiredStatus(workflow, "done");
    const qa = requiredStatus(workflow, statusKey);

    const checkedTransitions = panel.locator('input[type="checkbox"]:checked');
    while ((await checkedTransitions.count()) > 0) {
      await checkedTransitions.first().uncheck();
    }
    await panel.getByLabel("Allow Todo to QA review").check();
    await panel.getByLabel("Allow QA review to Done").check();
    await panel.getByRole("button", { name: "Save transitions" }).click();

    const issue = await createIssueViaApi(page, project, issueTitle, todo.id);
    await page.goto(`/board?projectId=${encodeURIComponent(project.id)}`);
    const card = page.locator(".issue-card").filter({ hasText: issueTitle });
    const boardStatus = card.getByLabel(`Status for ${issue.issue_key}`);
    await expect(boardStatus.getByRole("option", { name: "QA review" })).toHaveCount(1);
    await expect(boardStatus.getByRole("option", { name: "Done" })).toHaveCount(0);
    await boardStatus.selectOption(qa.id);
    await expect(boardStatus).toHaveValue(qa.id);

    const refreshedPanel = await openWorkflowSettings(page, projectName);
    const refreshedQaCard = statusCard(refreshedPanel, statusKey);
    await refreshedQaCard.getByRole("button", { name: "Archive" }).click();
    await refreshedPanel.getByLabel("Replacement status").selectOption(done.id);
    await refreshedPanel.getByRole("button", { name: "Confirm archive" }).click();
    await expect(
      refreshedPanel.getByRole("region", { name: "Archived workflow statuses" }),
    ).toContainText("QA review");

    const replacedIssue = await getIssueViaApi(page, issue.id);
    expect(replacedIssue.workflow_status.id).toBe(done.id);
  } finally {
    if (projectId) {
      await archiveProjectViaApi(page, projectId).catch(() => undefined);
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

function statusCard(panel: Locator, key: string) {
  return panel.locator(".workflow-status-card").filter({ hasText: key });
}

async function login(page: Page) {
  await page.goto("/");
  await page.getByLabel("Username or email").fill(adminLogin);
  await page.getByLabel("Password").fill(adminPassword);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("heading", { name: /Good to see you/ })).toBeVisible();
}

async function openNav(page: Page, sectionName: string) {
  await page
    .getByRole("navigation", { name: "Main navigation" })
    .getByRole("button", { name: sectionName, exact: true })
    .click();
}

async function createProjectViaApi(page: Page, key: string, name: string) {
  return expectJson<Project>(
    await page.request.post(`${apiBaseURL}/api/v1/projects`, {
      headers: await csrfHeaders(page),
      data: { key, name, description: "V4 workflow settings browser e2e." },
    }),
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
        description: "Created by V4 workflow settings browser e2e.",
        issue_type: "task",
        workflow_status_id: workflowStatusId,
        priority: "medium",
        story_points: 3,
        assignee_id: "",
        due_date: "",
        label_ids: [],
      },
    }),
  );
}

async function getIssueViaApi(page: Page, issueId: string) {
  return expectJson<Issue>(
    await page.request.get(`${apiBaseURL}/api/v1/issues/${issueId}`),
  );
}

async function archiveProjectViaApi(page: Page, projectId: string) {
  await expectOk(
    await page.request.post(`${apiBaseURL}/api/v1/projects/${projectId}/archive`, {
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
  const status = workflow.statuses.find((item) => item.key === key);
  if (!status) {
    throw new Error(`Workflow status ${key} is missing`);
  }
  return status as ProjectWorkflowStatus;
}

function newRunId() {
  return `${Date.now().toString(36).slice(-4)}${Math.random()
    .toString(36)
    .slice(2, 4)}`;
}
