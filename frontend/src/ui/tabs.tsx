import { type ReactNode } from "react";

export type TabItem<T extends string> = {
  id: T;
  label: string;
};

type TabsProps<T extends string> = {
  activeTab: T;
  ariaLabel: string;
  items: Array<TabItem<T>>;
  onChange: (tab: T) => void;
};

export function Tabs<T extends string>({
  activeTab,
  ariaLabel,
  items,
  onChange,
}: TabsProps<T>) {
  return (
    <div aria-label={ariaLabel} className="ui-tabs" role="tablist">
      {items.map((item) => (
        <button
          aria-selected={activeTab === item.id}
          className="ui-tab"
          key={item.id}
          onClick={() => onChange(item.id)}
          role="tab"
          type="button"
        >
          {item.label}
        </button>
      ))}
    </div>
  );
}

export function TabPanel({ children, labelledBy }: { children: ReactNode; labelledBy?: string }) {
  return (
    <div aria-labelledby={labelledBy} role="tabpanel">
      {children}
    </div>
  );
}
