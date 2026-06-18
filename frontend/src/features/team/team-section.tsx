import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import {
  type CurrentUser,
  type EmailDiagnostics,
  type TeamInvite,
  type TeamMember,
  type UpdateTeamMemberInput,
} from "../../lib/api-types";
import {
  inviteDeliveryStatusLabel,
  inviteStatusLabel,
} from "../../lib/invite-view";
import { TEAM_PERMISSION_NOTE } from "../../lib/permissions";
import { memberInitials } from "../../lib/team-view";

type TeamSectionProps = {
  canCreateTeamInvite: boolean;
  canCreateTeamMember: boolean;
  canResetTeamMemberPassword: boolean;
  copiedTeamInviteId: string;
  currentUser: CurrentUser;
  emailDiagnostics: EmailDiagnostics | null;
  emailDiagnosticsError: string;
  isCreatingTeamInvite: boolean;
  isActive: boolean;
  isCreatingTeamMember: boolean;
  isLoadingEmailDiagnostics: boolean;
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
  onRefreshEmailDiagnostics: () => void;
  onResendTeamInvite: (invite: TeamInvite) => void;
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
  resendingTeamInviteIds: string[];
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
  emailDiagnostics,
  emailDiagnosticsError,
  isCreatingTeamInvite,
  isActive,
  isCreatingTeamMember,
  isLoadingEmailDiagnostics,
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
  onRefreshEmailDiagnostics,
  onResendTeamInvite,
  onResetPassword,
  onResetPasswordChange,
  onRoleChange,
  onStartResetPassword,
  onUpdateTeamMember,
  onUsernameChange,
  passwordResetMemberId,
  revokingTeamInviteIds,
  resendingTeamInviteIds,
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
              Create an invite email and keep the raw link available as a copy
              fallback right after creation.
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
                  const isResendingInvite = resendingTeamInviteIds.includes(invite.id);
                  const canRevokeInvite =
                    invite.status === "pending" && !isRevokingInvite;
                  const canResendInvite =
                    invite.status === "pending" && !isResendingInvite;

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
                      <span
                        className={`invite-delivery-status invite-delivery-status-${invite.email_delivery_status}`}
                      >
                        Email: {inviteDeliveryStatusLabel(invite.email_delivery_status)}
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
                            className="small-button"
                            disabled={!canResendInvite}
                            onClick={() => onResendTeamInvite(invite)}
                            type="button"
                          >
                            {isResendingInvite ? "Resending..." : "Resend email"}
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

          <section
            className="email-diagnostics-panel"
            aria-label="Email delivery diagnostics"
          >
            <header className="section-header">
              <div>
                <p className="eyebrow">Email diagnostics</p>
                <h2>Delivery health</h2>
              </div>
              <button
                className="small-button"
                disabled={isLoadingEmailDiagnostics}
                onClick={onRefreshEmailDiagnostics}
                type="button"
              >
                {isLoadingEmailDiagnostics ? "Refreshing..." : "Refresh"}
              </button>
            </header>

            <p className="muted">
              Read-only outbox diagnostics for local email troubleshooting. Raw
              template data and provider secrets are not shown.
            </p>

            <FormError message={emailDiagnosticsError} />

            {emailDiagnostics ? (
              <>
                <div className="email-diagnostics-counts">
                  <article>
                    <span>Total</span>
                    <strong>{emailDiagnostics.total}</strong>
                  </article>
                  <article>
                    <span>Pending</span>
                    <strong>{emailDiagnostics.counts.pending}</strong>
                  </article>
                  <article>
                    <span>Processing</span>
                    <strong>{emailDiagnostics.counts.processing}</strong>
                  </article>
                  <article>
                    <span>Sent</span>
                    <strong>{emailDiagnostics.counts.sent}</strong>
                  </article>
                  <article>
                    <span>Failed</span>
                    <strong>{emailDiagnostics.counts.failed}</strong>
                  </article>
                </div>

                <div className="email-diagnostics-age">
                  <span>
                    Oldest pending:{" "}
                    {formatEmailDiagnosticsDate(emailDiagnostics.oldest_pending_at)}
                  </span>
                  <span>
                    Oldest processing:{" "}
                    {formatEmailDiagnosticsDate(
                      emailDiagnostics.oldest_processing_started_at,
                    )}
                  </span>
                </div>

                {emailDiagnostics.recent_terminal_failures.length > 0 ? (
                  <div className="email-failure-list">
                    <h3>Recent terminal failures</h3>
                    {emailDiagnostics.recent_terminal_failures.map((failure) => (
                      <article className="email-failure-card" key={failure.id}>
                        <div>
                          <strong>{failure.email_type}</strong>
                          <span>{failure.recipient_email}</span>
                        </div>
                        <p>{failure.last_error || "No sanitized error message"}</p>
                        <small>
                          Attempts: {failure.attempt_count} · Updated{" "}
                          {formatEmailDiagnosticsDate(failure.updated_at)}
                        </small>
                      </article>
                    ))}
                  </div>
                ) : (
                  <div className="project-empty">No terminal failures</div>
                )}
              </>
            ) : (
              <div className="project-empty">
                {isLoadingEmailDiagnostics
                  ? "Loading email diagnostics"
                  : "No email diagnostics loaded"}
              </div>
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

function formatEmailDiagnosticsDate(value: string | null) {
  if (!value) {
    return "None";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "Unknown";
  }

  return date.toLocaleString();
}
