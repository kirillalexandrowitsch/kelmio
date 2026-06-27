import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { AppSidebar } from "./app-shell";

function renderSidebar(overrides: Record<string, unknown> = {}) {
  const props = {
    activeSection: "issues" as const,
    onNavigate: vi.fn(),
    onOpenCommandPalette: vi.fn(),
    displayName: "Kirill Alexandrov",
    role: "admin",
    unreadNotificationsCount: 0,
    isLoggingOut: false,
    onSignOut: vi.fn(),
    ...overrides,
  };
  render(<AppSidebar {...(props as never)} />);
  return props;
}

describe("AppSidebar", () => {
  it("keeps the Main navigation contract with exact section button names", async () => {
    const props = renderSidebar();
    const nav = screen.getByRole("navigation", { name: "Main navigation" });

    for (const name of [
      "Dashboard",
      "Projects",
      "Issues",
      "Board",
      "Sprints",
      "Notifications",
      "Team",
      "Labels",
    ]) {
      expect(within(nav).getByRole("button", { name, exact: true })).toBeTruthy();
    }

    await userEvent.click(
      within(nav).getByRole("button", { name: "Board", exact: true }),
    );
    expect(props.onNavigate).toHaveBeenCalledWith("board");
  });

  it("does not let the unread badge pollute the Notifications accessible name", () => {
    renderSidebar({ unreadNotificationsCount: 3 });
    const nav = screen.getByRole("navigation", { name: "Main navigation" });

    expect(
      within(nav).getByRole("button", { name: "Notifications", exact: true }),
    ).toBeTruthy();
    expect(screen.getAllByText("3").length).toBeGreaterThan(0);
  });

  it("navigates to account via the user card and signs out", async () => {
    const props = renderSidebar();

    await userEvent.click(
      screen.getByRole("button", { name: /Kirill Alexandrov/ }),
    );
    expect(props.onNavigate).toHaveBeenCalledWith("account");

    await userEvent.click(screen.getByRole("button", { name: "Sign out" }));
    expect(props.onSignOut).toHaveBeenCalledTimes(1);
  });
});
