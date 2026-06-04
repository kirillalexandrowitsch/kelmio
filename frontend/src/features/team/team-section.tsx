import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import {
  type CurrentUser,
  type TeamInvite,
  type TeamMember,
  type UpdateTeamMemberInput,
} from "../../lib/api-types";
import { inviteStatusLabel } from "../../lib/invite-view";
import { TEAM_PERMISSION_NOTE } from "../../lib/permissions";
import { memberInitials } from "../../lib/team-view";

type TeamSectionProps = {
  canCreateTeamInvite: boolean;
  canCreateTeamMember: boolean;
  canResetTeamMemberPassword: boolean;
  copiedTeamInviteId: string;
  currentUser: CurrentUser;
  isCreatingTeamInvite: boolean;
  isActive: boolean;
  isCreatingTeamMember: boolean;
  isLoadingTeamInvites: boolean;
  isLoadingTeamMembers: boolean;
  onCancelResetPassword: () => void;
  onCopyTeamInviteLink: (inviteId: string) => void;
  onCreateTeamInvite: (event: FormEvent<HTMLFormElement>) => void;
  onCreateTeamMember: (event: FormEvent<HTMLFormElement>) => void;
  onDisplayNameChange: (value: string) => void;
  onEmailChange: (value: string) => void;
  onInviteEmailChange: (value: string) => void;
  onInviteRoleChange: (value: TeamMember["role"]) => void;
  onPasswordChange: (value: string) => void;
  onRevokeTeamInvite: (invite: TeamInvite) => void;
  onResetPassword: (
    event: FormEvent<HTMLFormElement>,
    memberId: string,
  ) => void;
  onResetPasswordChange: (value: string) => void;
  onRoleChange: (value: TeamMember["role"]) => void;
  onStartResetPassword: (memberId: string) => void;
  onUpdateTeamMember: (memberId: string, input: UpdateTeamMemberInput) => void;
  onUsernameChange: (value: string) => void;
  passwordResetMemberId: string;
  revokingTeamInviteIds: string[];
  resettingTeamMemberPasswordIds: string[];
  teamInviteEmail: string;
  teamInviteFormError: string;
  teamInviteLinksById: Record<string, string>;
  teamInviteRole: TeamMember["role"];
  teamInvites: TeamInvite[];
  teamInvitesError: string;
  teamMemberDisplayName: string;
  teamMemberEmail: string;
  teamMemberFormError: string;
  teamMemberPassword: string;
  teamMemberResetPassword: string;
  teamMemberRole: TeamMember["role"];
  teamMemberUsername: string;
  teamMembers: TeamMember[];
  teamMembersError: string;
  updatingTeamMemberIds: string[];
};

