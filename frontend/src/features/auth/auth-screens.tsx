import { type FormEvent } from "react";
import { ArrowRight, KeyRound, LoaderCircle, UserRound } from "lucide-react";

import { FormError } from "../../components/form-feedback";
import { Button, Field, Input } from "../../ui";
import { AuthLayout, KelmioMark } from "./auth-layout";

type SignInScreenProps = {
  canSignIn: boolean;
  error: string;
  isSubmitting: boolean;
  loginValue: string;
  onForgotPassword: () => void;
  onLoginChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  password: string;
};

export function BootingScreen() {
  return (
    <main className="auth-shell auth-booting">
      <section aria-live="polite" className="auth-panel auth-panel-compact">
        <KelmioMark />
        <LoaderCircle className="auth-spinner" size={20} />
        <p>Checking session</p>
      </section>
    </main>
  );
}

export function SignInScreen({
  canSignIn,
  error,
  isSubmitting,
  loginValue,
  onForgotPassword,
  onLoginChange,
  onPasswordChange,
  onSubmit,
  password,
}: SignInScreenProps) {
  return (
    <AuthLayout
      description="Enter your workspace credentials to continue."
      eyebrow="Workspace access"
      title="Welcome back"
    >
      <form className="auth-form" onSubmit={onSubmit}>
          <Field label="Username or email">
            <Input
              autoComplete="username"
              autoFocus
              name="login"
              onChange={(event) => onLoginChange(event.target.value)}
              placeholder="admin or member@example.com"
              value={loginValue}
            />
          </Field>

          <Field label="Password">
            <Input
              autoComplete="current-password"
              name="password"
              onChange={(event) => onPasswordChange(event.target.value)}
              placeholder="Enter your password"
              type="password"
              value={password}
            />
          </Field>

          <FormError message={error} />

          <Button
            disabled={!canSignIn}
            icon={isSubmitting ? <LoaderCircle className="auth-spinner" size={17} /> : <ArrowRight size={17} />}
            type="submit"
          >
            {isSubmitting ? "Signing in..." : "Sign in"}
          </Button>

          <Button
            className="auth-link-button"
            icon={<KeyRound size={15} />}
            onClick={onForgotPassword}
            size="sm"
            variant="ghost"
          >
            Forgot password?
          </Button>
        </form>
      <div className="auth-security-note">
        <UserRound size={15} />
        <span>Your session remains inside this Kelmio installation.</span>
      </div>
    </AuthLayout>
  );
}
