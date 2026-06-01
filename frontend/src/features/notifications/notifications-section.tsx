import { FormError } from "../../components/form-feedback";
import { type AppNotification } from "../../lib/api-types";
import {
  notificationDescription,
  notificationPreview,
  notificationTimeLabel,
  notificationTitle,
  unreadBadgeLabel,
} from "../../lib/notification-view";

type NotificationsSectionProps = {
  error: string;
  isActive: boolean;
  isLoading: boolean;
  notifications: AppNotification[];
  onMarkAllRead: () => void;
  onMarkRead: (notification: AppNotification) => void;
  onOpenIssue: (notification: AppNotification) => void;
  unreadCount: number;
};

export function NotificationsSection({
  error,
  isActive,
  isLoading,
  notifications,
  onMarkAllRead,
  onMarkRead,
  onOpenIssue,
  unreadCount,
}: NotificationsSectionProps) {
  if (!isActive) {
    return null;
  }

  return (
    <section className="section-card notifications-section" aria-label="Notifications">
      <header className="section-header">
        <div>
          <p className="eyebrow">In-app notifications</p>
          <h2>Notifications</h2>
          <p>{unreadBadgeLabel(unreadCount)}</p>
        </div>
        <button
          className="small-button"
          disabled={unreadCount === 0}
          onClick={onMarkAllRead}
          type="button"
        >
          Mark all read
        </button>
      </header>

      {isLoading ? <p className="muted">Loading notifications</p> : null}
      <FormError message={error} />

      {notifications.length > 0 ? (
        <div className="notification-list">
          {notifications.map((notification) => (
            <NotificationCard
              key={notification.id}
              notification={notification}
              onMarkRead={onMarkRead}
              onOpenIssue={onOpenIssue}
            />
          ))}
        </div>
      ) : (
        <div className="project-empty">No notifications yet</div>
      )}
    </section>
  );
}

type NotificationCardProps = {
  notification: AppNotification;
  onMarkRead: (notification: AppNotification) => void;
  onOpenIssue: (notification: AppNotification) => void;
};

function NotificationCard({
  notification,
  onMarkRead,
  onOpenIssue,
}: NotificationCardProps) {
  const preview = notificationPreview(notification);
  const isUnread = notification.read_at === null;

  return (
    <article className={`notification-card ${isUnread ? "notification-unread" : ""}`}>
      <div>
        <div className="notification-card-heading">
          <h3>{notificationTitle(notification)}</h3>
          {isUnread ? <span className="status-pill">Unread</span> : null}
        </div>
        <p>{notificationDescription(notification)}</p>
        {preview ? <p className="notification-preview">{preview}</p> : null}
        <span className="muted">{notificationTimeLabel(notification)}</span>
      </div>
      <div className="notification-actions">
        {notification.issue_id ? (
          <button
            className="small-button"
            onClick={() => onOpenIssue(notification)}
            type="button"
          >
            Open issue
          </button>
        ) : null}
        <button
          className="small-button"
          disabled={!isUnread}
          onClick={() => onMarkRead(notification)}
          type="button"
        >
          {isUnread ? "Mark read" : "Read"}
        </button>
      </div>
    </article>
  );
}
