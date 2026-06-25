import { type ReactNode } from "react";

type AuthLayoutProps = {
  children: ReactNode;
};

export function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <main className="kl-auth">
      <div className="kl-auth__card">
        <aside className="kl-auth__brand">
          <div className="kl-auth__brand-top">
            <span className="kl-auth__mark">K</span>
            <span>Kelmio</span>
          </div>
          <div>
            <p className="kl-auth__headline">
              Ship work,
              <br />
              not busywork.
            </p>
            <p className="kl-auth__sub">
              Command-first issue tracking with custom workflows, sprints and
              automation — in one calm, light workspace.
            </p>
          </div>
        </aside>
        <section className="kl-auth__content">{children}</section>
      </div>
    </main>
  );
}
