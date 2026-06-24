import { type HTMLAttributes, type ReactNode } from "react";

type CardProps = HTMLAttributes<HTMLElement> & {
  as?: "article" | "div" | "section";
  padding?: "sm" | "md" | "lg";
};

export function Card({
  as: Component = "div",
  className = "",
  padding = "md",
  ...props
}: CardProps) {
  return (
    <Component
      className={`ui-card ${className}`.trim()}
      data-padding={padding}
      {...props}
    />
  );
}

type BadgeProps = HTMLAttributes<HTMLSpanElement> & {
  icon?: ReactNode;
  tone?: "neutral" | "success" | "warning" | "danger";
};

export function Badge({
  children,
  className = "",
  icon,
  tone = "neutral",
  ...props
}: BadgeProps) {
  return (
    <span
      className={`ui-badge ${className}`.trim()}
      data-tone={tone}
      {...props}
    >
      {icon}
      {children}
    </span>
  );
}
