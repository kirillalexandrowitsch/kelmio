import {
  type DragEvent,
  type FormEvent,
  useEffect,
  useState,
} from "react";

import { FormError } from "../../components/form-feedback";
import type {
  CreateWorkflowStatusInput,
  ProjectWorkflow,
  ProjectWorkflowStatus,
  UpdateWorkflowStatusInput,
  WorkflowStatusCategory,
  WorkflowTransitionInput,
} from "../../lib/api-types";
import { activeWorkflowStatuses, workflowStatusStyle } from "../../lib/workflow-model";
import {
  moveWorkflowStatus,
  normalizeWorkflowStatusInput,
  transitionDraftFromWorkflow,
  transitionKey,
  transitionsFromDraft,
  validateWorkflowStatusInput,
  workflowStatusCategories,
} from "../../lib/workflow-settings-model";
import { Button, Field, Input, Select } from "../../ui";

type StatusDraft = {
  name: string;
  color: string;
  category: WorkflowStatusCategory;
};

type WorkflowSettingsPanelProps = {
  archivingStatusIds: string[];
  creatingStatus: boolean;
  error: string;
  isLoading: boolean;
  isReordering: boolean;
  isSavingTransitions: boolean;
  onArchiveStatus: (
    status: ProjectWorkflowStatus,
    replacementStatusId: string,
  ) => Promise<boolean>;
  onCreateStatus: (input: CreateWorkflowStatusInput) => Promise<boolean>;
  onReorderStatuses: (statusIds: string[]) => Promise<boolean>;
  onReplaceTransitions: (transitions: WorkflowTransitionInput[]) => Promise<boolean>;
  onUpdateStatus: (
    status: ProjectWorkflowStatus,
    input: UpdateWorkflowStatusInput,
  ) => Promise<boolean>;
  updatingStatusIds: string[];
  workflow?: ProjectWorkflow;
};

const emptyCreateStatus: CreateWorkflowStatusInput = {
  key: "",
  name: "",
  color: "#0ea5e9",
  category: "todo",
};

