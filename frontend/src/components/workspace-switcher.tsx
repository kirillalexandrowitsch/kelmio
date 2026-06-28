import { useEffect, useRef, useState } from "react";
import { Check, ChevronsUpDown } from "lucide-react";

import { type Workspace } from "../lib/api-types";
import { Icon } from "../ui";

type WorkspaceSwitcherProps = {
  workspaces: Workspace[];
  activeWorkspaceId: string;
  onSwitch: (workspaceId: string) => void;
  isSwitching: boolean;
};

export function WorkspaceSwitcher({
  workspaces,
  activeWorkspaceId,
  onSwitch,
  isSwitching,
}: WorkspaceSwitcherProps) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    function handlePointer(event: MouseEvent) {
      if (
        containerRef.current &&
        !containerRef.current.contains(event.target as Node)
      ) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handlePointer);
    return () => document.removeEventListener("mousedown", handlePointer);
  }, [open]);

  if (workspaces.length === 0) {
    return null;
  }

  const active =
    workspaces.find((workspace) => workspace.id === activeWorkspaceId) ??
    workspaces.find((workspace) => workspace.is_active) ??
    workspaces[0];
  const canSwitch = workspaces.length > 1;

  function handleSelect(workspaceId: string) {
    setOpen(false);
    if (workspaceId !== active.id) {
      onSwitch(workspaceId);
    }
  }

  return (
    <div className="kl-workspace-switcher" ref={containerRef}>
      <button
        aria-label="Switch workspace"
        aria-expanded={canSwitch ? open : undefined}
        aria-haspopup={canSwitch ? "menu" : undefined}
        className="kl-workspace-switcher__trigger"
        disabled={!canSwitch || isSwitching}
        onClick={() => setOpen((value) => !value)}
        type="button"
      >
        <span className="kl-workspace-switcher__label">
          <span className="kl-workspace-switcher__eyebrow">Workspace</span>
          <strong>{active.name}</strong>
        </span>
        {canSwitch ? <Icon icon={ChevronsUpDown} size={16} /> : null}
      </button>
      {open && canSwitch ? (
        <div
          aria-label="Workspaces"
          className="kl-workspace-switcher__menu"
          role="menu"
        >
          {workspaces.map((workspace) => (
            <button
              aria-current={workspace.id === active.id ? "true" : undefined}
              className="kl-workspace-switcher__item"
              key={workspace.id}
              onClick={() => handleSelect(workspace.id)}
              role="menuitem"
              type="button"
            >
              <span>{workspace.name}</span>
              {workspace.id === active.id ? (
                <Icon icon={Check} size={16} />
              ) : null}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}
