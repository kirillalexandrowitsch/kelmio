import { type Issue, type Sprint, type TeamMember } from "../../lib/api-types";
import { type AppSection } from "../../lib/routing";
import { memberDisplayName } from "../../lib/team-view";
import { isIssueDone } from "../../lib/issue-model";
import { EmptyState } from "../../ui";

type DashboardSectionProps = {
  activeSprint: Sprint | null;
  activeSprintIssues: Issue[];
  activeSprintError: string;
  isLoadingActiveSprint: boolean;
  dueSoonIssuesCount: number;
  overdueIssuesCount: number;
  openIssuesCount: number;
  projectsCount: number;
  teamMembersCount: number;
  teamMembers: TeamMember[];
  myWorkIssues: Issue[];
  displayName: string;
  isActive: boolean;
  onNavigate: (section: AppSection) => void;
  onOpenIssue: (issueId: string) => void;
};

const DAY_MS = 86400000;

function daysBetween(from: Date, to: Date) {
  return Math.ceil((to.getTime() - from.getTime()) / DAY_MS);
}

function initials(name: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

function shortDate(value: Date) {
  return value.toLocaleDateString("en-GB", { day: "numeric", month: "short" });
}

function dueLabel(dueDate: string | null, today: Date) {
  if (!dueDate) {
    return { text: "—", overdue: false };
  }
  const diff = daysBetween(today, new Date(dueDate));
  if (diff < 0) {
    return { text: `${Math.abs(diff)}d late`, overdue: true };
  }
  if (diff === 0) {
    return { text: "Today", overdue: false };
  }
  return { text: `${diff}d`, overdue: false };
}

function Burndown({ sprint }: { sprint: Sprint }) {
  const committed = sprint.points_total;
  const remaining = sprint.points_open;
  const startDate = sprint.start_date ? new Date(sprint.start_date) : null;
  const endDate = sprint.end_date ? new Date(sprint.end_date) : null;

  if (committed <= 0 || !startDate || !endDate) {
    return (
      <p className="kl-card__muted">
        Burndown appears once the active sprint has dates and estimated points.
      </p>
    );
  }

  const totalSpan = Math.max(1, daysBetween(startDate, endDate));
  const elapsed = Math.min(totalSpan, Math.max(0, daysBetween(startDate, new Date())));
  const elapsedFrac = elapsed / totalSpan;

  const padL = 40;
  const padR = 16;
  const padT = 16;
  const padB = 28;
  const width = 600;
  const height = 240;
  const plotW = width - padL - padR;
  const plotH = height - padT - padB;

  const x = (frac: number) => padL + frac * plotW;
  const y = (points: number) => padT + (1 - points / committed) * plotH;

  const todayX = x(elapsedFrac);
  const todayY = y(remaining);

  return (
    <div className="kl-burndown">
      <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label="Sprint burndown">
        {/* baseline */}
        <line x1={padL} y1={y(0)} x2={x(1)} y2={y(0)} className="kl-burndown__axis" />
        <line x1={padL} y1={padT} x2={padL} y2={y(0)} className="kl-burndown__axis" />
        {/* y ticks */}
        {[0, committed / 2, committed].map((tick) => (
          <text
            key={tick}
            x={padL - 8}
            y={y(tick) + 4}
            textAnchor="end"
            className="kl-burndown__tick"
          >
            {Math.round(tick)}
          </text>
        ))}
        {/* ideal guideline */}
        <line
          x1={x(0)}
          y1={y(committed)}
          x2={x(1)}
          y2={y(0)}
          className="kl-burndown__ideal"
        />
        {/* today guide + marker */}
        <line
          x1={todayX}
          y1={todayY}
          x2={todayX}
          y2={y(0)}
          className="kl-burndown__guide"
        />
        <circle cx={todayX} cy={todayY} r={5} className="kl-burndown__marker" />
        {/* x labels */}
        <text x={padL} y={height - 8} className="kl-burndown__tick">
          {shortDate(startDate)}
        </text>
        <text x={x(1)} y={height - 8} textAnchor="end" className="kl-burndown__tick">
          {shortDate(endDate)}
        </text>
      </svg>
      <div className="kl-burndown__legend">
        <span className="kl-burndown__legend-ideal">Ideal</span>
        <span className="kl-burndown__legend-remaining">
          Remaining today · {remaining} pts
        </span>
      </div>
    </div>
  );
}

export function DashboardSection({
  activeSprint,
  activeSprintIssues,
  activeSprintError,
  isLoadingActiveSprint,
  dueSoonIssuesCount,
  overdueIssuesCount,
  openIssuesCount,
  projectsCount,
  teamMembersCount,
  teamMembers,
  myWorkIssues,
  displayName,
  isActive,
  onNavigate,
  onOpenIssue,
}: DashboardSectionProps) {
  const today = new Date();
  const dateChip = today.toLocaleDateString("en-GB", {
    weekday: "short",
    day: "numeric",
    month: "short",
  });

  const workload = activeSprintIssues
    .filter((issue) => !isIssueDone(issue))
    .reduce<Record<string, number>>((totals, issue) => {
      const key = issue.assignee_id ?? "unassigned";
      return { ...totals, [key]: (totals[key] ?? 0) + issue.story_points };
    }, {});
  const workloadRows = Object.entries(workload)
    .sort(([, leftPoints], [, rightPoints]) => rightPoints - leftPoints)
    .slice(0, 6);
  const maxWorkload = Math.max(1, ...workloadRows.map(([, points]) => points));

  const committed = activeSprint?.points_total ?? 0;
  const done = activeSprint?.points_done ?? 0;
  const pctComplete = committed > 0 ? Math.round((done / committed) * 100) : 0;
  const endDate = activeSprint?.end_date ? new Date(activeSprint.end_date) : null;
  const daysRemaining = endDate ? Math.max(0, daysBetween(today, endDate)) : null;

  const cycleStatus = activeSprint
    ? [
        activeSprint.name,
        daysRemaining !== null ? `${daysRemaining} days remaining` : null,
        `${pctComplete}% complete`,
      ]
        .filter(Boolean)
        .join(" · ")
    : "No active sprint — start one to track delivery.";

  const stats = [
    { label: "Projects", value: projectsCount, tone: "" },
    { label: "Open", value: openIssuesCount, tone: "" },
    { label: "Overdue", value: overdueIssuesCount, tone: overdueIssuesCount > 0 ? "critical" : "" },
    { label: "Due soon", value: dueSoonIssuesCount, tone: dueSoonIssuesCount > 0 ? "overdue" : "" },
    { label: "Team", value: teamMembersCount, tone: "" },
  ];

  return (
    <section className="kl-dashboard" aria-label="Dashboard" hidden={!isActive}>
      <header className="kl-dashboard__intro">
        <div>
          <h1 className="kl-dashboard__greeting">Good to see you, {displayName}</h1>
          <p className="kl-dashboard__cycle">{cycleStatus}</p>
        </div>
        <span className="kl-dashboard__date">{dateChip}</span>
      </header>

      <div className="kl-stats">
        {stats.map((stat) => (
          <article
            className={stat.tone ? `kl-stat kl-stat--${stat.tone}` : "kl-stat"}
            key={stat.label}
          >
            <span className="kl-stat__label">{stat.label}</span>
            <strong className="kl-stat__value">{stat.value}</strong>
          </article>
        ))}
      </div>

      <div className="kl-dashboard__grid">
        <section className="kl-card" aria-label="Sprint burndown">
          <header className="kl-card__head">
            <h2>{activeSprint ? `${activeSprint.name} burndown` : "Burndown"}</h2>
            <span className="kl-card__meta">
              {committed > 0 ? `${done} / ${committed} pts done` : ""}
            </span>
          </header>
          {activeSprint ? (
            <Burndown sprint={activeSprint} />
          ) : (
            <p className="kl-card__muted">No active sprint to chart.</p>
          )}
        </section>

        <section className="kl-card" aria-label="Workload">
          <header className="kl-card__head">
            <h2>Workload · open points</h2>
          </header>
          {activeSprintError ? (
            <p className="kl-card__muted">{activeSprintError}</p>
          ) : isLoadingActiveSprint ? (
            <p className="kl-card__muted">Loading…</p>
          ) : workloadRows.length === 0 ? (
            <p className="kl-card__muted">No open sprint workload.</p>
          ) : (
            <ul className="kl-workload">
              {workloadRows.map(([memberId, points]) => {
                const name =
                  memberId === "unassigned"
                    ? "Unassigned"
                    : memberDisplayName(teamMembers, memberId);
                return (
                  <li className="kl-workload__row" key={memberId}>
                    <span
                      className={
                        memberId === "unassigned"
                          ? "kl-workload__avatar kl-workload__avatar--empty"
                          : "kl-workload__avatar"
                      }
                    >
                      {memberId === "unassigned" ? "?" : initials(name)}
                    </span>
                    <span className="kl-workload__name">{name}</span>
                    <span className="kl-workload__bar">
                      <span
                        className="kl-workload__fill"
                        style={{ width: `${(points / maxWorkload) * 100}%` }}
                      />
                    </span>
                    <strong className="kl-workload__points">{points}</strong>
                  </li>
                );
              })}
            </ul>
          )}
        </section>
      </div>

      <section className="kl-card" aria-label="My work">
        <header className="kl-card__head">
          <h2>My work</h2>
          <button
            className="kl-card__link"
            onClick={() => onNavigate("issues")}
            type="button"
          >
            View all →
          </button>
        </header>
        {myWorkIssues.length === 0 ? (
          <EmptyState
            title="Nothing assigned to you"
            description="Work assigned to you will show up here."
          />
        ) : (
          <ul className="kl-mywork">
            {myWorkIssues.map((issue) => {
              const due = dueLabel(issue.due_date, today);
              return (
                <li key={issue.id}>
                  <button
                    className="kl-mywork__row"
                    onClick={() => onOpenIssue(issue.id)}
                    type="button"
                  >
                    <span
                      aria-hidden="true"
                      className={`kl-pri kl-pri--${issue.priority}`}
                    />
                    <span className="kl-mywork__key">{issue.issue_key}</span>
                    <span className="kl-mywork__title">{issue.title}</span>
                    <span className="kl-mywork__meta">
                      <span
                        className={
                          due.overdue
                            ? "kl-mywork__due kl-mywork__due--late"
                            : "kl-mywork__due"
                        }
                      >
                        {due.text}
                      </span>
                      {issue.story_points > 0 ? (
                        <span className="kl-mywork__pts">{issue.story_points}</span>
                      ) : null}
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </section>
    </section>
  );
}