export function WorkflowSettingsPanel({
  archivingStatusIds,
  creatingStatus,
  error,
  isLoading,
  isReordering,
  isSavingTransitions,
  onArchiveStatus,
  onCreateStatus,
  onReorderStatuses,
  onReplaceTransitions,
  onUpdateStatus,
  updatingStatusIds,
  workflow,
}: WorkflowSettingsPanelProps) {
  const statuses = activeWorkflowStatuses(workflow);
  const archivedStatuses = workflow?.statuses
    .filter((status) => status.archived_at !== null)
    .sort((left, right) => left.position - right.position) ?? [];
  const doneStatusCount = statuses.filter((status) => status.category === "done").length;
  const [createStatus, setCreateStatus] =
    useState<CreateWorkflowStatusInput>(emptyCreateStatus);
  const [formError, setFormError] = useState("");
  const [statusDrafts, setStatusDrafts] = useState<Record<string, StatusDraft>>({});
  const [transitionDraft, setTransitionDraft] = useState<Set<string>>(new Set());
  const [draggedStatusId, setDraggedStatusId] = useState("");
  const [archiveStatusId, setArchiveStatusId] = useState("");
  const [replacementStatusId, setReplacementStatusId] = useState("");

  useEffect(() => {
    setStatusDrafts(
      Object.fromEntries(
        statuses.map((status) => [
          status.id,
          {
            name: status.name,
            color: status.color,
            category: status.category,
          },
        ]),
      ),
    );
    setTransitionDraft(transitionDraftFromWorkflow(workflow));
    setFormError("");
    setArchiveStatusId((currentStatusId) =>
      statuses.some((status) => status.id === currentStatusId)
        ? currentStatusId
        : "",
    );
    setReplacementStatusId((currentStatusId) =>
      statuses.some((status) => status.id === currentStatusId)
        ? currentStatusId
        : "",
    );
  }, [workflow]);

  async function handleCreateStatus(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalized = normalizeWorkflowStatusInput(createStatus);
    const validationError = validateWorkflowStatusInput(normalized);
    if (validationError) {
      setFormError(validationError);
      return;
    }
    setFormError("");
    if (await onCreateStatus(normalized)) {
      setCreateStatus(emptyCreateStatus);
    }
  }

  async function handleUpdateStatus(status: ProjectWorkflowStatus) {
    const draft = statusDrafts[status.id];
    if (!draft) {
      return;
    }
    const validationError = validateWorkflowStatusInput({
      key: status.key,
      ...draft,
    });
    if (validationError) {
      setFormError(validationError);
      return;
    }
    setFormError("");
    await onUpdateStatus(status, draft);
  }

  async function handleMoveStatus(statusId: string, direction: -1 | 1) {
    await onReorderStatuses(moveWorkflowStatus(statuses, statusId, direction));
  }

  async function handleDropStatus(
    event: DragEvent<HTMLElement>,
    targetStatusId: string,
  ) {
    event.preventDefault();
    const sourceStatusId =
      draggedStatusId || event.dataTransfer.getData("text/plain");
    if (!sourceStatusId || sourceStatusId === targetStatusId) {
      setDraggedStatusId("");
      return;
    }
    const ids = statuses.map((status) => status.id);
    const sourceIndex = ids.indexOf(sourceStatusId);
    const targetIndex = ids.indexOf(targetStatusId);
    if (sourceIndex < 0 || targetIndex < 0) {
      setDraggedStatusId("");
      return;
    }
    ids.splice(sourceIndex, 1);
    ids.splice(targetIndex, 0, sourceStatusId);
    setDraggedStatusId("");
    await onReorderStatuses(ids);
  }

  function toggleTransition(fromStatusId: string, toStatusId: string) {
    const key = transitionKey(fromStatusId, toStatusId);
    setTransitionDraft((currentDraft) => {
      const nextDraft = new Set(currentDraft);
      if (nextDraft.has(key)) {
        nextDraft.delete(key);
      } else {
        nextDraft.add(key);
      }
      return nextDraft;
    });
  }

  async function handleSaveTransitions() {
    const transitions = transitionsFromDraft(statuses, transitionDraft);
    if (
      transitions.length === 0 &&
      !window.confirm(
        "Save an empty transition graph? Issues will remain in their current statuses until transitions are added.",
      )
    ) {
      return;
    }
    await onReplaceTransitions(transitions);
  }

  function resetTransitions() {
    setTransitionDraft(transitionDraftFromWorkflow(workflow));
  }

  async function handleArchiveStatus() {
    const status = statuses.find((item) => item.id === archiveStatusId);
    if (!status || !replacementStatusId) {
      setFormError("Choose a replacement status.");
      return;
    }
    setFormError("");
    if (await onArchiveStatus(status, replacementStatusId)) {
      setArchiveStatusId("");
      setReplacementStatusId("");
    }
  }

  if (isLoading && !workflow) {
    return <div className="kl-empty-block">Loading project workflow</div>;
  }
  if (!workflow) {
    return (
      <section className="kl-wf" aria-label="Workflow settings">
        <FormError message={error || "Project workflow is unavailable."} />
      </section>
    );
  }

  const archiveStatus = statuses.find((status) => status.id === archiveStatusId);
  const archiveReplacements = statuses.filter(
    (status) => status.id !== archiveStatusId,
  );

  return (
    <section className="kl-wf" aria-label="Workflow settings">
      <header className="kl-section-head">
        <div>
          <h3>Workflow settings</h3>
          <p className="kl-muted">
            Statuses define board columns. Transitions control allowed moves.
          </p>
        </div>
        {isLoading ? <span className="kl-muted">Refreshing</span> : null}
      </header>

      <FormError message={formError || error} />

      <form className="kl-wf__create" onSubmit={handleCreateStatus}>
        <h4>Create status</h4>
        <div className="kl-wf__create-fields">
          <Field label="Key" htmlFor="wf-new-key">
            <Input
              id="wf-new-key"
              aria-label="New status key"
              maxLength={32}
              onChange={(event) =>
                setCreateStatus((current) => ({ ...current, key: event.target.value }))
              }
              placeholder="ready_for_review"
              value={createStatus.key}
            />
          </Field>
          <Field label="Name" htmlFor="wf-new-name">
            <Input
              id="wf-new-name"
              aria-label="New status name"
              maxLength={60}
              onChange={(event) =>
                setCreateStatus((current) => ({ ...current, name: event.target.value }))
              }
              placeholder="Ready for review"
              value={createStatus.name}
            />
          </Field>
          <Field label="Color" htmlFor="wf-new-color">
            <Input
              id="wf-new-color"
              aria-label="New status color"
              className="kl-color-input"
              onChange={(event) =>
                setCreateStatus((current) => ({ ...current, color: event.target.value }))
              }
              type="color"
              value={createStatus.color}
            />
          </Field>
          <Field label="Category" htmlFor="wf-new-category">
            <Select
              id="wf-new-category"
              aria-label="New status category"
              onChange={(event) =>
                setCreateStatus((current) => ({
                  ...current,
                  category: event.target.value as WorkflowStatusCategory,
                }))
              }
              value={createStatus.category}
            >
              <WorkflowCategoryOptions />
            </Select>
          </Field>
        </div>
        <Button variant="primary" disabled={creatingStatus} type="submit">
          {creatingStatus ? "Creating" : "Create status"}
        </Button>
      </form>

      <section className="kl-wf__statuses" aria-label="Active workflow statuses">
        <header className="kl-section-head">
          <div>
            <h4>Active statuses</h4>
            <p className="kl-muted">
              Drag rows or use move buttons to change board column order.
            </p>
          </div>
          {isReordering ? <span className="kl-muted">Saving order</span> : null}
        </header>
        <div className="kl-wf-status-list">
          {statuses.map((status, index) => {
            const draft = statusDrafts[status.id] ?? status;
            const isUpdating = updatingStatusIds.includes(status.id);
            const isArchiving = archivingStatusIds.includes(status.id);
            const protectsLastDone =
              status.category === "done" && doneStatusCount === 1;
            return (
              <article
                className="kl-wf-status"
                draggable={!isReordering}
                key={status.id}
                onDragOver={(event) => event.preventDefault()}
                onDragStart={(event) => {
                  setDraggedStatusId(status.id);
                  event.dataTransfer.setData("text/plain", status.id);
                }}
                onDrop={(event) => void handleDropStatus(event, status.id)}
                style={{ borderLeftColor: status.color }}
              >
                <div className="kl-wf-status__key">
                  <span
                    className="kl-wf-status__dot"
                    style={{ background: status.color }}
                  />
                  <strong>{status.key}</strong>
                  <small>Immutable key</small>
                </div>
                <div className="kl-wf-status__fields">
                  <Field label="Name" htmlFor={`wf-name-${status.id}`}>
                    <Input
                      id={`wf-name-${status.id}`}
                      aria-label={`Name for ${status.name}`}
                      maxLength={60}
                      onChange={(event) =>
                        setStatusDrafts((current) => ({
                          ...current,
                          [status.id]: { ...draft, name: event.target.value },
                        }))
                      }
                      value={draft.name}
                    />
                  </Field>
                  <Field label="Color" htmlFor={`wf-color-${status.id}`}>
                    <Input
                      id={`wf-color-${status.id}`}
                      aria-label={`Color for ${status.name}`}
                      className="kl-color-input"
                      onChange={(event) =>
                        setStatusDrafts((current) => ({
                          ...current,
                          [status.id]: { ...draft, color: event.target.value },
                        }))
                      }
                      type="color"
                      value={draft.color}
                    />
                  </Field>
                  <Field label="Category" htmlFor={`wf-category-${status.id}`}>
                    <Select
                      id={`wf-category-${status.id}`}
                      aria-label={`Category for ${status.name}`}
                      onChange={(event) =>
                        setStatusDrafts((current) => ({
                          ...current,
                          [status.id]: {
                            ...draft,
                            category: event.target.value as WorkflowStatusCategory,
                          },
                        }))
                      }
                      value={draft.category}
                    >
                      <WorkflowCategoryOptions />
                    </Select>
                  </Field>
                </div>
                <div className="kl-wf-status__actions">
                  <Button
                    variant="ghost"
                    size="sm"
                    aria-label={`Move ${status.name} up`}
                    disabled={index === 0 || isReordering}
                    onClick={() => void handleMoveStatus(status.id, -1)}
                  >
                    ↑
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    aria-label={`Move ${status.name} down`}
                    disabled={index === statuses.length - 1 || isReordering}
                    onClick={() => void handleMoveStatus(status.id, 1)}
                  >
                    ↓
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={isUpdating}
                    onClick={() => void handleUpdateStatus(status)}
                  >
                    {isUpdating ? "Saving" : "Save"}
                  </Button>
                  <Button
                    variant="danger"
                    size="sm"
                    disabled={isArchiving || protectsLastDone}
                    onClick={() => {
                      setArchiveStatusId(status.id);
                      setReplacementStatusId("");
                    }}
                    title={
                      protectsLastDone
                        ? "The workflow requires at least one active done status."
                        : undefined
                    }
                  >
                    Archive
                  </Button>
                </div>
                {protectsLastDone ? (
                  <p className="kl-wf-status__note">
                    This is the last active done status and cannot be archived.
                  </p>
                ) : null}
              </article>
            );
          })}
        </div>
      </section>

      <section className="kl-wf__transitions" aria-label="Transition matrix">
        <header className="kl-section-head">
          <div>
            <h4>Allowed transitions</h4>
            <p className="kl-muted">
              Rows are current statuses. Columns are allowed target statuses.
            </p>
          </div>
          {isSavingTransitions ? (
            <span className="kl-muted">Saving transitions</span>
          ) : null}
        </header>
        <div className="kl-wf-matrix__scroll">
          <table className="kl-wf-matrix">
            <thead>
              <tr>
                <th>From \ To</th>
                {statuses.map((status) => (
                  <th key={status.id} style={workflowStatusStyle(status)}>
                    {status.name}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {statuses.map((fromStatus) => (
                <tr key={fromStatus.id}>
                  <th style={workflowStatusStyle(fromStatus)}>{fromStatus.name}</th>
                  {statuses.map((toStatus) => {
                    const isSelf = fromStatus.id === toStatus.id;
                    const key = transitionKey(fromStatus.id, toStatus.id);
                    return (
                      <td key={toStatus.id}>
                        <input
                          aria-label={`Allow ${fromStatus.name} to ${toStatus.name}`}
                          checked={!isSelf && transitionDraft.has(key)}
                          disabled={isSelf || isSavingTransitions}
                          onChange={() => toggleTransition(fromStatus.id, toStatus.id)}
                          type="checkbox"
                        />
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="kl-wf__transition-actions">
          <Button
            variant="primary"
            disabled={isSavingTransitions}
            onClick={() => void handleSaveTransitions()}
          >
            Save transitions
          </Button>
          <Button
            variant="ghost"
            disabled={isSavingTransitions}
            onClick={resetTransitions}
          >
            Reset
          </Button>
        </div>
      </section>

      {archivedStatuses.length > 0 ? (
        <section className="kl-wf__archived" aria-label="Archived workflow statuses">
          <h4>Archived statuses</h4>
          <div className="kl-wf__archived-list">
            {archivedStatuses.map((status) => (
              <span
                className="kl-wf__archived-chip"
                key={status.id}
                style={workflowStatusStyle(status)}
              >
                {status.name} · {status.key}
              </span>
            ))}
          </div>
        </section>
      ) : null}

      {archiveStatus ? (
        <section
          aria-label={`Archive ${archiveStatus.name}`}
          aria-modal="true"
          className="kl-wf-archive"
          role="dialog"
        >
          <h4>Archive {archiveStatus.name}?</h4>
          <p className="kl-muted">
            All issues in this status will move to the replacement. Saved filters may
            show a missing status until they are updated.
          </p>
          <Field label="Replacement status" htmlFor="wf-replacement">
            <Select
              id="wf-replacement"
              aria-label="Replacement status"
              onChange={(event) => setReplacementStatusId(event.target.value)}
              value={replacementStatusId}
            >
              <option value="">Select replacement</option>
              {archiveReplacements.map((status) => (
                <option key={status.id} value={status.id}>
                  {status.name}
                </option>
              ))}
            </Select>
          </Field>
          <div className="kl-wf-archive__actions">
            <Button
              variant="danger"
              disabled={
                !replacementStatusId || archivingStatusIds.includes(archiveStatus.id)
              }
              onClick={() => void handleArchiveStatus()}
            >
              {archivingStatusIds.includes(archiveStatus.id)
                ? "Archiving"
                : "Confirm archive"}
            </Button>
            <Button
              variant="ghost"
              onClick={() => {
                setArchiveStatusId("");
                setReplacementStatusId("");
              }}
            >
              Cancel
            </Button>
          </div>
        </section>
      ) : null}
    </section>
  );
}

function WorkflowCategoryOptions() {
  return (
    <>
      {workflowStatusCategories.map((category) => (
        <option key={category} value={category}>
          {category.replace("_", " ")}
        </option>
      ))}
    </>
  );
}
