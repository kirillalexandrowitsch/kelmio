import { type LucideIcon } from "lucide-react";

type IconProps = {
  icon: LucideIcon;
  size?: number;
  className?: string;
  /** Accessible label. When omitted, the icon is decorative (aria-hidden). */
  label?: string;
};

export function Icon({ icon: Glyph, size = 16, className, label }: IconProps) {
  return (
    <Glyph
      className={["kl-icon", className].filter(Boolean).join(" ")}
      size={size}
      strokeWidth={1.75}
      aria-hidden={label ? undefined : true}
      aria-label={label}
      role={label ? "img" : undefined}
    />
  );
}
