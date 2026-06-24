import { FormError } from "../../components/form-feedback";
import { type AppNotification } from "../../lib/api-types";
import {
  notificationDescription,
  notificationPreview,
  notificationTimeLabel,
  notificationTitle,
  unreadBadgeLabel,
} from "../../lib/notification-view";
import { Bell, BellRing, Check, ExternalLink } from "lucide-react";
import { Badge, Button, EmptyState } from "../../ui";

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
        <Button
          icon={<Check size={15} />}
          disabled={unreadCount === 0}
          onClick={onMarkAllRead}
          size="sm"
          variant="secondary"
        >
          Mark all read
        </Button>
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
        <EmptyState
          description="Assignments, mentions and automation updates will appear here."
          icon={<Bell size={20} />}
          title="No notifications yet"
        />
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
      <span className="notification-card-icon">
        {isUnread ? <BellRing size={17} /> : <Bell size={17} />}
      </span>
      <div>
        <div className="notification-card-heading">
          <h3>{notificationTitle(notification)}</h3>
          {isUnread ? <Badge tone="success">Unread</Badge> : null}
        </div>
        <p>{notificationDescription(notification)}</p>
        {preview ? <p className="notification-preview">{preview}</p> : null}
        <span className="muted">{notificationTimeLabel(notification)}</span>
      </div>
      <div className="notification-actions">
        {notification.issue_id ? (
          <Button
            icon={<ExternalLink size={14} />}
            onClick={() => onOpenIssue(notification)}
            size="sm"
            variant="secondary"
          >
            Open issue
          </Button>
        ) : null}
        <Button
          icon={<Check size={14} />}
          disabled={!isUnread}
          onClick={() => onMarkRead(notification)}
          size="sm"
          variant="ghost"
        >
          {isUnread ? "Mark read" : "Read"}
        </Button>
      </div>
    </article>
  );
}
