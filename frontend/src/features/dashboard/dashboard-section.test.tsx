import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { DashboardSection } from "./dashboard-section";

function renderDashboard(overrides: Record<string, unknown> = {}) {
  const props = {
    activeSprint: {
      name: "Cycle 14",
      points_total: 47,
      points_done: 31,
      points_open: 16,
      start_date: "2026-06-12",
      end_date: "2026-06-28",
    },
    activeSprintIssues: [
      { assignee_id: "u1", story_points: 8, status: "in_progress" },
      { assignee_id: null, story_points: 3, status: "todo" },
    ],
    activeSprintError: "",
    isLoadingActiveSprint: false,
    dueSoonIssuesCount: 8,
    overdueIssuesCount: 3,
    openIssuesCount: 47,
    projectsCount: 6,
    teamMembersCount: 9,
    teamMembers: [{ id: "u1", display_name: "Kirill A" }],
    displayName: "Admin",
    myWorkIssues: [
      {
        id: "i1",
        issue_key: "KLM-2481",
        title: "Aurora command palette",
        priority: "critical",
        story_points: 8,
        due_date: null,
      },
    ],
    isActive: true,
    onNavigate: vi.fn(),
    onOpenIssue: vi.fn(),
    ...overrides,
  };
  render(<DashboardSection {...(props as never)} />);
  return props;
}

describe("DashboardSection", () => {
  it("renders the sign-in greeting heading that e2e depends on", () => {
    renderDashboard();
    expect(
      screen.getByRole("heading", { name: /Good to see you/ }).textContent,
    ).toContain("Admin");
  });

  it("renders stat cards, burndown header, workload and my work", () => {
    renderDashboard();

    for (const label of ["Projects", "Open", "Overdue", "Due soon", "Team"]) {
      expect(screen.getByText(label)).toBeTruthy();
    }
    expect(screen.getByText("Cycle 14 burndown")).toBeTruthy();
    expect(screen.getByText("31 / 47 pts done")).toBeTruthy();
    expect(screen.getByText("Kirill A")).toBeTruthy();
    expect(screen.getByText("Unassigned")).toBeTruthy();
    expect(screen.getByText("Aurora command palette")).toBeTruthy();
    expect(screen.getByText("KLM-2481")).toBeTruthy();
  });

  it("opens an issue and navigates to all issues", async () => {
    const props = renderDashboard();

    await userEvent.click(
      screen.getByRole("button", { name: /Aurora command palette/ }),
    );
    expect(props.onOpenIssue).toHaveBeenCalledWith("i1");

    await userEvent.click(screen.getByRole("button", { name: /View all/ }));
    expect(props.onNavigate).toHaveBeenCalledWith("issues");
  });

  it("shows an empty state when nothing is assigned", () => {
    renderDashboard({ myWorkIssues: [] });
    expect(screen.getByText("Nothing assigned to you")).toBeTruthy();
  });
});
