import { type FormEvent, useEffect, useState } from "react";

import { FormError } from "../../components/form-feedback";
import { Button, Field, Input } from "../../ui";
import { AuthLayout } from "./auth-layout";
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
    <AuthLayout>
      <header className="kl-auth__heading">
        <p className="kl-auth__eyebrow">Invite onboarding</p>
        <h1>Accept workspace invite</h1>
      </header>

      {isLoadingPreview ? (
        <p className="kl-auth__muted">Loading invite...</p>
      ) : null}
      <FormError message={previewError} />

      {preview && !acceptedInvite ? (
        <>
          <article className="kl-auth__preview">
            <span className="kl-auth__preview-label">Workspace</span>
            <strong>{preview.workspace_name}</strong>
            <p>
              Invited email: {preview.email} · Role: {preview.role}
            </p>
          </article>

          <form className="kl-auth__form" onSubmit={handleSubmit}>
            <Field label="Username" htmlFor="invite-username">
              <Input
                id="invite-username"
                autoComplete="username"
                maxLength={32}
                onChange={(event) =>
                  setUsername(event.target.value.toLowerCase())
                }
                placeholder="member_name"
                value={username}
              />
            </Field>

            <Field label="Display name" htmlFor="invite-display-name">
              <Input
                id="invite-display-name"
                autoComplete="name"
                maxLength={80}
                onChange={(event) => setDisplayName(event.target.value)}
                placeholder="Member Name"
                value={displayName}
              />
            </Field>

            <Field label="Password" htmlFor="invite-password">
              <Input
                id="invite-password"
                autoComplete="new-password"
                minLength={8}
                onChange={(event) => setPassword(event.target.value)}
                placeholder="At least 8 characters"
                type="password"
                value={password}
              />
            </Field>

            <Field label="Confirm password" htmlFor="invite-confirm-password">
              <Input
                id="invite-confirm-password"
                autoComplete="new-password"
                minLength={8}
                onChange={(event) => setConfirmPassword(event.target.value)}
                placeholder="Repeat password"
                type="password"
                value={confirmPassword}
              />
            </Field>

            <FormError message={formError} />

            <Button
              variant="primary"
              size="lg"
              block
              disabled={!canSubmit}
              type="submit"
            >
              {isSubmitting ? "Accepting..." : "Accept invite"}
            </Button>
          </form>
        </>
      ) : null}

      {acceptedInvite ? (
        <div className="kl-auth__success">
          <p className="kl-auth__eyebrow">Invite accepted</p>
          <h2>Account created for @{acceptedInvite.username}</h2>
          <p>
            Sign in with username <strong>{acceptedInvite.username}</strong> and
            the password you just created.
          </p>
          <Button variant="secondary" onClick={onGoToSignIn}>
            Go to sign in
          </Button>
        </div>
      ) : null}

      {!acceptedInvite ? (
        <button className="kl-auth__link" onClick={onGoToSignIn} type="button">
          Back to sign in
        </button>
      ) : null}
    </AuthLayout>
  );
}
