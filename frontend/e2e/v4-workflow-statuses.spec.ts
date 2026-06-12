import { expect, test, type APIResponse, type Page } from "@playwright/test";

import type {
  Project,
  ProjectWorkflow,
  ProjectWorkflowStatus,
} from "../src/lib/api-types";

const adminLogin = process.env.E2E_ADMIN_LOGIN ?? "admin";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "admin12345";
const apiBaseURL = process.env.E2E_API_BASE_URL ?? "http://localhost:8080";

test.setTimeout(60_000);

test("V4 issue controls use project workflow statuses and allowed transitions", async ({
  page,
}) => {
  const runId = newRunId();
  const projectName = `V4 Workflow ${runId}`;
  const issueTitle = `Custom workflow issue ${runId}`;
  let projectId = "";

  try {
    await login(page);
    const project = await createProjectViaApi(
      page,
      `WF${runId.slice(-8).toUpperCase()}`,
      projectName,
    );
    projectId = project.id;
    const review = await createWorkflowStatusViaApi(page, project.id, {
      key: `review_${runId}`,
      name: "Ready for review",
      color: "#0ea5e9",
      category: "in_progress",
    });
    const workflow = await getWorkflowViaApi(page, project.id);
    const done = workflow.statuses.find((status) => status.key === "done");
    if (!done) {
      throw new Error("Default done workflow status is missing");
    }
    await replaceTransitionsViaApi(page, project.id, [
      { from_status_id: review.id, to_status_id: done.id },
    ]);

    await page.reload();
    await openNav(page, "Issues");

    const createForm = page.locator(".issue-form");
    await createForm.getByLabel("Project").selectOption(project.id, { timeout: 5_000 });
    await expect(createForm.getByLabel("Status")).toContainText("Ready for review", {
      timeout: 5_000,
    });
    await createForm.getByLabel("Status").selectOption(review.id, { timeout: 5_000 });
    await createForm.getByLabel("Title").fill(issueTitle, { timeout: 5_000 });
    await createForm
      .getByRole("button", { name: "Create issue" })
      .click({ timeout: 5_000 });

    const detailStatus = page.locator(".issue-detail-status").first().getByRole("combobox");
    await expect(detailStatus).toHaveValue(review.id, { timeout: 5_000 });
    await expect(detailStatus.getByRole("option", { name: "Ready for review" })).toHaveCount(1, { timeout: 5_000 });
    await expect(detailStatus.getByRole("option", { name: "Done" })).toHaveCount(1, { timeout: 5_000 });
    await expect(detailStatus.getByRole("option", { name: "Todo" })).toHaveCount(0, { timeout: 5_000 });

    const filters = page.locator(".issue-filters");
    await filters.getByLabel("Project").selectOption(project.id, { timeout: 5_000 });
    await filters.getByLabel("Status").selectOption(review.id, { timeout: 5_000 });
    await expect(page.locator(".issue-row").filter({ hasText: issueTitle })).toBeVisible({
      timeout: 5_000,
    });

    await detailStatus.selectOption(done.id, { timeout: 5_000 });
    await expect(detailStatus).toHaveValue(done.id, { timeout: 5_000 });
    await expect(page.locator(".issue-row").filter({ hasText: issueTitle })).toHaveCount(0, {
      timeout: 5_000,
    });
  } finally {
    if (projectId) {
      await ensureAdminSession(page);
      await archiveProjectViaApi(page, projectId).catch(() => undefined);
    }
  }
});

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
    .click({ timeout: 5_000 });
}

async function createProjectViaApi(page: Page, key: string, name: string) {
  return expectJson<Project>(
    await page.request.post(`${apiBaseURL}/api/v1/projects`, {
      headers: await csrfHeaders(page),
      data: { key, name, description: "V4 dynamic workflow controls e2e." },
    }),
  );
}

async function createWorkflowStatusViaApi(
  page: Page,
  projectId: string,
  input: Pick<ProjectWorkflowStatus, "key" | "name" | "color" | "category">,
) {
  return expectJson<ProjectWorkflowStatus>(
    await page.request.post(`${apiBaseURL}/api/v1/projects/${projectId}/workflow/statuses`, {
      headers: await csrfHeaders(page),
      data: input,
    }),
  );
}

async function getWorkflowViaApi(page: Page, projectId: string) {
  return expectJson<ProjectWorkflow>(
    await page.request.get(`${apiBaseURL}/api/v1/projects/${projectId}/workflow`),
  );
}

async function replaceTransitionsViaApi(
  page: Page,
  projectId: string,
  transitions: Array<{ from_status_id: string; to_status_id: string }>,
) {
  await expectOk(
    await page.request.put(`${apiBaseURL}/api/v1/projects/${projectId}/workflow/transitions`, {
      headers: await csrfHeaders(page),
      data: { transitions },
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

async function ensureAdminSession(page: Page) {
  const me = await page.request.get(`${apiBaseURL}/api/v1/auth/me`);
  if (me.ok()) {
    return;
  }
  await login(page);
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

function newRunId() {
  return `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`;
}
