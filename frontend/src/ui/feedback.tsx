import { type HTMLAttributes, type ReactNode } from "react";

type BadgeTone =
  | "default"
  | "critical"
  | "overdue"
  | "done"
  | "info"
  | "accent";

type BadgeProps = HTMLAttributes<HTMLSpanElement> & {
  tone?: BadgeTone;
  children: ReactNode;
};

export function Badge({
  tone = "default",
  className,
  children,
  ...rest
}: BadgeProps) {
  const classes = [
    "kl-badge",
    tone !== "default" ? `kl-badge--${tone}` : null,
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <span className={classes} {...rest}>
      {children}
    </span>
  );
}

type PillProps = HTMLAttributes<HTMLSpanElement> & {
  children: ReactNode;
};

export function Pill({ className, children, ...rest }: PillProps) {
  return (
    <span className={["kl-pill", className].filter(Boolean).join(" ")} {...rest}>
      {children}
    </span>
  );
}

type EmptyStateProps = {
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  className?: string;
};

export function EmptyState({
  title,
  description,
  action,
  className,
}: EmptyStateProps) {
  return (
    <div className={["kl-empty", className].filter(Boolean).join(" ")}>
      <p className="kl-empty__title">{title}</p>
      {description ? <p>{description}</p> : null}
      {action}
    </div>
  );
}
