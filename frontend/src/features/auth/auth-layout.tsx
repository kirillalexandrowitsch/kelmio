import { ArrowUpRight, Boxes, ShieldCheck, Sparkles } from "lucide-react";
import { type ReactNode } from "react";
import { KelmioMark } from "../../components/kelmio-mark";

type AuthLayoutProps = {
  children: ReactNode;
  description: string;
  eyebrow: string;
  title: string;
};

export function AuthLayout({
  children,
  description,
  eyebrow,
  title,
}: AuthLayoutProps) {
  return (
    <main className="auth-shell auth-experience">
      <aside className="auth-story" aria-label="About Kelmio">
        <div className="auth-story-topline">
          <div className="auth-wordmark">
            <KelmioMark />
            <strong>Kelmio</strong>
          </div>
          <span className="auth-local-pill">
            <ShieldCheck size={14} /> Local-first
          </span>
        </div>

        <div className="auth-story-copy">
          <span className="auth-story-kicker">
            <Sparkles size={15} /> Connected work, without the noise
          </span>
          <h2>
            Plan deeply.
            <br />
            Move <em>clearly.</em>
          </h2>
          <p>
            Projects, workflows and team decisions stay in one private,
            focused workspace.
          </p>
        </div>

        <div className="auth-story-card">
          <span className="auth-story-card-icon">
            <Boxes size={18} />
          </span>
          <div>
            <strong>One connected workspace</strong>
            <span>Issues, sprints, workflows and automation</span>
          </div>
          <ArrowUpRight size={18} />
        </div>
      </aside>

      <section className="auth-stage">
        <div className="auth-mobile-brand">
          <KelmioMark />
          <strong>Kelmio</strong>
        </div>
        <section className="auth-panel">
          <header className="auth-heading">
            <p className="eyebrow">{eyebrow}</p>
            <h1>{title}</h1>
            <p>{description}</p>
          </header>
          {children}
        </section>
        <p className="auth-footnote">Private workspace · Local control</p>
      </section>
    </main>
  );
}
