import { expect, test, type APIResponse, type Locator, type Page } from "@playwright/test";

import type { Issue, Project, ProjectWorkflow } from "../src/lib/api-types";

const adminLogin = process.env.E2E_ADMIN_LOGIN ?? "admin";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "admin12345";
const apiBaseURL = process.env.E2E_API_BASE_URL ?? "http://localhost:8080";

test.setTimeout(75_000);

test("V4 automation settings UI manages rules and runs synchronous automation", async ({
  page,
}) => {
  const runId = newRunId();
  const projectName = `V4 Automation ${runId}`;
  let projectId = "";

  try {
    await login(page);
    const project = await createProjectViaApi(
      page,
      `AU${runId.toUpperCase()}`,
      projectName,
    );
    projectId = project.id;
    const workflow = await getWorkflowViaApi(page, project.id);
    const todoId = requiredStatusID(workflow, "todo");

    await page.reload();
    const panel = await openAutomationSettings(page, projectName);
    await createPriorityRule(panel, `Escalate ${runId}`, "critical");

    const automatedIssue = await createIssueViaApi(
      page,
      project,
      `Automated issue ${runId}`,
      todoId,
    );
    expect(automatedIssue.priority).toBe("critical");

    const ruleCard = panel
      .locator(".automation-rule-card")
      .filter({ hasText: `Escalate ${runId}` });
    await ruleCard.getByRole("button", { name: "Disable" }).click();
    await expect(ruleCard.getByText("Disabled", { exact: true })).toBeVisible();

    const unchangedIssue = await createIssueViaApi(
      page,
      project,
      `Unchanged issue ${runId}`,
      todoId,
    );
    expect(unchangedIssue.priority).toBe("medium");

    await ruleCard.getByRole("button", { name: "Edit" }).click();
    await panel.getByLabel("Automation rule name").fill(`Escalate updated ${runId}`);
    await panel.getByLabel("Enabled after save").check();
    await panel.getByRole("button", { name: "Save rule" }).click();
    await expect(panel.getByText(`Escalate updated ${runId}`)).toBeVisible();

    await createPriorityRule(panel, `Second ${runId}`, "high");
    await panel
      .getByRole("button", { name: `Move Second ${runId} up` })
      .click();

    const updatedCard = panel
      .locator(".automation-rule-card")
      .filter({ hasText: `Escalate updated ${runId}` });
    page.once("dialog", (dialog) => dialog.accept());
    await updatedCard.getByRole("button", { name: "Delete" }).click();
    await expect(updatedCard).toHaveCount(0);
  } finally {
    if (projectId) {
      await archiveProjectViaApi(page, projectId).catch(() => undefined);
    }
  }
});

async function openAutomationSettings(page: Page, projectName: string) {
  await openNav(page, "Projects");
  const projectRow = page.locator(".project-row").filter({ hasText: projectName });
  await expect(projectRow).toBeVisible();
  await projectRow.getByRole("button", { name: "Details" }).click();
  await page.getByRole("tab", { name: "Automation" }).click();
  const panel = page.getByRole("region", { name: "Automation settings" });
  await expect(panel).toBeVisible();
  await expect(panel.getByText("Refreshing")).toBeHidden();
  return panel;
}

async function createPriorityRule(
  panel: Locator,
  name: string,
  priority: "high" | "critical",
) {
  await panel.getByLabel("Automation rule name").fill(name);
  await panel.getByRole("button", { name: "Add action" }).click();
  const actionCount = await panel.locator(".automation-item-row").count();
  await panel
    .getByLabel(`Action ${actionCount} type`)
    .selectOption("change_priority");
  await panel
    .getByLabel("change priority value")
    .last()
    .selectOption(priority);
  await panel.getByRole("button", { name: "Create rule" }).click();
  await expect(panel.getByText(name)).toBeVisible();
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
      data: { key, name, description: "V4 automation settings browser e2e." },
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
        description: "Created by V4 automation settings browser e2e.",
        issue_type: "task",
        workflow_status_id: workflowStatusId,
        priority: "medium",
        story_points: 1,
        assignee_id: "",
        due_date: "",
        label_ids: [],
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

function requiredStatusID(workflow: ProjectWorkflow, key: string) {
  const status = workflow.statuses.find((candidate) => candidate.key === key);
  if (!status) {
    throw new Error(`Workflow status ${key} is missing`);
  }
  return status.id;
}

function newRunId() {
  return `${Date.now().toString(36).slice(-4)}${Math.random()
    .toString(36)
    .slice(2, 4)}`;
}
