import { type DragEvent, useEffect, useState } from "react";

import { FormError } from "../../components/form-feedback";
import type {
  AutomationAction,
  AutomationCondition,
  AutomationRule,
  AutomationTriggerType,
  CreateAutomationRuleInput,
  IssuePriority,
  IssueType,
  Label,
  ProjectMember,
  ProjectWorkflow,
  TeamMember,
} from "../../lib/api-types";
import {
  automationActionTypes,
  automationConditionTypes,
  automationDisabledReasonLabel,
  automationItemSummary,
  automationProjectUsers,
  automationRuleInput,
  automationTriggerLabel,
  automationTriggerTypes,
  emptyAutomationRuleInput,
  hasMissingAutomationDependency,
  moveAutomationRule,
  validateAutomationRuleInput,
} from "../../lib/automation-model";
import { activeWorkflowStatuses } from "../../lib/workflow-model";

type AutomationSettingsPanelProps = {
  creatingRule: boolean;
  deletingRuleIds: string[];
  error: string;
  isLoading: boolean;
  isReordering: boolean;
  labels: Label[];
  members: ProjectMember[];
  onCreateRule: (input: CreateAutomationRuleInput) => Promise<boolean>;
  onDeleteRule: (rule: AutomationRule) => Promise<boolean>;
  onReorderRules: (ruleIds: string[]) => Promise<boolean>;
  onUpdateRule: (
    rule: AutomationRule,
    input: CreateAutomationRuleInput | { is_enabled: boolean },
  ) => Promise<boolean>;
  rules: AutomationRule[];
  teamMembers: TeamMember[];
  updatingRuleIds: string[];
  workflow?: ProjectWorkflow;
};

const issueTypes: IssueType[] = ["task", "bug", "story", "epic", "subtask"];
const priorities: IssuePriority[] = ["low", "medium", "high", "critical"];

