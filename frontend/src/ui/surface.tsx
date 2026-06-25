import { type HTMLAttributes, type ReactNode } from "react";

type SurfaceProps = HTMLAttributes<HTMLDivElement> & {
  raised?: boolean;
  muted?: boolean;
  pad?: boolean;
  children?: ReactNode;
};

export function Surface({
  raised = false,
  muted = false,
  pad = false,
  className,
  children,
  ...rest
}: SurfaceProps) {
  const classes = [
    "kl-surface",
    raised ? "kl-surface--raised" : null,
    muted ? "kl-surface--muted" : null,
    pad ? "kl-surface--pad" : null,
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={classes} {...rest}>
      {children}
    </div>
  );
}
