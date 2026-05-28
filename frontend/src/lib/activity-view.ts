import { type IssueActivity, type TeamMember } from "./api-types.ts";
import { statusLabel } from "./issue-model.ts";
import { memberDisplayName } from "./team-view.ts";

export function activityTitle(activity: IssueActivity) {
  if (activity.action === "issue_created") {
    return "Created issue";
  }
  if (activity.action === "issue_updated") {
    return "Updated issue";
  }
  if (activity.action === "status_changed") {
    return "Changed status";
  }
  if (activity.action === "assignee_changed") {
    return "Changed assignee";
  }
  if (activity.action === "labels_changed") {
    return "Changed labels";
  }
  if (activity.action === "issue_parent_changed") {
    return "Changed parent";
  }
  if (activity.action === "issue_archived") {
    return "Archived issue";
  }
  if (activity.action === "comment_added") {
    return "Added comment";
  }
  if (activity.action === "comment_updated") {
    return "Updated comment";
  }
  if (activity.action === "comment_deleted") {
    return "Deleted comment";
  }

  return activity.action.replaceAll("_", " ");
}

export function activityDescription(
  activity: IssueActivity,
  members: TeamMember[],
) {
  if (activity.action === "status_changed") {
    return `${statusLabel(activity.payload.from_status)} -> ${statusLabel(
      activity.payload.to_status,
    )}`;
  }
  if (activity.action === "assignee_changed") {
    return `${memberDisplayName(
      members,
      activity.payload.from_assignee_id || null,
    )} -> ${memberDisplayName(members, activity.payload.to_assignee_id || null)}`;
  }
  if (activity.action === "comment_added") {
    return activity.payload.preview ? `"${activity.payload.preview}"` : "";
  }
  if (activity.action === "comment_updated") {
    return activity.payload.preview ? `"${activity.payload.preview}"` : "";
  }
  if (activity.action === "comment_deleted") {
    return activity.payload.preview ? `"${activity.payload.preview}"` : "";
  }
  if (activity.action === "issue_created") {
    return activity.payload.title ?? "";
  }
  if (activity.action === "issue_updated") {
    return activity.payload.fields
      ? `Fields: ${activity.payload.fields.replaceAll(",", ", ")}`
      : "";
  }
  if (activity.action === "labels_changed") {
    return "Labels updated";
  }
  if (activity.action === "issue_parent_changed") {
    return `${activity.payload.from_parent_issue_id || "No parent"} -> ${
      activity.payload.to_parent_issue_id || "No parent"
    }`;
  }

  return "";
}
