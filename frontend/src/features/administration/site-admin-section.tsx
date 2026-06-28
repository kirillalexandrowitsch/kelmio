import { type FormEvent, useEffect, useState } from "react";

import { FormError } from "../../components/form-feedback";
import { Badge, Button, EmptyState, Field, Input } from "../../ui";
import {
  ApiError,
  createOrganization,
  listOrganizations,
  updateOrganization,
} from "../../lib/api";
import { type Organization } from "../../lib/api-types";

type SiteAdminSectionProps = {
  isActive: boolean;
};

function adminErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback;
}

function replaceOrganization(
  organizations: Organization[],
  updated: Organization,
) {
  return organizations.map((organization) =>
    organization.id === updated.id ? updated : organization,
  );
}

export function SiteAdminSection({ isActive }: SiteAdminSectionProps) {
  const [organizations, setOrganizations] = useState<Organization[]>([]);
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

    listOrganizations()
      .then((response) => {
        if (isMounted) {
          setOrganizations(response.organizations);
        }
      })
      .catch((error: unknown) => {
        if (isMounted) {
          setLoadError(
            adminErrorMessage(error, "Could not load organizations."),
          );
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
      const created = await createOrganization(name);
      setOrganizations((current) => [created, ...current]);
      setNewName("");
    } catch (error: unknown) {
      setCreateError(
        adminErrorMessage(error, "Could not create the organization."),
      );
    } finally {
      setIsCreating(false);
    }
  }

  function startEditing(organization: Organization) {
    setEditingId(organization.id);
    setEditingName(organization.name);
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
      const updated = await updateOrganization(editingId, { name });
      setOrganizations((current) => replaceOrganization(current, updated));
      cancelEditing();
    } catch (error: unknown) {
      setRowError(adminErrorMessage(error, "Could not rename the organization."));
    } finally {
      setPendingId("");
    }
  }

  async function handleStatusChange(
    organization: Organization,
    status: Organization["status"],
  ) {
    if (pendingId) {
      return;
    }

    setPendingId(organization.id);
    setRowError("");
    try {
      const updated = await updateOrganization(organization.id, { status });
      setOrganizations((current) => replaceOrganization(current, updated));
    } catch (error: unknown) {
      setRowError(
        adminErrorMessage(error, "Could not update the organization."),
      );
    } finally {
      setPendingId("");
    }
  }

  return (
    <section
      className="site-admin"
      aria-label="Site administration"
      hidden={!isActive}
    >
      <header className="section-header">
        <div>
          <p className="eyebrow">Administration</p>
          <h2>Organizations</h2>
        </div>
        {isLoading ? <span className="muted">Loading</span> : null}
      </header>

      <FormError message={loadError} />

      <form className="site-admin__create" onSubmit={handleCreate}>
        <Field label="Organization name" htmlFor="new-organization-name">
          <Input
            id="new-organization-name"
            value={newName}
            onChange={(event) => setNewName(event.target.value)}
            placeholder="Acme Inc."
            autoComplete="off"
          />
        </Field>
        <Button
          type="submit"
          variant="primary"
          disabled={isCreating || newName.trim() === ""}
        >
          {isCreating ? "Creating…" : "Create organization"}
        </Button>
        <FormError message={createError} />
      </form>

      {organizations.length > 0 ? (
        <ul className="site-admin__list">
          {organizations.map((organization) => {
            const isEditing = editingId === organization.id;
            const isPending = pendingId === organization.id;

            return (
              <li className="site-admin__item" key={organization.id}>
                {isEditing ? (
                  <form className="site-admin__rename" onSubmit={handleRename}>
                    <Field
                      label="Organization name"
                      htmlFor={`rename-${organization.id}`}
                    >
                      <Input
                        id={`rename-${organization.id}`}
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
                      <strong>{organization.name}</strong>
                      <span className="muted">/{organization.slug}</span>
                      <Badge
                        tone={
                          organization.status === "active" ? "done" : "default"
                        }
                      >
                        {organization.status}
                      </Badge>
                    </div>
                    <div className="site-admin__actions">
                      <Button
                        type="button"
                        onClick={() => startEditing(organization)}
                        disabled={isPending}
                      >
                        Rename
                      </Button>
                      {organization.status === "active" ? (
                        <Button
                          type="button"
                          variant="danger"
                          onClick={() =>
                            void handleStatusChange(organization, "archived")
                          }
                          disabled={isPending}
                        >
                          {isPending ? "Archiving…" : "Archive"}
                        </Button>
                      ) : (
                        <Button
                          type="button"
                          onClick={() =>
                            void handleStatusChange(organization, "active")
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
          title="No organizations yet"
          description="Create the first organization to start administering Kelmio."
        />
      )}

      <FormError message={rowError} />
    </section>
  );
}
