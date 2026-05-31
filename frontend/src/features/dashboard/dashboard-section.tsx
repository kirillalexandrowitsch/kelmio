import { DASHBOARD_ACTION_COPY } from "../../lib/permissions";
import {
  type CurrentUser,
  type Issue,
  type Sprint,
  type TeamMember,
} from "../../lib/api-types";
import { type AppSection } from "../../lib/routing";
import { memberDisplayName } from "../../lib/team-view";

type DashboardSectionProps = {
  activeSprint: Sprint | null;
  activeSprintIssues: Issue[];
  activeSprintError: string;
  dueSoonIssuesCount: number;
  isActive: boolean;
  isLoadingActiveSprint: boolean;
  onNavigate: (section: AppSection) => void;
  openIssuesCount: number;
  overdueIssuesCount: number;
  projectsCount: number;
  role: CurrentUser["workspace"]["role"];
  teamMembers: TeamMember[];
  teamMembersCount: number;
};

export function DashboardSection({
  activeSprint,
  activeSprintIssues,
  activeSprintError,
  dueSoonIssuesCount,
  isActive,
  isLoadingActiveSprint,
  onNavigate,
  openIssuesCount,
  overdueIssuesCount,
  projectsCount,
  role,
  teamMembers,
  teamMembersCount,
}: DashboardSectionProps) {
  const workload = activeSprintIssues
    .filter((issue) => issue.status !== "done")
    .reduce<Record<string, number>>((totals, issue) => {
      const key = issue.assignee_id ?? "unassigned";
      return {
        ...totals,
        [key]: (totals[key] ?? 0) + issue.story_points,
      };
    }, {});
  const workloadRows = Object.entries(workload)
    .sort(([, leftPoints], [, rightPoints]) => rightPoints - leftPoints)
    .slice(0, 3);
  const activeSprintIssueCount = activeSprintIssues.length;
  const activeSprintDoneCount = activeSprintIssues.filter(
    (issue) => issue.status === "done",
  ).length;
  const activeSprintPointsTotal = activeSprintIssues.reduce(
    (sum, issue) => sum + issue.story_points,
    0,
  );
  const activeSprintPointsDone = activeSprintIssues
    .filter((issue) => issue.status === "done")
    .reduce((sum, issue) => sum + issue.story_points, 0);
  const activeSprintPointsOpen = activeSprintPointsTotal - activeSprintPointsDone;
  const sprintProgress =
    activeSprint && activeSprintIssueCount > 0
      ? Math.round((activeSprintDoneCount / activeSprintIssueCount) * 100)
      : 0;

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
        className="dashboard-sprint-summary"
        aria-label="Sprint progress summary"
        hidden={!isActive}
      >
        <article>
          <span>Active sprint</span>
          <strong>{activeSprint ? activeSprint.name : "No active sprint"}</strong>
          <p>
            {activeSprint
              ? `${sprintProgress}% complete · ${activeSprintDoneCount}/${activeSprintIssueCount} issues`
              : "Start a sprint to track progress here."}
          </p>
        </article>

        <article>
          <span>Sprint points</span>
          <strong>
            {activeSprint ? `${activeSprintPointsDone}/${activeSprintPointsTotal}` : "0/0"}
          </strong>
          <p>
            {activeSprint
              ? `${activeSprintPointsOpen} open points`
              : "No active sprint points yet."}
          </p>
        </article>

        <article>
          <span>Workload</span>
          <strong>{isLoadingActiveSprint ? "Loading" : "Open points"}</strong>
          {activeSprintError ? <p>{activeSprintError}</p> : null}
          {!activeSprintError && workloadRows.length > 0 ? (
            <ul>
              {workloadRows.map(([memberId, points]) => (
                <li key={memberId}>
                  <span>
                    {memberId === "unassigned"
                      ? "Unassigned"
                      : memberDisplayName(teamMembers, memberId)}
                  </span>
                  <strong>{points}</strong>
                </li>
              ))}
            </ul>
          ) : null}
          {!activeSprintError && workloadRows.length === 0 ? (
            <p>No open sprint workload.</p>
          ) : null}
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
