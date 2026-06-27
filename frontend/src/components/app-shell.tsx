import {
  Bell,
  Columns3,
  FolderKanban,
  LayoutDashboard,
  ListTodo,
  LogOut,
  Rocket,
  Search,
  Tag,
  UserRound,
  Users,
  type LucideIcon,
} from "lucide-react";

import { appSections, navGroups, type AppSection } from "../lib/routing";
import { type AppNotification } from "../lib/api-types";
import {
  notificationDescription,
  notificationTitle,
  unreadBadgeLabel,
} from "../lib/notification-view";
import { Icon } from "../ui";

const sectionIcons: Record<AppSection, LucideIcon> = {
  dashboard: LayoutDashboard,
  projects: FolderKanban,
  issues: ListTodo,
  board: Columns3,
  sprints: Rocket,
  notifications: Bell,
  team: Users,
  labels: Tag,
  account: UserRound,
};

function sectionTitle(id: AppSection) {
  return appSections.find((section) => section.id === id)?.title ?? id;
}

function userInitials(name: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

type AppSidebarProps = {
  activeSection: AppSection;
  onNavigate: (section: AppSection) => void;
  onOpenCommandPalette: () => void;
  displayName: string;
  role: string;
  unreadNotificationsCount: number;
  isLoggingOut: boolean;
  onSignOut: () => void;
  notifications: AppNotification[];
  notificationsError: string;
  isNotificationsOpen: boolean;
  onToggleNotifications: () => void;
  onMarkAllNotificationsRead: () => void;
  onMarkNotificationRead: (notification: AppNotification) => void;
  onOpenNotificationIssue: (notification: AppNotification) => void;
  onOpenNotifications: () => void;
};

export function AppSidebar({
  activeSection,
  onNavigate,
  onOpenCommandPalette,
  displayName,
  role,
  unreadNotificationsCount,
  isLoggingOut,
  onSignOut,
  notifications,
  notificationsError,
  isNotificationsOpen,
  onToggleNotifications,
  onMarkAllNotificationsRead,
  onMarkNotificationRead,
  onOpenNotificationIssue,
  onOpenNotifications,
}: AppSidebarProps) {
  const previewNotifications = (notifications ?? []).slice(0, 5);

  return (
    <aside className="kl-sidebar">
      <div className="kl-sidebar__brand">
        <span className="kl-sidebar__mark">K</span>
        <span>Kelmio</span>
        <div className="kl-sidebar__notify notification-menu">
          <button
            aria-label="Notifications"
            className="kl-sidebar__bell notification-toggle"
            onClick={onToggleNotifications}
            type="button"
          >
            <Icon icon={Bell} size={18} />
            {unreadNotificationsCount > 0 ? (
              <span className="notification-badge">{unreadNotificationsCount}</span>
            ) : null}
          </button>
          {isNotificationsOpen ? (
            <section
              className="notification-dropdown"
              aria-label="Notification dropdown"
            >
              <header>
                <strong>{unreadBadgeLabel(unreadNotificationsCount)}</strong>
                <button
                  className="small-button"
                  disabled={unreadNotificationsCount === 0}
                  onClick={onMarkAllNotificationsRead}
                  type="button"
                >
                  Mark all read
                </button>
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
                        onClick={() => onOpenNotificationIssue(notification)}
                        type="button"
                      >
                        <strong>{notificationTitle(notification)}</strong>
                        <span>{notificationDescription(notification)}</span>
                      </button>
                      {notification.read_at === null ? (
                        <button
                          className="small-button"
                          onClick={() => onMarkNotificationRead(notification)}
                          type="button"
                        >
                          Read
                        </button>
                      ) : null}
                    </article>
                  ))}
                </div>
              ) : (
                <p className="notification-preview">No notifications yet.</p>
              )}
              <button
                className="small-button notification-view-all"
                onClick={onOpenNotifications}
                type="button"
              >
                View all
              </button>
            </section>
          ) : null}
        </div>
      </div>

      <button
        className="kl-sidebar__search"
        onClick={onOpenCommandPalette}
        type="button"
      >
        <Icon icon={Search} size={16} />
        <span>Search</span>
        <kbd className="kl-sidebar__kbd">⌘K</kbd>
      </button>

      <nav className="kl-sidebar__nav" aria-label="Main navigation">
        {navGroups.map((group) => (
          <div className="kl-sidebar__group" key={group.id}>
            <p className="kl-sidebar__group-label">{group.label}</p>
            {group.sections.map((section) => (
              <button
                aria-current={activeSection === section ? "page" : undefined}
                className="kl-sidebar__item"
                key={section}
                onClick={() => onNavigate(section)}
                type="button"
              >
                <Icon icon={sectionIcons[section]} size={18} />
                <span>{sectionTitle(section)}</span>
                {section === "notifications" && unreadNotificationsCount > 0 ? (
                  <span aria-hidden="true" className="kl-sidebar__count">
                    {unreadNotificationsCount}
                  </span>
                ) : null}
              </button>
            ))}
          </div>
        ))}
      </nav>

      <div className="kl-sidebar__footer">
        <button
          aria-current={activeSection === "account" ? "page" : undefined}
          className="kl-sidebar__user"
          onClick={() => onNavigate("account")}
          type="button"
        >
          <span className="kl-sidebar__avatar">{userInitials(displayName)}</span>
          <span className="kl-sidebar__user-text">
            <strong>{displayName}</strong>
            <span>{role}</span>
          </span>
        </button>
        <button
          className="kl-sidebar__signout"
          disabled={isLoggingOut}
          onClick={onSignOut}
          type="button"
        >
          <Icon icon={LogOut} size={16} />
          {isLoggingOut ? "Signing out..." : "Sign out"}
        </button>
      </div>
    </aside>
  );
}
