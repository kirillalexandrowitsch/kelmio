import { type FormEvent, useEffect, useState } from "react";

import { FormError } from "../../components/form-feedback";
import { Badge, Button, EmptyState, Field, Input } from "../../ui";
import {
  ApiError,
  createWorkspace,
  listOrganizationWorkspaces,
  updateWorkspace,
} from "../../lib/api";
import { type Workspace } from "../../lib/api-types";

type OrganizationAdminSectionProps = {
  isActive: boolean;
};

function adminErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback;
}

function replaceWorkspace(workspaces: Workspace[], updated: Workspace) {
  return workspaces.map((workspace) =>
    workspace.id === updated.id ? updated : workspace,
  );
}

// Shares the .site-admin layout styles; both are list-and-create admin screens.
export function OrganizationAdminSection({
  isActive,
}: OrganizationAdminSectionProps) {
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [loadError, setLoadError] = useState("");

  const [newName, setNewName] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [createError, setCreateError] = useState("");

  const [editingId, setEditingId] = useState("");
  const [editingName, setEditingName] = useState("");
  const [rowError, setRowError] = useState("");
  const [pendingId, setPendingId] = useState("");

  useEffect(() => {
    if (!isActive) {
      return;
    }

    let isMounted = true;
    setIsLoading(true);
    setLoadError("");

    listOrganizationWorkspaces()
      .then((response) => {
        if (isMounted) {
          setWorkspaces(response.workspaces);
        }
      })
      .catch((error: unknown) => {
        if (isMounted) {
          setLoadError(adminErrorMessage(error, "Could not load workspaces."));
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
      const created = await createWorkspace(name);
      setWorkspaces((current) => [created, ...current]);
      setNewName("");
    } catch (error: unknown) {
      setCreateError(adminErrorMessage(error, "Could not create the workspace."));
    } finally {
      setIsCreating(false);
    }
  }

  function startEditing(workspace: Workspace) {
    setEditingId(workspace.id);
    setEditingName(workspace.name);
    setRowError("");
  }

  function cancelEditing() {
    setEditingId("");
    setEditingName("");
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
      const updated = await updateWorkspace(editingId, { name });
      setWorkspaces((current) => replaceWorkspace(current, updated));
      cancelEditing();
    } catch (error: unknown) {
      setRowError(adminErrorMessage(error, "Could not rename the workspace."));
    } finally {
      setPendingId("");
    }
  }

  async function handleStatusChange(
    workspace: Workspace,
    status: Workspace["status"],
  ) {
    if (pendingId) {
      return;
    }

    setPendingId(workspace.id);
    setRowError("");
    try {
      const updated = await updateWorkspace(workspace.id, { status });
      setWorkspaces((current) => replaceWorkspace(current, updated));
    } catch (error: unknown) {
      setRowError(adminErrorMessage(error, "Could not update the workspace."));
    } finally {
      setPendingId("");
    }
  }

  return (
    <section
      className="site-admin"
      aria-label="Organization administration"
      hidden={!isActive}
    >
      <header className="section-header">
        <div>
          <p className="eyebrow">Organization</p>
          <h2>Workspaces</h2>
        </div>
        {isLoading ? <span className="muted">Loading</span> : null}
      </header>

      <FormError message={loadError} />

      <form className="site-admin__create" onSubmit={handleCreate}>
        <Field label="Workspace name" htmlFor="new-workspace-name">
          <Input
            id="new-workspace-name"
            value={newName}
            onChange={(event) => setNewName(event.target.value)}
            placeholder="Marketing"
            autoComplete="off"
          />
        </Field>
        <Button
          type="submit"
          variant="primary"
          disabled={isCreating || newName.trim() === ""}
        >
          {isCreating ? "Creating…" : "Create workspace"}
        </Button>
        <FormError message={createError} />
      </form>

      {workspaces.length > 0 ? (
        <ul className="site-admin__list">
          {workspaces.map((workspace) => {
            const isEditing = editingId === workspace.id;
            const isPending = pendingId === workspace.id;

            return (
              <li className="site-admin__item" key={workspace.id}>
                {isEditing ? (
                  <form className="site-admin__rename" onSubmit={handleRename}>
                    <Field
                      label="Workspace name"
                      htmlFor={`rename-${workspace.id}`}
                    >
                      <Input
                        id={`rename-${workspace.id}`}
                        value={editingName}
                        onChange={(event) => setEditingName(event.target.value)}
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
                      <strong>{workspace.name}</strong>
                      <span className="muted">/{workspace.slug}</span>
                      <Badge
                        tone={
                          workspace.status === "active" ? "done" : "default"
                        }
                      >
                        {workspace.status}
                      </Badge>
                      {workspace.is_active ? <Badge tone="info">current</Badge> : null}
                    </div>
                    <div className="site-admin__actions">
                      <Button
                        type="button"
                        onClick={() => startEditing(workspace)}
                        disabled={isPending}
                      >
                        Rename
                      </Button>
                      {workspace.status === "active" ? (
                        <Button
                          type="button"
                          variant="danger"
                          onClick={() =>
                            void handleStatusChange(workspace, "archived")
                          }
                          disabled={isPending}
                        >
                          {isPending ? "Archiving…" : "Archive"}
                        </Button>
                      ) : (
                        <Button
                          type="button"
                          onClick={() =>
                            void handleStatusChange(workspace, "active")
                          }
                          disabled={isPending}
                        >
                          {isPending ? "Restoring…" : "Restore"}
                        </Button>
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
          title="No workspaces yet"
          description="Create the first workspace for this organization."
        />
      )}

      <FormError message={rowError} />
    </section>
  );
}
