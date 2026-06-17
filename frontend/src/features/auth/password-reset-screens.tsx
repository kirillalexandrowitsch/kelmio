import { type FormEvent, useEffect, useState } from "react";

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
    <main className="auth-shell">
      <section className="auth-panel">
        <div className="brand auth-brand">
          <span className="brand-mark">TT</span>
          <div>
            <strong>Team Task Tracker</strong>
            <span>Account recovery</span>
          </div>
        </div>

        <div>
          <p className="eyebrow">Password reset</p>
          <h1>Reset your password</h1>
        </div>

        {isSuccess ? (
          <div className="invite-success-card">
            <p className="eyebrow">Check your email</p>
            <h2>Password reset instructions sent</h2>
            <p>
              If an active account exists for that email, password reset
              instructions will be sent.
            </p>
            <button onClick={onGoToSignIn} type="button">
              Back to sign in
            </button>
          </div>
        ) : (
          <form className="auth-form" onSubmit={handleSubmit}>
            <label>
              <span>Email</span>
              <input
                autoComplete="email"
                autoFocus
                name="email"
                onChange={(event) => setEmail(event.target.value)}
                placeholder="member@example.com"
                type="email"
                value={email}
              />
            </label>

            <FormError message={formError} />

            <button disabled={!hasText(email) || isSubmitting} type="submit">
              {isSubmitting ? "Sending..." : "Send reset link"}
            </button>
          </form>
        )}

        {!isSuccess ? (
          <button className="ghost-button" onClick={onGoToSignIn} type="button">
            Back to sign in
          </button>
        ) : null}
      </section>
    </main>
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
    <main className="auth-shell">
      <section className="auth-panel">
        <div className="brand auth-brand">
          <span className="brand-mark">TT</span>
          <div>
            <strong>Team Task Tracker</strong>
            <span>Account recovery</span>
          </div>
        </div>

        <div>
          <p className="eyebrow">Password reset</p>
          <h1>Choose a new password</h1>
        </div>

        {isLoadingPreview ? <p className="muted">Loading reset link...</p> : null}
        <FormError message={previewError} />

        {preview && !isSuccess ? (
          <>
            <article className="invite-preview-card">
              <span>Reset request</span>
              <strong>{preview.email}</strong>
              <p>{passwordResetPreviewText(preview.email, preview.expires_at)}</p>
            </article>

            <form className="auth-form" onSubmit={handleSubmit}>
              <label>
                <span>New password</span>
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
                {isSubmitting ? "Resetting..." : "Reset password"}
              </button>
            </form>
          </>
        ) : null}

        {isSuccess ? (
          <div className="invite-success-card">
            <p className="eyebrow">Password updated</p>
            <h2>Your password has been reset</h2>
            <p>Sign in with your new password to continue.</p>
            <button onClick={onGoToSignIn} type="button">
              Go to sign in
            </button>
          </div>
        ) : null}

        {!isSuccess ? (
          <button className="ghost-button" onClick={onGoToSignIn} type="button">
            Back to sign in
          </button>
        ) : null}
      </section>
    </main>
  );
}
