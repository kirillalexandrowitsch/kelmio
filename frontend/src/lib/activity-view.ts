import { type IssueActivity, type Label, type TeamMember } from "./api-types.ts";
import { priorityLabels, statusLabel } from "./issue-model.ts";

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
  if (activity.action === "issue_link_created") {
    return "Linked issue";
  }
  if (activity.action === "issue_link_deleted") {
    return "Removed issue link";
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
  if (activity.action === "automation_applied") {
    return "Automation applied";
  }

  return activity.action.replaceAll("_", " ");
}

export function activityDescription(
  activity: IssueActivity,
  members: TeamMember[],
  labels: Label[] = [],
) {
  if (activity.action === "status_changed") {
    return `${statusLabel(activity.payload.from_status)} -> ${statusLabel(
      activity.payload.to_status,
    )}`;
  }
  if (activity.action === "assignee_changed") {
    return `${activityMemberName(members, activity.payload.from_assignee_id)} -> ${activityMemberName(
      members,
      activity.payload.to_assignee_id,
    )}`;
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
  if (
    activity.action === "issue_link_created" ||
    activity.action === "issue_link_deleted"
  ) {
    const sourceIssue = activity.payload.source_issue_key || "source issue";
    const targetIssue = activity.payload.target_issue_key || "target issue";
    const linkType = activity.payload.link_type || "linked";

    return `${sourceIssue} ${linkType} ${targetIssue}`;
  }
  if (activity.action === "automation_applied") {
    const changes: string[] = [];
    const changedFields = activity.payload.changed_fields?.split(",") ?? [];
    if (changedFields.includes("status")) {
      changes.push(
        `status ${statusLabel(activity.payload.from_status)} -> ${statusLabel(
          activity.payload.to_status,
        )}`,
      );
    }
    if (changedFields.includes("assignee")) {
      changes.push(
        `assignee ${activityMemberName(
          members,
          activity.payload.from_assignee_id,
        )} -> ${activityMemberName(members, activity.payload.to_assignee_id)}`,
      );
    }
    if (changedFields.includes("priority")) {
      changes.push(
        `priority ${activityPriority(activity.payload.from_priority)} -> ${activityPriority(
          activity.payload.to_priority,
        )}`,
      );
    }
    if (changedFields.includes("labels")) {
      const added = activityLabelNames(labels, activity.payload.added_label_ids);
      const removed = activityLabelNames(labels, activity.payload.removed_label_ids);
      changes.push(
        `labels ${[
          ...added.map((name) => `+${name}`),
          ...removed.map((name) => `-${name}`),
        ].join(", ")}`,
      );
    }

    const ruleName = activity.payload.rule_name || "Unnamed rule";
    return changes.length > 0 ? `${ruleName} · ${changes.join("; ")}` : ruleName;
  }

  return "";
}

function activityMemberName(members: TeamMember[], memberId?: string) {
  if (!memberId) {
    return "Unassigned";
  }
  return members.find((member) => member.id === memberId)?.display_name ?? "Missing member";
}

function activityPriority(priority?: string) {
  return priority && priority in priorityLabels
    ? priorityLabels[priority as keyof typeof priorityLabels]
    : priority || "Unknown";
}

function activityLabelNames(labels: Label[], value?: string) {
  if (!value) {
    return [];
  }
  return value.split(",").map(
    (labelId) => labels.find((label) => label.id === labelId)?.name ?? "Missing label",
  );
}
