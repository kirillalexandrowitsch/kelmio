import { type FormEvent, useEffect, useState } from "react";

import { FormError } from "../../components/form-feedback";
import {
  ApiError,
  acceptTeamInvite,
  getTeamInvitePreview,
  type AcceptInviteResponse,
  type InvitePreview,
} from "../../lib/api";
import {
  normalizedInviteAcceptInput,
  validateInviteAcceptForm,
} from "../../lib/invite-view";

type InviteAcceptScreenProps = {
  onGoToSignIn: () => void;
  token: string;
};

function inviteErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback;
}

export function InviteAcceptScreen({
  onGoToSignIn,
  token,
}: InviteAcceptScreenProps) {
  const [preview, setPreview] = useState<InvitePreview | null>(null);
  const [acceptedInvite, setAcceptedInvite] =
    useState<AcceptInviteResponse | null>(null);
  const [previewError, setPreviewError] = useState("");
  const [formError, setFormError] = useState("");
  const [isLoadingPreview, setIsLoadingPreview] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [username, setUsername] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  useEffect(() => {
    setPreview(null);
    setAcceptedInvite(null);
    setPreviewError("");
    setFormError("");
    setUsername("");
    setDisplayName("");
    setPassword("");
    setConfirmPassword("");

    if (!token) {
      setPreviewError("Invite link is missing a token.");
      setIsLoadingPreview(false);
      return;
    }

    let isMounted = true;
    setIsLoadingPreview(true);

    getTeamInvitePreview(token)
      .then((response) => {
        if (isMounted) {
          setPreview(response);
        }
      })
      .catch((error) => {
        if (isMounted) {
          setPreviewError(inviteErrorMessage(error, "Could not load invite."));
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingPreview(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [token]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError("");

    const formValues = {
      username,
      displayName,
      password,
      confirmPassword,
    };
    const validationError = validateInviteAcceptForm(formValues);
    if (validationError) {
      setFormError(validationError);
      return;
    }

    setIsSubmitting(true);

    try {
      const response = await acceptTeamInvite(
        token,
        normalizedInviteAcceptInput(formValues),
      );
      setAcceptedInvite(response);
      setPassword("");
      setConfirmPassword("");
    } catch (error) {
      setFormError(inviteErrorMessage(error, "Could not accept invite."));
    } finally {
      setIsSubmitting(false);
    }
  }

  const canSubmit =
    preview !== null &&
    !acceptedInvite &&
    username.trim() !== "" &&
    displayName.trim() !== "" &&
    password.trim().length >= 8 &&
    confirmPassword.trim().length >= 8 &&
    !isSubmitting;

  return (
    <main className="auth-shell">
      <section className="auth-panel invite-accept-panel">
        <div className="brand auth-brand">
          <span className="brand-mark">K</span>
          <div>
            <strong>Kelmio</strong>
            <span>Workspace invite</span>
          </div>
        </div>

        <div>
          <p className="eyebrow">Invite onboarding</p>
          <h1>Accept workspace invite</h1>
        </div>

        {isLoadingPreview ? <p className="muted">Loading invite...</p> : null}
        <FormError message={previewError} />

        {preview && !acceptedInvite ? (
          <>
            <article className="invite-preview-card">
              <span>Workspace</span>
              <strong>{preview.workspace_name}</strong>
              <p>
                Invited email: {preview.email} · Role: {preview.role}
              </p>
            </article>

            <form className="auth-form" onSubmit={handleSubmit}>
              <label>
                <span>Username</span>
                <input
                  autoComplete="username"
                  maxLength={32}
                  onChange={(event) => setUsername(event.target.value.toLowerCase())}
                  placeholder="member_name"
                  value={username}
                />
              </label>

              <label>
                <span>Display name</span>
                <input
                  autoComplete="name"
                  maxLength={80}
                  onChange={(event) => setDisplayName(event.target.value)}
                  placeholder="Member Name"
                  value={displayName}
                />
              </label>

              <label>
                <span>Password</span>
                <input
                  autoComplete="new-password"
                  minLength={8}
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder="At least 8 characters"
                  type="password"
                  value={password}
                />
              </label>

              <label>
                <span>Confirm password</span>
                <input
                  autoComplete="new-password"
                  minLength={8}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  placeholder="Repeat password"
                  type="password"
                  value={confirmPassword}
                />
              </label>

              <FormError message={formError} />

              <button disabled={!canSubmit} type="submit">
                {isSubmitting ? "Accepting..." : "Accept invite"}
              </button>
            </form>
          </>
        ) : null}

        {acceptedInvite ? (
          <div className="invite-success-card">
            <p className="eyebrow">Invite accepted</p>
            <h2>Account created for @{acceptedInvite.username}</h2>
            <p>
              Sign in with username <strong>{acceptedInvite.username}</strong> and
              the password you just created.
            </p>
            <button onClick={onGoToSignIn} type="button">
              Go to sign in
            </button>
          </div>
        ) : null}

        {!acceptedInvite ? (
          <button className="ghost-button" onClick={onGoToSignIn} type="button">
            Back to sign in
          </button>
        ) : null}
      </section>
    </main>
  );
}
