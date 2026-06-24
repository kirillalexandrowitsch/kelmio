import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { AppSidebar, WorkspaceTopbar } from "./app-shell";

describe("application shell", () => {
  it("keeps grouped navigation accessible and closes mobile navigation", async () => {
    const onNavigate = vi.fn();
    const onMobileClose = vi.fn();
    const user = userEvent.setup();

    render(
      <AppSidebar
        activeSection="dashboard"
        collapsed={false}
        displayName="Admin"
        isMobileOpen
        onCollapseToggle={vi.fn()}
        onMobileClose={onMobileClose}
        onNavigate={onNavigate}
        role="admin"
        username="admin"
      />,
    );

    expect(screen.getByRole("navigation", { name: "Main navigation" })).not.toBeNull();
    await user.click(screen.getByRole("button", { name: "Projects" }));
    expect(onNavigate).toHaveBeenCalledWith("projects");
    expect(onMobileClose).toHaveBeenCalledOnce();
  });

  it("opens notification details without exposing implementation controls", async () => {
    const onOpenNotificationIssue = vi.fn();
    const user = userEvent.setup();

    render(
      <WorkspaceTopbar
        displayName="Admin"
        heading="Dashboard"
        isLoggingOut={false}
        isNotificationsOpen
        notifications={[
          {
            id: "notification-1",
            workspace_id: "workspace-1",
            user_id: "user-1",
            actor_id: "user-2",
            actor_display_name: "Demo Member",
            issue_id: "issue-1",
            issue_key: "DEMO-1",
            issue_title: "Prepare release",
            notification_type: "issue_assigned",
            payload: {},
            read_at: null,
            created_at: "2026-06-24T12:00:00Z",
          },
        ]}
        notificationsError=""
        onLogout={vi.fn()}
        onMarkAllNotificationsRead={vi.fn()}
        onMarkNotificationRead={vi.fn()}
        onMobileMenuOpen={vi.fn()}
        onOpenNotifications={vi.fn()}
        onOpenNotificationIssue={onOpenNotificationIssue}
        onToggleNotifications={vi.fn()}
        role="admin"
        subtitle="Local workspace"
        unreadNotificationsCount={1}
        username="admin"
      />,
    );

    expect(screen.getByLabelText("Notification dropdown")).not.toBeNull();
    await user.click(
      screen.getByRole("button", { name: "Demo Member assigned you an issue" }),
    );
    expect(onOpenNotificationIssue).toHaveBeenCalledOnce();
  });
});
