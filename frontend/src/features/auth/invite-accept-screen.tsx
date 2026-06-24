import { type FormEvent, useEffect, useState } from "react";
import { ArrowLeft, ArrowRight, CheckCircle2, Mail, UsersRound } from "lucide-react";

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
import { Badge, Button, Card, Field, Input } from "../../ui";
import { AuthLayout } from "./auth-layout";

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
    <AuthLayout
      description="Join the workspace and create your private Kelmio account."
      eyebrow="Invite onboarding"
      title="Accept workspace invite"
    >
        {isLoadingPreview ? <p className="muted">Loading invite...</p> : null}
        <FormError message={previewError} />

        {preview && !acceptedInvite ? (
          <>
            <Card as="article" className="auth-preview" padding="sm">
              <span className="auth-preview-icon"><UsersRound size={18} /></span>
              <div>
                <span>Workspace</span>
                <strong>{preview.workspace_name}</strong>
                <p><Mail size={13} /> {preview.email}</p>
              </div>
              <Badge tone="success">{preview.role}</Badge>
            </Card>

            <form className="auth-form" onSubmit={handleSubmit}>
              <Field label="Username">
                <Input
                  autoComplete="username"
                  maxLength={32}
                  onChange={(event) => setUsername(event.target.value.toLowerCase())}
                  placeholder="member_name"
                  value={username}
                />
              </Field>

              <Field label="Display name">
                <Input
                  autoComplete="name"
                  maxLength={80}
                  onChange={(event) => setDisplayName(event.target.value)}
                  placeholder="Member Name"
                  value={displayName}
                />
              </Field>

              <Field label="Password">
                <Input
                  autoComplete="new-password"
                  minLength={8}
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder="At least 8 characters"
                  type="password"
                  value={password}
                />
              </Field>

              <Field label="Confirm password">
                <Input
                  autoComplete="new-password"
                  minLength={8}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  placeholder="Repeat password"
                  type="password"
                  value={confirmPassword}
                />
              </Field>

              <FormError message={formError} />

              <Button disabled={!canSubmit} icon={<ArrowRight size={17} />} type="submit">
                {isSubmitting ? "Accepting..." : "Accept invite"}
              </Button>
            </form>
          </>
        ) : null}

        {acceptedInvite ? (
          <Card className="auth-state auth-state-success">
            <span className="auth-state-icon"><CheckCircle2 size={22} /></span>
            <p className="eyebrow">Invite accepted</p>
            <h2>Account created for @{acceptedInvite.username}</h2>
            <p>
              Sign in with username <strong>{acceptedInvite.username}</strong> and
              the password you just created.
            </p>
            <Button icon={<ArrowRight size={16} />} onClick={onGoToSignIn}>
              Go to sign in
            </Button>
          </Card>
        ) : null}

        {!acceptedInvite ? (
          <Button icon={<ArrowLeft size={15} />} onClick={onGoToSignIn} size="sm" variant="ghost">
            Back to sign in
          </Button>
        ) : null}
      </AuthLayout>
  );
}
