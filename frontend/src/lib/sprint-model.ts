import { type Sprint, type SprintStatus } from "./api-types.ts";

export const sprintStatusLabels: Record<SprintStatus, string> = {
  planned: "Planned",
  active: "Active",
  completed: "Completed",
};

export const sprintStatusOptions = [
  "planned",
  "active",
  "completed",
] satisfies SprintStatus[];

export function sprintMatchesFilters(
  sprint: Sprint,
  projectId: string,
  status: SprintStatus | "",
) {
  if (projectId && sprint.project_id !== projectId) {
    return false;
  }

  if (status && sprint.status !== status) {
    return false;
  }

  return true;
}

export function sprintDateRange(sprint: Pick<Sprint, "start_date" | "end_date">) {
  if (sprint.start_date && sprint.end_date) {
    return `${sprint.start_date} to ${sprint.end_date}`;
  }
  if (sprint.start_date) {
    return `Starts ${sprint.start_date}`;
  }
  if (sprint.end_date) {
    return `Ends ${sprint.end_date}`;
  }

  return "No dates planned";
}

export function sprintStatusCounts(sprints: Sprint[]) {
  return sprints.reduce(
    (counts, sprint) => ({
      ...counts,
      [sprint.status]: counts[sprint.status] + 1,
    }),
    {
      planned: 0,
      active: 0,
      completed: 0,
    } satisfies Record<SprintStatus, number>,
  );
}
