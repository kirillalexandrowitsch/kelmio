import { type FormEvent, useEffect, useState } from "react";

import { FormError } from "../../components/form-feedback";
import { Badge, Button, EmptyState, Field, Input, Select } from "../../ui";
import {
  ApiError,
  addGroupMember,
  createGroup,
  deleteGroup,
  listDirectory,
  listGroupMembers,
  listGroups,
  removeGroupMember,
  updateGroup,
} from "../../lib/api";
import {
  type DirectoryUser,
  type Group,
  type GroupMember,
} from "../../lib/api-types";

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

  const [selectedGroup, setSelectedGroup] = useState<Group | null>(null);
  const [members, setMembers] = useState<GroupMember[]>([]);
  const [directory, setDirectory] = useState<DirectoryUser[]>([]);
  const [isLoadingMembers, setIsLoadingMembers] = useState(false);
  const [membersError, setMembersError] = useState("");
  const [selectedUserId, setSelectedUserId] = useState("");
  const [isAddingMember, setIsAddingMember] = useState(false);
  const [removingMemberId, setRemovingMemberId] = useState("");

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
      if (selectedGroup?.id === group.id) {
        setSelectedGroup(null);
      }
    } catch (error: unknown) {
      setRowError(adminErrorMessage(error, "Could not delete the group."));
    } finally {
      setPendingId("");
    }
  }

  useEffect(() => {
    if (!isActive || !selectedGroup) {
      return;
    }

    let isMounted = true;
    setIsLoadingMembers(true);
    setMembersError("");
    setSelectedUserId("");

    Promise.all([listGroupMembers(selectedGroup.id), listDirectory()])
      .then(([membersResponse, directoryResponse]) => {
        if (isMounted) {
          setMembers(membersResponse.members);
          setDirectory(directoryResponse.users);
        }
      })
      .catch((error: unknown) => {
        if (isMounted) {
          setMembersError(adminErrorMessage(error, "Could not load members."));
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingMembers(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [isActive, selectedGroup]);

  function adjustMemberCount(groupID: string, delta: number) {
    setGroups((current) =>
      current.map((group) =>
        group.id === groupID
          ? { ...group, member_count: group.member_count + delta }
          : group,
      ),
    );
  }

  async function handleAddMember(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedGroup || !selectedUserId || isAddingMember) {
      return;
    }

    setIsAddingMember(true);
    setMembersError("");
    try {
      const member = await addGroupMember(selectedGroup.id, selectedUserId);
      setMembers((current) => [...current, member]);
      adjustMemberCount(selectedGroup.id, 1);
      setSelectedUserId("");
    } catch (error: unknown) {
      setMembersError(adminErrorMessage(error, "Could not add the member."));
    } finally {
      setIsAddingMember(false);
    }
  }

  async function handleRemoveMember(userID: string) {
    if (!selectedGroup || removingMemberId) {
      return;
    }

    setRemovingMemberId(userID);
    setMembersError("");
    try {
      await removeGroupMember(selectedGroup.id, userID);
      setMembers((current) => current.filter((member) => member.user_id !== userID));
      adjustMemberCount(selectedGroup.id, -1);
    } catch (error: unknown) {
      setMembersError(adminErrorMessage(error, "Could not remove the member."));
    } finally {
      setRemovingMemberId("");
    }
  }

  const availableDirectory = directory.filter(
    (person) => !members.some((member) => member.user_id === person.user_id),
  );

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
                            onClick={() => setSelectedGroup(group)}
                            disabled={isPending}
                          >
                            Members
                          </Button>
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

      {selectedGroup ? (
        <section className="site-admin__members" aria-label="Group members">
          <header className="section-header">
            <div>
              <p className="eyebrow">Members</p>
              <h3>{selectedGroup.name}</h3>
            </div>
            <div className="site-admin__actions">
              {isLoadingMembers ? <span className="muted">Loading</span> : null}
              <Button
                type="button"
                variant="ghost"
                onClick={() => setSelectedGroup(null)}
              >
                Close
              </Button>
            </div>
          </header>

          <FormError message={membersError} />

          <form className="site-admin__create" onSubmit={handleAddMember}>
            <Field label="Add member" htmlFor="add-group-member">
              <Select
                id="add-group-member"
                value={selectedUserId}
                onChange={(event) => setSelectedUserId(event.target.value)}
                disabled={isLoadingMembers || availableDirectory.length === 0}
              >
                <option value="">Select a person</option>
                {availableDirectory.map((person) => (
                  <option key={person.user_id} value={person.user_id}>
                    {person.display_name} (@{person.username})
                  </option>
                ))}
              </Select>
            </Field>
            <Button
              type="submit"
              variant="primary"
              disabled={isAddingMember || selectedUserId === ""}
            >
              {isAddingMember ? "Adding…" : "Add"}
            </Button>
          </form>

          {members.length > 0 ? (
            <ul className="site-admin__list">
              {members.map((member) => (
                <li className="site-admin__item" key={member.user_id}>
                  <div className="site-admin__details">
                    <strong>{member.display_name}</strong>
                    <span className="muted">@{member.username}</span>
                  </div>
                  <div className="site-admin__actions">
                    <Button
                      type="button"
                      variant="danger"
                      onClick={() => void handleRemoveMember(member.user_id)}
                      disabled={removingMemberId === member.user_id}
                    >
                      {removingMemberId === member.user_id ? "Removing…" : "Remove"}
                    </Button>
                  </div>
                </li>
              ))}
            </ul>
          ) : isLoadingMembers ? null : (
            <EmptyState
              title="No members yet"
              description="Add organization members to this group."
            />
          )}
        </section>
      ) : null}
    </section>
  );
}
