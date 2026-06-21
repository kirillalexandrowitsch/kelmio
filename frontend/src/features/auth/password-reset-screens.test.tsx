import assert from "node:assert/strict";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, test, vi } from "vitest";

const apiMocks = vi.hoisted(() => {
  class MockApiError extends Error {
    status: number;
    code: string;

    constructor(message: string, status: number, code = "") {
      super(message);
      this.name = "ApiError";
      this.status = status;
      this.code = code;
    }
  }

  return {
    ApiError: MockApiError,
    completePasswordReset: vi.fn(),
    getPasswordResetPreview: vi.fn(),
    requestPasswordReset: vi.fn(),
  };
});

vi.mock("../../lib/api", () => apiMocks);

import {
  ForgotPasswordScreen,
  ResetPasswordScreen,
} from "./password-reset-screens";

beforeEach(() => {
  apiMocks.completePasswordReset.mockReset();
  apiMocks.getPasswordResetPreview.mockReset();
  apiMocks.requestPasswordReset.mockReset();
});

test("forgot password screen submits normalized email and shows private success", async () => {
  const user = userEvent.setup();
  apiMocks.requestPasswordReset.mockResolvedValue({
    message: "If an active account exists, password reset instructions will be sent.",
  });

  render(<ForgotPasswordScreen onGoToSignIn={vi.fn()} />);

  await user.type(screen.getByLabelText("Email"), " Admin@Example.COM ");
  await user.click(screen.getByRole("button", { name: "Send reset link" }));

  await screen.findByText("Password reset instructions sent");
  assert.equal(apiMocks.requestPasswordReset.mock.calls[0]?.[0], "admin@example.com");
  assert.match(
    screen.getByText(/If an active account exists/).textContent ?? "",
    /password reset instructions will be sent/,
  );
});

test("forgot password screen preserves the form after an API failure", async () => {
  const user = userEvent.setup();
  apiMocks.requestPasswordReset.mockRejectedValue(new Error("Delivery unavailable"));

  render(<ForgotPasswordScreen onGoToSignIn={vi.fn()} />);

  await user.type(screen.getByLabelText("Email"), "member@example.com");
  await user.click(screen.getByRole("button", { name: "Send reset link" }));

  await screen.findByText("Delivery unavailable");
  assert.equal(screen.getByLabelText("Email").getAttribute("value"), "member@example.com");
  assert.equal(screen.queryByText("Password reset instructions sent"), null);
});

test("reset password screen shows missing token error", async () => {
  render(
    <ResetPasswordScreen
      onGoToSignIn={vi.fn()}
      onResetCompleted={vi.fn()}
      token=""
    />,
  );

  await screen.findByRole("alert");
  assert.equal(
    screen.getByRole("alert").textContent,
    "Password reset link is missing a token.",
  );
  assert.equal(apiMocks.getPasswordResetPreview.mock.calls.length, 0);
});

for (const tokenState of [
  [
    "password_reset_not_found",
    404,
    "Password reset link was not found. Request a new link.",
  ],
  [
    "password_reset_expired",
    400,
    "Password reset link has expired. Request a new link.",
  ],
  [
    "password_reset_used",
    400,
    "Password reset link was already used. Request a new link.",
  ],
  [
    "password_reset_revoked",
    400,
    "Password reset link was revoked. Request a new link.",
  ],
] as const) {
  test(`reset password screen presents ${tokenState[0]} safely`, async () => {
    apiMocks.getPasswordResetPreview.mockRejectedValue(
      new apiMocks.ApiError("provider detail", tokenState[1], tokenState[0]),
    );

    render(
      <ResetPasswordScreen
        onGoToSignIn={vi.fn()}
        onResetCompleted={vi.fn()}
        token="reset-token"
      />,
    );

    await screen.findByText(tokenState[2]);
    assert.equal(screen.queryByLabelText("New password"), null);
    assert.equal(screen.queryByText("provider detail"), null);
  });
}

test("reset password screen exposes preview loading state", () => {
  apiMocks.getPasswordResetPreview.mockReturnValue(new Promise(() => undefined));

  render(
    <ResetPasswordScreen
      onGoToSignIn={vi.fn()}
      onResetCompleted={vi.fn()}
      token="reset-token"
    />,
  );

  assert.ok(screen.getByText("Loading reset link..."));
  assert.equal(screen.queryByRole("button", { name: "Reset password" }), null);
});

test("reset password screen loads preview and validates password confirmation", async () => {
  const user = userEvent.setup();
  apiMocks.getPasswordResetPreview.mockResolvedValue({
    email: "admin@example.com",
    expires_at: "2026-06-17T12:00:00Z",
  });

  render(
    <ResetPasswordScreen
      onGoToSignIn={vi.fn()}
      onResetCompleted={vi.fn()}
      token="reset-token"
    />,
  );

  await screen.findByText("admin@example.com");
  await user.type(screen.getByLabelText("New password"), "new-password");
  await user.type(screen.getByLabelText("Confirm password"), "other-password");
  await user.click(screen.getByRole("button", { name: "Reset password" }));

  assert.equal(
    screen.getByRole("alert").textContent,
    "Password confirmation does not match.",
  );
  assert.equal(apiMocks.completePasswordReset.mock.calls.length, 0);
});

test("reset password screen completes reset and shows success state", async () => {
  const user = userEvent.setup();
  const onResetCompleted = vi.fn();
  apiMocks.getPasswordResetPreview.mockResolvedValue({
    email: "admin@example.com",
    expires_at: "2026-06-17T12:00:00Z",
  });
  apiMocks.completePasswordReset.mockResolvedValue(undefined);

  render(
    <ResetPasswordScreen
      onGoToSignIn={vi.fn()}
      onResetCompleted={onResetCompleted}
      token="reset-token"
    />,
  );

  await screen.findByText("admin@example.com");
  await user.type(screen.getByLabelText("New password"), "new-password");
  await user.type(screen.getByLabelText("Confirm password"), "new-password");
  await user.click(screen.getByRole("button", { name: "Reset password" }));

  await screen.findByText("Your password has been reset");
  await waitFor(() => {
    assert.equal(apiMocks.completePasswordReset.mock.calls.length, 1);
  });
  assert.deepEqual(apiMocks.completePasswordReset.mock.calls[0], [
    "reset-token",
    "new-password",
    "new-password",
  ]);
  assert.equal(onResetCompleted.mock.calls.length, 1);
});

test("reset password screen keeps entered values after completion failure", async () => {
  const user = userEvent.setup();
  apiMocks.getPasswordResetPreview.mockResolvedValue({
    email: "admin@example.com",
    expires_at: "2026-06-17T12:00:00Z",
  });
  apiMocks.completePasswordReset.mockRejectedValue(
    new apiMocks.ApiError("used", 400, "password_reset_used"),
  );

  render(
    <ResetPasswordScreen
      onGoToSignIn={vi.fn()}
      onResetCompleted={vi.fn()}
      token="reset-token"
    />,
  );

  await screen.findByText("admin@example.com");
  await user.type(screen.getByLabelText("New password"), "new-password");
  await user.type(screen.getByLabelText("Confirm password"), "new-password");
  await user.click(screen.getByRole("button", { name: "Reset password" }));

  await screen.findByText(
    "Password reset link was already used. Request a new link.",
  );
  assert.equal(screen.getByLabelText("New password").getAttribute("value"), "new-password");
  assert.equal(screen.queryByText("Your password has been reset"), null);
});