export function AutomationSettingsPanel({
  creatingRule,
  deletingRuleIds,
  error,
  isLoading,
  isReordering,
  labels,
  members,
  onCreateRule,
  onDeleteRule,
  onReorderRules,
  onUpdateRule,
  rules,
  teamMembers,
  updatingRuleIds,
  workflow,
}: AutomationSettingsPanelProps) {
  const orderedRules = [...rules].sort((left, right) => left.position - right.position);
  const [editingRuleId, setEditingRuleId] = useState("");
  const [draft, setDraft] = useState<CreateAutomationRuleInput>(
    emptyAutomationRuleInput,
  );
  const [formError, setFormError] = useState("");
  const [draggedRuleId, setDraggedRuleId] = useState("");
  const editingRule = orderedRules.find((rule) => rule.id === editingRuleId);
  const isSaving = editingRule
    ? updatingRuleIds.includes(editingRule.id)
    : creatingRule;

  useEffect(() => {
    if (!editingRuleId) {
      return;
    }
    const rule = rules.find((candidate) => candidate.id === editingRuleId);
    if (!rule) {
      setEditingRuleId("");
      setDraft(emptyAutomationRuleInput());
      setFormError("");
    }
  }, [editingRuleId, rules]);

  function startCreate() {
    setEditingRuleId("");
    setDraft(emptyAutomationRuleInput());
    setFormError("");
  }

  function startEdit(rule: AutomationRule) {
    setEditingRuleId(rule.id);
    setDraft(automationRuleInput(rule));
    setFormError("");
  }

  async function saveDraft() {
    const normalized = { ...draft, name: draft.name.trim() };
    const validationError = validateAutomationRuleInput(normalized);
    if (validationError) {
      setFormError(validationError);
      return;
    }
    if (
      hasMissingAutomationDependency(
        normalized,
        workflow,
        teamMembers,
        labels,
        members,
      )
    ) {
      setFormError("Repair missing dependencies before saving.");
      return;
    }
    setFormError("");
    const saved = editingRule
      ? await onUpdateRule(editingRule, normalized)
      : await onCreateRule(normalized);
    if (saved) {
      startCreate();
    }
  }

  async function toggleRule(rule: AutomationRule) {
    if (
      !rule.is_enabled &&
      hasMissingAutomationDependency(rule, workflow, teamMembers, labels, members)
    ) {
      startEdit(rule);
      setFormError("Repair missing dependencies before enabling this rule.");
      return;
    }
    await onUpdateRule(rule, { is_enabled: !rule.is_enabled });
  }

  async function moveRule(ruleId: string, direction: -1 | 1) {
    await onReorderRules(moveAutomationRule(orderedRules, ruleId, direction));
  }

  async function dropRule(event: DragEvent<HTMLElement>, targetRuleId: string) {
    event.preventDefault();
    const sourceRuleId = draggedRuleId || event.dataTransfer.getData("text/plain");
    if (!sourceRuleId || sourceRuleId === targetRuleId) {
      setDraggedRuleId("");
      return;
    }
    const ids = orderedRules.map((rule) => rule.id);
    const sourceIndex = ids.indexOf(sourceRuleId);
    const targetIndex = ids.indexOf(targetRuleId);
    if (sourceIndex < 0 || targetIndex < 0) {
      return;
    }
    ids.splice(sourceIndex, 1);
    ids.splice(targetIndex, 0, sourceRuleId);
    setDraggedRuleId("");
    await onReorderRules(ids);
  }

  return (
    <section className="automation-settings-panel" aria-label="Automation settings">
      <header className="automation-settings-header">
        <div>
          <h3>Automation rules</h3>
          <p>
            Rules run synchronously, atomically, and once per direct issue change.
          </p>
        </div>
        {isLoading ? <span className="muted">Refreshing</span> : null}
      </header>

      <FormError message={formError || error} />

      <RuleEditor
        draft={draft}
        editingRule={editingRule}
        isSaving={isSaving}
        labels={labels}
        members={members}
        onCancel={startCreate}
        onChange={setDraft}
        onSave={() => void saveDraft()}
        teamMembers={teamMembers}
        workflow={workflow}
      />

      <section className="automation-rule-section" aria-label="Automation rules list">
        <header>
          <div>
            <h4>Configured rules</h4>
            <p>Later rules and later actions have priority.</p>
          </div>
          {isReordering ? <span className="muted">Saving order</span> : null}
        </header>
        {orderedRules.length > 0 ? (
          <div className="automation-rule-list">
            {orderedRules.map((rule, index) => {
              const isUpdating = updatingRuleIds.includes(rule.id);
              const isDeleting = deletingRuleIds.includes(rule.id);
              return (
                <article
                  className="automation-rule-card"
                  draggable={!isReordering}
                  key={rule.id}
                  onDragOver={(event) => event.preventDefault()}
                  onDragStart={(event) => {
                    setDraggedRuleId(rule.id);
                    event.dataTransfer.setData("text/plain", rule.id);
                  }}
                  onDrop={(event) => void dropRule(event, rule.id)}
                >
                  <header>
                    <div>
                      <h4>{rule.name}</h4>
                      <p>When {automationTriggerLabel(rule.trigger_type)}</p>
                    </div>
                    <span
                      className={`automation-rule-state ${
                        rule.is_enabled ? "enabled" : "disabled"
                      }`}
                    >
                      {rule.is_enabled ? "Enabled" : "Disabled"}
                    </span>
                  </header>
                  <RuleSummary
                    labels={labels}
                    members={members}
                    rule={rule}
                    teamMembers={teamMembers}
                    workflow={workflow}
                  />
                  {rule.disabled_reason ? (
                    <p className="automation-disabled-reason">
                      {automationDisabledReasonLabel(rule.disabled_reason)}
                    </p>
                  ) : null}
                  <div className="automation-rule-actions">
                    <button
                      aria-label={`Move ${rule.name} up`}
                      className="ghost-button"
                      disabled={index === 0 || isReordering}
                      onClick={() => void moveRule(rule.id, -1)}
                      type="button"
                    >
                      ↑
                    </button>
                    <button
                      aria-label={`Move ${rule.name} down`}
                      className="ghost-button"
                      disabled={index === orderedRules.length - 1 || isReordering}
                      onClick={() => void moveRule(rule.id, 1)}
                      type="button"
                    >
                      ↓
                    </button>
                    <button
                      className="small-button"
                      disabled={isUpdating || isDeleting}
                      onClick={() => startEdit(rule)}
                      type="button"
                    >
                      Edit
                    </button>
                    <button
                      className="small-button"
                      disabled={isUpdating || isDeleting}
                      onClick={() => void toggleRule(rule)}
                      type="button"
                    >
                      {isUpdating
                        ? "Saving"
                        : rule.is_enabled
                          ? "Disable"
                          : "Enable"}
                    </button>
                    <button
                      className="small-button danger-button"
                      disabled={isUpdating || isDeleting}
                      onClick={() => {
                        if (window.confirm(`Delete automation rule "${rule.name}"?`)) {
                          void onDeleteRule(rule);
                        }
                      }}
                      type="button"
                    >
                      {isDeleting ? "Deleting" : "Delete"}
                    </button>
                  </div>
                </article>
              );
            })}
          </div>
        ) : isLoading ? null : (
          <div className="comments-empty">No automation rules</div>
        )}
      </section>
    </section>
  );
}

