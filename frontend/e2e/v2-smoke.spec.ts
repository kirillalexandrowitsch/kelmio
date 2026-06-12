import { expect, test, type APIResponse, type Page } from "@playwright/test";

import type {
  Issue,
  IssuePriority,
  IssueStatus,
  IssueType,
  Project,
  TeamMember,
} from "../src/lib/api-types";

const adminLogin = process.env.E2E_ADMIN_LOGIN ?? "admin";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "admin12345";
const apiBaseURL = process.env.E2E_API_BASE_URL ?? "http://localhost:8080";

test("V2 browser smoke: hierarchy and links", async ({ page }) => {
  const runId = newRunId();
  const projectKey = `VH${runId.toUpperCase()}`;
  const subtaskTitle = `V2 subtask ${runId}`;
  let projectId = "";

  try {
    await login(page);
    const project = await createProjectViaApi(
      page,
      projectKey,
      `V2 Hierarchy ${runId}`,
    );
    projectId = project.id;
    const epic = await createIssueViaApi(page, {
      project_id: project.id,
      title: `V2 epic ${runId}`,
      issue_type: "epic",
      priority: "high",
      story_points: 13,
    });
    const story = await createIssueViaApi(page, {
      project_id: project.id,
      parent_issue_id: epic.id,
      title: `V2 child story ${runId}`,
      issue_type: "story",
      priority: "high",
      story_points: 8,
    });
    const blocker = await createIssueViaApi(page, {
      project_id: project.id,
      title: `V2 linked blocker ${runId}`,
      issue_type: "bug",
      status: "blocked",
      priority: "critical",
      story_points: 3,
    });

    await reloadWorkspace(page);
    await openNav(page, "Issues");
    const storyRow = page.locator(".issue-row").filter({ hasText: story.title });
    await expect(storyRow).toBeVisible();
    await storyRow.getByRole("button", { name: "Open" }).click();

    await expect(
      page.getByRole("heading", {
        name: new RegExp(`${projectKey}-\\d+ · ${story.title}`),
      }),
    ).toBeVisible();
    await expect(page.locator(".hierarchy-parent-card")).toContainText(epic.title);

    const subtaskForm = page.locator(".subtask-form");
    await subtaskForm.getByLabel("New subtask").fill(subtaskTitle);
    await subtaskForm.getByLabel("Points").fill("2");
    await subtaskForm.getByRole("button", { name: "Create subtask" }).click();
    await expect(page.locator(".hierarchy-child-list")).toContainText(subtaskTitle);

    const linkForm = page.locator(".issue-link-form");
    await linkForm.getByLabel("Relationship").selectOption("blocks");
    await linkForm.getByLabel("Target issue").selectOption(blocker.id);
    await linkForm.getByRole("button", { name: "Add link" }).click();
    await expect(page.locator(".linked-issue-card")).toContainText(blocker.title);
    await expect(page.locator(".linked-issue-card")).toContainText("blocks");
    await expect(
      page.getByRole("region", { name: "Issue activity" }).getByText("Linked issue"),
    ).toBeVisible();
  } finally {
    if (projectId) {
      await archiveProjectViaApi(page, projectId);
    }
  }
});

test("V2 browser smoke: sprint workflow", async ({ page }) => {
  const runId = newRunId();
  const projectKey = `VS${runId.toUpperCase()}`;
  const sprintName = `V2 sprint ${runId}`;
  let projectId = "";

  try {
    await login(page);
    const project = await createProjectViaApi(
      page,
      projectKey,
      `V2 Sprint ${runId}`,
    );
    projectId = project.id;
    const firstIssue = await createIssueViaApi(page, {
      project_id: project.id,
      title: `V2 sprint story ${runId}`,
      issue_type: "story",
      priority: "high",
      story_points: 5,
    });
    const secondIssue = await createIssueViaApi(page, {
      project_id: project.id,
      title: `V2 sprint task ${runId}`,
      issue_type: "task",
      priority: "medium",
      story_points: 3,
    });

    await reloadWorkspace(page);
    await openNav(page, "Sprints");
    const sprintForm = page.locator(".sprint-form");
    await sprintForm.getByLabel("Project").selectOption(project.id);
    await sprintForm.getByLabel("Name").fill(sprintName);
    await sprintForm.getByLabel("Goal").fill("Exercise V2 sprint workflow.");
    await sprintForm.getByLabel("Start date").fill(dateOnly(0));
    await sprintForm.getByLabel("End date").fill(dateOnly(7));
    await sprintForm.getByRole("button", { name: "Create sprint" }).click();

    const sprintRow = page.locator(".sprint-row").filter({ hasText: sprintName });
    await expect(sprintRow).toBeVisible();
    await sprintRow.getByRole("button", { name: "Details" }).click();
    await expect(
      page.locator(".sprint-detail-panel").getByRole("heading", { name: sprintName }),
    ).toBeVisible();

    await addPlanningIssue(page, firstIssue.title);
    await addPlanningIssue(page, secondIssue.title);
    await expect(page.getByRole("region", { name: "Sprint points summary" })).toContainText(
      "Total points",
    );

    await page.getByRole("button", { name: "Start sprint" }).click();
    await expect(
      page.getByRole("region", { name: "Active sprint board" }),
    ).toContainText(firstIssue.title);

    const firstCard = page.locator(".active-sprint-card").filter({
      hasText: firstIssue.title,
    });
    await firstCard
      .getByLabel(new RegExp(`Status for ${projectKey}-\\d+`))
      .selectOption("done");
    await expect(firstCard.getByText("5 points")).toBeVisible();
    await expect(page.getByRole("region", { name: "Sprint points summary" })).toContainText(
      "Done points",
    );

    await page.getByRole("button", { name: "Complete sprint" }).click();
    await expect(
      page.getByRole("region", { name: "Active sprint board" }),
    ).toContainText("Completed sprint board is locked for history.");
  } finally {
    if (projectId) {
      await archiveProjectViaApi(page, projectId);
    }
  }
});

