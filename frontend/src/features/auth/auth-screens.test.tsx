import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { SignInScreen } from "./auth-screens";

function renderSignIn(overrides: Record<string, unknown> = {}) {
  const props = {
    canSignIn: true,
    error: "",
    isSubmitting: false,
    loginValue: "",
    onForgotPassword: vi.fn(),
    onLoginChange: vi.fn(),
    onPasswordChange: vi.fn(),
    onSubmit: vi.fn((event: { preventDefault: () => void }) =>
      event.preventDefault(),
    ),
    password: "",
    ...overrides,
  };
  render(<SignInScreen {...(props as never)} />);
  return props;
}

describe("SignInScreen", () => {
  it("exposes the sign-in selectors the e2e flow depends on", () => {
    renderSignIn();

    expect(screen.getByLabelText("Username or email")).toBeTruthy();
    expect(screen.getByLabelText("Password")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Sign in" })).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "Forgot password?" }),
    ).toBeTruthy();
  });

  it("reports credential edits and forgot-password intent", async () => {
    const props = renderSignIn();

    await userEvent.type(screen.getByLabelText("Username or email"), "a");
    await userEvent.type(screen.getByLabelText("Password"), "b");
    await userEvent.click(
      screen.getByRole("button", { name: "Forgot password?" }),
    );

    expect(props.onLoginChange).toHaveBeenCalledWith("a");
    expect(props.onPasswordChange).toHaveBeenCalledWith("b");
    expect(props.onForgotPassword).toHaveBeenCalledTimes(1);
  });

  it("surfaces errors through an alert and disables submit when blocked", () => {
    renderSignIn({ error: "Invalid credentials", canSignIn: false });

    expect(screen.getByRole("alert").textContent).toBe("Invalid credentials");
    expect(
      screen.getByRole("button", { name: "Sign in" }).hasAttribute("disabled"),
    ).toBe(true);
  });
});
