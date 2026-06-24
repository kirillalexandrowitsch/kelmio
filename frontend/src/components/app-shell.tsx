import {
  Archive,
  Bell,
  ChevronLeft,
  ChevronRight,
  CircleUserRound,
  FolderKanban,
  Gauge,
  KanbanSquare,
  ListChecks,
  LogOut,
  Menu,
  Tags,
  UsersRound,
  X,
  type LucideIcon,
} from "lucide-react";

import { appSections, type AppSection } from "../lib/routing";
import { type AppNotification, type CurrentUser } from "../lib/api-types";
import {
  notificationDescription,
  notificationTitle,
  unreadBadgeLabel,
} from "../lib/notification-view";
import { Badge, Button, IconButton } from "../ui";
import { KelmioMark } from "./kelmio-mark";

const sectionIcons: Record<AppSection, LucideIcon> = {
  account: CircleUserRound,
  board: KanbanSquare,
  dashboard: Gauge,
  issues: ListChecks,
  labels: Tags,
  notifications: Bell,
  projects: FolderKanban,
  sprints: Archive,
  team: UsersRound,
};

const navigationGroups: Array<{
  label: string;
  sections: AppSection[];
}> = [
  {
    label: "Workspace",
    sections: ["dashboard", "projects", "issues", "board", "sprints"],
  },
  {
    label: "Organization",
    sections: ["notifications", "team", "labels"],
  },
  {
    label: "Personal",
    sections: ["account"],
  },
];

type AppSidebarProps = {
  activeSection: AppSection;
  collapsed: boolean;
  displayName: string;
  isMobileOpen: boolean;
  onCollapseToggle: () => void;
  onMobileClose: () => void;
  onNavigate: (section: AppSection) => void;
  role: CurrentUser["workspace"]["role"];
  username: string;
};

export function AppSidebar({
  activeSection,
  collapsed,
  displayName,
  isMobileOpen,
  onCollapseToggle,
  onMobileClose,
  onNavigate,
  role,
  username,
}: AppSidebarProps) {
  function navigate(section: AppSection) {
    onNavigate(section);
    onMobileClose();
  }

  return (
    <>
      {isMobileOpen ? (
        <button
          aria-label="Close navigation"
          className="sidebar-backdrop"
          onClick={onMobileClose}
          type="button"
        />
      ) : null}
      <aside
        className="sidebar"
        data-collapsed={collapsed}
        data-mobile-open={isMobileOpen}
      >
        <header className="sidebar-header">
          <div className="brand">
            <KelmioMark />
            <div className="brand-copy">
              <strong>Kelmio</strong>
              <span>Local workspace</span>
            </div>
          </div>
          <IconButton
            className="sidebar-mobile-close"
            label="Close navigation"
            onClick={onMobileClose}
          >
            <X size={18} />
          </IconButton>
        </header>

        <nav className="nav-list" aria-label="Main navigation">
          {navigationGroups.map((group) => (
            <section className="nav-group" key={group.label}>
              <span className="nav-group-label">{group.label}</span>
              {group.sections.map((sectionId) => {
                const section = appSections.find((item) => item.id === sectionId);
                if (!section) {
                  return null;
                }
                const Icon = sectionIcons[section.id];
                return (
                  <button
                    aria-current={activeSection === section.id ? "page" : undefined}
                    aria-label={collapsed ? section.title : undefined}
                    data-tooltip={collapsed ? section.title : undefined}
                    key={section.id}
                    onClick={() => navigate(section.id)}
                    type="button"
                  >
                    <Icon aria-hidden="true" size={18} strokeWidth={1.8} />
                    <span>{section.title}</span>
                    {activeSection === section.id ? <i aria-hidden="true" /> : null}
                  </button>
                );
              })}
            </section>
          ))}
        </nav>

        <footer className="sidebar-footer">
          <div className="sidebar-profile">
            <span className="sidebar-avatar">{displayName.slice(0, 1).toUpperCase()}</span>
            <div>
              <strong>{displayName}</strong>
              <span>@{username} · {role}</span>
            </div>
          </div>
          <IconButton
            className="sidebar-collapse"
            label={collapsed ? "Expand navigation" : "Collapse navigation"}
            onClick={onCollapseToggle}
          >
            {collapsed ? <ChevronRight size={17} /> : <ChevronLeft size={17} />}
          </IconButton>
        </footer>
      </aside>
    </>
  );
}

