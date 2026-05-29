import { DASHBOARD_ACTION_COPY } from "../../lib/permissions";
import { type CurrentUser } from "../../lib/api-types";
import { type AppSection } from "../../lib/routing";

type DashboardSectionProps = {
  dueSoonIssuesCount: number;
  isActive: boolean;
  onNavigate: (section: AppSection) => void;
  openIssuesCount: number;
  overdueIssuesCount: number;
  projectsCount: number;
  role: CurrentUser["workspace"]["role"];
  teamMembersCount: number;
};

export function DashboardSection({
  dueSoonIssuesCount,
  isActive,
  onNavigate,
  openIssuesCount,
  overdueIssuesCount,
  projectsCount,
  role,
  teamMembersCount,
}: DashboardSectionProps) {
  return (
    <>
      <section
        className="summary-grid"
        aria-label="Project summary"
        hidden={!isActive}
      >
        <article>
          <span>Projects</span>
          <strong>{projectsCount}</strong>
        </article>
        <article>
          <span>Open issues</span>
          <strong>{openIssuesCount}</strong>
        </article>
        <article className={overdueIssuesCount > 0 ? "summary-alert" : undefined}>
          <span>Overdue</span>
          <strong>{overdueIssuesCount}</strong>
        </article>
        <article className={dueSoonIssuesCount > 0 ? "summary-warning" : undefined}>
          <span>Due soon</span>
          <strong>{dueSoonIssuesCount}</strong>
        </article>
        <article>
          <span>Team members</span>
          <strong>{teamMembersCount}</strong>
        </article>
      </section>

      <section
        className="dashboard-actions"
        aria-label="Dashboard quick actions"
        hidden={!isActive}
      >
        <article className="dashboard-action-card">
          <div>
            <p className="eyebrow">Planning</p>
            <h2>Projects</h2>
            <p>{DASHBOARD_ACTION_COPY.projects[role]}</p>
          </div>
          <button
            className="small-button"
            onClick={() => onNavigate("projects")}
            type="button"
          >
            Open projects
          </button>
        </article>

        <article className="dashboard-action-card">
          <div>
            <p className="eyebrow">Execution</p>
            <h2>Issues</h2>
            <p>Create tasks, inspect details, update status, comments, and labels.</p>
          </div>
          <button
            className="small-button"
            onClick={() => onNavigate("issues")}
            type="button"
          >
            Open issues
          </button>
        </article>

        <article className="dashboard-action-card">
          <div>
            <p className="eyebrow">Flow</p>
            <h2>Board</h2>
            <p>Move work between statuses with a kanban view backed by the same issue data.</p>
          </div>
          <button
            className="small-button"
            onClick={() => onNavigate("board")}
            type="button"
          >
            Open board
          </button>
        </article>

        <article className="dashboard-action-card">
          <div>
            <p className="eyebrow">Iterations</p>
            <h2>Sprints</h2>
            <p>Create lightweight sprint cycles, set goals, and run start/complete flow.</p>
          </div>
          <button
            className="small-button"
            onClick={() => onNavigate("sprints")}
            type="button"
          >
            Open sprints
          </button>
        </article>

        <article className="dashboard-action-card">
          <div>
            <p className="eyebrow">People</p>
            <h2>Team</h2>
            <p>{DASHBOARD_ACTION_COPY.team[role]}</p>
          </div>
          <button
            className="small-button"
            onClick={() => onNavigate("team")}
            type="button"
          >
            Open team
          </button>
        </article>

        <article className="dashboard-action-card">
          <div>
            <p className="eyebrow">Taxonomy</p>
            <h2>Labels</h2>
            <p>Keep issue categories clean so filtering and board scans stay useful.</p>
          </div>
          <button
            className="small-button"
            onClick={() => onNavigate("labels")}
            type="button"
          >
            Open labels
          </button>
        </article>
      </section>
    </>
  );
}
