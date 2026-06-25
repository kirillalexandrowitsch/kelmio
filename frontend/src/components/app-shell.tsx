import {
  Bell,
  Columns3,
  FolderKanban,
  LayoutDashboard,
  ListTodo,
  LogOut,
  Rocket,
  Search,
  Tag,
  UserRound,
  Users,
  type LucideIcon,
} from "lucide-react";

import { appSections, navGroups, type AppSection } from "../lib/routing";
import { Icon } from "../ui";

const sectionIcons: Record<AppSection, LucideIcon> = {
  dashboard: LayoutDashboard,
  projects: FolderKanban,
  issues: ListTodo,
  board: Columns3,
  sprints: Rocket,
  notifications: Bell,
  team: Users,
  labels: Tag,
  account: UserRound,
};

function sectionTitle(id: AppSection) {
  return appSections.find((section) => section.id === id)?.title ?? id;
}

function userInitials(name: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

type AppSidebarProps = {
  activeSection: AppSection;
  onNavigate: (section: AppSection) => void;
  onOpenCommandPalette: () => void;
  displayName: string;
  role: string;
  unreadNotificationsCount: number;
  isLoggingOut: boolean;
  onSignOut: () => void;
};

export function AppSidebar({
  activeSection,
  onNavigate,
  onOpenCommandPalette,
  displayName,
  role,
  unreadNotificationsCount,
  isLoggingOut,
  onSignOut,
}: AppSidebarProps) {
  return (
    <aside className="kl-sidebar">
      <div className="kl-sidebar__brand">
        <span className="kl-sidebar__mark">K</span>
        <span>Kelmio</span>
      </div>

      <button
        className="kl-sidebar__search"
        onClick={onOpenCommandPalette}
        type="button"
      >
        <Icon icon={Search} size={16} />
        <span>Search</span>
        <kbd className="kl-sidebar__kbd">⌘K</kbd>
      </button>

      <nav className="kl-sidebar__nav" aria-label="Main navigation">
        {navGroups.map((group) => (
          <div className="kl-sidebar__group" key={group.id}>
            <p className="kl-sidebar__group-label">{group.label}</p>
            {group.sections.map((section) => (
              <button
                aria-current={activeSection === section ? "page" : undefined}
                className="kl-sidebar__item"
                key={section}
                onClick={() => onNavigate(section)}
                type="button"
              >
                <Icon icon={sectionIcons[section]} size={18} />
                <span>{sectionTitle(section)}</span>
                {section === "notifications" && unreadNotificationsCount > 0 ? (
                  <span aria-hidden="true" className="kl-sidebar__count">
                    {unreadNotificationsCount}
                  </span>
                ) : null}
              </button>
            ))}
          </div>
        ))}
      </nav>

      <div className="kl-sidebar__footer">
        <button
          aria-current={activeSection === "account" ? "page" : undefined}
          className="kl-sidebar__user"
          onClick={() => onNavigate("account")}
          type="button"
        >
          <span className="kl-sidebar__avatar">{userInitials(displayName)}</span>
          <span className="kl-sidebar__user-text">
            <strong>{displayName}</strong>
            <span>{role}</span>
          </span>
        </button>
        <button
          className="kl-sidebar__signout"
          disabled={isLoggingOut}
          onClick={onSignOut}
          type="button"
        >
          <Icon icon={LogOut} size={16} />
          {isLoggingOut ? "Signing out..." : "Sign out"}
        </button>
      </div>
    </aside>
  );
}

type WorkspaceTopbarProps = {
  heading: string;
  subtitle: string;
};

export function WorkspaceTopbar({ heading, subtitle }: WorkspaceTopbarProps) {
  return (
    <header className="kl-topbar">
      <p className="kl-topbar__eyebrow">{subtitle}</p>
      <h1 className="kl-topbar__heading">{heading}</h1>
    </header>
  );
}