export function TeamSection({
  canCreateTeamInvite,
  canCreateTeamMember,
  canResetTeamMemberPassword,
  copiedTeamInviteId,
  currentUser,
  isCreatingTeamInvite,
  isActive,
  isCreatingTeamMember,
  isLoadingTeamInvites,
  isLoadingTeamMembers,
  onCancelResetPassword,
  onCopyTeamInviteLink,
  onCreateTeamInvite,
  onCreateTeamMember,
  onDisplayNameChange,
  onEmailChange,
  onInviteEmailChange,
  onInviteRoleChange,
  onPasswordChange,
  onRevokeTeamInvite,
  onResetPassword,
  onResetPasswordChange,
  onRoleChange,
  onStartResetPassword,
  onUpdateTeamMember,
  onUsernameChange,
  passwordResetMemberId,
  revokingTeamInviteIds,
  resettingTeamMemberPasswordIds,
  teamInviteEmail,
  teamInviteFormError,
  teamInviteLinksById,
  teamInviteRole,
  teamInvites,
  teamInvitesError,
  teamMemberDisplayName,
  teamMemberEmail,
  teamMemberFormError,
  teamMemberPassword,
  teamMemberResetPassword,
  teamMemberRole,
  teamMemberUsername,
  teamMembers,
  teamMembersError,
  updatingTeamMemberIds,
}: TeamSectionProps) {
  const isAdmin = currentUser.workspace.role === "admin";

  return (
    <section
      className="team-panel"
      aria-label="Team members"
      hidden={!isActive}
    >
      <header className="section-header">
        <div>
          <p className="eyebrow">Team</p>
          <h2>Workspace members</h2>
        </div>
        {isLoadingTeamMembers ? <span className="muted">Loading</span> : null}
      </header>

      <FormError message={teamMembersError} />

      {teamMembers.length > 0 ? (
        <div className="team-list">
          {teamMembers.map((member) => {
            const isSelf = member.id === currentUser.id;
            const isUpdatingMember = updatingTeamMemberIds.includes(member.id);
            const isResettingPassword = passwordResetMemberId === member.id;
            const isSubmittingPasswordReset =
              resettingTeamMemberPasswordIds.includes(member.id);

            return (
              <article className="team-member-row" key={member.id}>
                <span className="member-avatar">
                  {memberInitials(member.display_name)}
                </span>
                <div>
                  <h3>{member.display_name}</h3>
                  <p>
                    @{member.username} · {member.email}
                  </p>
                </div>
                <span className="member-role">{member.role}</span>
                {isAdmin ? (
                  <div className="member-controls">
                    <label>
                      <span>Role</span>
                      <select
                        disabled={isSelf || isUpdatingMember}
                        onChange={(event) => {
                          onUpdateTeamMember(member.id, {
                            role: event.target.value as TeamMember["role"],
                          });
                        }}
                        value={member.role}
                      >
                        <option value="member">Member</option>
                        <option value="admin">Admin</option>
                      </select>
                    </label>
                    <label className="member-active-toggle">
                      <input
                        checked={member.is_active}
                        disabled={isSelf || isUpdatingMember}
                        onChange={(event) => {
                          onUpdateTeamMember(member.id, {
                            is_active: event.target.checked,
                          });
                        }}
                        type="checkbox"
                      />
                      <span>{member.is_active ? "Active" : "Inactive"}</span>
                    </label>
                    <button
                      className="small-button"
                      disabled={isSelf || isUpdatingMember || isSubmittingPasswordReset}
                      onClick={() => {
                        if (isResettingPassword) {
                          onCancelResetPassword();
                        } else {
                          onStartResetPassword(member.id);
                        }
                      }}
                      type="button"
                    >
                      {isResettingPassword ? "Cancel reset" : "Reset password"}
                    </button>
                  </div>
                ) : null}
                {isAdmin && isResettingPassword ? (
                  <form
                    className="member-password-reset"
                    onSubmit={(event) => {
                      onResetPassword(event, member.id);
                    }}
                  >
                    <label>
                      <span>New password for @{member.username}</span>
                      <input
                        autoComplete="new-password"
                        minLength={8}
                        onChange={(event) => onResetPasswordChange(event.target.value)}
                        placeholder="At least 8 characters"
                        type="password"
                        value={teamMemberResetPassword}
                      />
                    </label>
                    <div className="form-actions">
                      <button
                        className="small-button"
                        disabled={
                          isSubmittingPasswordReset || !canResetTeamMemberPassword
                        }
                        type="submit"
                      >
                        {isSubmittingPasswordReset ? "Saving..." : "Save password"}
                      </button>
                      <button
                        className="ghost-button"
                        disabled={isSubmittingPasswordReset}
                        onClick={onCancelResetPassword}
                        type="button"
                      >
                        Cancel
                      </button>
                    </div>
                  </form>
                ) : null}
              </article>
            );
          })}
        </div>
      ) : (
        <div className="project-empty">No team members yet</div>
      )}

      {isAdmin ? (
        <>
          <section className="team-invite-panel" aria-label="Team invites">
            <header className="section-header">
              <div>
                <p className="eyebrow">Invites</p>
                <h2>Invite members</h2>
              </div>
              {isLoadingTeamInvites ? <span className="muted">Loading</span> : null}
            </header>

            <p className="muted">
              Create an invite link without sending email. The raw link is shown
              only right after creation.
            </p>

            <FormError message={teamInvitesError} />

            <form className="team-invite-form" onSubmit={onCreateTeamInvite}>
              <label>
                <span>Email</span>
                <input
                  autoComplete="off"
                  onChange={(event) => onInviteEmailChange(event.target.value)}
                  placeholder="member@example.com"
                  type="email"
                  value={teamInviteEmail}
                />
              </label>

              <label>
                <span>Role</span>
                <select
                  onChange={(event) =>
                    onInviteRoleChange(event.target.value as TeamMember["role"])
                  }
                  value={teamInviteRole}
                >
                  <option value="member">Member</option>
                  <option value="admin">Admin</option>
                </select>
              </label>

              <button disabled={!canCreateTeamInvite} type="submit">
                {isCreatingTeamInvite ? "Creating..." : "Create invite"}
              </button>

              <FormError message={teamInviteFormError} />
            </form>

            {teamInvites.length > 0 ? (
              <div className="team-invite-list">
                {teamInvites.map((invite) => {
                  const inviteLink = teamInviteLinksById[invite.id] ?? "";
                  const isRevokingInvite = revokingTeamInviteIds.includes(invite.id);
                  const canRevokeInvite =
                    invite.status === "pending" && !isRevokingInvite;

                  return (
                    <article className="team-invite-card" key={invite.id}>
                      <div>
                        <h3>{invite.email}</h3>
                        <p>
                          {invite.role} · Expires{" "}
                          {new Date(invite.expires_at).toLocaleDateString()}
                        </p>
                      </div>

                      <span className={`invite-status invite-status-${invite.status}`}>
                        {inviteStatusLabel(invite.status)}
                      </span>

                      {inviteLink ? (
                        <label className="invite-link-field">
                          <span>Invite link</span>
                          <input
                            aria-label={`Invite link for ${invite.email}`}
                            readOnly
                            value={inviteLink}
                          />
                        </label>
                      ) : (
                        <p className="invite-link-note">
                          Link is only available right after creation.
                        </p>
                      )}

                      <div className="invite-actions">
                        {inviteLink ? (
                          <button
                            className="small-button"
                            onClick={() => onCopyTeamInviteLink(invite.id)}
                            type="button"
                          >
                            {copiedTeamInviteId === invite.id ? "Copied" : "Copy link"}
                          </button>
                        ) : null}
                        {invite.status === "pending" ? (
                          <button
                            className="small-button danger-button"
                            disabled={!canRevokeInvite}
                            onClick={() => onRevokeTeamInvite(invite)}
                            type="button"
                          >
                            {isRevokingInvite ? "Revoking..." : "Revoke"}
                          </button>
                        ) : null}
                      </div>
                    </article>
                  );
                })}
              </div>
            ) : (
              <div className="project-empty">No invites yet</div>
            )}
          </section>

          <form className="team-member-form" onSubmit={onCreateTeamMember}>
            <label>
              <span>Email</span>
              <input
                autoComplete="off"
                onChange={(event) => onEmailChange(event.target.value)}
                placeholder="member@example.com"
                type="email"
                value={teamMemberEmail}
              />
            </label>

            <label>
              <span>Username</span>
              <input
                autoComplete="off"
                maxLength={32}
                onChange={(event) => onUsernameChange(event.target.value.toLowerCase())}
                placeholder="member_name"
                value={teamMemberUsername}
              />
            </label>

            <label>
              <span>Display name</span>
              <input
                maxLength={80}
                onChange={(event) => onDisplayNameChange(event.target.value)}
                placeholder="Member Name"
                value={teamMemberDisplayName}
              />
            </label>

            <label>
              <span>Role</span>
              <select
                onChange={(event) =>
                  onRoleChange(event.target.value as TeamMember["role"])
                }
                value={teamMemberRole}
              >
                <option value="member">Member</option>
                <option value="admin">Admin</option>
              </select>
            </label>

            <label>
              <span>Password</span>
              <input
                autoComplete="new-password"
                minLength={8}
                onChange={(event) => onPasswordChange(event.target.value)}
                placeholder="At least 8 characters"
                type="password"
                value={teamMemberPassword}
              />
            </label>

            <button disabled={!canCreateTeamMember} type="submit">
              {isCreatingTeamMember ? "Creating..." : "Create member"}
            </button>

            <FormError message={teamMemberFormError} />
          </form>
        </>
      ) : (
        <aside className="team-readonly-note permission-note">
          <header className="section-header">
            <div>
              <p className="eyebrow">{TEAM_PERMISSION_NOTE.eyebrow}</p>
              <h2>{TEAM_PERMISSION_NOTE.title}</h2>
            </div>
          </header>

          <p>{TEAM_PERMISSION_NOTE.body}</p>
        </aside>
      )}
    </section>
  );
}