function RuleEditor({
  draft,
  editingRule,
  isSaving,
  labels,
  members,
  onCancel,
  onChange,
  onSave,
  teamMembers,
  workflow,
}: {
  draft: CreateAutomationRuleInput;
  editingRule?: AutomationRule;
  isSaving: boolean;
  labels: Label[];
  members: ProjectMember[];
  onCancel: () => void;
  onChange: (input: CreateAutomationRuleInput) => void;
  onSave: () => void;
  teamMembers: TeamMember[];
  workflow?: ProjectWorkflow;
}) {
  return (
    <section className="automation-rule-editor" aria-label="Automation rule editor">
      <header>
        <h4>{editingRule ? `Edit ${editingRule.name}` : "Create rule"}</h4>
        {editingRule ? (
          <button className="ghost-button" onClick={onCancel} type="button">
            Cancel
          </button>
        ) : null}
      </header>
      <div className="automation-rule-basics">
        <label>
          <span>Name</span>
          <input
            aria-label="Automation rule name"
            maxLength={100}
            onChange={(event) => onChange({ ...draft, name: event.target.value })}
            placeholder="Move critical bugs to blocked"
            value={draft.name}
          />
        </label>
        <label>
          <span>Trigger</span>
          <select
            aria-label="Automation rule trigger"
            onChange={(event) =>
              onChange({
                ...draft,
                trigger_type: event.target.value as AutomationTriggerType,
              })
            }
            value={draft.trigger_type}
          >
            {automationTriggerTypes.map((trigger) => (
              <option key={trigger} value={trigger}>
                {automationTriggerLabel(trigger)}
              </option>
            ))}
          </select>
        </label>
        <label className="automation-enabled-field">
          <input
            checked={draft.is_enabled ?? true}
            onChange={(event) =>
              onChange({ ...draft, is_enabled: event.target.checked })
            }
            type="checkbox"
          />
          <span>Enabled after save</span>
        </label>
      </div>

      <RuleItems
        items={draft.conditions}
        kind="condition"
        labels={labels}
        members={members}
        onChange={(conditions) =>
          onChange({ ...draft, conditions: conditions as AutomationCondition[] })
        }
        teamMembers={teamMembers}
        workflow={workflow}
      />
      <RuleItems
        items={draft.actions}
        kind="action"
        labels={labels}
        members={members}
        onChange={(actions) =>
          onChange({ ...draft, actions: actions as AutomationAction[] })
        }
        teamMembers={teamMembers}
        workflow={workflow}
      />
      <button disabled={isSaving} onClick={onSave} type="button">
        {isSaving ? "Saving" : editingRule ? "Save rule" : "Create rule"}
      </button>
    </section>
  );
}

