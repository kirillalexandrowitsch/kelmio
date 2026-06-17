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
