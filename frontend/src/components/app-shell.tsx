import { appSections, type AppSection } from "../lib/routing";
import { type AppNotification, type CurrentUser } from "../lib/api-types";
import {
  notificationDescription,
  notificationTitle,
  unreadBadgeLabel,
} from "../lib/notification-view";

type AppSidebarProps = {
  activeSection: AppSection;
  onNavigate: (section: AppSection) => void;
};

export function AppSidebar({ activeSection, onNavigate }: AppSidebarProps) {
  return (
    <aside className="sidebar">
      <div className="brand">
        <span className="brand-mark">TT</span>
        <div>
          <strong>Team Task Tracker</strong>
          <span>Local workspace</span>
        </div>
      </div>

      <nav className="nav-list" aria-label="Main navigation">
        {appSections.map((section) => (
          <button
            aria-current={activeSection === section.id ? "page" : undefined}
            key={section.id}
            onClick={() => onNavigate(section.id)}
            type="button"
          >
            {section.title}
          </button>
        ))}
      </nav>
    </aside>
  );
}

type WorkspaceTopbarProps = {
  heading: string;
  isLoggingOut: boolean;
  isNotificationsOpen: boolean;
  notifications: AppNotification[];
  notificationsError: string;
  onMarkAllNotificationsRead: () => void;
  onMarkNotificationRead: (notification: AppNotification) => void;
  onLogout: () => void;
  onOpenNotifications: () => void;
  onOpenNotificationIssue: (notification: AppNotification) => void;
  onToggleNotifications: () => void;
  role: CurrentUser["workspace"]["role"];
  subtitle: string;
  unreadNotificationsCount: number;
};

export function WorkspaceTopbar({
  heading,
  isLoggingOut,
  isNotificationsOpen,
  notifications,
  notificationsError,
  onMarkAllNotificationsRead,
  onMarkNotificationRead,
  onLogout,
  onOpenNotifications,
  onOpenNotificationIssue,
  onToggleNotifications,
  role,
  subtitle,
  unreadNotificationsCount,
}: WorkspaceTopbarProps) {
  const previewNotifications = notifications.slice(0, 5);

  return (
    <header className="topbar">
      <div>
        <p className="eyebrow">{subtitle}</p>
        <h1>{heading}</h1>
      </div>
      <div className="topbar-actions">
        <div className="notification-menu">
          <button
            className="ghost-button notification-toggle"
            onClick={onToggleNotifications}
            type="button"
          >
            Notifications
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
        <div className="status-pill">{role}</div>
        <button
          className="ghost-button"
          disabled={isLoggingOut}
          onClick={onLogout}
          type="button"
        >
          {isLoggingOut ? "Logging out..." : "Log out"}
        </button>
      </div>
    </header>
  );
}