function RuleItems({
  items,
  kind,
  labels,
  members,
  onChange,
  teamMembers,
  workflow,
}: {
  items: AutomationCondition[] | AutomationAction[];
  kind: "condition" | "action";
  labels: Label[];
  members: ProjectMember[];
  onChange: (items: Array<AutomationCondition | AutomationAction>) => void;
  teamMembers: TeamMember[];
  workflow?: ProjectWorkflow;
}) {
  const types = kind === "condition" ? automationConditionTypes : automationActionTypes;
  return (
    <section className="automation-item-section">
      <header>
        <div>
          <h5>{kind === "condition" ? "Conditions" : "Actions"}</h5>
          <p>
            {kind === "condition"
              ? "All conditions must match. Empty means always."
              : "Actions run in this order."}
          </p>
        </div>
        <button
          className="small-button"
          disabled={items.length >= 20}
          onClick={() =>
            onChange([
              ...items,
              kind === "condition"
                ? defaultCondition(automationConditionTypes[0])
                : defaultAction(automationActionTypes[0]),
            ])
          }
          type="button"
        >
          Add {kind}
        </button>
      </header>
      <div className="automation-item-list">
        {items.map((item, index) => (
          <div className="automation-item-row" key={`${item.type}-${index}`}>
            <select
              aria-label={`${capitalize(kind)} ${index + 1} type`}
              onChange={(event) => {
                const next = [...items];
                next[index] =
                  kind === "condition"
                    ? defaultCondition(
                        event.target.value as (typeof automationConditionTypes)[number],
                      )
                    : defaultAction(
                        event.target.value as (typeof automationActionTypes)[number],
                      );
                onChange(next as Array<AutomationCondition | AutomationAction>);
              }}
              value={item.type}
            >
              {types.map((type) => (
                <option key={type} value={type}>
                  {automationTriggerLabel(type)}
                </option>
              ))}
            </select>
            <AutomationValueControl
              item={item}
              labels={labels}
              members={members}
              onChange={(nextItem) => {
                const next = [...items];
                next[index] = nextItem;
                onChange(next as Array<AutomationCondition | AutomationAction>);
              }}
              teamMembers={teamMembers}
              workflow={workflow}
            />
            <button
              aria-label={`Move ${kind} ${index + 1} up`}
              className="ghost-button"
              disabled={index === 0}
              onClick={() =>
                onChange(
                  moveItem(
                    items as Array<AutomationCondition | AutomationAction>,
                    index,
                    -1,
                  ),
                )
              }
              type="button"
            >
              ↑
            </button>
            <button
              aria-label={`Move ${kind} ${index + 1} down`}
              className="ghost-button"
              disabled={index === items.length - 1}
              onClick={() =>
                onChange(
                  moveItem(
                    items as Array<AutomationCondition | AutomationAction>,
                    index,
                    1,
                  ),
                )
              }
              type="button"
            >
              ↓
            </button>
            <button
              aria-label={`Remove ${kind} ${index + 1}`}
              className="ghost-button danger-button"
              onClick={() =>
                onChange(
                  items.filter(
                    (_, itemIndex) => itemIndex !== index,
                  ) as Array<AutomationCondition | AutomationAction>,
                )
              }
              type="button"
            >
              Remove
            </button>
          </div>
        ))}
      </div>
    </section>
  );
}

