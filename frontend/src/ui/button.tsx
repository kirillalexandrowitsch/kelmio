import { type ButtonHTMLAttributes, type ReactNode } from "react";
import { type LucideIcon } from "lucide-react";
import { Icon } from "./icon";

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
type ButtonSize = "sm" | "md" | "lg";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  size?: ButtonSize;
  block?: boolean;
  icon?: LucideIcon;
  iconEnd?: LucideIcon;
  children?: ReactNode;
};

export function Button({
  variant = "secondary",
  size = "md",
  block = false,
  icon,
  iconEnd,
  className,
  children,
  type = "button",
  ...rest
}: ButtonProps) {
  const classes = [
    "kl-btn",
    `kl-btn--${variant}`,
    size !== "md" ? `kl-btn--${size}` : null,
    block ? "kl-btn--block" : null,
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button className={classes} type={type} {...rest}>
      {icon ? <Icon icon={icon} /> : null}
      {children}
      {iconEnd ? <Icon icon={iconEnd} /> : null}
    </button>
  );
}
