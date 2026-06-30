import { type FormEvent, useEffect, useState } from "react";

import { FormError } from "../../components/form-feedback";
import { Badge, Button, EmptyState, Field, Input } from "../../ui";
import {
  ApiError,
  createGroup,
  deleteGroup,
  listGroups,
  updateGroup,
} from "../../lib/api";
import { type Group } from "../../lib/api-types";

type GroupsAdminSectionProps = {
  isActive: boolean;
};

function adminErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback;
}

function replaceGroup(groups: Group[], updated: Group) {
  return groups.map((group) => (group.id === updated.id ? updated : group));
}

function memberCountLabel(count: number) {
  return count === 1 ? "1 member" : `${count} members`;
}

// Shares the .site-admin layout styles with the other administration screens.
export function GroupsAdminSection({ isActive }: GroupsAdminSectionProps) {
  const [groups, setGroups] = useState<Group[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [loadError, setLoadError] = useState("");

  const [newName, setNewName] = useState("");
  const [newDescription, setNewDescription] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [createError, setCreateError] = useState("");

  const [editingId, setEditingId] = useState("");
  const [editingName, setEditingName] = useState("");
  const [editingDescription, setEditingDescription] = useState("");
  const [confirmingDeleteId, setConfirmingDeleteId] = useState("");
  const [rowError, setRowError] = useState("");
  const [pendingId, setPendingId] = useState("");

  useEffect(() => {
    if (!isActive) {
      return;
    }

    let isMounted = true;
    setIsLoading(true);
    setLoadError("");

    listGroups()
      .then((response) => {
        if (isMounted) {
          setGroups(response.groups);
        }
      })
      .catch((error: unknown) => {
        if (isMounted) {
          setLoadError(adminErrorMessage(error, "Could not load groups."));
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [isActive]);

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = newName.trim();
    if (!name || isCreating) {
      return;
    }

    setIsCreating(true);
    setCreateError("");
    try {
      const created = await createGroup(name, newDescription.trim());
      setGroups((current) => [created, ...current]);
      setNewName("");
      setNewDescription("");
    } catch (error: unknown) {
      setCreateError(adminErrorMessage(error, "Could not create the group."));
    } finally {
      setIsCreating(false);
    }
  }

  function startEditing(group: Group) {
    setEditingId(group.id);
    setEditingName(group.name);
    setEditingDescription(group.description);
    setConfirmingDeleteId("");
    setRowError("");
  }

  function cancelEditing() {
    setEditingId("");
    setEditingName("");
    setEditingDescription("");
    setRowError("");
  }

  async function handleRename(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = editingName.trim();
    if (!name || pendingId) {
      return;
    }

    setPendingId(editingId);
    setRowError("");
    try {
      const updated = await updateGroup(editingId, {
        name,
        description: editingDescription.trim(),
      });
      setGroups((current) => replaceGroup(current, updated));
      cancelEditing();
    } catch (error: unknown) {
      setRowError(adminErrorMessage(error, "Could not update the group."));
    } finally {
      setPendingId("");
    }
  }

  async function handleDelete(group: Group) {
    if (pendingId) {
      return;
    }

    setPendingId(group.id);
    setRowError("");
    try {
      await deleteGroup(group.id);
      setGroups((current) => current.filter((item) => item.id !== group.id));
      setConfirmingDeleteId("");
    } catch (error: unknown) {
      setRowError(adminErrorMessage(error, "Could not delete the group."));
    } finally {
      setPendingId("");
    }
  }

  return (
    <section
      className="site-admin"
      aria-label="Group administration"
      hidden={!isActive}
    >
      <header className="section-header">
        <div>
          <p className="eyebrow">Organization</p>
          <h2>Groups</h2>
        </div>
        {isLoading ? <span className="muted">Loading</span> : null}
      </header>

      <FormError message={loadError} />

      <form className="site-admin__create" onSubmit={handleCreate}>
        <Field label="Group name" htmlFor="new-group-name">
          <Input
            id="new-group-name"
            value={newName}
            onChange={(event) => setNewName(event.target.value)}
            placeholder="Engineers"
            autoComplete="off"
          />
        </Field>
        <Field label="Description" htmlFor="new-group-description">
          <Input
            id="new-group-description"
            value={newDescription}
            onChange={(event) => setNewDescription(event.target.value)}
            placeholder="Optional"
            autoComplete="off"
          />
        </Field>
        <Button
          type="submit"
          variant="primary"
          disabled={isCreating || newName.trim() === ""}
        >
          {isCreating ? "Creating…" : "Create group"}
        </Button>
        <FormError message={createError} />
      </form>

      {groups.length > 0 ? (
        <ul className="site-admin__list">
          {groups.map((group) => {
            const isEditing = editingId === group.id;
            const isPending = pendingId === group.id;
            const isConfirmingDelete = confirmingDeleteId === group.id;

            return (
              <li className="site-admin__item" key={group.id}>
                {isEditing ? (
                  <form className="site-admin__rename" onSubmit={handleRename}>
                    <Field label="Group name" htmlFor={`rename-${group.id}`}>
                      <Input
                        id={`rename-${group.id}`}
                        value={editingName}
                        onChange={(event) => setEditingName(event.target.value)}
                        autoComplete="off"
                      />
                    </Field>
                    <Field
                      label="Description"
                      htmlFor={`description-${group.id}`}
                    >
                      <Input
                        id={`description-${group.id}`}
                        value={editingDescription}
                        onChange={(event) =>
                          setEditingDescription(event.target.value)
                        }
                        autoComplete="off"
                      />
                    </Field>
                    <div className="site-admin__actions">
                      <Button
                        type="submit"
                        variant="primary"
                        disabled={isPending || editingName.trim() === ""}
                      >
                        {isPending ? "Saving…" : "Save"}
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        onClick={cancelEditing}
                        disabled={isPending}
                      >
                        Cancel
                      </Button>
                    </div>
                  </form>
                ) : (
                  <>
                    <div className="site-admin__details">
                      <strong>{group.name}</strong>
                      {group.description ? (
                        <span className="muted">{group.description}</span>
                      ) : null}
                      <Badge tone="info">
                        {memberCountLabel(group.member_count)}
                      </Badge>
                    </div>
                    <div className="site-admin__actions">
                      {isConfirmingDelete ? (
                        <>
                          <span className="muted">Delete group?</span>
                          <Button
                            type="button"
                            variant="danger"
                            onClick={() => void handleDelete(group)}
                            disabled={isPending}
                          >
                            {isPending ? "Deleting…" : "Confirm"}
                          </Button>
                          <Button
                            type="button"
                            variant="ghost"
                            onClick={() => setConfirmingDeleteId("")}
                            disabled={isPending}
                          >
                            Cancel
                          </Button>
                        </>
                      ) : (
                        <>
                          <Button
                            type="button"
                            onClick={() => startEditing(group)}
                            disabled={isPending}
                          >
                            Rename
                          </Button>
                          <Button
                            type="button"
                            variant="danger"
                            onClick={() => setConfirmingDeleteId(group.id)}
                            disabled={isPending}
                          >
                            Delete
                          </Button>
                        </>
                      )}
                    </div>
                  </>
                )}
              </li>
            );
          })}
        </ul>
      ) : isLoading ? null : (
        <EmptyState
          title="No groups yet"
          description="Create a group to bundle members for reusable access."
        />
      )}

      <FormError message={rowError} />
    </section>
  );
}
