import type {
  AutomationAction,
  AutomationCondition,
  AutomationRule,
  CreateAutomationRuleInput,
  Label,
  ProjectMember,
  ProjectWorkflow,
  TeamMember,
} from "./api-types";

export const automationTriggerTypes = [
  "issue_created",
  "status_changed",
  "assignee_changed",
  "priority_changed",
] as const;

export const automationConditionTypes = [
  "issue_type",
  "workflow_status",
  "priority",
  "assignee",
  "reporter",
  "label",
] as const;

export const automationActionTypes = [
  "change_workflow_status",
  "change_assignee",
  "change_priority",
  "add_label",
  "remove_label",
] as const;

export function emptyAutomationRuleInput(): CreateAutomationRuleInput {
  return {
    name: "",
    trigger_type: "issue_created",
    conditions: [],
    actions: [],
    is_enabled: true,
  };
}

export function automationRuleInput(rule: AutomationRule): CreateAutomationRuleInput {
  return {
    name: rule.name,
    trigger_type: rule.trigger_type,
    conditions: structuredClone(rule.conditions),
    actions: structuredClone(rule.actions),
    is_enabled: rule.is_enabled,
  };
}

export function validateAutomationRuleInput(input: CreateAutomationRuleInput) {
  if (!input.name.trim()) {
    return "Name is required.";
  }
  if (input.name.trim().length > 100) {
    return "Name must be 100 characters or fewer.";
  }
  if (input.conditions.length > 20 || input.actions.length > 20) {
    return "Rules support at most 20 conditions and 20 actions.";
  }
  if (input.actions.length === 0) {
    return "Add at least one action.";
  }
  const scalar = new Set<string>();
  const labels = new Set<string>();
  for (const condition of input.conditions) {
    if (condition.type === "label") {
      if (labels.has(condition.label_id)) {
        return "Label conditions must reference different labels.";
      }
      labels.add(condition.label_id);
      continue;
    }
    if (scalar.has(condition.type)) {
      return "Scalar condition types cannot be repeated.";
    }
    scalar.add(condition.type);
  }
  return "";
}

export function hasMissingAutomationDependency(
  input: Pick<CreateAutomationRuleInput, "conditions" | "actions">,
  workflow: ProjectWorkflow | undefined,
  users: TeamMember[],
  labels: Label[],
  projectMembers: ProjectMember[],
) {
  const statusIDs = new Set(
    workflow?.statuses
      .filter((status) => status.archived_at === null)
      .map((status) => status.id) ?? [],
  );
  const userIDs = new Set(automationProjectUsers(users, projectMembers).map((user) => user.id));
  const labelIDs = new Set(labels.map((label) => label.id));
  const items = [...input.conditions, ...input.actions];
  return items.some((item) => {
    if ("workflow_status_id" in item) {
      return !statusIDs.has(item.workflow_status_id);
    }
    if ("label_id" in item) {
      return !labelIDs.has(item.label_id);
    }
    if ("user_id" in item && item.user_id) {
      return !userIDs.has(item.user_id);
    }
    if ("user_id" in item && item.type === "reporter") {
      return true;
    }
    return false;
  });
}

export function automationProjectUsers(
  teamMembers: TeamMember[],
  projectMembers: ProjectMember[],
) {
  const memberIDs = new Set(
    projectMembers.filter((member) => member.is_active).map((member) => member.user_id),
  );
  return teamMembers.filter(
    (member) =>
      member.is_active && (member.role === "admin" || memberIDs.has(member.id)),
  );
}

export function moveAutomationRule(
  rules: AutomationRule[],
  ruleId: string,
  direction: -1 | 1,
) {
  const ids = rules.map((rule) => rule.id);
  const index = ids.indexOf(ruleId);
  const nextIndex = index + direction;
  if (index < 0 || nextIndex < 0 || nextIndex >= ids.length) {
    return ids;
  }
  [ids[index], ids[nextIndex]] = [ids[nextIndex], ids[index]];
  return ids;
}

export function automationTriggerLabel(value: string) {
  return value.replaceAll("_", " ");
}

export function automationDisabledReasonLabel(value: string | null) {
  const labels: Record<string, string> = {
    workflow_status_unavailable: "A workflow status is unavailable.",
    label_unavailable: "A label is unavailable.",
    user_unavailable: "A user is unavailable.",
    project_access_removed: "A user no longer has project access.",
  };
  return value ? labels[value] ?? value.replaceAll("_", " ") : "";
}

export function automationItemSummary(
  item: AutomationCondition | AutomationAction,
  workflow: ProjectWorkflow | undefined,
  users: TeamMember[],
  labels: Label[],
  projectMembers: ProjectMember[],
) {
  if ("workflow_status_id" in item) {
    const status = workflow?.statuses.find(
      (candidate) => candidate.id === item.workflow_status_id,
    );
    return `${automationTriggerLabel(item.type)}: ${status?.name ?? "Missing status"}`;
  }
  if ("label_id" in item) {
    const label = labels.find((candidate) => candidate.id === item.label_id);
    return `${automationTriggerLabel(item.type)}: ${label?.name ?? "Missing label"}`;
  }
  if ("user_id" in item) {
    const user = automationProjectUsers(users, projectMembers).find(
      (candidate) => candidate.id === item.user_id,
    );
    return `${automationTriggerLabel(item.type)}: ${
      item.user_id ? user?.display_name ?? "Missing user" : "Unassigned"
    }`;
  }
  return `${automationTriggerLabel(item.type)}: ${item.value.replaceAll("_", " ")}`;
}
