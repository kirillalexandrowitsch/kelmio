import { type CSSProperties, type ReactNode } from "react";
import { Inbox } from "lucide-react";

type EmptyStateProps = {
  action?: ReactNode;
  description: string;
  icon?: ReactNode;
  title: string;
};

export function EmptyState({ action, description, icon, title }: EmptyStateProps) {
  return (
    <div className="ui-empty-state">
      <span aria-hidden="true" className="ui-empty-state-icon">
        {icon ?? <Inbox size={20} />}
      </span>
      <h3>{title}</h3>
      <p>{description}</p>
      {action}
    </div>
  );
}

export function Skeleton({ height = 12, width = "100%" }: { height?: number; width?: CSSProperties["width"] }) {
  return <span aria-hidden="true" className="ui-skeleton" style={{ height, width }} />;
}

export function ToastRegion({ children }: { children: ReactNode }) {
  return (
    <div aria-live="polite" className="ui-toast-region">
      {children}
    </div>
  );
}

export function Toast({ children }: { children: ReactNode }) {
  return <div className="ui-toast">{children}</div>;
}
