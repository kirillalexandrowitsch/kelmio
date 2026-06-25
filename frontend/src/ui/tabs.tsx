import { type ReactNode } from "react";

export type TabItem<T extends string = string> = {
  id: T;
  label: ReactNode;
};

type TabsProps<T extends string> = {
  items: ReadonlyArray<TabItem<T>>;
  active: T;
  onChange: (id: T) => void;
  ariaLabel?: string;
  className?: string;
};

export function Tabs<T extends string>({
  items,
  active,
  onChange,
  ariaLabel,
  className,
}: TabsProps<T>) {
  return (
    <div
      className={["kl-tabs", className].filter(Boolean).join(" ")}
      role="tablist"
      aria-label={ariaLabel}
    >
      {items.map((item) => (
        <button
          key={item.id}
          type="button"
          role="tab"
          aria-selected={item.id === active}
          className="kl-tab"
          onClick={() => onChange(item.id)}
        >
          {item.label}
        </button>
      ))}
    </div>
  );
}
