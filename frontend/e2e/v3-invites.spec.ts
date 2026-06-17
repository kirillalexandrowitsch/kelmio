import { expect, test, type APIResponse, type Page } from "@playwright/test";

import type { AuthResponse, TeamMember } from "../src/lib/api-types";

const adminLogin = process.env.E2E_ADMIN_LOGIN ?? "admin";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "admin12345";
const apiBaseURL = process.env.E2E_API_BASE_URL ?? "http://localhost:8080";
const appBaseURL = process.env.E2E_BASE_URL ?? "http://localhost:5173";
const mailpitBaseURL = process.env.E2E_MAILPIT_BASE_URL ?? "http://localhost:8025";

test("V3 invite UI: admin creates invite and recipient accepts it", async ({
  page,
}) => {
  const runId = newRunId();
  const username = `v3_invite_${runId}`;
  const email = `${username}@example.com`;
  const displayName = `V3 Invite ${runId}`;
  const password = "invite12345";
  let memberId = "";

  try {
    await login(page);
    await openNav(page, "Team");

    const inviteSection = page.getByRole("region", { name: "Team invites" });
    await inviteSection.getByLabel("Email").fill(email);
    await inviteSection.getByLabel("Role").selectOption("member");
    await inviteSection.getByRole("button", { name: "Create invite" }).click();

    const inviteCard = inviteSection.locator(".team-invite-card").filter({
      hasText: email,
    });
    await expect(inviteCard).toContainText("Pending");
    const inviteLinkInput = inviteCard.getByLabel(`Invite link for ${email}`);
    await expect(inviteLinkInput).toHaveValue(/\/accept-invite\?token=/);
    await expect(inviteCard).toContainText("Email: Pending");
    const emailedInvite = await waitForValidInviteEmail(page, email);

    await logoutViaApi(page);
    await page.goto(emailedInvite.link);
    await expect(
      page.getByRole("heading", { name: "Accept workspace invite" }),
    ).toBeVisible();
    await expect(page.getByText(email)).toBeVisible();

    await page.getByLabel("Username").fill(username);
    await page.getByLabel("Display name").fill(displayName);
    await page.getByLabel("Password", { exact: true }).fill(password);
    await page.getByLabel("Confirm password").fill(password);
    await page.getByRole("button", { name: "Accept invite" }).click();

    await expect(page.getByText(`Account created for @${username}`)).toBeVisible();
    await page.getByRole("button", { name: "Go to sign in" }).click();
    await login(page, username, password);

    const me = await expectJson<AuthResponse>(
      await page.request.get(`${apiBaseURL}/api/v1/auth/me`),
    );
    memberId = me.user.id;

    await logoutViaApi(page);
    await login(page);
    await openNav(page, "Team");

    const memberRow = page.locator(".team-member-row").filter({
      hasText: email,
    });
    await expect(memberRow).toContainText(displayName);
    await expect(memberRow).toContainText(`@${username}`);
  } finally {
    if (memberId) {
      await logoutViaApi(page).catch(() => undefined);
      await login(page);
      await deactivateMemberViaApi(page, memberId);
    }
  }
});

test("V3 invite UI: admin revokes a pending invite", async ({ page }) => {
  const runId = newRunId();
  const email = `v3_revoke_${runId}@example.com`;

  await login(page);
  await openNav(page, "Team");

  const inviteSection = page.getByRole("region", { name: "Team invites" });
  await inviteSection.getByLabel("Email").fill(email);
  await inviteSection.getByLabel("Role").selectOption("member");
  await inviteSection.getByRole("button", { name: "Create invite" }).click();

  const inviteCard = inviteSection.locator(".team-invite-card").filter({
    hasText: email,
  });
  await expect(inviteCard).toContainText("Pending");
  const inviteLink = await inviteCard
    .getByLabel(`Invite link for ${email}`)
    .inputValue();
  const createEmail = await waitForValidInviteEmail(page, email);

  await inviteCard.getByRole("button", { name: "Resend email" }).click();
  const resentEmail = await waitForValidInviteEmail(
    page,
    email,
    new Set([createEmail.id]),
  );
  expect(resentEmail.link).toContain("/accept-invite?token=");

  await inviteCard.getByRole("button", { name: "Revoke" }).click();
  await expect(inviteCard).toContainText("Revoked");
  await expect(inviteCard.getByRole("button", { name: "Revoke" })).toHaveCount(0);

  await logoutViaApi(page);
  await page.goto(inviteLink);
  await expect(page.getByRole("alert")).toContainText("invite was revoked");
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

async function waitForValidInviteEmail(
  page: Page,
  email: string,
  seenMessageIds: Set<string> = new Set(),
) {
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

        const messageID = String(message.ID ?? message.Id ?? message.id ?? "");
        if (!messageID || seenMessageIds.has(messageID)) {
          continue;
        }

        const detailResponse = await page.request.get(
          `${mailpitBaseURL}/api/v1/message/${encodeURIComponent(messageID)}`,
        );
        if (!detailResponse.ok()) {
          continue;
        }

        const detail = await detailResponse.json().catch(() => null);
        const link = extractInviteLink(detail);
        if (link && (await invitePreviewIsValid(page, link))) {
          return { id: messageID, link };
        }
      }
    }

    await page.waitForTimeout(1000);
  }

  throw new Error(`Timed out waiting for invite email for ${email}`);
}

function messageTargetsEmail(message: unknown, email: string) {
  return collectStrings(message)
    .map((value) => value.toLowerCase())
    .some((value) => value.includes(email.toLowerCase()));
}

function extractInviteLink(payload: unknown) {
  const inviteLinkPattern =
    /(?:https?:\/\/[^\s"'<>]+)?\/accept-invite\?token=[A-Za-z0-9_-]+/;
  for (const value of collectStrings(payload)) {
    const match = value.match(inviteLinkPattern);
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

async function invitePreviewIsValid(page: Page, inviteLink: string) {
  const token = inviteTokenFromLink(inviteLink);
  const response = await page.request.get(
    `${apiBaseURL}/api/v1/auth/invites/${encodeURIComponent(token)}`,
  );
  return response.ok();
}

function inviteTokenFromLink(inviteLink: string) {
  const parsed = new URL(inviteLink);
  const token = parsed.searchParams.get("token") ?? "";
  if (!token) {
    throw new Error(`Invite link is missing token: ${inviteLink}`);
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

function newRunId() {
  return `${Date.now().toString(36).slice(-4)}${Math.random()
    .toString(36)
    .slice(2, 4)}`;
}
