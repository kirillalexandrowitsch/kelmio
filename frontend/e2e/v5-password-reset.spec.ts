import {
  expect,
  test,
  type APIResponse,
  type BrowserContext,
  type Page,
} from "@playwright/test";

import type { TeamMember } from "../src/lib/api-types";

const adminLogin = process.env.E2E_ADMIN_LOGIN ?? "admin";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "admin12345";
const apiBaseURL = process.env.E2E_API_BASE_URL ?? "http://localhost:8080";
const appBaseURL = process.env.E2E_BASE_URL ?? "http://localhost:5173";
const mailpitBaseURL = process.env.E2E_MAILPIT_BASE_URL ?? "http://localhost:8025";

test("V5 password reset UI: request email, reset password, and sign in", async ({
  browser,
  page,
}) => {
  const runId = newRunId();
  const username = `v5_reset_${runId}`;
  const email = `${username}@example.com`;
  const initialPassword = "initial12345";
  const temporaryPassword = `reset-${Date.now().toString(36)}-pass`;
  let memberId = "";
  let resetContext: BrowserContext | null = null;

  try {
    await login(page);
    const member = await createTeamMemberViaApi(page, {
      email,
      username,
      display_name: `V5 Reset ${runId}`,
      password: initialPassword,
      role: "member",
    });
    memberId = member.id;
    await logoutViaApi(page);
    await login(page, username, initialPassword);

    resetContext = await browser.newContext();
    const resetPage = await resetContext.newPage();

    await resetPage.goto("/");
    await resetPage.getByRole("button", { name: "Forgot password?" }).click();
    await expect(
      resetPage.getByRole("heading", { name: "Reset your password" }),
    ).toBeVisible();

    await resetPage.getByLabel("Email").fill(email);
    await resetPage.getByRole("button", { name: "Send reset link" }).click();
    await expect(
      resetPage.getByText("Password reset instructions sent"),
    ).toBeVisible();

    const resetLink = await waitForValidPasswordResetLink(resetPage, email);
    await resetPage.goto(resetLink);
    await expect(
      resetPage.getByRole("heading", { name: "Choose a new password" }),
    ).toBeVisible();
    await expect(resetPage.getByText(email, { exact: true })).toBeVisible();

    await resetPage.getByLabel("New password").fill(temporaryPassword);
    await resetPage.getByLabel("Confirm password").fill(temporaryPassword);
    await resetPage.getByRole("button", { name: "Reset password" }).click();
    await expect(resetPage.getByText("Your password has been reset")).toBeVisible();

    const staleSessionResponse = await page.request.get(`${apiBaseURL}/api/v1/auth/me`);
    expect(staleSessionResponse.status()).toBe(401);

    const reusePage = await resetContext.newPage();
    await reusePage.goto(resetLink);
    await expect(reusePage.getByRole("alert")).toContainText("already used");
    await reusePage.close();

    await resetPage.getByRole("button", { name: "Go to sign in" }).click();
    await login(resetPage, username, temporaryPassword);
  } finally {
    await resetContext?.close().catch(() => undefined);
    if (memberId) {
      await logoutViaApi(page).catch(() => undefined);
      await login(page);
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

async function createTeamMemberViaApi(
  page: Page,
  input: {
    email: string;
    username: string;
    display_name: string;
    password: string;
    role: "admin" | "member";
  },
) {
  return expectJson<TeamMember>(
    await page.request.post(`${apiBaseURL}/api/v1/team/members`, {
      headers: await csrfHeaders(page),
      data: input,
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

async function waitForValidPasswordResetLink(page: Page, email: string) {
  const deadline = Date.now() + 45000;

  while (Date.now() < deadline) {
    const messagesResponse = await page.request.get(`${mailpitBaseURL}/api/v1/messages`);
    if (messagesResponse.ok()) {
      const payload = await messagesResponse.json().catch(() => null);
      const messages = Array.isArray(payload?.messages) ? payload.messages : [];

      for (const message of messages) {
        if (!messageTargetsEmail(message, email)) {
          continue;
        }

        const messageID = message.ID ?? message.Id ?? message.id;
        if (!messageID) {
          continue;
        }

        const detailResponse = await page.request.get(
          `${mailpitBaseURL}/api/v1/message/${encodeURIComponent(String(messageID))}`,
        );
        if (!detailResponse.ok()) {
          continue;
        }

        const detail = await detailResponse.json().catch(() => null);
        const link = extractPasswordResetLink(detail);
        if (link && (await passwordResetPreviewIsValid(page, link))) {
          return link;
        }
      }
    }

    await page.waitForTimeout(1000);
  }

  throw new Error(`Timed out waiting for password reset email for ${email}`);
}

function messageTargetsEmail(message: unknown, email: string) {
  return collectStrings(message)
    .map((value) => value.toLowerCase())
    .some((value) => value.includes(email.toLowerCase()));
}

function extractPasswordResetLink(payload: unknown) {
  const resetLinkPattern =
    /(?:https?:\/\/[^\s"'<>]+)?\/reset-password\?token=[A-Za-z0-9_-]+/;
  for (const value of collectStrings(payload)) {
    const match = value.match(resetLinkPattern);
    if (!match) {
      continue;
    }

    const link = match[0];
    if (link.startsWith("http")) {
      return link;
    }
    return `${appBaseURL.replace(/\/$/, "")}${link}`;
  }

  return "";
}

function collectStrings(value: unknown): string[] {
  if (typeof value === "string") {
    return [value];
  }
  if (Array.isArray(value)) {
    return value.flatMap((item) => collectStrings(item));
  }
  if (value && typeof value === "object") {
    return Object.values(value).flatMap((item) => collectStrings(item));
  }
  return [];
}

async function passwordResetPreviewIsValid(page: Page, resetLink: string) {
  const token = passwordResetTokenFromLink(resetLink);
  const response = await page.request.get(
    `${apiBaseURL}/api/v1/auth/password-reset/${encodeURIComponent(token)}`,
  );
  return response.ok();
}

function passwordResetTokenFromLink(resetLink: string) {
  const parsed = new URL(resetLink);
  const token = parsed.searchParams.get("token") ?? "";
  if (!token) {
    throw new Error(`Password reset link is missing token: ${resetLink}`);
  }
  return token;
}

async function logoutViaApi(page: Page) {
  await expectOk(
    await page.request.post(`${apiBaseURL}/api/v1/auth/logout`, {
      headers: await csrfHeaders(page),
    }),
  );
}

async function expectJson<T = unknown>(response: APIResponse) {
  await expectOk(response);
  return (await response.json()) as T;
}

async function csrfHeaders(page: Page) {
  const response = await page.request.get(`${apiBaseURL}/api/v1/auth/csrf-token`);
  await expectOk(response);

  const payload = (await response.json()) as { csrf_token: string };
  return {
    "X-CSRF-Token": payload.csrf_token,
  };
}

async function expectOk(response: APIResponse) {
  if (!response.ok()) {
    throw new Error(
      `Expected API request to succeed, got ${response.status()}: ${await response.text()}`,
    );
  }
}

function newRunId() {
  return `${Date.now().toString(36).slice(-4)}${Math.random()
    .toString(36)
    .slice(2, 4)}`;
}
