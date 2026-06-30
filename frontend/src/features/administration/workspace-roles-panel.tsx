import { type FormEvent, useEffect, useState } from "react";

import { FormError } from "../../components/form-feedback";
import { Badge, Button, EmptyState, Field, Select } from "../../ui";
import {
  ApiError,
  createWorkspaceRoleAssignment,
  deleteWorkspaceRoleAssignment,
  listDirectory,
  listGroups,
  listWorkspaceRoleAssignments,
} from "../../lib/api";
import {
  type DirectoryUser,
  type Group,
  type RoleAssignment,
  type RoleAssignmentSubjectType,
  type WorkspaceRole,
} from "../../lib/api-types";

type WorkspaceRolesPanelProps = {
  workspaceId: string;
  workspaceName: string;
  onClose: () => void;
};

function adminErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback;
}

export function WorkspaceRolesPanel({
  workspaceId,
  workspaceName,
  onClose,
}: WorkspaceRolesPanelProps) {
  const [assignments, setAssignments] = useState<RoleAssignment[]>([]);
  const [directory, setDirectory] = useState<DirectoryUser[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");

  const [subjectType, setSubjectType] =
    useState<RoleAssignmentSubjectType>("user");
  const [subjectId, setSubjectId] = useState("");
  const [role, setRole] = useState<WorkspaceRole>("member");
  const [isAdding, setIsAdding] = useState(false);
  const [removingId, setRemovingId] = useState("");

  useEffect(() => {
    let isMounted = true;
    setIsLoading(true);
    setError("");
    setSubjectId("");

    Promise.all([
      listWorkspaceRoleAssignments(workspaceId),
      listDirectory(),
      listGroups(),
    ])
      .then(([assignmentsResponse, directoryResponse, groupsResponse]) => {
        if (isMounted) {
          setAssignments(assignmentsResponse.assignments);
          setDirectory(directoryResponse.users);
          setGroups(groupsResponse.groups);
        }
      })
      .catch((cause: unknown) => {
        if (isMounted) {
          setError(adminErrorMessage(cause, "Could not load role assignments."));
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
  }, [workspaceId]);

  async function handleAdd(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!subjectId || isAdding) {
      return;
    }

    setIsAdding(true);
    setError("");
    try {
      const assignment = await createWorkspaceRoleAssignment(workspaceId, {
        subject_type: subjectType,
        subject_id: subjectId,
        role,
      });
      setAssignments((current) => {
        const exists = current.some((item) => item.id === assignment.id);
        return exists
          ? current.map((item) =>
              item.id === assignment.id ? assignment : item,
            )
          : [...current, assignment];
      });
      setSubjectId("");
    } catch (cause: unknown) {
      setError(adminErrorMessage(cause, "Could not assign the role."));
    } finally {
      setIsAdding(false);
    }
  }

  async function handleRemove(assignmentId: string) {
    if (removingId) {
      return;
    }

    setRemovingId(assignmentId);
    setError("");
    try {
      await deleteWorkspaceRoleAssignment(workspaceId, assignmentId);
      setAssignments((current) =>
        current.filter((item) => item.id !== assignmentId),
      );
    } catch (cause: unknown) {
      setError(adminErrorMessage(cause, "Could not remove the assignment."));
    } finally {
      setRemovingId("");
    }
  }

  const subjectOptions =
    subjectType === "user"
      ? directory.map((person) => ({
          id: person.user_id,
          label: `${person.display_name} (@${person.username})`,
        }))
      : groups.map((group) => ({ id: group.id, label: group.name }));

  return (
    <section className="site-admin__members" aria-label="Workspace roles">
      <header className="section-header">
        <div>
          <p className="eyebrow">Roles</p>
          <h3>{workspaceName}</h3>
        </div>
        <div className="site-admin__actions">
          {isLoading ? <span className="muted">Loading</span> : null}
          <Button type="button" variant="ghost" onClick={onClose}>
            Close
          </Button>
        </div>
      </header>

      <FormError message={error} />

      <form className="site-admin__create" onSubmit={handleAdd}>
        <Field label="Subject" htmlFor="assignment-subject-type">
          <Select
            id="assignment-subject-type"
            value={subjectType}
            onChange={(event) => {
              setSubjectType(event.target.value as RoleAssignmentSubjectType);
              setSubjectId("");
            }}
            disabled={isLoading}
          >
            <option value="user">User</option>
            <option value="group">Group</option>
          </Select>
        </Field>
        <Field
          label={subjectType === "user" ? "Person" : "Group"}
          htmlFor="assignment-subject"
        >
          <Select
            id="assignment-subject"
            value={subjectId}
            onChange={(event) => setSubjectId(event.target.value)}
            disabled={isLoading || subjectOptions.length === 0}
          >
            <option value="">
              {subjectType === "user" ? "Select a person" : "Select a group"}
            </option>
            {subjectOptions.map((option) => (
              <option key={option.id} value={option.id}>
                {option.label}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Role" htmlFor="assignment-role">
          <Select
            id="assignment-role"
            value={role}
            onChange={(event) => setRole(event.target.value as WorkspaceRole)}
            disabled={isLoading}
          >
            <option value="member">Member</option>
            <option value="admin">Admin</option>
          </Select>
        </Field>
        <Button
          type="submit"
          variant="primary"
          disabled={isAdding || subjectId === ""}
        >
          {isAdding ? "Assigning…" : "Assign"}
        </Button>
      </form>

      {assignments.length > 0 ? (
        <ul className="site-admin__list">
          {assignments.map((assignment) => (
            <li className="site-admin__item" key={assignment.id}>
              <div className="site-admin__details">
                <strong>{assignment.subject_name}</strong>
                <Badge tone="default">{assignment.subject_type}</Badge>
                <Badge tone={assignment.role === "admin" ? "accent" : "info"}>
                  {assignment.role}
                </Badge>
              </div>
              <div className="site-admin__actions">
                <Button
                  type="button"
                  variant="danger"
                  onClick={() => void handleRemove(assignment.id)}
                  disabled={removingId === assignment.id}
                >
                  {removingId === assignment.id ? "Removing…" : "Remove"}
                </Button>
              </div>
            </li>
          ))}
        </ul>
      ) : isLoading ? null : (
        <EmptyState
          title="No role assignments yet"
          description="Assign a workspace role to a user or group."
        />
      )}
    </section>
  );
}
