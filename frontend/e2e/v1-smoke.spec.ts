import { expect, test } from "@playwright/test";

const adminLogin = process.env.E2E_ADMIN_LOGIN ?? "admin";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "admin12345";

test("V1 browser smoke: create and move issue with a comment", async ({ page }) => {
  const runId = Date.now().toString().slice(-7);
  const projectKey = `E${runId}`;
  const projectName = `E2E Project ${runId}`;
  const issueTitle = `E2E issue ${runId}`;
  const commentBody = `E2E comment ${runId}`;

  await page.goto("/");

  await page.getByLabel("Username or email").fill(adminLogin);
  await page.getByLabel("Password").fill(adminPassword);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("heading", { name: /Good to see you/ })).toBeVisible();

  const mainNav = page.getByRole("navigation", { name: "Main navigation" });

  await mainNav.getByRole("button", { name: "Projects", exact: true }).click();
  const projectForm = page.locator(".project-form").filter({ hasText: "Create project" });
  await projectForm.getByLabel("Key").fill(projectKey);
  await projectForm.getByLabel("Name").fill(projectName);
  await projectForm.getByLabel("Description").fill("Created by browser e2e smoke.");
  await projectForm.getByRole("button", { name: "Create project" }).click();

  await expect(page.getByRole("heading", { name: `${projectKey} · ${projectName}` })).toBeVisible();

  await mainNav.getByRole("button", { name: "Issues", exact: true }).click();
  const issueForm = page.locator(".issue-form");
  await expect(issueForm.getByLabel("Project")).toHaveValue(/.+/);
  await issueForm.getByLabel("Title").fill(issueTitle);
  await issueForm.getByLabel("Description").fill("Created by browser e2e smoke.");
  await issueForm.getByLabel("Priority").selectOption("high");
  await page.getByRole("button", { name: "Create issue" }).click();

  await expect(page.locator(".issue-row").filter({ hasText: issueTitle })).toBeVisible();
  await expect(
    page.getByRole("heading", { name: new RegExp(`${projectKey}-\\d+ · ${issueTitle}`) }),
  ).toBeVisible();

  await mainNav.getByRole("button", { name: "Board", exact: true }).click();
  const boardCard = page.locator(".issue-card").filter({ hasText: issueTitle });
  await expect(boardCard).toBeVisible();
  await boardCard.getByLabel(new RegExp(`Status for ${projectKey}-\\d+`)).selectOption("in_progress");
  await expect(boardCard.getByLabel(new RegExp(`Status for ${projectKey}-\\d+`))).toHaveValue("in_progress");

  await boardCard.getByRole("button", { name: "Open" }).click();
  await page.getByLabel("Add comment").fill(commentBody);
  await page.getByRole("button", { name: "Post comment" }).click();

  await expect(page.getByText(commentBody)).toBeVisible();
  await expect(page.getByText("Added comment")).toBeVisible();
});