type WorkspaceTopbarProps = {
  displayName: string;
  heading: string;
  isLoggingOut: boolean;
  isNotificationsOpen: boolean;
  notifications: AppNotification[];
  notificationsError: string;
  onMarkAllNotificationsRead: () => void;
  onMarkNotificationRead: (notification: AppNotification) => void;
  onLogout: () => void;
  onMobileMenuOpen: () => void;
  onOpenNotifications: () => void;
  onOpenNotificationIssue: (notification: AppNotification) => void;
  onToggleNotifications: () => void;
  role: CurrentUser["workspace"]["role"];
  subtitle: string;
  unreadNotificationsCount: number;
  username: string;
};

export function WorkspaceTopbar({
  displayName,
  heading,
  isLoggingOut,
  isNotificationsOpen,
  notifications,
  notificationsError,
  onMarkAllNotificationsRead,
  onMarkNotificationRead,
  onLogout,
  onMobileMenuOpen,
  onOpenNotifications,
  onOpenNotificationIssue,
  onToggleNotifications,
  role,
  subtitle,
  unreadNotificationsCount,
  username,
}: WorkspaceTopbarProps) {
  const previewNotifications = notifications.slice(0, 5);

  return (
    <header className="topbar">
      <div className="topbar-heading">
        <IconButton
          className="topbar-menu-button"
          label="Open navigation"
          onClick={onMobileMenuOpen}
        >
          <Menu size={19} />
        </IconButton>
        <div>
          <p className="eyebrow">{subtitle}</p>
          <h1>{heading}</h1>
        </div>
      </div>

      <div className="topbar-actions">
        <div className="notification-menu">
          <IconButton
            className="notification-toggle"
            label="Notifications"
            onClick={onToggleNotifications}
          >
            <Bell size={18} />
            {unreadNotificationsCount > 0 ? (
              <span className="notification-badge">{unreadNotificationsCount}</span>
            ) : null}
          </IconButton>
          {isNotificationsOpen ? (
            <section
              className="notification-dropdown"
              aria-label="Notification dropdown"
            >
              <header>
                <div>
                  <span className="notification-dropdown-kicker">Inbox</span>
                  <strong>{unreadBadgeLabel(unreadNotificationsCount)}</strong>
                </div>
                <Button
                  disabled={unreadNotificationsCount === 0}
                  onClick={onMarkAllNotificationsRead}
                  size="sm"
                  variant="ghost"
                >
                  Mark all read
                </Button>
              </header>
              {notificationsError ? (
                <p className="notification-preview">{notificationsError}</p>
              ) : null}
              {previewNotifications.length > 0 ? (
                <div className="notification-dropdown-list">
                  {previewNotifications.map((notification) => (
                    <article
                      className={
                        notification.read_at === null
                          ? "notification-dropdown-item notification-unread"
                          : "notification-dropdown-item"
                      }
                      key={notification.id}
                    >
                      <button
                        aria-label={notificationTitle(notification)}
                        onClick={() => onOpenNotificationIssue(notification)}
                        type="button"
                      >
                        <span className="notification-item-icon"><Bell size={14} /></span>
                        <span>
                          <strong>{notificationTitle(notification)}</strong>
                          <small>{notificationDescription(notification)}</small>
                        </span>
                      </button>
                      {notification.read_at === null ? (
                        <button
                          aria-label={`Mark ${notificationTitle(notification)} read`}
                          className="notification-read-dot"
                          onClick={() => onMarkNotificationRead(notification)}
                          type="button"
                        />
                      ) : null}
                    </article>
                  ))}
                </div>
              ) : (
                <p className="notification-preview">No notifications yet.</p>
              )}
              <Button
                className="notification-view-all"
                onClick={onOpenNotifications}
                size="sm"
                variant="secondary"
              >
                View all notifications
              </Button>
            </section>
          ) : null}
        </div>

        <Badge className="topbar-role" tone="success">{role}</Badge>
        <div className="topbar-user">
          <span className="topbar-avatar">{displayName.slice(0, 1).toUpperCase()}</span>
          <div>
            <strong>{displayName}</strong>
            <span>@{username}</span>
          </div>
        </div>
        <IconButton
          className="topbar-logout"
          disabled={isLoggingOut}
          label={isLoggingOut ? "Logging out..." : "Log out"}
          onClick={onLogout}
        >
          <LogOut size={18} />
        </IconButton>
      </div>
    </header>
  );
}