function AutomationValueControl({
  item,
  labels,
  members,
  onChange,
  teamMembers,
  workflow,
}: {
  item: AutomationCondition | AutomationAction;
  labels: Label[];
  members: ProjectMember[];
  onChange: (item: AutomationCondition | AutomationAction) => void;
  teamMembers: TeamMember[];
  workflow?: ProjectWorkflow;
}) {
  if ("workflow_status_id" in item) {
    const statuses = activeWorkflowStatuses(workflow);
    const missing =
      Boolean(item.workflow_status_id) &&
      !statuses.some((status) => status.id === item.workflow_status_id);
    return (
      <select
        aria-label={`${automationTriggerLabel(item.type)} value`}
        onChange={(event) =>
          onChange({ ...item, workflow_status_id: event.target.value })
        }
        value={item.workflow_status_id}
      >
        {missing ? (
          <option disabled value={item.workflow_status_id}>
            Missing status
          </option>
        ) : null}
        <option value="">Select status</option>
        {statuses.map((status) => (
          <option key={status.id} value={status.id}>
            {status.name}
          </option>
        ))}
      </select>
    );
  }
  if ("label_id" in item) {
    const missing =
      Boolean(item.label_id) && !labels.some((label) => label.id === item.label_id);
    return (
      <select
        aria-label={`${automationTriggerLabel(item.type)} value`}
        onChange={(event) => onChange({ ...item, label_id: event.target.value })}
        value={item.label_id}
      >
        {missing ? (
          <option disabled value={item.label_id}>
            Missing label
          </option>
        ) : null}
        <option value="">Select label</option>
        {labels.map((label) => (
          <option key={label.id} value={label.id}>
            {label.name}
          </option>
        ))}
      </select>
    );
  }
  if ("user_id" in item) {
    const users = automationProjectUsers(teamMembers, members);
    const missing =
      Boolean(item.user_id) && !users.some((user) => user.id === item.user_id);
    return (
      <select
        aria-label={`${automationTriggerLabel(item.type)} value`}
        onChange={(event) =>
          onChange({
            ...item,
            user_id: event.target.value || null,
          } as AutomationCondition | AutomationAction)
        }
        value={item.user_id ?? ""}
      >
        {missing ? (
          <option disabled value={item.user_id ?? ""}>
            Missing user
          </option>
        ) : null}
        {item.type !== "reporter" ? <option value="">Unassigned</option> : null}
        {item.type === "reporter" ? <option value="">Select reporter</option> : null}
        {users.map((user) => (
          <option key={user.id} value={user.id}>
            {user.display_name}
          </option>
        ))}
      </select>
    );
  }
  const values = item.type === "issue_type" ? issueTypes : priorities;
  return (
    <select
      aria-label={`${automationTriggerLabel(item.type)} value`}
      onChange={(event) =>
        onChange({ ...item, value: event.target.value } as AutomationCondition | AutomationAction)
      }
      value={item.value}
    >
      {values.map((value) => (
        <option key={value} value={value}>
          {automationTriggerLabel(value)}
        </option>
      ))}
    </select>
  );
}

function RuleSummary({
  labels,
  members,
  rule,
  teamMembers,
  workflow,
}: {
  labels: Label[];
  members: ProjectMember[];
  rule: AutomationRule;
  teamMembers: TeamMember[];
  workflow?: ProjectWorkflow;
}) {
  return (
    <div className="automation-rule-summary">
      <div>
        <strong>Conditions</strong>
        {rule.conditions.length > 0 ? (
          rule.conditions.map((item, index) => (
            <span key={`${item.type}-${index}`}>
              {automationItemSummary(item, workflow, teamMembers, labels, members)}
            </span>
          ))
        ) : (
          <span>Always</span>
        )}
      </div>
      <div>
        <strong>Actions</strong>
        {rule.actions.map((item, index) => (
          <span key={`${item.type}-${index}`}>
            {automationItemSummary(item, workflow, teamMembers, labels, members)}
          </span>
        ))}
      </div>
    </div>
  );
}

function defaultCondition(
  type: (typeof automationConditionTypes)[number],
): AutomationCondition {
  switch (type) {
    case "issue_type":
      return { type, value: "task" };
    case "workflow_status":
      return { type, workflow_status_id: "" };
    case "priority":
      return { type, value: "medium" };
    case "assignee":
      return { type, user_id: null };
    case "reporter":
      return { type, user_id: "" };
    case "label":
      return { type, label_id: "" };
  }
}

function defaultAction(type: (typeof automationActionTypes)[number]): AutomationAction {
  switch (type) {
    case "change_workflow_status":
      return { type, workflow_status_id: "" };
    case "change_assignee":
      return { type, user_id: null };
    case "change_priority":
      return { type, value: "medium" };
    case "add_label":
    case "remove_label":
      return { type, label_id: "" };
  }
}

function moveItem<T>(items: T[], index: number, direction: -1 | 1) {
  const next = [...items];
  const nextIndex = index + direction;
  if (nextIndex < 0 || nextIndex >= next.length) {
    return next;
  }
  [next[index], next[nextIndex]] = [next[nextIndex], next[index]];
  return next;
}

function capitalize(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
