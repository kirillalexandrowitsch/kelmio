import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";

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
    <main className="auth-shell">
      <section className="auth-panel auth-panel-compact">
        <span className="brand-mark">K</span>
        <p className="eyebrow">Checking session</p>
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
    <main className="auth-shell">
      <section className="auth-panel">
        <div className="brand auth-brand">
          <span className="brand-mark">K</span>
          <div>
            <strong>Kelmio</strong>
            <span>Local workspace</span>
          </div>
        </div>

        <div>
          <p className="eyebrow">Sign in</p>
          <h1>Welcome back</h1>
        </div>

        <form className="auth-form" onSubmit={onSubmit}>
          <label>
            <span>Username or email</span>
            <input
              autoComplete="username"
              autoFocus
              name="login"
              onChange={(event) => onLoginChange(event.target.value)}
              value={loginValue}
            />
          </label>

          <label>
            <span>Password</span>
            <input
              autoComplete="current-password"
              name="password"
              onChange={(event) => onPasswordChange(event.target.value)}
              type="password"
              value={password}
            />
          </label>

          <FormError message={error} />

          <button disabled={!canSignIn} type="submit">
            {isSubmitting ? "Signing in..." : "Sign in"}
          </button>

          <button
            className="auth-link-button"
            onClick={onForgotPassword}
            type="button"
          >
            Forgot password?
          </button>
        </form>
      </section>
    </main>
  );
}
