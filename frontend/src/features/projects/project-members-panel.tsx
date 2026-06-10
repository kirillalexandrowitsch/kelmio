import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import {
  type ProjectMember,
  type ProjectRole,
  type TeamMember,
} from "../../lib/api-types";
import { memberInitials } from "../../lib/team-view";

type ProjectMembersPanelProps = {
  error: string;
  isLoading: boolean;
  members: ProjectMember[];
  onAddMember: (event: FormEvent<HTMLFormElement>) => void;
  onMemberRoleChange: (member: ProjectMember, role: ProjectRole) => void;
  onRemoveMember: (member: ProjectMember) => void;
  onRoleChange: (role: ProjectRole) => void;
  onUserChange: (userId: string) => void;
  removingMemberIds: string[];
  role: ProjectRole;
  selectedUserId: string;
  teamMembers: TeamMember[];
  updatingMemberIds: string[];
};

export function ProjectMembersPanel({
  error,
  isLoading,
  members,
  onAddMember,
  onMemberRoleChange,
  onRemoveMember,
  onRoleChange,
  onUserChange,
  removingMemberIds,
  role,
  selectedUserId,
  teamMembers,
  updatingMemberIds,
}: ProjectMembersPanelProps) {
  const memberIds = new Set(members.map((member) => member.user_id));
  const availableMembers = teamMembers.filter(
    (member) => member.is_active && !memberIds.has(member.id),
  );
  const hasActiveWorkspaceAdmin = teamMembers.some(
    (member) => member.is_active && member.role === "admin",
  );
  const activeLeadCount = members.filter(
    (member) => member.is_active && member.role === "lead",
  ).length;

  return (
    <section className="project-members-panel" aria-label="Project members">
      <header className="project-members-header">
        <div>
          <h3>Project members</h3>
          <p>Leads manage access. Contributors write. Viewers are read-only.</p>
        </div>
        {isLoading ? <span className="muted">Loading</span> : null}
      </header>

      <FormError message={error} />

      <form className="project-member-add-form" onSubmit={onAddMember}>
        <label>
          <span>Workspace member</span>
          <select
            disabled={isLoading || availableMembers.length === 0}
            onChange={(event) => onUserChange(event.target.value)}
            value={selectedUserId}
          >
            <option value="">Select member</option>
            {availableMembers.map((member) => (
              <option key={member.id} value={member.id}>
                {member.display_name} (@{member.username})
              </option>
            ))}
          </select>
        </label>
        <label>
          <span>Project role</span>
          <select
            aria-label="New project member role"
            disabled={isLoading || availableMembers.length === 0}
            onChange={(event) => onRoleChange(event.target.value as ProjectRole)}
            value={role}
          >
            <ProjectRoleOptions />
          </select>
        </label>
        <button disabled={isLoading || !selectedUserId} type="submit">
          Add member
        </button>
      </form>

      {availableMembers.length === 0 ? (
        <p className="muted project-members-availability">
          All active workspace members already belong to this project.
        </p>
      ) : null}

      {members.length > 0 ? (
        <div className="project-member-list">
          {members.map((member) => {
            const isUpdating = updatingMemberIds.includes(member.user_id);
            const isRemoving = removingMemberIds.includes(member.user_id);
            const isProtectedLastLead =
              member.is_active &&
              member.role === "lead" &&
              activeLeadCount === 1 &&
              !hasActiveWorkspaceAdmin;
            const controlsDisabled =
              !member.is_active || isUpdating || isRemoving || isProtectedLastLead;

            return (
              <article className="project-member-card" key={member.user_id}>
                <span className="member-avatar">
                  {memberInitials(member.display_name)}
                </span>
                <div className="project-member-identity">
                  <h4>{member.display_name}</h4>
                  <p>
                    @{member.username} · {member.email}
                  </p>
                  <div className="project-member-badges">
                    <span className="member-role">{member.role}</span>
                    {member.workspace_role === "admin" ? (
                      <span className="member-role">Workspace admin</span>
                    ) : null}
                    {!member.is_active ? (
                      <span className="member-role member-role-inactive">Inactive</span>
                    ) : null}
                  </div>
                </div>
                <div className="project-member-controls">
                  <label>
                    <span>Role</span>
                    <select
                      aria-label={`Project role for ${member.display_name}`}
                      disabled={controlsDisabled}
                      onChange={(event) =>
                        onMemberRoleChange(
                          member,
                          event.target.value as ProjectRole,
                        )
                      }
                      value={member.role}
                    >
                      <ProjectRoleOptions />
                    </select>
                  </label>
                  <button
                    className="small-button danger-button"
                    disabled={isUpdating || isRemoving || isProtectedLastLead}
                    onClick={() => onRemoveMember(member)}
                    type="button"
                  >
                    {isRemoving ? "Removing" : "Remove"}
                  </button>
                </div>
                {member.workspace_role === "admin" ? (
                  <p className="project-member-note">
                    Workspace admins keep full project access even without this
                    membership row.
                  </p>
                ) : null}
                {isProtectedLastLead ? (
                  <p className="project-member-note">
                    Add another active lead before changing or removing this lead.
                  </p>
                ) : null}
                {!member.is_active ? (
                  <p className="project-member-note">
                    Inactive memberships remain visible for audit and can be removed.
                  </p>
                ) : null}
              </article>
            );
          })}
        </div>
      ) : isLoading ? null : (
        <div className="comments-empty">No project members</div>
      )}
    </section>
  );
}

function ProjectRoleOptions() {
  return (
    <>
      <option value="lead">Lead</option>
      <option value="contributor">Contributor</option>
      <option value="viewer">Viewer</option>
    </>
  );
}
