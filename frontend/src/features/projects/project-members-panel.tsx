import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import {
  type ProjectMember,
  type ProjectRole,
  type TeamMember,
} from "../../lib/api-types";
import { memberInitials } from "../../lib/team-view";
import { Badge, Button, Field, Select } from "../../ui";

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
    <section className="kl-members" aria-label="Project members">
      <header className="kl-section-head">
        <div>
          <h3>Project members</h3>
          <p className="kl-muted">
            Leads manage access. Contributors write. Viewers are read-only.
          </p>
        </div>
        {isLoading ? <span className="kl-muted">Loading</span> : null}
      </header>

      <FormError message={error} />

      <form className="kl-members__add" onSubmit={onAddMember}>
        <Field label="Workspace member" htmlFor="member-user">
          <Select
            id="member-user"
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
          </Select>
        </Field>
        <Field label="Project role" htmlFor="member-role">
          <Select
            id="member-role"
            aria-label="New project member role"
            disabled={isLoading || availableMembers.length === 0}
            onChange={(event) => onRoleChange(event.target.value as ProjectRole)}
            value={role}
          >
            <ProjectRoleOptions />
          </Select>
        </Field>
        <Button
          variant="primary"
          disabled={isLoading || !selectedUserId}
          type="submit"
        >
          Add member
        </Button>
      </form>

      {availableMembers.length === 0 ? (
        <p className="kl-muted">
          All active workspace members already belong to this project.
        </p>
      ) : null}

      {members.length > 0 ? (
        <div className="kl-members__list">
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
              <article className="project-member-card kl-member" key={member.user_id}>
                <span className="kl-member__avatar">
                  {memberInitials(member.display_name)}
                </span>
                <div className="kl-member__identity">
                  <h4>{member.display_name}</h4>
                  <p>
                    @{member.username} · {member.email}
                  </p>
                  <div className="kl-member__badges">
                    <Badge tone="accent">{member.role}</Badge>
                    {member.workspace_role === "admin" ? (
                      <Badge tone="info">Workspace admin</Badge>
                    ) : null}
                    {!member.is_active ? <Badge>Inactive</Badge> : null}
                  </div>
                </div>
                <div className="kl-member__controls">
                  <Field label="Role" htmlFor={`member-role-${member.user_id}`}>
                    <Select
                      id={`member-role-${member.user_id}`}
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
                    </Select>
                  </Field>
                  <Button
                    variant="danger"
                    size="sm"
                    disabled={isUpdating || isRemoving || isProtectedLastLead}
                    onClick={() => onRemoveMember(member)}
                  >
                    {isRemoving ? "Removing" : "Remove"}
                  </Button>
                </div>
                {member.workspace_role === "admin" ? (
                  <p className="kl-member__note">
                    Workspace admins keep full project access even without this
                    membership row.
                  </p>
                ) : null}
                {isProtectedLastLead ? (
                  <p className="kl-member__note">
                    Add another active lead before changing or removing this lead.
                  </p>
                ) : null}
                {!member.is_active ? (
                  <p className="kl-member__note">
                    Inactive memberships remain visible for audit and can be removed.
                  </p>
                ) : null}
              </article>
            );
          })}
        </div>
      ) : isLoading ? null : (
        <div className="kl-empty-block">No project members</div>
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