test("V2 browser smoke: saved filters", async ({ page }) => {
  const runId = newRunId();
  const projectKey = `VF${runId.toUpperCase()}`;
  const filterName = `V2 saved view ${runId}`;
  let projectId = "";

  try {
    await login(page);
    const project = await createProjectViaApi(
      page,
      projectKey,
      `V2 Filters ${runId}`,
    );
    projectId = project.id;
    const issue = await createIssueViaApi(page, {
      project_id: project.id,
      title: `V2 filtered issue ${runId}`,
      issue_type: "task",
      status: "blocked",
      priority: "critical",
      story_points: 2,
    });

    await reloadWorkspace(page);
    await openNav(page, "Issues");
    const filters = page.getByRole("region", { name: "Issue filters" });
    await filters.getByLabel("Search").fill(issue.title);
    await filters.getByLabel("Sort").selectOption("priority_desc");
    await filters.getByLabel("Project").selectOption(project.id);
    await filters.getByLabel("Status").selectOption({ label: "Blocked" });
    await expect(page.locator(".issue-row").filter({ hasText: issue.title })).toBeVisible();

    const savedFilters = page.getByRole("region", { name: "Saved issue filters" });
    await savedFilters.getByLabel("View name").fill(filterName);
    await savedFilters.getByRole("button", { name: "Save current view" }).click();
    const savedFilterCard = page.locator(".saved-filter-card").filter({
      hasText: filterName,
    });
    await expect(savedFilterCard).toBeVisible();

    await filters.getByRole("button", { name: "Clear" }).click();
    await filters.getByLabel("Search").fill(`missing-${runId}`);
    await expect(page.locator(".issue-row").filter({ hasText: issue.title })).toHaveCount(
      0,
    );

    await savedFilterCard.getByRole("button", { name: "Apply" }).click();
    await expect(filters.getByLabel("Search")).toHaveValue(issue.title);
    await expect(page.locator(".issue-row").filter({ hasText: issue.title })).toBeVisible();

    await savedFilterCard.getByRole("button", { name: "Delete" }).click();
    await expect(savedFilterCard).toHaveCount(0);
  } finally {
    if (projectId) {
      await archiveProjectViaApi(page, projectId);
    }
  }
});

