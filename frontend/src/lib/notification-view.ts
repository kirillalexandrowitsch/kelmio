import { type AppNotification } from "./api-types.ts";

export function unreadBadgeLabel(count: number) {
  if (count === 0) {
    return "No unread notifications";
  }
  return count === 1 ? "1 unread notification" : `${count} unread notifications`;
}

export function notificationActor(notification: AppNotification) {
  return notification.actor_display_name ?? "Someone";
}

export function notificationTitle(notification: AppNotification) {
  const actor = notificationActor(notification);
  switch (notification.notification_type) {
    case "issue_assigned":
      return `${actor} assigned you an issue`;
    case "issue_mentioned":
      return `${actor} mentioned you`;
    case "issue_commented":
      return `${actor} commented on your issue`;
    case "sprint_started":
      return `${actor} started a sprint`;
    case "sprint_completed":
      return `${actor} completed a sprint`;
  }
}

export function notificationDescription(notification: AppNotification) {
  if (notification.issue_key) {
    return `${notification.issue_key} · ${notification.issue_title ?? "Issue"}`;
  }

  const sprintName = notification.payload.sprint_name;
  const projectKey = notification.payload.project_key;
  if (sprintName && projectKey) {
    return `${sprintName} · ${projectKey}`;
  }
  if (sprintName) {
    return sprintName;
  }

  return notification.payload.message ?? "Workspace notification";
}

export function notificationPreview(notification: AppNotification) {
  return notification.payload.preview ?? "";
}

export function notificationTimeLabel(notification: AppNotification) {
  return new Date(notification.created_at).toLocaleString();
}
