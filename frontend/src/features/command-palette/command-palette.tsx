import {
  type KeyboardEvent as ReactKeyboardEvent,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Search } from "lucide-react";

import { Icon, Modal } from "../../ui";
import { type AppSection } from "../../lib/routing";

export type CommandPaletteIssue = {
  id: string;
  issue_key: string;
  title: string;
};

type Command = {
  id: string;
  group: string;
  label: string;
  hint?: string;
  keywords: string;
  run: () => void;
};

type CommandPaletteProps = {
  open: boolean;
  onClose: () => void;
  onNavigate: (section: AppSection) => void;
  onOpenIssue: (issueId: string) => void;
  recentIssues: CommandPaletteIssue[];
};

export function CommandPalette({
  open,
  onClose,
  onNavigate,
  onOpenIssue,
  recentIssues,
}: CommandPaletteProps) {
  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  const commands = useMemo<Command[]>(() => {
    const goTo = (section: AppSection) => () => {
      onNavigate(section);
      onClose();
    };

    const base: Command[] = [
      { id: "create-issue", group: "Create", label: "New issue", hint: "C", keywords: "new issue create", run: goTo("issues") },
      { id: "create-project", group: "Create", label: "New project", hint: "P", keywords: "new project create", run: goTo("projects") },
      { id: "start-sprint", group: "Create", label: "Start a sprint", hint: "S", keywords: "new sprint start", run: goTo("sprints") },
      { id: "go-dashboard", group: "Go to", label: "Dashboard", keywords: "dashboard home", run: goTo("dashboard") },
      { id: "go-board", group: "Go to", label: "Board", keywords: "board kanban", run: goTo("board") },
      { id: "go-issues", group: "Go to", label: "Issues", keywords: "issues list", run: goTo("issues") },
      { id: "go-projects", group: "Go to", label: "Projects", keywords: "projects", run: goTo("projects") },
      { id: "go-sprints", group: "Go to", label: "Sprints", keywords: "sprints cycles", run: goTo("sprints") },
      { id: "go-notifications", group: "Go to", label: "Notifications", keywords: "notifications inbox", run: goTo("notifications") },
      { id: "go-team", group: "Go to", label: "Team", keywords: "team members", run: goTo("team") },
      { id: "go-labels", group: "Go to", label: "Labels", keywords: "labels", run: goTo("labels") },
    ];

    const recents: Command[] = recentIssues.map((issue) => ({
      id: `issue-${issue.id}`,
      group: "Recent issues",
      label: issue.title,
      hint: issue.issue_key,
      keywords: `${issue.issue_key} ${issue.title}`,
      run: () => {
        onOpenIssue(issue.id);
        onClose();
      },
    }));

    return [...base, ...recents];
  }, [onNavigate, onOpenIssue, onClose, recentIssues]);

  const filtered = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) {
      return commands;
    }
    return commands.filter((command) =>
      `${command.label} ${command.keywords} ${command.group}`
        .toLowerCase()
        .includes(normalized),
    );
  }, [commands, query]);

  const indexById = useMemo(() => {
    const map = new Map<string, number>();
    filtered.forEach((command, index) => map.set(command.id, index));
    return map;
  }, [filtered]);

  useEffect(() => {
    if (!open) {
      return;
    }
    setQuery("");
    setActiveIndex(0);
    const focusTimer = window.setTimeout(() => inputRef.current?.focus(), 0);
    return () => window.clearTimeout(focusTimer);
  }, [open]);

  useEffect(() => {
    setActiveIndex(0);
  }, [query]);

  function onInputKeyDown(event: ReactKeyboardEvent<HTMLInputElement>) {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActiveIndex((index) => Math.min(index + 1, filtered.length - 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActiveIndex((index) => Math.max(index - 1, 0));
    } else if (event.key === "Enter") {
      event.preventDefault();
      filtered[activeIndex]?.run();
    }
  }

  const groups = useMemo(() => {
    const order: string[] = [];
    const map = new Map<string, Command[]>();
    for (const command of filtered) {
      if (!map.has(command.group)) {
        map.set(command.group, []);
        order.push(command.group);
      }
      map.get(command.group)!.push(command);
    }
    return order.map((group) => ({ group, items: map.get(group)! }));
  }, [filtered]);

  return (
    <Modal open={open} onClose={onClose} label="Command palette" panelClassName="kl-cmd">
      <div className="kl-cmd__search">
        <Icon icon={Search} size={18} />
        <input
          aria-label="Search or run a command"
          className="kl-cmd__input"
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={onInputKeyDown}
          placeholder="Search or run a command"
          ref={inputRef}
          value={query}
        />
        <kbd className="kl-cmd__esc">esc</kbd>
      </div>

      <div className="kl-cmd__results" role="listbox" aria-label="Commands">
        {filtered.length === 0 ? (
          <p className="kl-cmd__empty">No commands match “{query}”.</p>
        ) : null}
        {groups.map((group) => (
          <div className="kl-cmd__group" key={group.group}>
            <p className="kl-cmd__group-label">{group.group}</p>
            {group.items.map((command) => {
              const index = indexById.get(command.id) ?? 0;
              const isActive = index === activeIndex;
              return (
                <button
                  aria-selected={isActive}
                  className={
                    isActive ? "kl-cmd__item kl-cmd__item--active" : "kl-cmd__item"
                  }
                  key={command.id}
                  onClick={() => command.run()}
                  onMouseMove={() => setActiveIndex(index)}
                  role="option"
                  type="button"
                >
                  <span className="kl-cmd__item-label">{command.label}</span>
                  {command.hint ? (
                    <kbd className="kl-cmd__hint">{command.hint}</kbd>
                  ) : null}
                </button>
              );
            })}
          </div>
        ))}
      </div>

      <div className="kl-cmd__footer">
        <span>↑↓ navigate</span>
        <span>↵ open</span>
        <span className="kl-cmd__brand">Kelmio</span>
      </div>
    </Modal>
  );
}