test("V2 browser smoke: notifications", async ({ page }) => {
  const runId = newRunId();
  const projectKey = `VN${runId.toUpperCase()}`;
  const memberUsername = `v2_member_${runId}`;
  const memberPassword = "smoke12345";
  let projectId = "";
  let memberId = "";

  try {
    await login(page);
    const member = await createTeamMemberViaApi(
      page,
      memberUsername,
      `V2 Member ${runId}`,
      memberPassword,
    );
    memberId = member.id;
    const project = await createProjectViaApi(
      page,
      projectKey,
      `V2 Notifications ${runId}`,
    );
    projectId = project.id;
    await expectOk(
      await page.request.put(
        `${apiBaseURL}/api/v1/projects/${project.id}/members/${member.id}`,
        {
          headers: await csrfHeaders(page),
          data: { role: "contributor" },
        },
      ),
    );
    const issue = await createIssueViaApi(page, {
      project_id: project.id,
      title: `V2 notification issue ${runId}`,
      issue_type: "task",
      priority: "high",
      story_points: 1,
    });

    await expectJson<Issue>(
      await page.request.post(`${apiBaseURL}/api/v1/issues/${issue.id}/assign`, {
        headers: await csrfHeaders(page),
        data: { assignee_id: member.id },
      }),
    );
    await expectJson(
      await page.request.post(`${apiBaseURL}/api/v1/issues/${issue.id}/comments`, {
        headers: await csrfHeaders(page),
        data: { body: `@${memberUsername} please check V2 notifications.` },
      }),
    );

    await logoutViaApi(page);
    await login(page, memberUsername, memberPassword);

    const notificationToggle = page.locator(".notification-toggle");
    await expect(notificationToggle.locator(".notification-badge")).toHaveText("2");
    await notificationToggle.click();

    const dropdown = page.getByRole("region", { name: "Notification dropdown" });
    await expect(dropdown.getByText(/assigned you an issue/)).toBeVisible();
    await expect(dropdown.getByText(/mentioned you/)).toBeVisible();
    await dropdown.getByRole("button", { name: "View all" }).click();

    const notificationSection = page.getByRole("region", { name: "Notifications" });
    await expect(notificationSection).toContainText("2 unread notifications");
    const unreadCards = notificationSection.locator(".notification-unread");
    await expect(unreadCards).toHaveCount(2);
    await unreadCards.first().getByRole("button", { name: "Mark read" }).click();
    await expect(notificationSection).toContainText("1 unread notification");

    await notificationSection.getByRole("button", { name: "Mark all read" }).click();
    await expect(notificationSection).toContainText("No unread notifications");
    await expect(notificationToggle.locator(".notification-badge")).toHaveCount(0);
  } finally {
    if (projectId || memberId) {
      await ensureAdminSession(page);
    }
    if (projectId) {
      await archiveProjectViaApi(page, projectId);
    }
    if (memberId) {
      await deactivateMemberViaApi(page, memberId);
    }
  }
});

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

async function reloadWorkspace(page: Page) {
  await page.reload();
  await expect(page.getByRole("heading", { name: /Good to see you/ })).toBeVisible();
}

async function createProjectViaApi(page: Page, key: string, name: string) {
  return expectJson<Project>(
    await page.request.post(`${apiBaseURL}/api/v1/projects`, {
      headers: await csrfHeaders(page),
      data: {
        key,
        name,
        description: "Created by V2 browser e2e smoke.",
      },
    }),
  );
}

async function createIssueViaApi(
  page: Page,
  input: {
    project_id: string;
    parent_issue_id?: string;
    title: string;
    issue_type?: IssueType;
    status?: IssueStatus;
    priority?: IssuePriority;
    story_points?: number;
  },
) {
  return expectJson<Issue>(
    await page.request.post(`${apiBaseURL}/api/v1/issues`, {
      headers: await csrfHeaders(page),
      data: {
        description: "Created by V2 browser e2e smoke.",
        due_date: "",
        label_ids: [],
        issue_type: "task",
        status: "todo",
        priority: "medium",
        story_points: 0,
        ...input,
      },
    }),
  );
}

async function createTeamMemberViaApi(
  page: Page,
  username: string,
  displayName: string,
  password: string,
) {
  return expectJson<TeamMember>(
    await page.request.post(`${apiBaseURL}/api/v1/team/members`, {
      headers: await csrfHeaders(page),
      data: {
        email: `${username}@example.com`,
        username,
        display_name: displayName,
        password,
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

async function addPlanningIssue(page: Page, issueTitle: string) {
  const issueCard = page.locator(".planning-issue-card").filter({
    hasText: issueTitle,
  });
  await expect(issueCard).toBeVisible();
  await issueCard.getByRole("button", { name: "Add" }).click();
  await expect(issueCard.getByRole("button")).toHaveText("Remove");
}

async function ensureAdminSession(page: Page) {
  const response = await page.request.get(`${apiBaseURL}/api/v1/auth/me`);
  if (response.ok()) {
    const payload = (await response.json()) as { user: { username: string } };
    if (payload.user.username === adminLogin) {
      return;
    }
    await logoutViaApi(page);
  }

  await login(page);
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
  return {
    "X-CSRF-Token": payload.csrf_token,
  };
}

async function expectJson<T = unknown>(response: APIResponse) {
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

function dateOnly(offsetDays: number) {
  const date = new Date();
  date.setDate(date.getDate() + offsetDays);
  return date.toISOString().slice(0, 10);
}

function newRunId() {
  return `${Date.now().toString(36).slice(-4)}${Math.random()
    .toString(36)
    .slice(2, 4)}`;
}
