import { appSections, type AppSection } from "../lib/routing";
import { type CurrentUser } from "../lib/api-types";

type AppSidebarProps = {
  activeSection: AppSection;
  onNavigate: (section: AppSection) => void;
};

export function AppSidebar({ activeSection, onNavigate }: AppSidebarProps) {
  return (
    <aside className="sidebar">
      <div className="brand">
        <span className="brand-mark">TT</span>
        <div>
          <strong>Team Task Tracker</strong>
          <span>Local workspace</span>
        </div>
      </div>

      <nav className="nav-list" aria-label="Main navigation">
        {appSections.map((section) => (
          <button
            aria-current={activeSection === section.id ? "page" : undefined}
            key={section.id}
            onClick={() => onNavigate(section.id)}
            type="button"
          >
            {section.title}
          </button>
        ))}
      </nav>
    </aside>
  );
}

type WorkspaceTopbarProps = {
  heading: string;
  isLoggingOut: boolean;
  onLogout: () => void;
  role: CurrentUser["workspace"]["role"];
  subtitle: string;
};

export function WorkspaceTopbar({
  heading,
  isLoggingOut,
  onLogout,
  role,
  subtitle,
}: WorkspaceTopbarProps) {
  return (
    <header className="topbar">
      <div>
        <p className="eyebrow">{subtitle}</p>
        <h1>{heading}</h1>
      </div>
      <div className="topbar-actions">
        <div className="status-pill">{role}</div>
        <button
          className="ghost-button"
          disabled={isLoggingOut}
          onClick={onLogout}
          type="button"
        >
          {isLoggingOut ? "Logging out..." : "Log out"}
        </button>
      </div>
    </header>
  );
}
