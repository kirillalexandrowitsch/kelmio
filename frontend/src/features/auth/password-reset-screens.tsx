import { type FormEvent, useEffect, useState } from "react";
import { ArrowLeft, ArrowRight, CheckCircle2, KeyRound, Mail } from "lucide-react";

import { FormError } from "../../components/form-feedback";
import {
  completePasswordReset,
  getPasswordResetPreview,
  requestPasswordReset,
  type PasswordResetPreview,
} from "../../lib/api";
import {
  normalizedPasswordResetComplete,
  normalizedPasswordResetEmail,
  passwordResetPreviewText,
  passwordResetTokenErrorMessage,
  validatePasswordResetComplete,
  validatePasswordResetEmail,
} from "../../lib/password-reset-view";
import { hasText } from "../../lib/validation";
import { Button, Card, Field, Input } from "../../ui";
import { AuthLayout } from "./auth-layout";

type ForgotPasswordScreenProps = {
  onGoToSignIn: () => void;
};

type ResetPasswordScreenProps = {
  onGoToSignIn: () => void;
  onResetCompleted: () => void;
  token: string;
};

function requestErrorMessage(error: unknown) {
  return error instanceof Error
    ? error.message
    : "Could not request password reset.";
}

export function ForgotPasswordScreen({ onGoToSignIn }: ForgotPasswordScreenProps) {
  const [email, setEmail] = useState("");
  const [formError, setFormError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError("");

    const validationError = validatePasswordResetEmail(email);
    if (validationError) {
      setFormError(validationError);
      return;
    }

    setIsSubmitting(true);

    try {
      await requestPasswordReset(normalizedPasswordResetEmail(email));
      setIsSuccess(true);
    } catch (error) {
      setFormError(requestErrorMessage(error));
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <AuthLayout
      description="We will send a private reset link if the account is active."
      eyebrow="Account recovery"
      title="Reset your password"
    >
        {isSuccess ? (
          <Card className="auth-state auth-state-success">
            <span className="auth-state-icon"><CheckCircle2 size={22} /></span>
            <p className="eyebrow">Check your email</p>
            <h2>Password reset instructions sent</h2>
            <p>
              If an active account exists for that email, password reset
              instructions will be sent.
            </p>
            <Button icon={<ArrowLeft size={16} />} onClick={onGoToSignIn} variant="secondary">
              Back to sign in
            </Button>
          </Card>
        ) : (
          <form className="auth-form" onSubmit={handleSubmit}>
            <Field label="Email">
              <Input
                autoComplete="email"
                autoFocus
                name="email"
                onChange={(event) => setEmail(event.target.value)}
                placeholder="member@example.com"
                type="email"
                value={email}
              />
            </Field>

            <FormError message={formError} />

            <Button
              disabled={!hasText(email) || isSubmitting}
              icon={<Mail size={17} />}
              type="submit"
            >
              {isSubmitting ? "Sending..." : "Send reset link"}
            </Button>
          </form>
        )}

        {!isSuccess ? (
          <Button icon={<ArrowLeft size={15} />} onClick={onGoToSignIn} size="sm" variant="ghost">
            Back to sign in
          </Button>
        ) : null}
    </AuthLayout>
  );
}

export function ResetPasswordScreen({
  onGoToSignIn,
  onResetCompleted,
  token,
}: ResetPasswordScreenProps) {
  const [preview, setPreview] = useState<PasswordResetPreview | null>(null);
  const [previewError, setPreviewError] = useState("");
  const [formError, setFormError] = useState("");
  const [isLoadingPreview, setIsLoadingPreview] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  useEffect(() => {
    setPreview(null);
    setPreviewError("");
    setFormError("");
    setIsSuccess(false);
    setPassword("");
    setConfirmPassword("");

    if (!token) {
      setPreviewError("Password reset link is missing a token.");
      setIsLoadingPreview(false);
      return;
    }

    let isMounted = true;
    setIsLoadingPreview(true);

    getPasswordResetPreview(token)
      .then((response) => {
        if (isMounted) {
          setPreview(response);
        }
      })
      .catch((error) => {
        if (isMounted) {
          setPreviewError(passwordResetTokenErrorMessage(error));
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

    const normalizedValues = normalizedPasswordResetComplete({
      password,
      confirmPassword,
    });
    const validationError = validatePasswordResetComplete(normalizedValues);
    if (validationError) {
      setFormError(validationError);
      return;
    }

    setIsSubmitting(true);

    try {
      await completePasswordReset(
        token,
        normalizedValues.password,
        normalizedValues.confirmPassword,
      );
      setPassword("");
      setConfirmPassword("");
      setIsSuccess(true);
      onResetCompleted();
    } catch (error) {
      setFormError(passwordResetTokenErrorMessage(error));
    } finally {
      setIsSubmitting(false);
    }
  }

  const canSubmit =
    preview !== null &&
    !isSuccess &&
    password.trim().length >= 8 &&
    confirmPassword.trim().length >= 8 &&
    !isSubmitting;

  return (
    <AuthLayout
      description="Choose a new password for the account linked to this request."
      eyebrow="Secure reset"
      title="Choose a new password"
    >
        {isLoadingPreview ? <p className="muted">Loading reset link...</p> : null}
        <FormError message={previewError} />

        {preview && !isSuccess ? (
          <>
            <Card as="article" className="auth-preview" padding="sm">
              <span className="auth-preview-icon"><KeyRound size={18} /></span>
              <div>
              <span>Reset request</span>
              <strong>{preview.email}</strong>
              <p>{passwordResetPreviewText(preview.email, preview.expires_at)}</p>
              </div>
            </Card>

            <form className="auth-form" onSubmit={handleSubmit}>
              <Field label="New password">
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
                {isSubmitting ? "Resetting..." : "Reset password"}
              </Button>
            </form>
          </>
        ) : null}

        {isSuccess ? (
          <Card className="auth-state auth-state-success">
            <span className="auth-state-icon"><CheckCircle2 size={22} /></span>
            <p className="eyebrow">Password updated</p>
            <h2>Your password has been reset</h2>
            <p>Sign in with your new password to continue.</p>
            <Button icon={<ArrowRight size={16} />} onClick={onGoToSignIn}>
              Go to sign in
            </Button>
          </Card>
        ) : null}

        {!isSuccess ? (
          <Button icon={<ArrowLeft size={15} />} onClick={onGoToSignIn} size="sm" variant="ghost">
            Back to sign in
          </Button>
        ) : null}
    </AuthLayout>
  );
}
