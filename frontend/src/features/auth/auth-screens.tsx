import { type FormEvent } from "react";
import { ArrowRight } from "lucide-react";

import { FormError } from "../../components/form-feedback";
import { Button, Field, Input } from "../../ui";
import { AuthLayout } from "./auth-layout";

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
    <AuthLayout>
      <header className="kl-auth__heading">
        <p className="kl-auth__eyebrow">Checking session</p>
        <h1>Welcome to Kelmio</h1>
      </header>
      <p className="kl-auth__muted">Restoring your workspace…</p>
    </AuthLayout>
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
    <AuthLayout>
      <header className="kl-auth__heading">
        <p className="kl-auth__eyebrow">Sign in to your workspace</p>
        <h1>Welcome back</h1>
      </header>

      <form className="kl-auth__form" onSubmit={onSubmit}>
        <Field label="Username or email" htmlFor="signin-login">
          <Input
            id="signin-login"
            autoComplete="username"
            autoFocus
            name="login"
            onChange={(event) => onLoginChange(event.target.value)}
            value={loginValue}
          />
        </Field>

        <Field label="Password" htmlFor="signin-password">
          <Input
            id="signin-password"
            autoComplete="current-password"
            name="password"
            onChange={(event) => onPasswordChange(event.target.value)}
            type="password"
            value={password}
          />
        </Field>

        <div className="kl-auth__row-end">
          <button
            className="kl-auth__link"
            onClick={onForgotPassword}
            type="button"
          >
            Forgot password?
          </button>
        </div>

        <FormError message={error} />

        <Button
          variant="primary"
          size="lg"
          block
          disabled={!canSignIn}
          iconEnd={ArrowRight}
          type="submit"
        >
          {isSubmitting ? "Signing in..." : "Sign in"}
        </Button>
      </form>
    </AuthLayout>
  );
}
